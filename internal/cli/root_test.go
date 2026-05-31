package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootCommandName(t *testing.T) {
	orig := os.Args[0]
	t.Cleanup(func() { os.Args[0] = orig })

	for _, tc := range []struct{ argv0, want string }{
		{"/usr/local/bin/rk", "rk"},
		{"/usr/local/bin/rekord", "rekord"},
	} {
		os.Args[0] = tc.argv0
		if got := NewRootCommand(io.Discard, io.Discard).Name(); got != tc.want {
			t.Errorf("argv0 %q: root name = %q, want %q", tc.argv0, got, tc.want)
		}
	}
}

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
