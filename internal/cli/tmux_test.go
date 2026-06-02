package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
)

func tmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func newTmuxTestSession(t *testing.T) string {
	t.Helper()
	name := "rekord-cli-" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
	if err := exec.Command("tmux", "new-session", "-d", "-s", name).Run(); err != nil {
		t.Fatalf("new-session: %v", err)
	}
	t.Cleanup(func() { _ = exec.Command("tmux", "kill-session", "-t", name).Run() })
	return name
}

func TestTmuxStatusNotInside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tmux is not available on windows")
	}
	t.Setenv("TMUX", "")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"tmux", "status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Not inside a tmux session.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTmuxCapture(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}
	sess := newTmuxTestSession(t)
	if err := exec.Command("tmux", "send-keys", "-t", sess, "echo cap-marker", "Enter").Run(); err != nil {
		t.Fatalf("send-keys: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"tmux", "capture", "--pane", sess, "--name", "cap", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}

	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("session dirs = %d, want 1", len(entries))
	}
	evs, err := events.ReadAll(filepath.Join(root, entries[0].Name(), "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var joined strings.Builder
	for _, e := range evs {
		joined.WriteString(e.Data)
	}
	if !strings.Contains(joined.String(), "cap-marker") {
		t.Fatalf("captured events missing marker:\n%s", joined.String())
	}
}

func TestTmuxRecordStopsOnEOF(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}
	sess := newTmuxTestSession(t)
	root := t.TempDir()

	cmd := NewRootCommand(new(bytes.Buffer), new(bytes.Buffer))
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"tmux", "record", "--pane", sess, "--name", "rec", "--root", root})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("record: %v", err)
	}

	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("session dirs = %d, want 1", len(entries))
	}
	if _, err := os.Stat(filepath.Join(root, entries[0].Name(), "events.jsonl")); err != nil {
		t.Fatalf("events.jsonl missing: %v", err)
	}
}
