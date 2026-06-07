package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeProjectCanonicalizesToGitRoot(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(real, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	sub := filepath.Join(real, "internal", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}

	fromRoot, err := NormalizeProject(real)
	if err != nil {
		t.Fatalf("NormalizeProject(root): %v", err)
	}
	fromSub, err := NormalizeProject(sub)
	if err != nil {
		t.Fatalf("NormalizeProject(sub): %v", err)
	}
	if fromRoot != real {
		t.Fatalf("root: got %q want %q", fromRoot, real)
	}
	if fromSub != real {
		t.Fatalf("subdir did not canonicalize to git root: got %q want %q", fromSub, real)
	}
	if ProjectKey(fromRoot) != ProjectKey(fromSub) {
		t.Fatalf("project keys differ across subdirs of the same repo")
	}
}

func TestNormalizeProjectNonRepoUnchanged(t *testing.T) {
	dir := t.TempDir()
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	got, err := NormalizeProject(real)
	if err != nil {
		t.Fatalf("NormalizeProject: %v", err)
	}
	if got != real {
		t.Fatalf("non-repo path changed: got %q want %q", got, real)
	}
}

func TestListProjectsRecordsPath(t *testing.T) {
	store := NewFileStore(t.TempDir())
	project := t.TempDir()
	m := Memory{ID: "mem_1", Project: project, Status: StatusOpen, Type: TypeNote, Title: "t", Body: "b"}
	if err := store.AddMemory(context.Background(), m); err != nil {
		t.Fatalf("AddMemory: %v", err)
	}
	normalized, err := NormalizeProject(project)
	if err != nil {
		t.Fatalf("NormalizeProject: %v", err)
	}
	projects, err := store.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects want 1", len(projects))
	}
	if projects[0].Path != normalized {
		t.Fatalf("path: got %q want %q", projects[0].Path, normalized)
	}
	if projects[0].Key != ProjectKey(normalized) {
		t.Fatalf("key: got %q want %q", projects[0].Key, ProjectKey(normalized))
	}
}
