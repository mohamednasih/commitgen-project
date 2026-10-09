package commitgen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type GeminiClient struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTP       *http.Client
	MaxRetries int
	RetryDelay time.Duration
	OnRetry    func(failedAttempt, totalAttempts int, err error, nextDelay time.Duration)
}

type generateRequest struct {
	Contents []content `json:"contents"`
}

type generateResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	return &GeminiClient{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    geminiBaseURL,
		HTTP:       &http.Client{Timeout: 60 * time.Second},
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

func (c *GeminiClient) Generate(ctx context.Context, prompt string) (string, error) {
	if c.APIKey == "" {
		return "", nil
	}
	body, err := json.Marshal(generateRequest{Contents: []content{{Parts: []part{{Text: prompt}}}}})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}
	endpoint := strings.TrimRight(c.BaseURL, "/") + "/models/" + url.PathEscape(c.Model) + ":generateContent"
	totalAttempts := c.MaxRetries + 1
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		result, retryable, err := c.generateOnce(ctx, endpoint, body)
		if err == nil {
			return result, nil
		}
		if !retryable || attempt == totalAttempts {
			return "", err
		}

		delay := c.RetryDelay * time.Duration(1<<(attempt-1))
		if c.OnRetry != nil {
			c.OnRetry(attempt, totalAttempts, err, delay)
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
		}
	}
	return "", errors.New("Gemini request failed")
}

func (c *GeminiClient) generateOnce(ctx context.Context, endpoint string, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", ctx.Err() == nil, fmt.Errorf("call Gemini: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		retryable := resp.StatusCode == http.StatusRequestTimeout ||
			resp.StatusCode == http.StatusTooEarly ||
			resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode >= http.StatusInternalServerError
		return "", retryable, fmt.Errorf("Gemini returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, fmt.Errorf("decode Gemini response: %w", err)
	}
	if len(result.Candidates) == 0 {
		return "", false, nil
	}
	var text []string
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			text = append(text, part.Text)
		}
	}
	return strings.TrimSpace(strings.Join(text, "")), false, nil
}
