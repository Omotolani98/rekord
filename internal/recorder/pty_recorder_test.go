//go:build !windows

package recorder

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/creack/pty"
)

func TestResolveShell(t *testing.T) {
	t.Run("explicit override wins", func(t *testing.T) {
		t.Setenv("SHELL", "/from/env")
		if got := resolveShell("/explicit/sh"); got != "/explicit/sh" {
			t.Fatalf("got %q, want /explicit/sh", got)
		}
	})
	t.Run("env fallback", func(t *testing.T) {
		t.Setenv("SHELL", "/env/sh")
		if got := resolveShell(""); got != "/env/sh" {
			t.Fatalf("got %q, want /env/sh", got)
		}
	})
	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("SHELL", "")
		if got := resolveShell(""); got != defaultShell {
			t.Fatalf("got %q, want %q", got, defaultShell)
		}
	})
}

func TestPTYRecorderMissingEventsPathFails(t *testing.T) {
	r := NewPTYRecorder()
	_, err := r.Record(context.Background(), Options{
		Stdin:  strings.NewReader(""),
		Stdout: &bytes.Buffer{},
	})
	if err == nil || !strings.Contains(err.Error(), "EventsPath") {
		t.Fatalf("err = %v, want EventsPath error", err)
	}
}

func TestPTYRecorderRecordsOutputEvents(t *testing.T) {
	requireSh(t)

	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	stdinR, stdinW := io.Pipe()
	var stdout bytes.Buffer

	go func() {
		_, _ = stdinW.Write([]byte("echo rekord-test-marker\nexit\n"))
		_ = stdinW.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := NewPTYRecorder()
	res, err := r.Record(ctx, Options{
		Shell:      "/bin/sh",
		EventsPath: eventsPath,
		Stdin:      stdinR,
		Stdout:     &stdout,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if res.DurationMS < 0 {
		t.Fatalf("DurationMS = %d, want >= 0", res.DurationMS)
	}
	if res.StartedAt.IsZero() || res.EndedAt.IsZero() {
		t.Fatalf("timestamps zero: %+v", res)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}

	if !strings.Contains(stdout.String(), "rekord-test-marker") {
		t.Fatalf("stdout missing marker:\n%s", stdout.String())
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
		if e.Type != events.TypeOutput {
			t.Fatalf("unexpected event type %q", e.Type)
		}
		if e.TimeMS < 0 {
			t.Fatalf("negative TimeMS: %d", e.TimeMS)
		}
		joined.WriteString(e.Data)
	}
	if !strings.Contains(joined.String(), "rekord-test-marker") {
		t.Fatalf("events missing marker:\n%s", joined.String())
	}
}

func TestPTYRecorderContextCancelKillsShell(t *testing.T) {
	requireSh(t)

	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { _ = stdinW.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	r := NewPTYRecorder()
	_, err := r.Record(ctx, Options{
		Shell:      "/bin/sh",
		EventsPath: eventsPath,
		Stdin:      stdinR,
		Stdout:     io.Discard,
		KillGrace:  100 * time.Millisecond,
	})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Record took %v, want < 2s", elapsed)
	}
}

func TestPTYRecorderEmitsInitialResize(t *testing.T) {
	requireSh(t)
	master, slave := openPTYPair(t)

	if err := pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}); err != nil {
		t.Fatalf("Setsize master: %v", err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = master.Write([]byte("exit\n"))
	}()

	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := NewPTYRecorder()
	_, err := r.Record(ctx, Options{
		Shell:      "/bin/sh",
		EventsPath: eventsPath,
		Stdin:      slave,
		Stdout:     io.Discard,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	evs, err := events.ReadAll(eventsPath)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var first *events.Event
	for i := range evs {
		if evs[i].Type == events.TypeResize {
			first = &evs[i]
			break
		}
	}
	if first == nil {
		t.Fatal("no resize event found")
	}
	if first.TimeMS != 0 {
		t.Fatalf("initial resize TimeMS = %d, want 0", first.TimeMS)
	}
	if first.Cols != 120 || first.Rows != 40 {
		t.Fatalf("initial resize size = %dx%d, want 120x40", first.Cols, first.Rows)
	}
}

func TestPTYRecorderHandlesSIGWINCH(t *testing.T) {
	requireSh(t)
	master, slave := openPTYPair(t)

	if err := pty.Setsize(master, &pty.Winsize{Cols: 80, Rows: 24}); err != nil {
		t.Fatalf("Setsize master: %v", err)
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = pty.Setsize(master, &pty.Winsize{Cols: 100, Rows: 30})
		_ = syscall.Kill(os.Getpid(), syscall.SIGWINCH)
		time.Sleep(200 * time.Millisecond)
		_, _ = master.Write([]byte("exit\n"))
	}()

	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := NewPTYRecorder()
	_, err := r.Record(ctx, Options{
		Shell:      "/bin/sh",
		EventsPath: eventsPath,
		Stdin:      slave,
		Stdout:     io.Discard,
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	evs, err := events.ReadAll(eventsPath)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var resizes []events.Event
	for _, e := range evs {
		if e.Type == events.TypeResize {
			resizes = append(resizes, e)
		}
	}
	if len(resizes) < 2 {
		t.Fatalf("resize event count = %d, want >= 2; events=%v", len(resizes), resizes)
	}
	last := resizes[len(resizes)-1]
	if last.Cols != 100 || last.Rows != 30 {
		t.Fatalf("last resize = %dx%d, want 100x30", last.Cols, last.Rows)
	}
	if last.TimeMS <= 0 {
		t.Fatalf("last resize TimeMS = %d, want > 0", last.TimeMS)
	}
}

func openPTYPair(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = slave.Close()
		_ = master.Close()
	})
	return master, slave
}

func requireSh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}
}
