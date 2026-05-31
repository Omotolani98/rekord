package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newReplayCommand() *cobra.Command {
	var root string
	var speed float64

	cmd := &cobra.Command{
		Use:   "replay <session>",
		Short: "Replay a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplay(cmd, args[0], root, speed)
		},
	}

	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().Float64Var(&speed, "speed", 1.0, "playback speed multiplier")

	return cmd
}

func runReplay(cmd *cobra.Command, ref, root string, speed float64) error {
	if speed <= 0 {
		return errors.New("--speed must be positive")
	}

	ctx := cmd.Context()
	store := session.NewFileStore(root)
	m, err := store.Resolve(ctx, ref)
	if err != nil {
		return err
	}

	evs, err := events.ReadAll(filepath.Join(store.SessionDir(m.ID), "events.jsonl"))
	if err != nil {
		return fmt.Errorf("read events: %w", err)
	}

	out := cmd.OutOrStdout()
	start := time.Now()
	for _, e := range evs {
		if e.Type != events.TypeOutput {
			continue
		}
		target := time.Duration(float64(e.TimeMS) * float64(time.Millisecond) / speed)
		wait := target - time.Since(start)
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
		if _, err := fmt.Fprint(out, e.Data); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	return nil
}
