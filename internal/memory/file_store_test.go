package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStoreMemoryLifecycleAndSearch(t *testing.T) {
	store := NewFileStore(t.TempDir())
	project := t.TempDir()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	m := Memory{
		ID:           "mem_test",
		Project:      project,
		Agent:        "claude",
		SessionName:  "auth-claude",
		Type:         TypeBlocker,
		Status:       StatusOpen,
		Title:        "Auth blocker",
		Body:         "Refresh token test still failing",
		Tags:         []string{"auth"},
		RelatedFiles: []string{"internal/auth/session_test.go"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.AddMemory(context.Background(), m); err != nil {
		t.Fatalf("AddMemory: %v", err)
	}

	items, err := store.SearchMemories(context.Background(), "auth", Filter{Project: project, Agent: "claude", Session: "auth-claude"})
	if err != nil {
		t.Fatalf("SearchMemories: %v", err)
	}
	if len(items) != 1 || items[0].ID != m.ID {
		t.Fatalf("search = %+v, want %s", items, m.ID)
	}

	got, err := store.GetMemory(context.Background(), project, m.ID)
	if err != nil {
		t.Fatalf("GetMemory: %v", err)
	}
	got.Status = StatusResolved
	got.UpdatedAt = now.Add(time.Minute)
	if err := store.UpdateMemory(context.Background(), got); err != nil {
		t.Fatalf("UpdateMemory: %v", err)
	}
	resolved, err := store.GetMemory(context.Background(), project, m.ID)
	if err != nil {
		t.Fatalf("GetMemory resolved: %v", err)
	}
	if resolved.Status != StatusResolved {
		t.Fatalf("status = %q, want resolved", resolved.Status)
	}
}

func TestCreateSnapshotWritesGitPatches(t *testing.T) {
	requireGit(t)
	project := t.TempDir()
	runGit(t, project, "init")
	runGit(t, project, "config", "user.email", "test@example.com")
	runGit(t, project, "config", "user.name", "Test")
	writeFile(t, filepath.Join(project, "README.md"), "hello\n")
	runGit(t, project, "add", "README.md")
	runGit(t, project, "commit", "-m", "init")
	writeFile(t, filepath.Join(project, "README.md"), "hello\nworld\n")

	store := NewFileStore(t.TempDir())
	snap, err := CreateSnapshot(context.Background(), store, SnapshotOptions{Project: project, Agent: "claude", Session: "memory-claude", Note: "stopped here"})
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if snap.Agent != "claude" || snap.SessionName != "memory-claude" {
		t.Fatalf("snapshot linkage = agent %q session %q", snap.Agent, snap.SessionName)
	}
	if len(snap.Patches) != 1 {
		t.Fatalf("patches = %d, want 1", len(snap.Patches))
	}
	data, err := os.ReadFile(snap.Patches[0].Path)
	if err != nil {
		t.Fatalf("read patch: %v", err)
	}
	if !strings.Contains(string(data), "+world") {
		t.Fatalf("patch missing change:\n%s", data)
	}
}

func TestBuildResumeContextReturnsAllAgents(t *testing.T) {
	store := NewFileStore(t.TempDir())
	project := t.TempDir()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	for _, m := range []Memory{
		{ID: "mem_claude", Project: project, Agent: "claude", Status: StatusOpen, Type: TypeNote, Title: "Claude work", Body: "Claude stopped at parser tests", CreatedAt: now, UpdatedAt: now},
		{ID: "mem_codex", Project: project, Agent: "codex", Status: StatusOpen, Type: TypeNote, Title: "Codex work", Body: "Codex did unrelated cleanup", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
	} {
		if err := store.AddMemory(context.Background(), m); err != nil {
			t.Fatalf("AddMemory: %v", err)
		}
	}
	rc, err := BuildResumeContext(context.Background(), store, ResumeOptions{Project: project, FromAgent: "claude", ToAgent: "codex"})
	if err != nil {
		t.Fatalf("BuildResumeContext: %v", err)
	}
	if !strings.Contains(rc.Summary, "Claude stopped") {
		t.Fatalf("summary missing claude memory:\n%s", rc.Summary)
	}
	if !strings.Contains(rc.Summary, "unrelated cleanup") {
		t.Fatalf("from_agent must not filter out other agents' memories:\n%s", rc.Summary)
	}
	if !strings.Contains(rc.Summary, "Context from agent: claude") {
		t.Fatalf("summary missing from_agent label:\n%s", rc.Summary)
	}
}

func TestBuildResumeContextFiltersByExplicitAgent(t *testing.T) {
	store := NewFileStore(t.TempDir())
	project := t.TempDir()
	now := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	for _, m := range []Memory{
		{ID: "mem_claude", Project: project, Agent: "claude", Status: StatusOpen, Type: TypeNote, Title: "Claude work", Body: "Claude stopped at parser tests", CreatedAt: now, UpdatedAt: now},
		{ID: "mem_codex", Project: project, Agent: "codex", Status: StatusOpen, Type: TypeNote, Title: "Codex work", Body: "Codex did unrelated cleanup", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)},
	} {
		if err := store.AddMemory(context.Background(), m); err != nil {
			t.Fatalf("AddMemory: %v", err)
		}
	}
	rc, err := BuildResumeContext(context.Background(), store, ResumeOptions{Project: project, Agent: "claude"})
	if err != nil {
		t.Fatalf("BuildResumeContext: %v", err)
	}
	if !strings.Contains(rc.Summary, "Claude stopped") {
		t.Fatalf("summary missing claude memory:\n%s", rc.Summary)
	}
	if strings.Contains(rc.Summary, "unrelated cleanup") {
		t.Fatalf("explicit agent filter must exclude other agents:\n%s", rc.Summary)
	}
}
