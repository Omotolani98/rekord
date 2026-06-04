//go:build !windows

package live

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func requireCmd(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available", name)
	}
}

func TestSessionLaunchSendCapture(t *testing.T) {
	requireCmd(t, "cat")

	root := t.TempDir()
	h := NewHub(root, "test")
	defer h.Shutdown()

	s, err := h.Launch(context.Background(), LaunchOptions{
		Name:    "demo",
		Command: []string{"cat"},
		Cols:    40,
		Rows:    10,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if err := s.SendInput([]byte("hello-rekord\n")); err != nil {
		t.Fatalf("SendInput: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reason, f, err := s.WaitForText(ctx, "hello-rekord")
	if err != nil {
		t.Fatalf("WaitForText: %v", err)
	}
	if reason != "matched" {
		t.Fatalf("reason = %q, frame:\n%s", reason, f.Text())
	}
	if !strings.Contains(s.Capture().Text(), "hello-rekord") {
		t.Fatalf("capture missing text:\n%s", s.Capture().Text())
	}
	if !s.Status().Running {
		t.Fatal("status not running while cat alive")
	}

	id := s.ID()
	if err := h.Stop("demo"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	store := session.NewFileStore(root)
	m, err := store.ReadMetadata(context.Background(), id)
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if m.Status != session.StatusCompleted {
		t.Fatalf("status = %q, want completed", m.Status)
	}

	evs, err := events.ReadAll(filepath.Join(root, id, "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var sawInput bool
	for _, e := range evs {
		if e.Type == events.TypeInput && strings.Contains(e.Data, "hello-rekord") {
			sawInput = true
		}
	}
	if !sawInput {
		t.Fatal("no input event recorded")
	}
}

func TestSessionWaitForExit(t *testing.T) {
	requireCmd(t, "sh")

	root := t.TempDir()
	h := NewHub(root, "test")
	defer h.Shutdown()

	s, err := h.Launch(context.Background(), LaunchOptions{
		Name:    "quick",
		Command: []string{"sh", "-c", "echo done"},
		Cols:    20,
		Rows:    5,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	code, reason, err := s.WaitForExit(ctx)
	if err != nil {
		t.Fatalf("WaitForExit: %v", err)
	}
	if reason != "exited" {
		t.Fatalf("reason = %q, want exited", reason)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if st := s.Status(); st.Running || st.ExitCode == nil || *st.ExitCode != 0 {
		t.Fatalf("status = %+v, want stopped exit 0", st)
	}
}

func TestSessionWaitForIdle(t *testing.T) {
	requireCmd(t, "cat")

	root := t.TempDir()
	h := NewHub(root, "test")
	defer h.Shutdown()

	s, err := h.Launch(context.Background(), LaunchOptions{
		Name:    "idle",
		Command: []string{"cat"},
		Cols:    20,
		Rows:    5,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reason, _, err := s.WaitForIdle(ctx, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForIdle: %v", err)
	}
	if reason != "idle" {
		t.Fatalf("reason = %q, want idle", reason)
	}
}

func TestHubLaunchValidation(t *testing.T) {
	h := NewHub(t.TempDir(), "test")
	defer h.Shutdown()

	if _, err := h.Launch(context.Background(), LaunchOptions{Name: "x", Command: nil}); err == nil {
		t.Fatal("expected error for missing command")
	}

	requireCmd(t, "cat")
	if _, err := h.Launch(context.Background(), LaunchOptions{Name: "dup", Command: []string{"cat"}}); err != nil {
		t.Fatalf("first launch: %v", err)
	}
	if _, err := h.Launch(context.Background(), LaunchOptions{Name: "dup", Command: []string{"cat"}}); err == nil {
		t.Fatal("expected duplicate-name error")
	}
}
