//go:build !windows

package recorder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/ptyx"
	"github.com/creack/pty"
	"golang.org/x/term"
)

const (
	ptyReadBufSize   = 4096
	defaultKillGrace = 2 * time.Second
	exitCodeUnknown  = -1
)

type PTYRecorder struct{}

func NewPTYRecorder() *PTYRecorder {
	return &PTYRecorder{}
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

	shell := resolveShell(opts.Shell)
	result := Result{ExitCode: exitCodeUnknown}
	if len(opts.Command) == 0 {
		result.Shell = shell
	}

	writer, err := events.NewWriter(opts.EventsPath)
	if err != nil {
		return result, fmt.Errorf("open events writer: %w", err)
	}
	defer func() { _ = writer.Close() }()

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

	var ttyStdin *os.File
	if stdinFile, ok := opts.Stdin.(*os.File); ok {
		fd := int(stdinFile.Fd())
		if term.IsTerminal(fd) {
			oldState, terr := term.MakeRaw(fd)
			if terr == nil {
				defer func() { _ = term.Restore(fd, oldState) }()
			}
			if rows, cols, gerr := pty.Getsize(stdinFile); gerr == nil {
				_ = handle.Resize(cols, rows)
			}
			ttyStdin = stdinFile
		}
	}

	result.StartedAt = time.Now()

	if ttyStdin != nil {
		if rows, cols, gerr := pty.Getsize(ttyStdin); gerr == nil {
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
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		go func() {
			defer signal.Stop(sigCh)
			for {
				select {
				case <-sigCh:
					rows, cols, gerr := pty.Getsize(ttyStdin)
					if gerr != nil {
						continue
					}
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
		_ = handle.Signal(syscall.SIGTERM)
		grace := opts.KillGrace
		if grace <= 0 {
			grace = defaultKillGrace
		}
		select {
		case <-time.After(grace):
			_ = handle.Kill()
		case <-done:
		}
	}()

	exitCode, waitErr := handle.Wait()
	close(done)
	close(resizeDone)
	wg.Wait()
	_ = handle.Close()

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
