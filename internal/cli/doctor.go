package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var doctorTools = []string{"ffmpeg", "agg", "asciinema", "tmux", "git"}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check for optional external tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd)
		},
	}
}

func runDoctor(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	st := newStyler(out)
	for _, tool := range doctorTools {
		path, err := exec.LookPath(tool)
		if err != nil {
			fmt.Fprintf(out, "%s    %s\n", st.red("missing"), tool)
			continue
		}
		fmt.Fprintf(out, "%s  %s %s\n", st.green("available"), tool, st.dim("("+path+")"))
	}
	return nil
}
