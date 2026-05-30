package cli

import (
	"fmt"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/export"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newExportCommand() *cobra.Command {
	var format, output, root string

	cmd := &cobra.Command{
		Use:   "export <session>",
		Short: "Export a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd, args[0], format, output, root)
		},
	}

	cmd.Flags().StringVar(&format, "to", "cast", "export format: cast")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path")
	cmd.Flags().StringVar(&root, "root", filepath.Join(".rekord", "sessions"), "sessions root directory")

	return cmd
}

func runExport(cmd *cobra.Command, ref, format, output, root string) error {
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

	outPath := output
	if outPath == "" {
		outPath = filepath.Join(store.SessionDir(m.ID), "exports", m.Name+"."+exp.Format())
	}

	if err := exp.Export(ctx, m, evs, outPath); err != nil {
		return fmt.Errorf("export %s: %w", exp.Format(), err)
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), outPath)
	return err
}
