package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func TestCommandsCommandExtractsAndWrites(t *testing.T) {
	root := t.TempDir()
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ echo hi\r\n"},
		{TimeMS: 5, Type: events.TypeOutput, Data: "hi\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"commands", "demo", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("echo hi")) {
		t.Fatalf("stdout missing command:\n%s", stdout.String())
	}

	store := session.NewFileStore(root)
	data, err := os.ReadFile(filepath.Join(store.SessionDir(m.ID), "commands.json"))
	if err != nil {
		t.Fatalf("read commands.json: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal commands.json: %v", err)
	}
	if len(got) != 1 || got[0]["command"] != "echo hi" {
		t.Fatalf("commands.json = %v", got)
	}
}

func TestCommandsCommandJSON(t *testing.T) {
	root := t.TempDir()
	seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ ls\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"commands", "demo", "--root", root, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout not JSON array: %v\n%s", err, stdout.String())
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestCommandsCommandNoCommands(t *testing.T) {
	root := t.TempDir()
	m := seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "plain output\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"commands", "demo", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("No commands extracted.")) {
		t.Fatalf("stdout = %q", stdout.String())
	}

	store := session.NewFileStore(root)
	data, err := os.ReadFile(filepath.Join(store.SessionDir(m.ID), "commands.json"))
	if err != nil {
		t.Fatalf("read commands.json: %v", err)
	}
	if string(bytes.TrimSpace(data)) != "[]" {
		t.Fatalf("commands.json = %q, want []", string(data))
	}
}

func TestCommandsCommandMissingSession(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"commands", "nope", "--root", root}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for missing session")
	}
}
