//go:build !windows

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
)

func TestSkillsListShowsBuiltins(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"skills", "list", "--skills-dir", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "go-project-demo") {
		t.Fatalf("list missing builtin:\n%s", stdout.String())
	}
}

func TestSkillsRunRecordsSession(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	skillsDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(skillsDir, "demo.yaml"),
		[]byte("name: demo\ndescription: d\nsteps:\n  - run: echo skill-marker\n"),
		0o600,
	); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"skills", "run", "demo", "--name", "s", "--root", root, "--skills-dir", skillsDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}

	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("session dirs = %d, want 1", len(entries))
	}
	evs, err := events.ReadAll(filepath.Join(root, entries[0].Name(), "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var joined strings.Builder
	for _, e := range evs {
		joined.WriteString(e.Data)
	}
	if !strings.Contains(joined.String(), "skill-marker") {
		t.Fatalf("events missing marker:\n%s", joined.String())
	}
}
