package commitgen

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const progressRefreshInterval = 100 * time.Millisecond

type progressDisplay struct {
	out      io.Writer
	animated bool
	mu       sync.Mutex
}

type progressTask struct {
	display *progressDisplay
	label   string
	started time.Time
	done    chan struct{}
	wg      sync.WaitGroup
}

func newProgressDisplay(out io.Writer) *progressDisplay {
	return &progressDisplay{out: out, animated: isCharacterDevice(out)}
}

func isCharacterDevice(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (p *progressDisplay) start(label string) *progressTask {
	task := &progressTask{
		display: p,
		label:   label,
		started: time.Now(),
		done:    make(chan struct{}),
	}
	if !p.animated {
		p.printf("[commitgen] Generating %s...\n", label)
		return task
	}

	task.wg.Add(1)
	go func() {
		defer task.wg.Done()
		frames := []byte{'|', '/', '-', '\\'}
		ticker := time.NewTicker(progressRefreshInterval)
		defer ticker.Stop()
		frame := 0
		for {
			task.render(frames[frame%len(frames)])
			frame++
			select {
			case <-task.done:
				return
			case <-ticker.C:
			}
		}
	}()
	return task
}

func (task *progressTask) render(frame byte) {
	task.display.mu.Lock()
	defer task.display.mu.Unlock()
	fmt.Fprintf(task.display.out, "\r\033[2K[commitgen] %c Generating %s... %s", frame, task.label, formatElapsed(time.Since(task.started)))
}

func (task *progressTask) stop(err error) {
	if task.display.animated {
		close(task.done)
		task.wg.Wait()
	}

	status := "Generated"
	if err != nil {
		status = "Failed to generate"
	}
	task.display.mu.Lock()
	defer task.display.mu.Unlock()
	if task.display.animated {
		fmt.Fprint(task.display.out, "\r\033[2K")
	}
	fmt.Fprintf(task.display.out, "[commitgen] %s %s (%s)\n", status, task.label, formatElapsed(time.Since(task.started)))
}

func (p *progressDisplay) logf(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.animated {
		fmt.Fprint(p.out, "\r\033[2K")
	}
	fmt.Fprintf(p.out, format, args...)
}

func (p *progressDisplay) printf(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, format, args...)
}

func formatElapsed(elapsed time.Duration) string {
	if elapsed < time.Second {
		return fmt.Sprintf("%.1fs", elapsed.Seconds())
	}
	return elapsed.Round(100 * time.Millisecond).String()
}
