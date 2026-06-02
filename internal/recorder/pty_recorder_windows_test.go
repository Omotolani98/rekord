//go:build windows

package recorder

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
)

func TestResolveShellWindows(t *testing.T) {
	if got := resolveShellWindows("pwsh.exe"); got != "pwsh.exe" {
		t.Fatalf("got %q, want pwsh.exe", got)
	}
	if got := resolveShellWindows(""); got != defaultWindowsShell {
		t.Fatalf("got %q, want %q", got, defaultWindowsShell)
	}
}

func TestPTYRecorderWindowsMissingEventsPathFails(t *testing.T) {
	r := NewPTYRecorder()
	_, err := r.Record(context.Background(), Options{
		Stdin:  strings.NewReader(""),
		Stdout: &bytes.Buffer{},
	})
	if err == nil || !strings.Contains(err.Error(), "EventsPath") {
		t.Fatalf("err = %v, want EventsPath error", err)
	}
}

func TestPTYRecorderWindowsRecordsOutputEvents(t *testing.T) {
	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { _ = stdinW.Close() })
	var stdout bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	r := NewPTYRecorder()
	res, err := r.Record(ctx, Options{
		Command:    []string{"cmd.exe", "/c", "echo rekord-test-marker"},
		EventsPath: eventsPath,
		Stdin:      stdinR,
		Stdout:     &stdout,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.StartedAt.IsZero() || res.EndedAt.IsZero() {
		t.Fatalf("timestamps zero: %+v", res)
	}

	evs, err := events.ReadAll(eventsPath)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(evs) == 0 {
		t.Fatal("no events recorded")
	}
	var joined strings.Builder
	for _, e := range evs {
		if e.Type == events.TypeOutput {
			joined.WriteString(e.Data)
		}
	}
	if !strings.Contains(joined.String(), "rekord-test-marker") {
		t.Fatalf("events missing marker:\n%s", joined.String())
	}
}
