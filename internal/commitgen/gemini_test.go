package commitgen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGeminiGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-api-key"); got != "secret" {
			t.Errorf("API key header = %q", got)
		}
		if r.URL.Path != "/models/gemini-test:generateContent" {
			t.Errorf("path = %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("empty request body")
		}
		fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"feat(cli): port to Go"}]}}]}`)
	}))
	defer server.Close()

	client := NewGeminiClient("secret", "gemini-test")
	client.BaseURL = server.URL
	got, err := client.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "feat(cli): port to Go" {
		t.Errorf("result = %q", got)
	}
}

func TestGeminiGenerateRetriesTransientErrorsThreeTimes(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests <= 3 {
			http.Error(w, `{"error":{"message":"high demand"}}`, http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"fix(ai): retry transient failures"}]}}]}`)
	}))
	defer server.Close()

	retryNotices := 0
	client := NewGeminiClient("secret", "gemini-test")
	client.BaseURL = server.URL
	client.RetryDelay = 0
	client.OnRetry = func(_, _ int, _ error, _ time.Duration) {
		retryNotices++
	}

	got, err := client.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "fix(ai): retry transient failures" {
		t.Errorf("result = %q", got)
	}
	if requests != 4 {
		t.Errorf("requests = %d, want 4", requests)
	}
	if retryNotices != 3 {
		t.Errorf("retry notices = %d, want 3", retryNotices)
	}
}

func TestGeminiGenerateDoesNotRetryClientErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		http.Error(w, `{"error":{"message":"invalid key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewGeminiClient("bad-secret", "gemini-test")
	client.BaseURL = server.URL
	client.RetryDelay = 0
	if _, err := client.Generate(context.Background(), "prompt"); err == nil {
		t.Fatal("Generate() error = nil, want an authentication error")
	}
	if requests != 1 {
		t.Errorf("requests = %d, want 1", requests)
	}
}

func TestGeminiGenerateReturnsErrorAfterRetriesAreExhausted(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		http.Error(w, `{"error":{"message":"high demand"}}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewGeminiClient("secret", "gemini-test")
	client.BaseURL = server.URL
	client.RetryDelay = 0
	if _, err := client.Generate(context.Background(), "prompt"); err == nil {
		t.Fatal("Generate() error = nil after retries were exhausted")
	}
	if requests != 4 {
		t.Errorf("requests = %d, want 4", requests)
	}
}

func TestDescriptionGenerationRejectsEmptyAIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"candidates":[]}`)
	}))
	defer server.Close()

	client := NewGeminiClient("secret", "gemini-test")
	client.BaseURL = server.URL
	app := &App{ai: client}
	if _, err := app.generateDescription(context.Background(), "diff", []string{"file.go"}, "", "feat: test"); err == nil {
		t.Fatal("generateDescription() error = nil for an empty AI response")
	}
}
