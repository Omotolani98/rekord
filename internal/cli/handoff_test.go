package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func TestHandoffCommandGeneratesContext(t *testing.T) {
	root := t.TempDir()
	secret := "sk-abcdef0123456789ABCDEF"
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ echo " + secret + "\r\n"},
		{TimeMS: 5, Type: events.TypeOutput, Data: secret + "\r\n"},
		{TimeMS: 9, Type: events.TypeOutput, Data: "error: something failed\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"handoff", "demo", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}

	store := session.NewFileStore(root)
	data, err := os.ReadFile(filepath.Join(store.SessionDir(m.ID), "handoff", "context.md"))
	if err != nil {
		t.Fatalf("read context.md: %v", err)
	}
	s := string(data)
	if !bytes.Contains(data, []byte("# Rekord AI Context")) {
		t.Fatalf("missing heading:\n%s", s)
	}
	if bytes.Contains(data, []byte(secret)) {
		t.Fatalf("context leaked secret:\n%s", s)
	}
	if !bytes.Contains(data, []byte("[REDACTED]")) {
		t.Fatalf("context missing redaction:\n%s", s)
	}
	if !bytes.Contains(data, []byte("something failed")) {
		t.Fatalf("context missing error line:\n%s", s)
	}
}

func TestHandoffCommandIncludeTree(t *testing.T) {
	root := t.TempDir()
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "hi\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"handoff", "demo", "--root", root, "--include-tree"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}
	store := session.NewFileStore(root)
	if _, err := os.Stat(filepath.Join(store.SessionDir(m.ID), "handoff", "tree.txt")); err != nil {
		t.Fatalf("tree.txt not written: %v", err)
	}
}

func TestHandoffCommandMissingSession(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"handoff", "nope", "--root", root}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for missing session")
	}
}
