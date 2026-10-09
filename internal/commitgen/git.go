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

func stagedDiff(ctx context.Context) (string, error) {
	output, err := gitOutput(ctx, "diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("get staged diff: %w", err)
	}
	if output == "" {
		return "", errors.New("no staged changes found; stage changes using 'git add .' first")
	}
	return output, nil
}

func stagedFiles(ctx context.Context) []string {
	output, err := gitOutput(ctx, "diff", "--cached", "--name-only")
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files
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
