package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Omotolani98/rekord/internal/config"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newScanCommand() *cobra.Command {
	var root, cfgPath string
	var strict bool

	cmd := &cobra.Command{
		Use:   "scan <session>",
		Short: "Scan a session for possible secrets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, args[0], root, cfgPath, strict)
		},
	}

	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&cfgPath, "config", "rekord.yaml", "config file with redaction patterns")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero if secrets are found")

	return cmd
}

func runScan(cmd *cobra.Command, ref, root, cfgPath string, strict bool) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	red, err := buildRedactor(cfg)
	if err != nil {
		return err
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

	var b strings.Builder
	for _, e := range evs {
		b.WriteString(e.Data)
		b.WriteByte('\n')
	}
	cmds, err := extractWithConfig(cfg, evs)
	if err != nil {
		return err
	}
	for _, c := range cmds {
		b.WriteString(c.Command)
		b.WriteByte('\n')
	}

	categories := red.Scan(b.String())

	out := cmd.OutOrStdout()
	if len(categories) == 0 {
		_, err := fmt.Fprintln(out, "No secrets detected.")
		return err
	}

	fmt.Fprintf(out, "Possible secrets found (%d categories):\n", len(categories))
	for _, c := range categories {
		fmt.Fprintf(out, "  - %s\n", c)
	}

	if strict {
		return &exitCodeError{code: 1}
	}
	return nil
}
