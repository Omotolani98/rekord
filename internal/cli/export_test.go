package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestExportCommandDocFormats(t *testing.T) {
	root := t.TempDir()
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ echo hi\r\n"},
		{TimeMS: 5, Type: events.TypeOutput, Data: "hi\r\n"},
	})
	store := session.NewFileStore(root)

	for _, tc := range []struct{ format, ext string }{
		{"json", "json"},
		{"markdown", "md"},
		{"script", "sh"},
	} {
		var stdout, stderr bytes.Buffer
		code := Execute([]string{"export", "demo", "--root", root, "--to", tc.format}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%s: exit code = %d; stderr=%s", tc.format, code, stderr.String())
		}
		path := filepath.Join(store.SessionDir(m.ID), "exports", "demo."+tc.ext)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: stat %s: %v", tc.format, path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s: file empty", tc.format)
		}
	}
}

func TestExportCommandRedact(t *testing.T) {
	root := t.TempDir()
	secret := "sk-abcdef0123456789ABCDEF"
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ echo " + secret + "\r\n"},
		{TimeMS: 5, Type: events.TypeOutput, Data: secret + "\r\n"},
	})
	store := session.NewFileStore(root)

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"export", "demo", "--root", root, "--to", "json", "--redact"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(store.SessionDir(m.ID), "exports", "demo.json"))
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if bytes.Contains(data, []byte(secret)) {
		t.Fatalf("export leaked secret:\n%s", data)
	}
	if !bytes.Contains(data, []byte("[REDACTED]")) {
		t.Fatalf("export missing redaction:\n%s", data)
	}

	raw, err := os.ReadFile(filepath.Join(store.SessionDir(m.ID), "events.jsonl"))
	if err != nil {
		t.Fatalf("read raw events: %v", err)
	}
	if !bytes.Contains(raw, []byte(secret)) {
		t.Fatal("raw events.jsonl was modified; should retain secret")
	}
}

func TestExportCommandGif(t *testing.T) {
	root := t.TempDir()
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "hi\r\n"},
	})
	store := session.NewFileStore(root)

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"export", "demo", "--root", root, "--to", "gif"}, &stdout, &stderr)

	if _, err := exec.LookPath("agg"); err != nil {
		if code == 0 {
			t.Fatal("exit code = 0, want non-zero when agg missing")
		}
		if !strings.Contains(stderr.String(), "agg") {
			t.Fatalf("error should mention agg:\n%s", stderr.String())
		}
		return
	}

	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(store.SessionDir(m.ID), "exports", "demo.gif")); err != nil {
		t.Fatalf("gif not written: %v", err)
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
