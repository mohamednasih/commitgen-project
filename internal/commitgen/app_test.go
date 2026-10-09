package commitgen

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadNotesStopsAtDelimiterAndLeavesInputOpen(t *testing.T) {
	input := strings.NewReader("focus on retries\nmention the Docker path\nEOF\ny\n")
	var output bytes.Buffer
	app := New(input, &output, &output)

	notes, err := app.readNotes()
	if err != nil {
		t.Fatal(err)
	}
	if notes != "focus on retries\nmention the Docker path" {
		t.Errorf("notes = %q", notes)
	}
	answer, err := app.ask("Continue? ", "yn")
	if err != nil {
		t.Fatal(err)
	}
	if answer != 'y' {
		t.Errorf("answer = %q, want y", answer)
	}
}

func TestReadNotesRequiresDelimiter(t *testing.T) {
	app := New(strings.NewReader("unfinished notes"), &bytes.Buffer{}, &bytes.Buffer{})
	if _, err := app.readNotes(); err == nil {
		t.Fatal("readNotes() error = nil, want missing delimiter error")
	}
}

func TestDefaultModel(t *testing.T) {
	t.Setenv("COMMITGEN_MODEL", "")
	app := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if app.ai.Model != "gemini-3.1-flash-lite" {
		t.Errorf("default model = %q, want gemini-3.1-flash-lite", app.ai.Model)
	}
}

func TestConfiguredAPIKeyUsesEmbeddedFallback(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	previous := embeddedAPIKey
	embeddedAPIKey = "embedded-key"
	t.Cleanup(func() { embeddedAPIKey = previous })
	if got := configuredAPIKey(); got != "embedded-key" {
		t.Errorf("configuredAPIKey() = %q, want embedded-key", got)
	}
}

func TestConfiguredAPIKeyPrefersEnvironment(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "runtime-key")
	previous := embeddedAPIKey
	embeddedAPIKey = "embedded-key"
	t.Cleanup(func() { embeddedAPIKey = previous })
	if got := configuredAPIKey(); got != "runtime-key" {
		t.Errorf("configuredAPIKey() = %q, want runtime-key", got)
	}
}
