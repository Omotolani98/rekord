package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestMemoryCLIRememberRecallResume(t *testing.T) {
	root := t.TempDir()
	project := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"remember", "--memory-root", root, "--project", project, "--agent", "claude", "--session", "memory-claude", "Refresh token test still failing"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("remember code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "remembered mem_") {
		t.Fatalf("remember output = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"recall", "--memory-root", root, "--project", project, "--agent", "claude", "--session", "memory-claude", "refresh"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("recall code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Refresh token test still failing") {
		t.Fatalf("recall output = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"resume", "--memory-root", root, "--project", project, "--agent", "claude"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resume code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Context from agent: claude") || !strings.Contains(stdout.String(), "Refresh token test still failing") {
		t.Fatalf("resume output = %q", stdout.String())
	}
}
