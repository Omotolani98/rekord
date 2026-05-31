//go:build !windows

package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func TestStartCommandRecordsSession(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	root := t.TempDir()
	stdinR, stdinW := io.Pipe()
	go func() {
		_, _ = stdinW.Write([]byte("echo cli-marker\nexit\n"))
		_ = stdinW.Close()
	}()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetIn(stdinR)
	cmd.SetArgs([]string{"start", "--name", "demo", "--shell", "/bin/sh", "--root", root})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("Execute: %v; stderr=%s", err, stderr.String())
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
	if m.Name != "demo" {
		t.Fatalf("Name = %q, want demo", m.Name)
	}
	if m.RekordVersion != Version() {
		t.Fatalf("RekordVersion = %q, want %q", m.RekordVersion, Version())
	}
	if m.Shell != "/bin/sh" {
		t.Fatalf("Shell = %q, want /bin/sh", m.Shell)
	}
	if m.EndedAt == nil {
		t.Fatal("EndedAt nil, want set")
	}
	if m.DurationMS < 0 {
		t.Fatalf("DurationMS = %d, want >= 0", m.DurationMS)
	}

	evs, err := events.ReadAll(filepath.Join(sessionDir, "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll events: %v", err)
	}
	if len(evs) == 0 {
		t.Fatal("no events recorded")
	}
	var joined strings.Builder
	for _, e := range evs {
		joined.WriteString(e.Data)
	}
	if !strings.Contains(joined.String(), "cli-marker") {
		t.Fatalf("events missing marker:\n%s", joined.String())
	}
}

func TestStartCommandTimerStops(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	root := t.TempDir()
	stdinR, stdinW := io.Pipe()
	defer stdinW.Close()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetIn(stdinR)
	cmd.SetArgs([]string{"start", "--name", "demo", "--shell", "/bin/sh", "--root", root, "--timer", "300ms"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("Execute: %v; stderr=%s", err, stderr.String())
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
	if m.EndedAt == nil {
		t.Fatal("EndedAt nil, want set")
	}
}

func TestStartCommandInvalidTimer(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"start", "--name", "demo", "--root", root, "--timer", "bogus"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute err = nil, want error")
	}
	if !strings.Contains(err.Error(), "timer") {
		t.Fatalf("err = %v, want timer message", err)
	}

	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatalf("created %d entries, want 0", len(entries))
	}
}

func TestStartCommandRequiresName(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"start", "--root", root})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute err = nil, want error")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("err = %v, want name-required message", err)
	}

	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatalf("created %d entries, want 0", len(entries))
	}
}

func TestStartCommandFailedStatusOnRecorderError(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"start", "--name", "demo", "--shell", "/does/not/exist", "--root", root})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute err = nil, want recorder error")
	}

	entries, derr := os.ReadDir(root)
	if derr != nil {
		t.Fatalf("ReadDir: %v", derr)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	store := session.NewFileStore(root)
	m, merr := store.ReadMetadata(context.Background(), entries[0].Name())
	if merr != nil {
		t.Fatalf("ReadMetadata: %v", merr)
	}
	if m.Status != session.StatusFailed {
		t.Fatalf("Status = %q, want %q", m.Status, session.StatusFailed)
	}
	if m.EndedAt == nil {
		t.Fatal("EndedAt nil, want set")
	}
}
