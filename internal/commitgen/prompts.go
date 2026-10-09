package commitgen

import (
	"fmt"
	"strings"
)

const maxDiffChars = 8000

func titlePrompt(diff string, files []string) string {
	return fmt.Sprintf(`
You are a senior software engineer generating a **Conventional Commit title** based on code changes.

Guidelines:
- Output ONLY ONE line — no explanations, no extra text.
- Format: type(scope): summary
- Use one of these types: feat, fix, refactor, style, docs, test, chore, perf, ci, build, revert.
- The scope should be concise and relevant (e.g., a filename, folder, or feature).
- The summary should describe what changed, using imperative mood ("add", "update", "fix", "remove").
- Keep the entire title under 70 characters.
- Do NOT include punctuation at the end, emojis, code snippets, backticks, or quotes around the output.

Example output:
feat(auth): add JWT token refresh on session expiry

Context:
- Changed files: %s
- Git diff:
%s

Return only the final commit title as plain text.
`, fileList(files), truncateDiff(diff))
}

func descriptionPrompt(diff string, files []string, notes, title string) string {
	if notes == "" {
		notes = "None"
	}
	return fmt.Sprintf(`
You are a professional assistant writing a **Conventional Commit description** that complements this title:
"%s"

Guidelines:
- Expand on the title — explain what changed and why.
- Use bullet points (starting with "- ") for clarity and structure.
- Mention affected files or modules if relevant.
- Do NOT repeat the title verbatim; provide supporting detail instead.
- Keep a professional and concise tone.
- Do NOT include markdown headers, code blocks, backticks, or commit hashes.
- Do NOT wrap your output in markdown formatting of any kind.
- If user notes exist, use them to enrich the context.

Context:
- Changed files: %s
- User notes: %s
- Git diff:
%s

Return only the formatted bullet-point description as plain text.
`, title, fileList(files), notes, truncateDiff(diff))
}

func fileList(files []string) string {
	if len(files) == 0 {
		return "multiple files"
	}
	return strings.Join(files, ", ")
}

func truncateDiff(diff string) string {
	runes := []rune(diff)
	if len(runes) <= maxDiffChars {
		return diff
	}
	return string(runes[:maxDiffChars]) + "\n\n[... diff truncated for brevity ...]"
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
