package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func Version() string { return version }

// Execute runs the CLI and returns a process exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	cmd := NewRootCommand(stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		var ece *exitCodeError
		if errors.As(err, &ece) {
			return ece.code
		}
		fmt.Fprintln(stderr, err)
		return 2
	}

	return 0
}

type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string {
	return fmt.Sprintf("command exited with status %d", e.code)
}

// NewRootCommand builds the Rekord command tree.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "rekord",
		Short:         "Record terminal workflows as structured session data",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	cmd.AddCommand(
		newVersionCommand(),
		newStartCommand(),
		newRunCommand(),
		newListCommand(),
		newReplayCommand(),
		newExportCommand(),
		newScanCommand(),
		newHandoffCommand(),
		newTmuxCommand(),
		newSkillsCommand(),
	)

	return cmd
}

func notImplemented(command string) error {
	return fmt.Errorf("rekord %s is not implemented yet", command)
}

func commandPath(parts ...string) string {
	return strings.Join(parts, " ")
}
