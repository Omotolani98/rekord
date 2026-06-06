package memory

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type SnapshotOptions struct {
	Project     string
	Agent       string
	Actor       string
	Source      string
	Session     string
	SessionID   string
	SessionName string
	Title       string
	Note        string
	Now         time.Time
}

func CreateSnapshot(ctx context.Context, store Store, opts SnapshotOptions) (Snapshot, error) {
	project, err := NormalizeProject(opts.Project)
	if err != nil {
		return Snapshot{}, err
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = strings.TrimSpace(opts.Note)
	}
	if title == "" {
		title = "snapshot"
	}
	sessionID, sessionName := splitSession(opts.Session, opts.SessionID, opts.SessionName)
	snap := Snapshot{
		ID:          NewID("snap", title, now),
		Project:     project,
		Agent:       strings.TrimSpace(opts.Agent),
		Actor:       strings.TrimSpace(opts.Actor),
		Source:      defaultString(opts.Source, SourceSnapshot),
		SessionID:   sessionID,
		SessionName: sessionName,
		Title:       title,
		Note:        strings.TrimSpace(opts.Note),
		CreatedAt:   now,
	}
	snap.Summary = snapshotSummary(snap)
	if git, unstaged, staged, ok := GatherGit(ctx, project); ok {
		snap.Git = git
		if len(unstaged) > 0 {
			path := filepath.Join(store.PatchDir(project), snap.ID+".patch")
			bytes, err := WritePatch(path, unstaged)
			if err != nil {
				return Snapshot{}, fmt.Errorf("write unstaged patch: %w", err)
			}
			snap.Patches = append(snap.Patches, PatchRef{Kind: "unstaged", Path: path, Bytes: bytes})
		}
		if len(staged) > 0 {
			path := filepath.Join(store.PatchDir(project), snap.ID+".staged.patch")
			bytes, err := WritePatch(path, staged)
			if err != nil {
				return Snapshot{}, fmt.Errorf("write staged patch: %w", err)
			}
			snap.Patches = append(snap.Patches, PatchRef{Kind: "staged", Path: path, Bytes: bytes})
		}
	}
	if err := store.CreateSnapshot(ctx, snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

func snapshotSummary(s Snapshot) string {
	parts := []string{s.Title}
	if s.Agent != "" {
		parts = append(parts, "agent: "+s.Agent)
	}
	if s.SessionName != "" {
		parts = append(parts, "session: "+s.SessionName)
	}
	return strings.Join(parts, " · ")
}

func splitSession(session, sessionID, sessionName string) (string, string) {
	if sessionName == "" && sessionID == "" {
		if strings.HasPrefix(session, "sess_") || strings.Contains(session, "_") {
			sessionID = session
		} else {
			sessionName = session
		}
	}
	return strings.TrimSpace(sessionID), strings.TrimSpace(sessionName)
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
