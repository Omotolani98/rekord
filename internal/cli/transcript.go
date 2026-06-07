package cli

import (
	"fmt"
	"strings"

	mem "github.com/Omotolani98/rekord/internal/memory"
	"github.com/Omotolani98/rekord/internal/redact"
	"github.com/Omotolani98/rekord/internal/transcript"
	"github.com/spf13/cobra"
)

func newTranscriptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcript",
		Short: "Read prior coding-agent session transcripts for context handoff",
	}
	cmd.AddCommand(
		newTranscriptSourcesCommand(),
		newTranscriptListCommand(),
		newTranscriptShowCommand(),
		newTranscriptSearchCommand(),
	)
	return cmd
}

func newTranscriptSourcesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sources",
		Short: "List available agent transcript sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sources := transcript.Sources()
			out := cmd.OutOrStdout()
			if len(sources) == 0 {
				_, err := fmt.Fprintln(out, "no agent transcript sources found")
				return err
			}
			for _, s := range sources {
				fmt.Fprintln(out, s)
			}
			return nil
		},
	}
}

func newTranscriptListCommand() *cobra.Command {
	var project, source string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List prior agent transcripts for this project, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			norm, err := mem.NormalizeProject(project)
			if err != nil {
				return err
			}
			items, err := transcript.List(norm)
			if err != nil {
				return err
			}
			return printTranscripts(cmd, filterBySource(items, source))
		},
	}
	cmd.Flags().StringVar(&project, "project", ".", "project directory")
	cmd.Flags().StringVar(&source, "source", "", "limit to one source: claude or codex")
	return cmd
}

func newTranscriptShowCommand() *cobra.Command {
	var project string
	var lastN, maxBytes int
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <source> <id>",
		Short: "Show one transcript as an agent-ready digest",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			norm, err := mem.NormalizeProject(project)
			if err != nil {
				return err
			}
			tr, err := transcript.Read(norm, args[0], args[1])
			if err != nil {
				return err
			}
			if !raw {
				tr = tr.Redact(redact.NewDefault())
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), transcript.Digest(tr, lastN, maxBytes))
			return err
		},
	}
	cmd.Flags().StringVar(&project, "project", ".", "project directory")
	cmd.Flags().IntVar(&lastN, "last", 0, "keep only the last N turns (default 20)")
	cmd.Flags().IntVar(&maxBytes, "max-bytes", 0, "max digest bytes (default 8000)")
	cmd.Flags().BoolVar(&raw, "raw", false, "do not redact secrets")
	return cmd
}

func newTranscriptSearchCommand() *cobra.Command {
	var project, source string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search prior agent transcripts for this project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			norm, err := mem.NormalizeProject(project)
			if err != nil {
				return err
			}
			items, err := transcript.Search(norm, strings.Join(args, " "))
			if err != nil {
				return err
			}
			return printTranscripts(cmd, filterBySource(items, source))
		},
	}
	cmd.Flags().StringVar(&project, "project", ".", "project directory")
	cmd.Flags().StringVar(&source, "source", "", "limit to one source: claude or codex")
	return cmd
}

func filterBySource(items []transcript.Summary, source string) []transcript.Summary {
	source = strings.TrimSpace(source)
	if source == "" {
		return items
	}
	var out []transcript.Summary
	for _, s := range items {
		if s.Source == source {
			out = append(out, s)
		}
	}
	return out
}

func printTranscripts(cmd *cobra.Command, items []transcript.Summary) error {
	out := cmd.OutOrStdout()
	if len(items) == 0 {
		_, err := fmt.Fprintln(out, "no transcripts found")
		return err
	}
	r := redact.NewDefault()
	for _, s := range items {
		parts := []string{s.Source, s.SessionID}
		if s.Branch != "" {
			parts = append(parts, "branch="+s.Branch)
		}
		parts = append(parts, fmt.Sprintf("msgs=%d", s.Messages))
		if !s.EndedAt.IsZero() {
			parts = append(parts, s.EndedAt.Format("2006-01-02 15:04"))
		}
		fmt.Fprintln(out, strings.Join(parts, "  "))
		prompt := strings.TrimSpace(r.Redact(firstNonEmpty(s.Title, s.FirstPrompt)))
		if prompt != "" {
			fmt.Fprintf(out, "  %s\n", prompt)
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
