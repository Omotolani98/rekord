package tmux

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func skipNoTmux(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("tmux not available")
	}
}

func TestSessionLifecycle(t *testing.T) {
	skipNoTmux(t)
	ctx := context.Background()
	name := "rekord-test-" + time.Now().Format("150405.000000")
	name = strings.ReplaceAll(name, ".", "")

	if err := NewSession(ctx, name); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer KillSession(ctx, name)

	if !HasSession(ctx, name) {
		t.Fatal("HasSession = false after NewSession")
	}

	if err := exec.Command("tmux", "send-keys", "-t", name, "echo rekord-marker", "Enter").Run(); err != nil {
		t.Fatalf("send-keys: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	text, err := CapturePane(ctx, name)
	if err != nil {
		t.Fatalf("CapturePane: %v", err)
	}
	if !strings.Contains(text, "rekord-marker") {
		t.Fatalf("captured text missing marker:\n%s", text)
	}

	if err := KillSession(ctx, name); err != nil {
		t.Fatalf("KillSession: %v", err)
	}
	if HasSession(ctx, name) {
		t.Fatal("HasSession = true after KillSession")
	}
}
