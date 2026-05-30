package cli

import (
	"github.com/spf13/cobra"
)

func newSkillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Run reusable recording recipes",
	}

	cmd.AddCommand(newSkillsListCommand(), newSkillsRunCommand())

	return cmd
}

func newSkillsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented(commandPath("skills", "list"))
		},
	}
}

func newSkillsRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <skill>",
		Short: "Run a recording skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented(commandPath("skills", "run"))
		},
	}
}
