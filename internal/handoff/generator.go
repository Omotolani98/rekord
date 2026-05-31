package handoff

import (
	"fmt"
	"strings"
	"time"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/session"
)

type Input struct {
	Metadata session.Metadata
	Commands []commands.Command
	Output   string
	Errors   []string
	Git      *GitContext
	Tree     string
}

func Generate(in Input) string {
	var b strings.Builder
	m := in.Metadata

	b.WriteString("# Rekord AI Context\n\n")

	b.WriteString("## Session\n\n")
	fmt.Fprintf(&b, "- Name: %s\n", m.Name)
	fmt.Fprintf(&b, "- Shell: %s\n", orDash(m.Shell))
	fmt.Fprintf(&b, "- Working directory: %s\n", m.CWD)
	fmt.Fprintf(&b, "- Duration: %s\n", formatDuration(m.DurationMS))
	fmt.Fprintf(&b, "- Status: %s\n", m.Status)
	if len(m.Command) > 0 {
		fmt.Fprintf(&b, "- Command: %s\n", strings.Join(m.Command, " "))
	}
	b.WriteString("\n")

	b.WriteString("## Commands Run\n\n")
	if len(in.Commands) == 0 {
		b.WriteString("_No commands extracted._\n\n")
	} else {
		for _, c := range in.Commands {
			fmt.Fprintf(&b, "%d. %s\n", c.Index, c.Command)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Observed Output\n\n")
	if strings.TrimSpace(in.Output) == "" {
		b.WriteString("_No output captured._\n\n")
	} else {
		fmt.Fprintf(&b, "```text\n%s\n```\n\n", strings.TrimRight(in.Output, "\n"))
	}

	b.WriteString("## Possible Errors\n\n")
	if len(in.Errors) == 0 {
		b.WriteString("_None detected._\n\n")
	} else {
		for _, e := range in.Errors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Suggested Summary\n\n")
	fmt.Fprintf(&b, "Recorded session %q with %d extracted command(s).\n\n", m.Name, len(in.Commands))

	if in.Git != nil {
		b.WriteString("## Git\n\n")
		fmt.Fprintf(&b, "- Branch: %s\n\n", orDash(in.Git.Branch))
		if strings.TrimSpace(in.Git.Status) != "" {
			fmt.Fprintf(&b, "Status:\n\n```text\n%s\n```\n\n", strings.TrimRight(in.Git.Status, "\n"))
		}
		if strings.TrimSpace(in.Git.Diff) != "" {
			fmt.Fprintf(&b, "Diff:\n\n```diff\n%s\n```\n\n", strings.TrimRight(in.Git.Diff, "\n"))
		}
	}

	if strings.TrimSpace(in.Tree) != "" {
		b.WriteString("## Project Tree\n\n")
		fmt.Fprintf(&b, "```text\n%s\n```\n\n", strings.TrimRight(in.Tree, "\n"))
	}

	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
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
