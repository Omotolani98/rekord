package handoff

import (
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/session"
)

func TestGenerateSections(t *testing.T) {
	in := Input{
		Metadata: session.Metadata{Name: "demo", Shell: "/bin/zsh", CWD: "/p", DurationMS: 1000, Status: session.StatusCompleted},
		Commands: []commands.Command{{Index: 1, Command: "go test ./..."}},
		Output:   "ok run\n",
		Errors:   []string{"error: boom"},
	}
	out := Generate(in)
	for _, want := range []string{
		"# Rekord AI Context",
		"## Session",
		"- Name: demo",
		"## Commands Run",
		"1. go test ./...",
		"## Observed Output",
		"ok run",
		"## Possible Errors",
		"- error: boom",
		"## Suggested Summary",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "## Git") || strings.Contains(out, "## Project Tree") {
		t.Fatalf("unexpected optional section:\n%s", out)
	}
}

func TestGenerateOptionalSections(t *testing.T) {
	in := Input{
		Metadata: session.Metadata{Name: "demo"},
		Git:      &GitContext{Branch: "main", Status: " M file.go"},
		Tree:     "a.go\nb.go\n",
	}
	out := Generate(in)
	if !strings.Contains(out, "## Git") || !strings.Contains(out, "- Branch: main") {
		t.Fatalf("missing git section:\n%s", out)
	}
	if !strings.Contains(out, "## Project Tree") || !strings.Contains(out, "a.go") {
		t.Fatalf("missing tree section:\n%s", out)
	}
}

func TestGenerateNoCommands(t *testing.T) {
	out := Generate(Input{Metadata: session.Metadata{Name: "demo"}})
	if !strings.Contains(out, "_No commands extracted._") {
		t.Fatalf("missing placeholder:\n%s", out)
	}
}
