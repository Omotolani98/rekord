package cli

import (
	"fmt"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/config"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/export"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	var format, output, root, cfgPath string

	cmd := &cobra.Command{
		Use:   "export <session>",
		Short: "Export a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd, args[0], format, output, root, cfgPath)
		},
	}

	cmd.Flags().StringVar(&format, "to", "cast", "export format: cast, json, markdown, script")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path")
	cmd.Flags().StringVar(&root, "root", filepath.Join(".rekord", "sessions"), "sessions root directory")
	cmd.Flags().StringVar(&cfgPath, "config", "rekord.yaml", "config file with prompt patterns")

	return cmd
}

func runExport(cmd *cobra.Command, ref, format, output, root, cfgPath string) error {
	exp, err := export.Get(format)
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

	cmds, err := extractCommands(cfgPath, evs)
	if err != nil {
		return err
	}

	outPath := output
	if outPath == "" {
		outPath = filepath.Join(store.SessionDir(m.ID), "exports", m.Name+"."+exp.Ext())
	}

	if err := exp.Export(ctx, m, evs, cmds, outPath); err != nil {
		return fmt.Errorf("export %s: %w", exp.Format(), err)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), outPath)
	return err
}

func extractCommands(cfgPath string, evs []events.Event) ([]commands.Command, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	patterns := cfg.Commands.PromptPatterns
	if len(patterns) == 0 {
		patterns = commands.DefaultPatterns()
	}
	compiled, err := commands.CompilePatterns(patterns)
	if err != nil {
		return nil, err
	}
	return commands.NewExtractor(compiled).Extract(evs), nil
}
