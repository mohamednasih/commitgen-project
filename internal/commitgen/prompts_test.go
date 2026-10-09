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
