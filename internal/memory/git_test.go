package memory

import (
	"os"
	"os/exec"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestChangedFilesKeepsFirstPathCharacter(t *testing.T) {
	got := changedFiles(" M internal/agent/server.go\n?? internal/memory/\nR  old.go -> new.go\n")
	want := []string{"internal/agent/server.go", "internal/memory/", "new.go"}
	if len(got) != len(want) {
		t.Fatalf("changedFiles = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("changedFiles[%d] = %q, want %q (all: %#v)", i, got[i], want[i], got)
		}
	}
}
