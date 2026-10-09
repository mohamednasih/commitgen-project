package commitgen

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func ensureRepository(ctx context.Context) error {
	output, err := gitOutput(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return fmt.Errorf("inspect Git repository: %w", err)
	}
	if strings.TrimSpace(output) != "true" {
		return errors.New("not a Git repository")
	}
	return nil
}

type stagedFileDiff struct {
	name    string
	content string
}

func stagedFiles(ctx context.Context) ([]string, error) {
	output, err := gitOutput(ctx, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return nil, fmt.Errorf("get staged files: %w", err)
	}
	if output == "" {
		return nil, errors.New("no staged changes found; stage changes using 'git add .' first")
	}
	var files []string
	for _, name := range strings.Split(output, "\x00") {
		if name != "" {
			files = append(files, name)
		}
	}
	return files, nil
}

func stagedDiffContext(ctx context.Context, files []string) (string, error) {
	stat, err := gitOutput(ctx, "diff", "--cached", "--stat")
	if err != nil {
		return "", fmt.Errorf("get staged diff summary: %w", err)
	}
	diffs := make([]stagedFileDiff, 0, len(files))
	for _, name := range files {
		content, err := gitOutput(ctx, "diff", "--cached", "--no-ext-diff", "--unified=3", "--", name)
		if err != nil {
			return "", fmt.Errorf("get staged diff for %q: %w", name, err)
		}
		diffs = append(diffs, stagedFileDiff{name: name, content: content})
	}
	return sampleDiffContext(stat, diffs, maxDiffChars), nil
}

func sampleDiffContext(stat string, files []stagedFileDiff, limit int) string {
	if limit <= 0 {
		return ""
	}
	statBudget := minInt(1200, limit/5)
	stat = truncateToLimit(strings.TrimSpace(stat), statBudget, "\n[... summary truncated ...]")
	prefix := "Diff summary:\n" + stat
	if len(files) == 0 {
		return truncateToLimit(prefix, limit, "")
	}

	headings := make([]string, len(files))
	headingSize := 0
	for i, file := range files {
		name := truncateToLimit(file.name, 120, "...")
		headings[i] = "\n\n--- " + name + " ---\n"
		headingSize += len([]rune(headings[i]))
	}
	contentBudget := limit - len([]rune(prefix)) - headingSize
	if contentBudget < 0 {
		var onlyHeadings strings.Builder
		onlyHeadings.WriteString(prefix)
		for _, heading := range headings {
			onlyHeadings.WriteString(heading)
		}
		return truncateToLimit(onlyHeadings.String(), limit, "")
	}

	allocations := fairAllocations(files, contentBudget)
	var result strings.Builder
	result.WriteString(prefix)
	for i, file := range files {
		result.WriteString(headings[i])
		result.WriteString(truncateToLimit(file.content, allocations[i], "\n[... file diff truncated ...]"))
	}
	return result.String()
}

func fairAllocations(files []stagedFileDiff, budget int) []int {
	allocations := make([]int, len(files))
	remaining := budget
	active := make([]int, len(files))
	for i := range files {
		active[i] = i
	}
	for len(active) > 0 && remaining > 0 {
		share := remaining / len(active)
		if share == 0 {
			for _, index := range active[:remaining] {
				allocations[index]++
			}
			break
		}
		var next []int
		allocatedShortFile := false
		for _, index := range active {
			size := len([]rune(files[index].content))
			if size <= share {
				allocations[index] = size
				remaining -= size
				allocatedShortFile = true
			} else {
				next = append(next, index)
			}
		}
		if !allocatedShortFile {
			for _, index := range active {
				allocations[index] += share
				remaining -= share
			}
			for _, index := range active {
				if remaining == 0 {
					break
				}
				allocations[index]++
				remaining--
			}
			break
		}
		active = next
	}
	return allocations
}

func truncateToLimit(value string, limit int, marker string) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	markerRunes := []rune(marker)
	if len(markerRunes) >= limit {
		return string(runes[:limit])
	}
	return string(runes[:limit-len(markerRunes)]) + marker

}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func gitCommit(ctx context.Context, title, description string) error {
	args := []string{"commit", "-m", title}
	if strings.TrimSpace(description) != "" {
		args = append(args, "-m", description)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("git commit failed: %s", message)
	} else if len(output) > 0 {
		fmt.Print(string(output))
	}
	return nil
}

func gitOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return string(output), nil
}
