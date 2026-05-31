package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Omotolani98/rekord/internal/recorder"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var name, cwd, root string

	cmd := &cobra.Command{
		Use:   "run --name <name> -- <command> [args...]",
		Short: "Record a single command",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRun(cmd, name, cwd, root, args)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "recording name (required)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the recorded command")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")

	return cmd
}

func runRun(cmd *cobra.Command, name, cwdOverride, root string, args []string) error {
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

	cols, rows := terminalSize(cmd.InOrStdin())

	m := session.Metadata{
		ID:            id,
		Name:          name,
		CreatedAt:     now,
		Command:       args,
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
		Command:    args,
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

	if recErr != nil {
		return recErr
	}
	if res.ExitCode != 0 {
		return &exitCodeError{code: res.ExitCode}
	}
	return nil
}
