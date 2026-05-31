package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Rekord version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			st := newStyler(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "rekord %s\n", st.bold(version))
		},
	}
}
