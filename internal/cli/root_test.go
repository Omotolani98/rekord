package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Execute returned %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help output missing usage:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "start") {
		t.Fatalf("help output missing commands:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestExecuteVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Execute returned %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "rekord "+version {
		t.Fatalf("version output = %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"missing"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("Execute returned success for unknown command")
	}
	if !strings.Contains(stderr.String(), `unknown command "missing"`) {
		t.Fatalf("stderr missing unknown command message:\n%s", stderr.String())
	}
}

func TestExecuteCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"export", "--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Execute returned %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "--to") {
		t.Fatalf("export help missing --to flag:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestExecutePlaceholderCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"replay", "some-session"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("Execute returned %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "rekord replay is not implemented yet") {
		t.Fatalf("stderr missing not implemented message:\n%s", stderr.String())
	}
}

func TestExecuteNestedPlaceholderCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"tmux", "start", "--session", "demo"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("Execute returned %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "rekord tmux start is not implemented yet") {
		t.Fatalf("stderr missing nested not implemented message:\n%s", stderr.String())
	}
}
