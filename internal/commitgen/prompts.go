package commitgen

import (
	"strings"

	"github.com/N0ViP/commitgen-project/prompts"
)

const maxDiffChars = 8000

func titlePrompt(diff string, files []string) string {
	return strings.NewReplacer(
		"{{FILES}}", fileList(files),
		"{{DIFF}}", truncateDiff(diff),
	).Replace(prompts.Title)
}

func descriptionPrompt(diff string, files []string, notes, title string) string {
	if notes == "" {
		notes = "None"
	}
	return strings.NewReplacer(
		"{{TITLE}}", title,
		"{{FILES}}", fileList(files),
		"{{NOTES}}", notes,
		"{{DIFF}}", truncateDiff(diff),
	).Replace(prompts.Description)
}

func fileList(files []string) string {
	if len(files) == 0 {
		return "multiple files"
	}
	return strings.Join(files, ", ")
}

func truncateDiff(diff string) string {
	return truncateToLimit(diff, maxDiffChars, "\n\n[... diff truncated for brevity ...]")
}

func cleanFirstLine(text string) string {
	text = strings.TrimSpace(strings.SplitN(text, "```", 2)[0])
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}
