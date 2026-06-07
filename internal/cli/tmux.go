package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/Omotolani98/rekord/internal/tmux"
	"github.com/spf13/cobra"
)

const tmuxRawLog = "tmux-raw.log"

func newTmuxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tmux",
		Short: "Work with tmux sessions",
	}
	cmd.AddCommand(
		newTmuxStatusCommand(),
		newTmuxCaptureCommand(),
		newTmuxRecordCommand(),
		newTmuxStartCommand(),
	)
	return cmd
}

func newTmuxStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether the current shell is inside tmux",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTmuxStatus(cmd)
		},
	}
}

func runTmuxStatus(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	if !tmux.Available() {
		_, err := fmt.Fprintln(out, "tmux is not installed.")
		return err
	}
	if !tmux.Inside() {
		_, err := fmt.Fprintln(out, "Not inside a tmux session.")
		return err
	}
	name, err := tmux.CurrentSession(cmd.Context())
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Inside tmux session: %s\n", name)
	return err
}

func newTmuxCaptureCommand() *cobra.Command {
	var pane, name, root string
	cmd := &cobra.Command{
		Use:   "capture --pane <pane> --name <name>",
		Short: "Capture a tmux pane's current contents as a session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTmuxCapture(cmd, pane, name, root)
		},
	}
	cmd.Flags().StringVar(&pane, "pane", "", "tmux pane or session target (required)")
	cmd.Flags().StringVar(&name, "name", "", "recording name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	return cmd
}

func runTmuxCapture(cmd *cobra.Command, pane, name, root string) error {
	if err := requireTmux(); err != nil {
		return err
	}
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--name is required: %w", err)
	}
	if pane == "" {
		return errors.New("--pane is required")
	}

	ctx := cmd.Context()
	text, err := tmux.CapturePane(ctx, pane)
	if err != nil {
		return fmt.Errorf("capture pane: %w", err)
	}

	now := time.Now().UTC()
	id := session.NewID(name, now)
	cols, rows := paneSize(ctx, pane)
	m := session.Metadata{
		ID:            id,
		Name:          name,
		CreatedAt:     now,
		EndedAt:       &now,
		TmuxPane:      pane,
		Cols:          cols,
		Rows:          rows,
		Status:        session.StatusCompleted,
		RekordVersion: Version(),
	}
	store := session.NewFileStore(root)
	if err := store.Create(ctx, m); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	w, err := events.NewWriter(filepath.Join(store.SessionDir(id), "events.jsonl"))
	if err != nil {
		return err
	}
	if err := w.Append(events.Event{TimeMS: 0, Type: events.TypeOutput, Data: text}); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), store.SessionDir(id))
	return err
}

