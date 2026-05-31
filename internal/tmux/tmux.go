package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func Inside() bool {
	return os.Getenv("TMUX") != ""
}

func CurrentSession(ctx context.Context) (string, error) {
	out, err := output(ctx, "display-message", "-p", "#S")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func CapturePane(ctx context.Context, pane string) (string, error) {
	return output(ctx, "capture-pane", "-p", "-t", pane)
}

func PaneSize(ctx context.Context, pane string) (cols, rows int, err error) {
	out, err := output(ctx, "display-message", "-p", "-t", pane, "#{pane_width} #{pane_height}")
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("tmux pane size: unexpected output %q", strings.TrimSpace(out))
	}
	cols, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("tmux pane size: %w", err)
	}
	rows, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("tmux pane size: %w", err)
	}
	return cols, rows, nil
}

func HasSession(ctx context.Context, name string) bool {
	return run(ctx, "has-session", "-t", name) == nil
}

func NewSession(ctx context.Context, name string) error {
	return run(ctx, "new-session", "-d", "-s", name)
}

func KillSession(ctx context.Context, name string) error {
	return run(ctx, "kill-session", "-t", name)
}

func PipePane(ctx context.Context, target, shellCmd string) error {
	return run(ctx, "pipe-pane", "-t", target, shellCmd)
}

func StopPipe(ctx context.Context, target string) error {
	return run(ctx, "pipe-pane", "-t", target)
}

func Attach(ctx context.Context, name string, in, out, errOut *os.File) error {
	cmd := exec.CommandContext(ctx, "tmux", "attach", "-t", name)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

func run(ctx context.Context, args ...string) error {
	return exec.CommandContext(ctx, "tmux", args...).Run()
}

func output(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "tmux", args...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
