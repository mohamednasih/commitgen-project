package commitgen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
