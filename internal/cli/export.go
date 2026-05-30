package cli

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/config"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/export"
	"github.com/Omotolani98/rekord/internal/redact"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	var format, output, root, cfgPath string
	var doRedact, noRedact bool

	cmd := &cobra.Command{
		Use:   "export <session>",
		Short: "Export a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd, args[0], format, output, root, cfgPath, doRedact, noRedact)
		},
	}

	cmd.Flags().StringVar(&format, "to", "cast", "export format: cast, json, markdown, script")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path")
	cmd.Flags().StringVar(&root, "root", filepath.Join(".rekord", "sessions"), "sessions root directory")
	cmd.Flags().StringVar(&cfgPath, "config", "rekord.yaml", "config file with prompt and redaction patterns")
	cmd.Flags().BoolVar(&doRedact, "redact", false, "redact secrets in the export")
	cmd.Flags().BoolVar(&noRedact, "no-redact", false, "disable redaction even if enabled in config")

	return cmd
}

func runExport(cmd *cobra.Command, ref, format, output, root, cfgPath string, doRedact, noRedact bool) error {
	exp, err := export.Get(format)
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgPath)
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

	cmds, err := extractWithConfig(cfg, evs)
	if err != nil {
		return err
	}

	if redactEnabled(cfg, doRedact, noRedact) {
		r, err := buildRedactor(cfg)
		if err != nil {
			return err
		}
		evs = redactEvents(r, evs)
		cmds = redactCommands(r, cmds)
		m = redactMetadata(r, m)
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
	return extractWithConfig(cfg, evs)
}

func extractWithConfig(cfg config.Config, evs []events.Event) ([]commands.Command, error) {
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

func buildRedactor(cfg config.Config) (*redact.Redactor, error) {
	patterns := redact.DefaultPatterns()
	for _, p := range cfg.Privacy.RedactPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid redact pattern %q: %w", p, err)
		}
		patterns = append(patterns, redact.Custom("custom", re))
	}
	return redact.New(patterns), nil
}

func redactEnabled(cfg config.Config, doRedact, noRedact bool) bool {
	if noRedact {
		return false
	}
	if doRedact {
		return true
	}
	return cfg.Privacy.Redact
}

func redactEvents(r *redact.Redactor, evs []events.Event) []events.Event {
	out := make([]events.Event, len(evs))
	copy(out, evs)
	for i := range out {
		if out[i].Data != "" {
			out[i].Data = r.Redact(out[i].Data)
		}
	}
	return out
}

func redactMetadata(r *redact.Redactor, m session.Metadata) session.Metadata {
	if len(m.Command) > 0 {
		redacted := make([]string, len(m.Command))
		for i, c := range m.Command {
			redacted[i] = r.Redact(c)
		}
		m.Command = redacted
	}
	return m
}

func redactCommands(r *redact.Redactor, cmds []commands.Command) []commands.Command {
	out := make([]commands.Command, len(cmds))
	copy(out, cmds)
	for i := range out {
		out[i].Command = r.Redact(out[i].Command)
		out[i].OutputPreview = r.Redact(out[i].OutputPreview)
	}
	return out
}
