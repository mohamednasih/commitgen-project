package commitgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type GeminiClient struct {
	APIKey  string
	Model   string
	BaseURL string
	HTTP    *http.Client
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
		APIKey:  apiKey,
		Model:   model,
		BaseURL: geminiBaseURL,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("call Gemini: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		return "", fmt.Errorf("Gemini returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Gemini response: %w", err)
	}
	if len(result.Candidates) == 0 {
		return "", nil
	}
	var text []string
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			text = append(text, part.Text)
		}
	}
	return strings.TrimSpace(strings.Join(text, "")), nil
}
