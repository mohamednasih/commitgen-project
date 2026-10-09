package commitgen

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const outputWidth = 60

var errInputClosed = errors.New("input closed")

type App struct {
	out   io.Writer
	input *bufio.Reader
	ai    *GeminiClient
}

func New(in io.Reader, out, errOut io.Writer) *App {
	client := NewGeminiClient(
		os.Getenv("GEMINI_API_KEY"),
		envOrDefault("COMMITGEN_MODEL", "gemini-3.1-flash-lite"),
	)
	client.OnRetry = func(failedAttempt, totalAttempts int, err error, nextDelay time.Duration) {
		fmt.Fprintf(errOut, "\n[commitgen] AI request failed (attempt %d/%d): %v\n", failedAttempt, totalAttempts, err)
		fmt.Fprintf(errOut, "[commitgen] Retrying in %s (%d retries remaining)...\n", nextDelay, totalAttempts-failedAttempt)
	}
	return &App{
		out:   out,
		input: bufio.NewReader(in),
		ai:    client,
	}
}

func (a *App) Run(ctx context.Context, interactive bool) error {
	if a.ai.APIKey == "" {
		return errors.New("Gemini API key not found; set GEMINI_API_KEY")
	}
	if err := ensureRepository(ctx); err != nil {
		return err
	}

	files, err := stagedFiles(ctx)
	if err != nil {
		return err
	}
	diff, err := stagedDiffContext(ctx, files)
	if err != nil {
		return err
	}
	if !interactive {
		return a.runAutomatic(ctx, diff, files)
	}

	fmt.Fprintln(a.out, "\nGenerating initial commit suggestion, please wait...")
	title, err := a.generateTitle(ctx, diff, files)
	if err != nil {
		return fmt.Errorf("generate commit title: %w", err)
	}
	for {
		fmt.Fprintf(a.out, "\n--- Proposed Commit Title ---\n\n%s\n\n%s\n\n", title, strings.Repeat("-", outputWidth))
		choice, askErr := a.ask("Accept (y), Regenerate (n), Edit (e), or Quit (q)? ", "yneq")
		if askErr != nil {
			return a.finishOnInputError(askErr)
		}
		switch choice {
		case 'y':
			goto description
		case 'n':
			title, err = a.generateTitle(ctx, diff, files)
			if err != nil {
				return fmt.Errorf("regenerate commit title: %w", err)
			}
		case 'e':
			edited, editErr := openInEditor(title, "# Edit your commit title below. Lines starting with # will be ignored.\n\n")
			if editErr != nil {
				return fmt.Errorf("edit commit title: %w", editErr)
			}
			if cleaned := cleanFirstLine(edited); cleaned != "" {
				title = cleaned
			} else {
				fmt.Fprintln(a.out, "Empty title. Keeping previous title.")
			}
		case 'q':
			return a.cancel("Commit process cancelled by user.")
		}
	}

description:
	wantDescription, err := a.ask("\nDo you want a description? (y/n/q): ", "ynq")
	if err != nil {
		return a.finishOnInputError(err)
	}
	if wantDescription == 'q' {
		return a.cancel("Commit process cancelled by user.")
	}

	description := ""
	if wantDescription == 'y' {
		fmt.Fprintln(a.out, "\n(Optional) Add keywords/notes for the description.")
		fmt.Fprintln(a.out, "Type EOF on its own line to finish.")
		notes, readErr := a.readNotes()
		if readErr != nil {
			return fmt.Errorf("read description notes: %w", readErr)
		}
		fmt.Fprintln(a.out, "\nNotes captured. Please wait...")

		for {
			description, err = a.generateDescription(ctx, diff, files, notes, title)
			if err != nil {
				return fmt.Errorf("generate commit description: %w", err)
			}
			fmt.Fprintf(a.out, "\n--- Proposed Description ---\n\n%s\n\n%s\n\n", description, strings.Repeat("-", outputWidth))
			choice, askErr := a.ask("Accept (y), Regenerate (n), Edit (e), or Skip (s)? ", "ynes")
			if askErr != nil {
				return a.finishOnInputError(askErr)
			}
			switch choice {
			case 'y':
				goto commit
			case 'n':
				continue
			case 'e':
				edited, editErr := openInEditor(description, "# Edit your commit description below. Lines starting with # will be ignored.\n\n")
				if editErr != nil {
					return fmt.Errorf("edit commit description: %w", editErr)
				}
				description = strings.TrimSpace(edited)
				if description == "" {
					fmt.Fprintln(a.out, "Empty description. Skipping.")
					goto commit
				}
				fmt.Fprintf(a.out, "\n--- Edited Description ---\n\n%s\n\n%s\n\n", description, strings.Repeat("-", outputWidth))
				secondChoice, secondErr := a.ask("Accept (y), Edit again (e), or Skip (s)? ", "yes")
				if secondErr != nil {
					return a.finishOnInputError(secondErr)
				}
				if secondChoice == 'y' {
					goto commit
				}
				if secondChoice == 's' {
					description = ""
					goto commit
				}
			case 's':
				description = ""
				goto commit
			}
		}
	}

commit:
	if err := gitCommit(ctx, title, description); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "\nCommit successful!")
	return nil
}

