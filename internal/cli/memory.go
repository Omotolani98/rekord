package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	mem "github.com/Omotolani98/rekord/internal/memory"
	"github.com/spf13/cobra"
)

type memoryFlags struct {
	root      string
	project   string
	agent     string
	fromAgent string
	toAgent   string
	session   string
	typeName  string
	tags      []string
	limit     int
}

func newRememberCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "remember <text>",
		Short: "Store a project memory for humans and agents",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := strings.TrimSpace(strings.Join(args, " "))
			m, err := addMemory(cmd.Context(), flags, body, body, mem.SourceCLI)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "remembered %s\n", m.ID)
			return err
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	cmd.Flags().StringVar(&flags.typeName, "type", mem.TypeNote, "memory type: note, fact, decision, todo, blocker, warning")
	cmd.Flags().StringSliceVar(&flags.tags, "tag", nil, "memory tag (repeatable or comma-separated)")
	return cmd
}

func newRecallCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "recall [query]",
		Short: "Search project memory",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(strings.Join(args, " "))
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			items, err := store.SearchMemories(cmd.Context(), query, mem.Filter{Project: project, Agent: flags.agent, Session: flags.session, Limit: flags.limit})
			if err != nil {
				return err
			}
			return printMemories(cmd, items)
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	cmd.Flags().IntVar(&flags.limit, "limit", 10, "maximum memories to show")
	return cmd
}

func newSnapshotCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "snapshot [note]",
		Short: "Capture a resumable project checkpoint with git patches",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			note := strings.TrimSpace(strings.Join(args, " "))
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			snap, err := mem.CreateSnapshot(cmd.Context(), store, mem.SnapshotOptions{
				Project: project,
				Agent:   flags.agent,
				Actor:   "agent",
				Source:  mem.SourceCLI,
				Session: flags.session,
				Title:   note,
				Note:    note,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "snapshot %s\n", snap.ID)
			if len(snap.Patches) > 0 {
				for _, patch := range snap.Patches {
					fmt.Fprintf(out, "%s patch: %s\n", patch.Kind, patch.Path)
				}
			}
			return nil
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	return cmd
}

func newResumeCommand() *cobra.Command {
	var flags memoryFlags
	var query string
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Generate agent-ready context from Rekord memory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			rc, err := mem.BuildResumeContext(cmd.Context(), store, mem.ResumeOptions{
				Project:   project,
				Agent:     flags.agent,
				FromAgent: flags.fromAgent,
				ToAgent:   flags.toAgent,
				Session:   flags.session,
				Query:     query,
				Limit:     flags.limit,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), rc.Summary)
			return err
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	cmd.Flags().StringVar(&flags.fromAgent, "from-agent", "", "source agent to resume from")
	cmd.Flags().StringVar(&flags.toAgent, "to-agent", "", "destination agent for handoff context")
	cmd.Flags().StringVar(&query, "query", "", "optional query to rank memories")
	cmd.Flags().IntVar(&flags.limit, "limit", 8, "maximum memories to include")
	return cmd
}

func newMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage Rekord project memory",
	}
	cmd.AddCommand(newMemoryAddCommand(), newMemoryListCommand(), newMemorySearchCommand(), newMemoryShowCommand(), newMemoryResolveCommand(), newMemoryProjectsCommand())
	return cmd
}

func newMemoryProjectsCommand() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects with stored memory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mem.NewFileStore(root)
			projects, err := store.ListProjects(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(projects) == 0 {
				_, err := fmt.Fprintln(out, "no projects found")
				return err
			}
			for _, p := range projects {
				path := p.Path
				if path == "" {
					path = "(unknown path)"
				}
				fmt.Fprintf(out, "%s  %s\n", p.Key, path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "memory-root", mem.DefaultRoot(), "memory root directory")
	return cmd
}

func newMemoryAddCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "add <text>",
		Short: "Add a memory",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := strings.TrimSpace(strings.Join(args, " "))
			m, err := addMemory(cmd.Context(), flags, body, body, mem.SourceCLI)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", m.ID)
			return err
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	cmd.Flags().StringVar(&flags.typeName, "type", mem.TypeNote, "memory type")
	cmd.Flags().StringSliceVar(&flags.tags, "tag", nil, "memory tag")
	return cmd
}

