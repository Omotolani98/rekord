package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
)

func TestReplayCommandOutputsInOrder(t *testing.T) {
	root := t.TempDir()
	seedSession(t, root, "demo", []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "one\n"},
		{TimeMS: 1, Type: events.TypeResize, Cols: 80, Rows: 24},
		{TimeMS: 2, Type: events.TypeOutput, Data: "two\n"},
	})

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"replay", "demo", "--root", root, "--speed", "1000"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	got := stdout.String()
	if got != "one\ntwo\n" {
		t.Fatalf("output = %q, want %q", got, "one\ntwo\n")
	}
	if strings.Contains(got, "resize") {
		t.Fatalf("output leaked non-output event: %q", got)
	}
}

func TestReplayCommandMissingSession(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"replay", "nope", "--root", root}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for missing session")
	}
}
