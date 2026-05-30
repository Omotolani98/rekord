package cli

import (
	"fmt"
	"io"
	"strings"
)

const version = "0.1.0-dev"

// Execute runs the CLI and returns a process exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return 0
	case "version":
		fmt.Fprintf(stdout, "rekord %s\n", version)
		return 0
	case "start", "run", "list", "replay", "export", "scan", "handoff", "tmux", "skills":
		fmt.Fprintf(stderr, "rekord %s is not implemented yet\n", args[0])
		return 2
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	commands := []string{
		"start     Record an interactive terminal session",
		"run       Record a single command",
		"list      List recorded sessions",
		"replay    Replay a recorded session",
		"export    Export a session",
		"scan      Scan a session for sensitive data",
		"handoff   Generate AI-ready context",
		"tmux      Record or export tmux sessions",
		"skills    Run reusable recording recipes",
		"version   Print the Rekord version",
	}

	fmt.Fprintln(w, "rekord records terminal workflows as structured session data.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  rekord <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintf(w, "  %s\n", strings.Join(commands, "\n  "))
}
