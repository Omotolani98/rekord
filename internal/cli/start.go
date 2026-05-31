package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Omotolani98/rekord/internal/config"
	"github.com/Omotolani98/rekord/internal/recorder"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newStartCommand() *cobra.Command {
	var name, shell, cwd, root, timer, stopKey, cfgPath string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Record an interactive terminal session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStart(cmd, name, shell, cwd, root, timer, stopKey, cfgPath)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "recording name (required)")
	cmd.Flags().StringVar(&shell, "shell", "", "shell to record (default: $SHELL)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the recorded shell")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&timer, "timer", "", "auto-stop after duration (e.g. 40s, 5m)")
	cmd.Flags().StringVar(&stopKey, "stop-key", "", "hotkey to stop recording (e.g. ctrl-]); overrides config")
	cmd.Flags().StringVar(&cfgPath, "config", "rekord.yaml", "config file with the stop-key default")

	return cmd
}

func runStart(cmd *cobra.Command, name, shellOverride, cwdOverride, root, timer, stopKey, cfgPath string) error {
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--name is required: %w", err)
	}

	keySpec := stopKey
	if keySpec == "" {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		keySpec = cfg.Recording.StopKey
	}
	stopByte, keyLabel, err := parseStopKey(keySpec)
	if err != nil {
		return err
	}

	var timeout time.Duration
	if timer != "" {
		d, err := time.ParseDuration(timer)
		if err != nil {
			return fmt.Errorf("--timer invalid: %w", err)
		}
		if d <= 0 {
			return errors.New("--timer must be positive")
		}
		timeout = d
	}

	ctx := cmd.Context()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
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

	fmt.Fprintf(cmd.ErrOrStderr(), "rekord: recording %q — press %s to stop\n", name, keyLabel)

	eventsPath := filepath.Join(root, id, "events.jsonl")
	rec := recorder.NewPTYRecorder()
	res, recErr := rec.Record(ctx, recorder.Options{
		Shell:      shellOverride,
		CWD:        cwdOverride,
		EventsPath: eventsPath,
		Stdin:      cmd.InOrStdin(),
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
		StopKey:    stopByte,
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
	cleanStop := errors.Is(recErr, context.Canceled) || errors.Is(recErr, context.DeadlineExceeded)
	if recErr != nil && !cleanStop {
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

	if cleanStop {
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
