package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var root string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recorded sessions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := session.NewFileStore(root)
			sessions, err := store.List(cmd.Context())
			if err != nil {
				return err
			}
			return renderSessionTable(cmd.OutOrStdout(), sessions)
		},
	}

	cmd.Flags().StringVar(&root, "root", filepath.Join(".rekord", "sessions"), "sessions root directory")

	return cmd
}

func renderSessionTable(out io.Writer, sessions []session.Metadata) error {
	if len(sessions) == 0 {
		_, err := fmt.Fprintln(out, "No sessions recorded yet.")
		return err
	}

	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tDURATION\tSTATUS\tCREATED"); err != nil {
		return err
	}
	for _, m := range sessions {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			m.Name,
			formatDuration(m.DurationMS),
			m.Status,
			m.CreatedAt.Local().Format("2006-01-02 15:04"),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	d := time.Duration(ms) * time.Millisecond
	if d >= time.Second {
		d = d.Round(time.Second)
	}
	return d.String()
}
