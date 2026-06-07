package export

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func sampleMeta() session.Metadata {
	return session.Metadata{Name: "demo", Shell: "/bin/zsh", CWD: "/project", DurationMS: 42000}
}

func sampleCmds() []commands.Command {
	return []commands.Command{
		{Index: 1, Command: "go test ./...", OutputPreview: "ok github.com/example/app 0.231s"},
		{Index: 2, Command: "ls", OutputPreview: "a.go b.go"},
	}
}

func TestJSONExporter(t *testing.T) {
	out := filepath.Join(t.TempDir(), "exports", "demo.json")
	evs := []events.Event{{TimeMS: 0, Type: events.TypeOutput, Data: "hello\r\n"}}
	if err := (JSONExporter{}).Export(context.Background(), sampleMeta(), evs, sampleCmds(), out); err != nil {
		t.Fatalf("Export: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got struct {
		Metadata      session.Metadata   `json:"metadata"`
		Commands      []commands.Command `json:"commands"`
		OutputSummary string             `json:"outputSummary"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Metadata.Name != "demo" {
		t.Fatalf("metadata.name = %q", got.Metadata.Name)
	}
	if len(got.Commands) != 2 {
		t.Fatalf("commands = %d, want 2", len(got.Commands))
	}
	if !strings.Contains(got.OutputSummary, "hello") {
		t.Fatalf("outputSummary = %q", got.OutputSummary)
	}
}

func TestMarkdownExporter(t *testing.T) {
	out := filepath.Join(t.TempDir(), "exports", "demo.md")
	if err := (MarkdownExporter{}).Export(context.Background(), sampleMeta(), nil, sampleCmds(), out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, _ := os.ReadFile(out)
	s := string(data)
	for _, want := range []string{
		"# Rekord Session: demo",
		"## Summary",
		"- Duration: 42s",
		"- Shell: /bin/zsh",
		"### 1. go test ./...",
		"ok github.com/example/app 0.231s",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("markdown missing %q:\n%s", want, s)
		}
	}
}

func TestMarkdownExporterNoCommands(t *testing.T) {
	out := filepath.Join(t.TempDir(), "demo.md")
	if err := (MarkdownExporter{}).Export(context.Background(), sampleMeta(), nil, nil, out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "_No commands extracted._") {
		t.Fatalf("missing placeholder:\n%s", data)
	}
}

func TestScriptExporter(t *testing.T) {
	out := filepath.Join(t.TempDir(), "exports", "demo.sh")
	if err := (ScriptExporter{}).Export(context.Background(), sampleMeta(), nil, sampleCmds(), out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	data, _ := os.ReadFile(out)
	s := string(data)
	if !strings.HasPrefix(s, "#!/usr/bin/env bash\n") {
		t.Fatalf("missing shebang:\n%s", s)
	}
	if !strings.Contains(s, "set -e") {
		t.Fatalf("missing set -e:\n%s", s)
	}
	i1 := strings.Index(s, "go test ./...")
	i2 := strings.Index(s, "ls")
	if i1 < 0 || i2 < 0 || i1 > i2 {
		t.Fatalf("commands out of order:\n%s", s)
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755", info.Mode().Perm())
	}
}
