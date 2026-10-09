package commitgen

import (
	"strings"
	"testing"
)

func TestCleanFirstLine(t *testing.T) {
	got := cleanFirstLine("\n\nfeat(cli): port to Go\nextra")
	if got != "feat(cli): port to Go" {
		t.Errorf("cleanFirstLine() = %q", got)
	}
}

func TestTruncateDiff(t *testing.T) {
	diff := strings.Repeat("x", maxDiffChars+1)
	got := truncateDiff(diff)
	if !strings.HasSuffix(got, "[... diff truncated for brevity ...]") {
		t.Errorf("diff was not truncated: %q", got[len(got)-50:])
	}
}

func TestFileList(t *testing.T) {
	if got := fileList([]string{"one.go", "two.go"}); got != "one.go, two.go" {
		t.Errorf("fileList() = %q", got)
	}
}

func TestTitlePromptReplacesExternalTemplateVariables(t *testing.T) {
	got := titlePrompt("+new line", []string{"one.go", "two.go"})
	for _, want := range []string{"one.go, two.go", "+new line"} {
		if !strings.Contains(got, want) {
			t.Errorf("titlePrompt() does not contain %q", want)
		}
	}
	if strings.Contains(got, "{{FILES}}") || strings.Contains(got, "{{DIFF}}") {
		t.Error("titlePrompt() contains an unreplaced required variable")
	}
}

func TestDescriptionPromptReplacesExternalTemplateVariables(t *testing.T) {
	got := descriptionPrompt("+new line", []string{"one.go"}, "focus on tests", "test(cli): add coverage")
	for _, want := range []string{"one.go", "+new line", "focus on tests", "test(cli): add coverage"} {
		if !strings.Contains(got, want) {
			t.Errorf("descriptionPrompt() does not contain %q", want)
		}
	}
	for _, variable := range []string{"{{TITLE}}", "{{FILES}}", "{{NOTES}}", "{{DIFF}}"} {
		if strings.Contains(got, variable) {
			t.Errorf("descriptionPrompt() contains unreplaced variable %s", variable)
		}
	}
}
