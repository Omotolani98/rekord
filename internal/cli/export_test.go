package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func seedSession(t *testing.T, root, name string, evs []events.Event) session.Metadata {
	t.Helper()
	now := time.Now().UTC()
	id := session.NewID(name, now)
	m := session.Metadata{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		Cols:      80,
		Rows:      24,
		Status:    session.StatusCompleted,
	}
	store := session.NewFileStore(root)
	if err := store.Create(context.Background(), m); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	w, err := events.NewWriter(filepath.Join(store.SessionDir(id), "events.jsonl"))
	if err != nil {
		t.Fatalf("seed NewWriter: %v", err)
	}
	for _, e := range evs {
		if err := w.Append(e); err != nil {
			t.Fatalf("seed Append: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("seed Close: %v", err)
	}
	return m
}

func TestExportCommandWritesCast(t *testing.T) {
	root := t.TempDir()
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "hello\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"export", "demo", "--root", root, "--to", "cast"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	store := session.NewFileStore(root)
	castPath := filepath.Join(store.SessionDir(m.ID), "exports", "demo.cast")
	info, err := os.Stat(castPath)
	if err != nil {
		t.Fatalf("stat cast: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("cast file empty")
	}
}

func TestExportCommandUnknownFormat(t *testing.T) {
	root := t.TempDir()
	seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "hi\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"export", "demo", "--root", root, "--to", "bogus"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for unknown format")
	}
}

func TestExportCommandMissingSession(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"export", "nope", "--root", root, "--to", "cast"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for missing session")
	}
}
