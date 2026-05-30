//go:build !windows

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func TestRunCommandRecordsOutput(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute(
		[]string{"run", "--name", "hi", "--root", root, "--", "/bin/sh", "-c", "echo run-marker"},
		&stdout, &stderr,
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("session dir count = %d, want 1", len(entries))
	}
	sessionDir := filepath.Join(root, entries[0].Name())

	store := session.NewFileStore(root)
	m, err := store.ReadMetadata(context.Background(), entries[0].Name())
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Status != session.StatusCompleted {
		t.Fatalf("Status = %q, want %q", m.Status, session.StatusCompleted)
	}
	wantCmd := []string{"/bin/sh", "-c", "echo run-marker"}
	if strings.Join(m.Command, "\x00") != strings.Join(wantCmd, "\x00") {
		t.Fatalf("Command = %v, want %v", m.Command, wantCmd)
	}

	evs, err := events.ReadAll(filepath.Join(sessionDir, "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll events: %v", err)
	}
	var joined strings.Builder
	for _, e := range evs {
		joined.WriteString(e.Data)
	}
	if !strings.Contains(joined.String(), "run-marker") {
		t.Fatalf("events missing marker:\n%s", joined.String())
	}
}

func TestRunCommandPropagatesExitCode(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute(
		[]string{"run", "--name", "fail", "--root", root, "--", "/bin/sh", "-c", "exit 3"},
		&stdout, &stderr,
	)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%s", code, stderr.String())
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("session dir count = %d, want 1", len(entries))
	}

	store := session.NewFileStore(root)
	m, err := store.ReadMetadata(context.Background(), entries[0].Name())
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Status != session.StatusCompleted {
		t.Fatalf("Status = %q, want %q", m.Status, session.StatusCompleted)
	}
}

func TestRunCommandRequiresName(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute(
		[]string{"run", "--root", root, "--", "/bin/sh", "-c", "echo hi"},
		&stdout, &stderr,
	)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}

	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatalf("created %d entries, want 0", len(entries))
	}
}
