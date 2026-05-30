package cli

import (
	"github.com/spf13/cobra"
)

func newStartCommand() *cobra.Command {
	var name, timer, shell, cwd string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Record an interactive terminal session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("start")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "recording name")
	cmd.Flags().StringVar(&timer, "timer", "", "optional recording duration, for example 40s or 5m")
	cmd.Flags().StringVar(&shell, "shell", "", "shell to record")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the recorded shell")

	return cmd
}

func newRunCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "run --name <name> -- <command> [args...]",
		Short: "Record a single command",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("run")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "recording name")

	return cmd
}

func newReplayCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "replay <session>",
		Short: "Replay a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("replay")
		},
	}
}

func newExportCommand() *cobra.Command {
	var format, output string

	cmd := &cobra.Command{
		Use:   "export <session>",
		Short: "Export a recorded session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("export")
		},
	}

	cmd.Flags().StringVar(&format, "to", "markdown", "export format: markdown, cast, json, script, mp4, or gif")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path")

	return cmd
}

func newScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan <session>",
		Short: "Scan a session for sensitive data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("scan")
		},
	}
}

func newHandoffCommand() *cobra.Command {
	var includeGit, includeTree, includeLogs bool

	cmd := &cobra.Command{
		Use:   "handoff <session>",
		Short: "Generate AI-ready context from a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented("handoff")
		},
	}

	cmd.Flags().BoolVar(&includeGit, "include-git", false, "include git status and diff context")
	cmd.Flags().BoolVar(&includeTree, "include-tree", false, "include a repository tree snapshot")
	cmd.Flags().BoolVar(&includeLogs, "include-logs", false, "include captured session logs")

	return cmd
}

func newTmuxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tmux",
		Short: "Record or export tmux sessions",
	}

	cmd.AddCommand(newTmuxStartCommand(), newTmuxExportCommand())

	return cmd
}

func newTmuxStartCommand() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Record a tmux session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented(commandPath("tmux", "start"))
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "tmux session name")

	return cmd
}

func newTmuxExportCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "export <session>",
		Short: "Export a recorded tmux session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notImplemented(commandPath("tmux", "export"))
		},
	}

	cmd.Flags().StringVar(&format, "to", "markdown", "export format")

	return cmd
}

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
