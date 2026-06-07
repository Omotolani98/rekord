package memory

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

type ResumeOptions struct {
	Project   string
	Agent     string
	FromAgent string
	ToAgent   string
	Session   string
	Query     string
	Limit     int
}

func BuildResumeContext(ctx context.Context, store Store, opts ResumeOptions) (ResumeContext, error) {
	project, err := NormalizeProject(opts.Project)
	if err != nil {
		return ResumeContext{}, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 8
	}
	f := Filter{Project: project, Agent: opts.Agent, Session: opts.Session, Limit: limit}
	latest, err := store.LatestSnapshot(ctx, f)
	var latestPtr *Snapshot
	if err == nil {
		latestPtr = &latest
	} else if !errors.Is(err, fs.ErrNotExist) {
		return ResumeContext{}, err
	}
	open, err := store.ListMemories(ctx, Filter{Project: project, Agent: opts.Agent, Session: opts.Session, Status: StatusOpen, Limit: limit})
	if err != nil {
		return ResumeContext{}, err
	}
	var recent []Memory
	if strings.TrimSpace(opts.Query) != "" {
		recent, err = store.SearchMemories(ctx, opts.Query, f)
	} else {
		recent, err = store.ListMemories(ctx, f)
	}
	if err != nil {
		return ResumeContext{}, err
	}
	rc := ResumeContext{
		Project:        project,
		Agent:          opts.Agent,
		FromAgent:      opts.FromAgent,
		ToAgent:        opts.ToAgent,
		SessionName:    opts.Session,
		LatestSnapshot: latestPtr,
		OpenMemories:   open,
		RecentMemories: recent,
	}
	rc.Summary = FormatResume(rc)
	return rc, nil
}

func FormatResume(rc ResumeContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\n", rc.Project)
	if rc.Project != "" {
		fmt.Fprintf(&b, "Storage key: %s\n", ProjectKey(rc.Project))
	}
	if rc.FromAgent != "" || rc.Agent != "" {
		agent := rc.FromAgent
		if agent == "" {
			agent = rc.Agent
		}
		fmt.Fprintf(&b, "Context from agent: %s\n", agent)
	}
	if rc.ToAgent != "" {
		fmt.Fprintf(&b, "Intended next agent: %s\n", rc.ToAgent)
	}
	if rc.SessionName != "" {
		fmt.Fprintf(&b, "Session: %s\n", rc.SessionName)
	}
	if rc.LatestSnapshot != nil {
		fmt.Fprintf(&b, "\nLatest snapshot: %s\n", rc.LatestSnapshot.Title)
		if rc.LatestSnapshot.Note != "" {
			fmt.Fprintf(&b, "%s\n", rc.LatestSnapshot.Note)
		}
		if rc.LatestSnapshot.Git.Branch != "" {
			fmt.Fprintf(&b, "Branch: %s\n", rc.LatestSnapshot.Git.Branch)
		}
		if len(rc.LatestSnapshot.Git.ChangedFiles) > 0 {
			b.WriteString("Changed files:\n")
			for _, file := range rc.LatestSnapshot.Git.ChangedFiles {
				fmt.Fprintf(&b, "- %s\n", file)
			}
		}
		if len(rc.LatestSnapshot.Patches) > 0 {
			b.WriteString("Patch files:\n")
			for _, patch := range rc.LatestSnapshot.Patches {
				fmt.Fprintf(&b, "- %s: %s (%d bytes)\n", patch.Kind, patch.Path, patch.Bytes)
			}
		}
	}
	if len(rc.OpenMemories) > 0 {
		b.WriteString("\nOpen memories:\n")
		for _, m := range rc.OpenMemories {
			fmt.Fprintf(&b, "- %s: %s\n", displayTitle(m), m.Body)
		}
	}
	if len(rc.RecentMemories) > 0 {
		b.WriteString("\nRecent memories:\n")
		for _, m := range rc.RecentMemories {
			fmt.Fprintf(&b, "- %s: %s\n", displayTitle(m), m.Body)
		}
	}
	if rc.LatestSnapshot == nil && len(rc.OpenMemories) == 0 && len(rc.RecentMemories) == 0 {
		b.WriteString("\nNo Rekord memory found for this scope.\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func displayTitle(m Memory) string {
	if m.Title != "" {
		return m.Title
	}
	return m.ID
}
