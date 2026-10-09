package commitgen

import (
	"strings"
	"testing"
)

func TestSampleDiffContextIncludesSummaryAndEveryFile(t *testing.T) {
	files := []stagedFileDiff{
		{name: "cmd/main.go", content: strings.Repeat("A", 500)},
		{name: "internal/git.go", content: strings.Repeat("B", 500)},
		{name: "README.md", content: strings.Repeat("C", 500)},
	}
	context := sampleDiffContext("3 files changed, 20 insertions(+)", files, 500)

	if len([]rune(context)) > 500 {
		t.Fatalf("context length = %d, want at most 500", len([]rune(context)))
	}
	for _, want := range []string{"Diff summary:", "3 files changed", "cmd/main.go", "internal/git.go", "README.md", "AAA", "BBB", "CCC"} {
		if !strings.Contains(context, want) {
			t.Errorf("context does not contain %q", want)
		}
	}
}

func TestSampleDiffContextRedistributesUnusedBudget(t *testing.T) {
	files := []stagedFileDiff{
		{name: "small.go", content: "short"},
		{name: "large.go", content: strings.Repeat("L", 1000)},
	}
	context := sampleDiffContext("2 files changed", files, 400)

	if strings.Count(context, "L") < 250 {
		t.Errorf("large file did not receive unused context budget")
	}
	if len([]rune(context)) > 400 {
		t.Fatalf("context length = %d, want at most 400", len([]rune(context)))
	}
}

func TestSampleDiffContextCountsUnicodeCharacters(t *testing.T) {
	files := []stagedFileDiff{{name: "emoji.txt", content: strings.Repeat("界", 500)}}
	context := sampleDiffContext("1 file changed", files, 200)
	if len([]rune(context)) > 200 {
		t.Fatalf("context length = %d, want at most 200", len([]rune(context)))
	}
}
