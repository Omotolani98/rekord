package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
)

const secretValue = "sk-abcdef0123456789ABCDEF"

func TestScanCommandReportsCategories(t *testing.T) {
	root := t.TempDir()
	seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "export OPENAI_API_KEY=" + secretValue + "\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"scan", "demo", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "env-secret") && !strings.Contains(out, "openai-key") {
		t.Fatalf("no category reported:\n%s", out)
	}
	if strings.Contains(out, secretValue) {
		t.Fatalf("scan leaked raw secret:\n%s", out)
	}
}

func TestScanCommandStrict(t *testing.T) {
	root := t.TempDir()
	seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "password=hunter2\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"scan", "demo", "--root", root, "--strict"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero under --strict with secrets")
	}
}

func TestScanCommandClean(t *testing.T) {
	root := t.TempDir()
	seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "just normal output\r\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"scan", "demo", "--root", root, "--strict"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 for clean session; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No secrets detected.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
