package handoff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGatherGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	gc, ok := GatherGit(context.Background(), dir, 1000)
	if !ok {
		t.Fatal("ok = false, want true for git repo")
	}
	if gc.Branch != "main" {
		t.Fatalf("branch = %q, want main", gc.Branch)
	}
	if gc.Status == "" {
		t.Fatal("status empty, want modified file")
	}
}

func TestGatherGitNonRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if _, ok := GatherGit(context.Background(), t.TempDir(), 1000); ok {
		t.Fatal("ok = true, want false for non-git dir")
	}
}