func newTmuxRecordCommand() *cobra.Command {
	var pane, name, root string
	cmd := &cobra.Command{
		Use:   "record --pane <pane> --name <name>",
		Short: "Stream a tmux pane into a recording via pipe-pane",
		Long: `Stream a tmux pane into a recording using tmux pipe-pane.

Limitations: only output produced after recording starts is captured; input
keystrokes are not recorded and sub-second timing is approximate.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTmuxRecord(cmd, pane, name, root)
		},
	}
	cmd.Flags().StringVar(&pane, "pane", "", "tmux pane or session target (required)")
	cmd.Flags().StringVar(&name, "name", "", "recording name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	return cmd
}

func runTmuxRecord(cmd *cobra.Command, pane, name, root string) error {
	if err := requireTmux(); err != nil {
		return err
	}
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--name is required: %w", err)
	}
	if pane == "" {
		return errors.New("--pane is required")
	}

	ctx := cmd.Context()
	store, m, w, rawPath, err := newTmuxSession(ctx, root, name, pane)
	if err != nil {
		return err
	}

	stop, err := startPipeRecording(ctx, rawPath, pane, w, m.CreatedAt)
	if err != nil {
		_ = w.Close()
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Recording tmux pane %s... press Enter to stop.\n", pane)
	waitForStop(ctx, cmd.InOrStdin())
	stop()

	return finalizeTmux(store, w, m)
}

func newTmuxStartCommand() *cobra.Command {
	var name, root string
	cmd := &cobra.Command{
		Use:   "start --session <name>",
		Short: "Create a tmux session, record it, and attach",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTmuxStart(cmd, name, root)
		},
	}
	cmd.Flags().StringVar(&name, "session", "", "tmux session name (required)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	return cmd
}

func runTmuxStart(cmd *cobra.Command, name, root string) error {
	if err := requireTmux(); err != nil {
		return err
	}
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--session is required: %w", err)
	}

	ctx := cmd.Context()
	if tmux.HasSession(ctx, name) {
		return fmt.Errorf("tmux session %q already exists", name)
	}
	if err := tmux.NewSession(ctx, name); err != nil {
		return fmt.Errorf("create tmux session: %w", err)
	}

	store, m, w, rawPath, err := newTmuxSession(ctx, root, name, name)
	if err != nil {
		return err
	}

	stop, err := startPipeRecording(ctx, rawPath, name, w, m.CreatedAt)
	if err != nil {
		_ = w.Close()
		return err
	}

	attachErr := tmux.Attach(ctx, name, os.Stdin, os.Stdout, os.Stderr)
	stop()
	if ferr := finalizeTmux(store, w, m); ferr != nil {
		return ferr
	}
	return attachErr
}

const (
	defaultTmuxCols = 80
	defaultTmuxRows = 24
)

func paneSize(ctx context.Context, pane string) (cols, rows int) {
	cols, rows, err := tmux.PaneSize(ctx, pane)
	if err != nil || cols <= 0 || rows <= 0 {
		return defaultTmuxCols, defaultTmuxRows
	}
	return cols, rows
}

func requireTmux() error {
	if !tmux.Available() {
		return errors.New("tmux is not installed")
	}
	return nil
}

func newTmuxSession(ctx context.Context, root, name, pane string) (*session.FileStore, session.Metadata, *events.Writer, string, error) {
	now := time.Now().UTC()
	id := session.NewID(name, now)
	cols, rows := paneSize(ctx, pane)
	m := session.Metadata{
		ID:            id,
		Name:          name,
		CreatedAt:     now,
		TmuxPane:      pane,
		Cols:          cols,
		Rows:          rows,
		Status:        session.StatusRecording,
		RekordVersion: Version(),
	}
	store := session.NewFileStore(root)
	if err := store.Create(ctx, m); err != nil {
		return nil, m, nil, "", fmt.Errorf("create session: %w", err)
	}

	rawPath := filepath.Join(store.SessionDir(id), tmuxRawLog)
	f, err := os.Create(rawPath)
	if err != nil {
		return nil, m, nil, "", fmt.Errorf("create raw log: %w", err)
	}
	_ = f.Close()

	w, err := events.NewWriter(filepath.Join(store.SessionDir(id), "events.jsonl"))
	if err != nil {
		return nil, m, nil, "", err
	}
	return store, m, w, rawPath, nil
}

func startPipeRecording(ctx context.Context, rawPath, target string, w *events.Writer, start time.Time) (func(), error) {
	if err := tmux.PipePane(ctx, target, "cat >> "+shellQuote(rawPath)); err != nil {
		return nil, fmt.Errorf("pipe-pane: %w", err)
	}

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		f, err := os.Open(rawPath)
		if err != nil {
			return
		}
		defer f.Close()

		buf := make([]byte, 4096)
		drain := func() {
			for {
				n, rerr := f.Read(buf)
				if n > 0 {
					_ = w.Append(events.Event{
						TimeMS: time.Since(start).Milliseconds(),
						Type:   events.TypeOutput,
						Data:   string(buf[:n]),
					})
				}
				if rerr != nil {
					return
				}
			}
		}

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				drain()
			case <-done:
				drain()
				return
			case <-ctx.Done():
				drain()
				return
			}
		}
	}()

	stop := func() {
		_ = tmux.StopPipe(context.Background(), target)
		close(done)
		<-finished
	}
	return stop, nil
}

func finalizeTmux(store *session.FileStore, w *events.Writer, m session.Metadata) error {
	if err := w.Close(); err != nil {
		return err
	}
	ended := time.Now().UTC()
	m.EndedAt = &ended
	m.DurationMS = ended.Sub(m.CreatedAt).Milliseconds()
	m.Status = session.StatusCompleted
	return store.WriteMetadata(context.Background(), m)
}

func waitForStop(ctx context.Context, in io.Reader) {
	ch := make(chan struct{})
	go func() {
		_, _ = bufio.NewReader(in).ReadString('\n')
		close(ch)
	}()
	select {
	case <-ch:
	case <-ctx.Done():
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