func newMemoryListCommand() *cobra.Command {
	var flags memoryFlags
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List memories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			items, err := store.ListMemories(cmd.Context(), mem.Filter{Project: project, Agent: flags.agent, Session: flags.session, Status: status, Limit: flags.limit})
			if err != nil {
				return err
			}
			return printMemories(cmd, items)
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().IntVar(&flags.limit, "limit", 20, "maximum memories to show")
	return cmd
}

func newMemorySearchCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search memories",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			items, err := store.SearchMemories(cmd.Context(), strings.Join(args, " "), mem.Filter{Project: project, Agent: flags.agent, Session: flags.session, Limit: flags.limit})
			if err != nil {
				return err
			}
			return printMemories(cmd, items)
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	cmd.Flags().IntVar(&flags.limit, "limit", 20, "maximum memories to show")
	return cmd
}

func newMemoryShowCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show one memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			m, err := store.GetMemory(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s\n", m.ID, m.Body)
			return err
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	return cmd
}

func newMemoryResolveCommand() *cobra.Command {
	var flags memoryFlags
	cmd := &cobra.Command{
		Use:   "resolve <id>",
		Short: "Mark a memory resolved",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, store, err := memoryStore(flags)
			if err != nil {
				return err
			}
			m, err := store.GetMemory(cmd.Context(), project, args[0])
			if err != nil {
				return err
			}
			m.Status = mem.StatusResolved
			m.UpdatedAt = time.Now()
			if err := store.UpdateMemory(cmd.Context(), m); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "resolved %s\n", m.ID)
			return err
		},
	}
	addMemoryCommonFlags(cmd, &flags)
	return cmd
}

func addMemoryCommonFlags(cmd *cobra.Command, flags *memoryFlags) {
	cmd.Flags().StringVar(&flags.root, "memory-root", mem.DefaultRoot(), "memory root directory")
	cmd.Flags().StringVar(&flags.project, "project", ".", "project directory")
	cmd.Flags().StringVar(&flags.agent, "agent", "", "agent name that produced the memory")
	cmd.Flags().StringVar(&flags.session, "session", "", "session name or id")
}

func addMemory(ctx context.Context, flags memoryFlags, title, body, source string) (mem.Memory, error) {
	project, store, err := memoryStore(flags)
	if err != nil {
		return mem.Memory{}, err
	}
	now := time.Now()
	sessionID, sessionName := sessionParts(flags.session)
	m := mem.Memory{
		ID:          mem.NewID("mem", title, now),
		Project:     project,
		Agent:       strings.TrimSpace(flags.agent),
		Actor:       "agent",
		Source:      source,
		SessionID:   sessionID,
		SessionName: sessionName,
		Type:        defaultMemoryType(flags.typeName),
		Status:      mem.StatusOpen,
		Title:       title,
		Body:        body,
		Tags:        flags.tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.AddMemory(ctx, m); err != nil {
		return mem.Memory{}, err
	}
	return m, nil
}

func memoryStore(flags memoryFlags) (string, *mem.FileStore, error) {
	project, err := mem.NormalizeProject(flags.project)
	if err != nil {
		return "", nil, err
	}
	return project, mem.NewFileStore(flags.root), nil
}

func sessionParts(session string) (string, string) {
	if strings.HasPrefix(session, "sess_") || strings.Contains(session, "_") {
		return session, ""
	}
	return "", session
}

func defaultMemoryType(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return mem.TypeNote
	}
	return v
}

func printMemories(cmd *cobra.Command, items []mem.Memory) error {
	out := cmd.OutOrStdout()
	if len(items) == 0 {
		_, err := fmt.Fprintln(out, "no memories found")
		return err
	}
	for _, m := range items {
		parts := []string{m.ID}
		if m.Agent != "" {
			parts = append(parts, "agent="+m.Agent)
		}
		if m.SessionName != "" {
			parts = append(parts, "session="+m.SessionName)
		}
		parts = append(parts, "status="+m.Status)
		fmt.Fprintf(out, "%s\n", strings.Join(parts, " "))
		if m.Title != "" && m.Title != m.Body {
			fmt.Fprintf(out, "  %s\n", m.Title)
		}
		fmt.Fprintf(out, "  %s\n", m.Body)
	}
	return nil
}
