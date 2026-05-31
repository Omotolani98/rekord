package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

const commandsFilePerm = 0o600

func newCommandsCommand() *cobra.Command {
	var root, cfgPath string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "commands <session>",
		Short: "Show commands extracted from a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommands(cmd, args[0], root, cfgPath, asJSON)
		},
	}

	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&cfgPath, "config", defaultConfigPath(), "config file with prompt patterns")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")

	return cmd
}

func runCommands(cmd *cobra.Command, ref, root, cfgPath string, asJSON bool) error {
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

	extracted, err := extractCommands(cfgPath, evs)
	if err != nil {
		return err
	}
	if extracted == nil {
		extracted = []commands.Command{}
	}

	data, err := json.MarshalIndent(extracted, "", "  ")
	if err != nil {
		return fmt.Errorf("encode commands: %w", err)
	}
	if err := os.WriteFile(filepath.Join(store.SessionDir(m.ID), "commands.json"), data, commandsFilePerm); err != nil {
		return fmt.Errorf("write commands.json: %w", err)
	}

	out := cmd.OutOrStdout()
	if asJSON {
		_, err := out.Write(append(data, '\n'))
		return err
	}

	if len(extracted) == 0 {
		_, err := fmt.Fprintln(out, "No commands extracted.")
		return err
	}
	for _, c := range extracted {
		if _, err := fmt.Fprintf(out, "%d  %s\n", c.Index, c.Command); err != nil {
			return err
		}
	}
	return nil
}
