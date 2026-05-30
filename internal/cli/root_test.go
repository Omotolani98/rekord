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
