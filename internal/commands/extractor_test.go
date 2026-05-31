package commands

import (
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
)

func mustExtractor(t *testing.T) *Extractor {
	t.Helper()
	compiled, err := CompilePatterns(DefaultPatterns())
	if err != nil {
		t.Fatalf("CompilePatterns: %v", err)
	}
	return NewExtractor(compiled)
}

func TestExtractBasicCommands(t *testing.T) {
	evs := []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ go te"},
		{TimeMS: 10, Type: events.TypeOutput, Data: "st ./...\r\n"},
		{TimeMS: 20, Type: events.TypeOutput, Data: "ok github.com/example/app 0.231s\r\n"},
		{TimeMS: 30, Type: events.TypeOutput, Data: "$ \r\n"},
		{TimeMS: 40, Type: events.TypeOutput, Data: "❯ ls\r\n"},
		{TimeMS: 50, Type: events.TypeOutput, Data: "a.go b.go\r\n"},
	}

	cmds := mustExtractor(t).Extract(evs)
	if len(cmds) != 2 {
		t.Fatalf("len(cmds) = %d, want 2 (empty prompt ignored)", len(cmds))
	}

	if cmds[0].Index != 1 || cmds[0].Command != "go test ./..." {
		t.Fatalf("cmd0 = %+v, want index 1 'go test ./...'", cmds[0])
	}
	if cmds[0].StartedAtMs != 0 {
		t.Fatalf("cmd0 StartedAtMs = %d, want 0", cmds[0].StartedAtMs)
	}
	if cmds[0].OutputPreview != "ok github.com/example/app 0.231s" {
		t.Fatalf("cmd0 preview = %q", cmds[0].OutputPreview)
	}
	if cmds[0].ExitCode != nil {
		t.Fatalf("cmd0 ExitCode = %v, want nil", *cmds[0].ExitCode)
	}

	if cmds[1].Index != 2 || cmds[1].Command != "ls" {
		t.Fatalf("cmd1 = %+v, want index 2 'ls'", cmds[1])
	}
	if cmds[1].StartedAtMs != 40 {
		t.Fatalf("cmd1 StartedAtMs = %d, want 40", cmds[1].StartedAtMs)
	}
	if cmds[0].EndedAtMs != cmds[1].StartedAtMs {
		t.Fatalf("cmd0 EndedAtMs = %d, want %d", cmds[0].EndedAtMs, cmds[1].StartedAtMs)
	}
}

func TestExtractNoCommands(t *testing.T) {
	evs := []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "just some output\r\n"},
	}
	if cmds := mustExtractor(t).Extract(evs); len(cmds) != 0 {
		t.Fatalf("len(cmds) = %d, want 0", len(cmds))
	}
}

func TestCompilePatternsInvalid(t *testing.T) {
	if _, err := CompilePatterns([]string{"("}); err == nil {
		t.Fatal("CompilePatterns err = nil, want error for invalid regex")
	}
}
