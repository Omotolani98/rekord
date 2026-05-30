package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Omotolani98/rekord/internal/recorder"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newStartCommand() *cobra.Command {
	var name, shell, cwd, root string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Record an interactive terminal session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStart(cmd, name, shell, cwd, root)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "recording name (required)")
	cmd.Flags().StringVar(&shell, "shell", "", "shell to record (default: $SHELL)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the recorded shell")
	cmd.Flags().StringVar(&root, "root", filepath.Join(".rekord", "sessions"), "sessions root directory")

	return cmd
}

func runStart(cmd *cobra.Command, name, shellOverride, cwdOverride, root string) error {
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--name is required: %w", err)
	}

	ctx := cmd.Context()
	now := time.Now().UTC()
	id := session.NewID(name, now)

	cwdResolved := cwdOverride
	if cwdResolved == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}
		cwdResolved = wd
	}

	shellInitial := shellOverride
	if shellInitial == "" {
		shellInitial = os.Getenv("SHELL")
	}

	cols, rows := terminalSize(cmd.InOrStdin())

	m := session.Metadata{
		ID:            id,
		Name:          name,
		CreatedAt:     now,
		Shell:         shellInitial,
		CWD:           cwdResolved,
		Cols:          cols,
		Rows:          rows,
		Status:        session.StatusRecording,
		RekordVersion: Version(),
	}

	store := session.NewFileStore(root)
	if err := store.Create(ctx, m); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	eventsPath := filepath.Join(root, id, "events.jsonl")
	rec := recorder.NewPTYRecorder()
	res, recErr := rec.Record(ctx, recorder.Options{
		Shell:      shellOverride,
		CWD:        cwdOverride,
		EventsPath: eventsPath,
		Stdin:      cmd.InOrStdin(),
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
	})

	ended := res.EndedAt
	if ended.IsZero() {
		ended = time.Now()
	}
	ended = ended.UTC()
	m.EndedAt = &ended
	m.DurationMS = res.DurationMS
	if res.Shell != "" {
		m.Shell = res.Shell
	}
	if recErr != nil {
		m.Status = session.StatusFailed
	} else {
		m.Status = session.StatusCompleted
	}

	if err := store.WriteMetadata(context.Background(), m); err != nil {
		if recErr != nil {
			return fmt.Errorf("recorder failed: %w; also failed to update metadata: %v", recErr, err)
		}
		return fmt.Errorf("update metadata: %w", err)
	}

	if errors.Is(recErr, context.Canceled) {
		return nil
	}
	return recErr
}

func terminalSize(stdin any) (int, int) {
	f, ok := stdin.(*os.File)
	if !ok {
		return 0, 0
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return 0, 0
	}
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		return 0, 0
	}
	return cols, rows
}