func (a *App) runAutomatic(ctx context.Context, diff string, files []string) error {
	fmt.Fprintln(a.out, "\nGenerating commit title and description, please wait...")
	title, err := a.generateTitle(ctx, diff, files)
	if err != nil {
		return fmt.Errorf("generate commit title: %w", err)
	}
	description, err := a.generateDescription(ctx, diff, files, "", title)
	if err != nil {
		return fmt.Errorf("generate commit description: %w", err)
	}

	fmt.Fprintf(a.out, "\n--- Generated Commit ---\n\n%s\n\n%s\n\n%s\n", title, description, strings.Repeat("-", outputWidth))
	if err := gitCommit(ctx, title, description); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "\nCommit successful!")
	return nil
}

func (a *App) readNotes() (string, error) {
	var lines []string
	for {
		line, err := a.input.ReadString('\n')
		if strings.TrimSpace(line) == "EOF" {
			return strings.TrimSpace(strings.Join(lines, "\n")), nil
		}
		if line != "" {
			lines = append(lines, strings.TrimRight(line, "\r\n"))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", errors.New("input closed before the EOF delimiter")
			}
			return "", err
		}
	}
}

func (a *App) ask(prompt, allowed string) (byte, error) {
	for {
		fmt.Fprint(a.out, prompt)
		answer, err := a.input.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "" && strings.ContainsRune(allowed, rune(answer[0])) {
			return answer[0], nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0, errInputClosed
			}
			return 0, err
		}
		fmt.Fprintf(a.out, "Invalid choice. Allowed: %s.\n", strings.Join(strings.Split(allowed, ""), ", "))
	}
}

func (a *App) generateTitle(ctx context.Context, diff string, files []string) (string, error) {
	result, err := a.ai.Generate(ctx, titlePrompt(diff, files))
	if err != nil {
		return "", err
	}
	if title := cleanFirstLine(result); title != "" {
		return title, nil
	}
	return "", errors.New("Gemini returned an empty commit title")
}

func (a *App) generateDescription(ctx context.Context, diff string, files []string, notes, title string) (string, error) {
	result, err := a.ai.Generate(ctx, descriptionPrompt(diff, files, notes, title))
	if err != nil {
		return "", err
	}
	result = strings.TrimSpace(strings.SplitN(result, "```", 2)[0])
	if result == "" {
		return "", errors.New("Gemini returned an empty commit description")
	}
	return result, nil
}

func (a *App) cancel(message string) error {
	fmt.Fprintf(a.out, "\n%s\n", message)
	return nil
}

func (a *App) finishOnInputError(err error) error {
	if errors.Is(err, errInputClosed) {
		return a.cancel("Input closed. Exiting.")
	}
	return err
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
