package commitgen

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func openInEditor(initial, hint string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "nano"
		}
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "", fmt.Errorf("EDITOR is empty")
	}

	file, err := os.CreateTemp("", "commitgen-*.tmp")
	if err != nil {
		return "", err
	}
	path := file.Name()
	defer os.Remove(path)

	content := strings.TrimRight(hint+initial, "\n") + "\n"
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	edited, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var kept []string
	for _, line := range strings.Split(string(edited), "\n") {
		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			kept = append(kept, line)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), nil
}
