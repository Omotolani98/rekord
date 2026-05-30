//go:build !windows

package recorder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/creack/pty"
	"golang.org/x/term"
)

const (
	ptyReadBufSize      = 4096
	defaultKillGrace    = 2 * time.Second
	exitCodeUnknown     = -1
	exitCodeUnavailable = -2
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
	result := Result{Shell: shell, ExitCode: exitCodeUnknown}

	writer, err := events.NewWriter(opts.EventsPath)
	if err != nil {
		return result, fmt.Errorf("open events writer: %w", err)
	}
	defer func() { _ = writer.Close() }()

	cmd := exec.Command(shell)
	if opts.CWD != "" {
		cmd.Dir = opts.CWD
	}
	if opts.Env != nil {
		cmd.Env = opts.Env
	}

	master, err := pty.Start(cmd)
	if err != nil {
		return result, fmt.Errorf("start pty: %w", err)
	}
	defer func() { _ = master.Close() }()

	if stdinFile, ok := opts.Stdin.(*os.File); ok {
		fd := int(stdinFile.Fd())
		if term.IsTerminal(fd) {
			oldState, terr := term.MakeRaw(fd)
			if terr == nil {
				defer func() { _ = term.Restore(fd, oldState) }()
			}
			_ = pty.InheritSize(stdinFile, master)
		}
	}

	result.StartedAt = time.Now()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, ptyReadBufSize)
		for {
			n, rerr := master.Read(buf)
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

	go func() {
		_, _ = io.Copy(master, opts.Stdin)
	}()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if cmd.Process == nil {
				return
			}
			_ = cmd.Process.Signal(syscall.SIGTERM)
			grace := opts.KillGrace
			if grace <= 0 {
				grace = defaultKillGrace
			}
			select {
			case <-time.After(grace):
				_ = cmd.Process.Kill()
			case <-done:
			}
		case <-done:
		}
	}()

	waitErr := cmd.Wait()
	close(done)
	_ = master.Close()
	wg.Wait()

	result.EndedAt = time.Now()
	result.DurationMS = result.EndedAt.Sub(result.StartedAt).Milliseconds()

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = exitCodeUnavailable
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return result, ctxErr
	}

	var exitErr *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitErr) {
		return result, fmt.Errorf("wait shell: %w", waitErr)
	}

	return result, nil
}
