package commitgen

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const bannerWidth = 60

var errInputClosed = errors.New("input closed")

type App struct {
	in     io.Reader
	out    io.Writer
	errOut io.Writer
	input  *bufio.Reader
	ai     *GeminiClient
}

func New(in io.Reader, out, errOut io.Writer) *App {
	return &App{
		in:     in,
		out:    out,
		errOut: errOut,
		input:  bufio.NewReader(in),
		ai: NewGeminiClient(
			os.Getenv("GEMINI_API_KEY"),
			envOrDefault("COMMITGEN_MODEL", "gemini-3.8-flash"),
		),
	}
}

func (a *App) Run(ctx context.Context) error {
	PrintHeader(a.out)
	if a.ai.APIKey == "" {
		return errors.New("Gemini API key not found; set GEMINI_API_KEY")
	}
	if err := ensureRepository(ctx); err != nil {
		return err
	}

	diff, err := stagedDiff(ctx)
	if err != nil {
		return err
	}
	files := stagedFiles(ctx)

	fmt.Fprintln(a.out, "\nGenerating initial commit suggestion, please wait...")
	title := a.generateTitle(ctx, diff, files)
	for {
		fmt.Fprintf(a.out, "\n--- Proposed Commit Title ---\n\n%s\n\n%s\n\n", title, strings.Repeat("-", bannerWidth))
		choice, askErr := a.ask("Accept (y), Regenerate (n), Edit (e), or Quit (q)? ", "yneq")
		if askErr != nil {
			return a.finishOnInputError(askErr)
		}
		switch choice {
		case 'y':
			goto description
		case 'n':
			title = a.generateTitle(ctx, diff, files)
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
		fmt.Fprintln(a.out, "Press Ctrl+D (Unix/Mac) or Ctrl+Z then Enter (Windows) to finish.")
		notesBytes, readErr := io.ReadAll(a.input)
		if readErr != nil {
			return fmt.Errorf("read description notes: %w", readErr)
		}
		notes := strings.TrimSpace(string(notesBytes))
		fmt.Fprintln(a.out, "\nNotes captured. Please wait...")

		for {
			description = a.generateDescription(ctx, diff, files, notes, title)
			fmt.Fprintf(a.out, "\n--- Proposed Description ---\n\n%s\n\n%s\n\n", description, strings.Repeat("-", bannerWidth))
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
				fmt.Fprintf(a.out, "\n--- Edited Description ---\n\n%s\n\n%s\n\n", description, strings.Repeat("-", bannerWidth))
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
	PrintFooter(a.out)
	return nil
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

func (a *App) generateTitle(ctx context.Context, diff string, files []string) string {
	result, err := a.ai.Generate(ctx, titlePrompt(diff, files))
	if err != nil {
		fmt.Fprintf(a.errOut, "\n[commitgen] AI error: %v\n", err)
	}
	if title := cleanFirstLine(result); title != "" {
		return title
	}
	return "chore(core): update changes"
}

func (a *App) generateDescription(ctx context.Context, diff string, files []string, notes, title string) string {
	result, err := a.ai.Generate(ctx, descriptionPrompt(diff, files, notes, title))
	if err != nil {
		fmt.Fprintf(a.errOut, "\n[commitgen] AI error: %v\n", err)
	}
	result = strings.TrimSpace(strings.SplitN(result, "```", 2)[0])
	if result == "" {
		return "- Describe changes (AI unavailable)\n- Provide purpose/impact"
	}
	return result
}

func (a *App) cancel(message string) error {
	fmt.Fprintf(a.out, "\n%s\n", message)
	PrintFooter(a.out)
	return nil
}

func (a *App) finishOnInputError(err error) error {
	if errors.Is(err, errInputClosed) {
		return a.cancel("Input closed. Exiting.")
	}
	return err
}

func PrintHeader(w io.Writer) {
	bar := strings.Repeat("=", bannerWidth)
	fmt.Fprintf(w, "\n%s\n     AI-POWERED GIT COMMIT GENERATOR (Conventional Commits)\n%s\n\n", bar, bar)
}

func PrintFooter(w io.Writer) {
	bar := strings.Repeat("=", bannerWidth)
	fmt.Fprintf(w, "\n%s\n                 Commit Process Finished\n%s\n\n", bar, bar)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
