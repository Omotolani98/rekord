//go:build windows

package recorder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/ptyx"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

const (
	ptyReadBufSize      = 4096
	exitCodeUnknown     = -1
	resizePollInterval  = 250 * time.Millisecond
	defaultWindowsShell = "powershell.exe"
)

type PTYRecorder struct{}

func NewPTYRecorder() *PTYRecorder {
	return &PTYRecorder{}
}

func resolveShellWindows(override string) string {
	if override != "" {
		return override
	}
	return defaultWindowsShell
}

func enableVirtualTerminal(w io.Writer) func() {
	f, ok := w.(*os.File)
	if !ok {
		return func() {}
	}
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return func() {}
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return func() {}
	}
	if err := windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return func() {}
	}
	return func() { _ = windows.SetConsoleMode(handle, mode) }
}

func (r *PTYRecorder) Record(ctx context.Context, opts Options) (Result, error) {
	if opts.EventsPath == "" {
		return Result{}, errors.New("recorder: EventsPath is required")
	}
	if opts.Stdin == nil {
		return Result{}, errors.New("recorder: Stdin is required")
	}
	if opts.Stdout == nil {
		return Result{}, errors.New("recorder: Stdout is required")
	}

	shell := resolveShellWindows(opts.Shell)
	result := Result{ExitCode: exitCodeUnknown}
	if len(opts.Command) == 0 {
		result.Shell = shell
	}

	writer, err := events.NewWriter(opts.EventsPath)
	if err != nil {
		return result, fmt.Errorf("open events writer: %w", err)
	}
	defer func() { _ = writer.Close() }()

	restoreVT := enableVirtualTerminal(opts.Stdout)
	defer restoreVT()

	var ttyStdin *os.File
	if stdinFile, ok := opts.Stdin.(*os.File); ok {
		fd := int(stdinFile.Fd())
		if term.IsTerminal(fd) {
			oldState, terr := term.MakeRaw(fd)
			if terr == nil {
				defer func() { _ = term.Restore(fd, oldState) }()
			}
			ttyStdin = stdinFile
		}
	}

	handle, err := ptyx.Start(ptyx.Options{
		Command: opts.Command,
		Shell:   shell,
		CWD:     opts.CWD,
		Env:     opts.Env,
	})
	if err != nil {
		return result, fmt.Errorf("start pty: %w", err)
	}
	defer func() { _ = handle.Close() }()

	result.StartedAt = time.Now()

	lastCols, lastRows := 0, 0
	if ttyStdin != nil {
		if cols, rows, gerr := term.GetSize(int(ttyStdin.Fd())); gerr == nil {
			lastCols, lastRows = cols, rows
			_ = handle.Resize(cols, rows)
			if aerr := writer.Append(events.Event{
				TimeMS: 0,
				Type:   events.TypeResize,
				Cols:   cols,
				Rows:   rows,
			}); aerr != nil && opts.Stderr != nil {
				fmt.Fprintf(opts.Stderr, "recorder: append initial resize: %v\n", aerr)
			}
		}
	}

	resizeDone := make(chan struct{})
	if ttyStdin != nil {
		go func() {
			ticker := time.NewTicker(resizePollInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					cols, rows, gerr := term.GetSize(int(ttyStdin.Fd()))
					if gerr != nil || (cols == lastCols && rows == lastRows) {
						continue
					}
					lastCols, lastRows = cols, rows
					if serr := handle.Resize(cols, rows); serr != nil && opts.Stderr != nil {
						fmt.Fprintf(opts.Stderr, "recorder: pty resize: %v\n", serr)
					}
					if aerr := writer.Append(events.Event{
						TimeMS: time.Since(result.StartedAt).Milliseconds(),
						Type:   events.TypeResize,
						Cols:   cols,
						Rows:   rows,
					}); aerr != nil && opts.Stderr != nil {
						fmt.Fprintf(opts.Stderr, "recorder: append resize: %v\n", aerr)
					}
				case <-resizeDone:
					return
				}
			}
		}()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, ptyReadBufSize)
		for {
			n, rerr := handle.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				if _, werr := opts.Stdout.Write(chunk); werr != nil && opts.Stderr != nil {
					fmt.Fprintf(opts.Stderr, "recorder: stdout write: %v\n", werr)
				}
				ev := events.Event{
					TimeMS: time.Since(result.StartedAt).Milliseconds(),
					Type:   events.TypeOutput,
					Data:   string(chunk),
				}
				if aerr := writer.Append(ev); aerr != nil && opts.Stderr != nil {
					fmt.Fprintf(opts.Stderr, "recorder: append event: %v\n", aerr)
				}
			}
			if rerr != nil {
				return
			}
		}
	}()

	stopCh := make(chan struct{})
	var stopOnce sync.Once
	requestStop := func() { stopOnce.Do(func() { close(stopCh) }) }

	go func() {
		if opts.StopKey == 0 {
			_, _ = io.Copy(handle, opts.Stdin)
			return
		}
		buf := make([]byte, ptyReadBufSize)
		for {
			n, rerr := opts.Stdin.Read(buf)
			if n > 0 {
				data := buf[:n]
				if idx := bytes.IndexByte(data, opts.StopKey); idx >= 0 {
					_, _ = handle.Write(data[:idx])
					requestStop()
					return
				}
				_, _ = handle.Write(data)
			}
			if rerr != nil {
				return
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-stopCh:
		case <-done:
			return
		}
		_ = handle.Kill()
	}()

	exitCode, waitErr := handle.Wait()
	close(done)
	close(resizeDone)
	_ = handle.Close()
	wg.Wait()

	result.EndedAt = time.Now()
	result.DurationMS = result.EndedAt.Sub(result.StartedAt).Milliseconds()
	result.ExitCode = exitCode

	if ctxErr := ctx.Err(); ctxErr != nil {
		return result, ctxErr
	}

	var exitErr *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitErr) {
		return result, fmt.Errorf("wait shell: %w", waitErr)
	}

	return result, nil
}
