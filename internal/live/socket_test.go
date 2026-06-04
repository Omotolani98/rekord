//go:build !windows

package live

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServeAndClient(t *testing.T) {
	requireCmd(t, "cat")

	root := t.TempDir()
	h := NewHub(root, "test")
	defer h.Shutdown()

	s, err := h.Launch(context.Background(), LaunchOptions{
		Name:    "sock",
		Command: []string{"cat"},
		Cols:    40,
		Rows:    8,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	sock := filepath.Join(root, "sock.sock")
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, sock, s) }()

	waitSocket(t, sock)

	if resp, err := Do(sock, Request{Op: "send", Text: "hello-sock\n"}); err != nil || resp.Sent == 0 {
		t.Fatalf("send: resp=%+v err=%v", resp, err)
	}

	resp, err := Do(sock, Request{Op: "wait_text", Sub: "hello-sock", TimeoutMs: 3000})
	if err != nil {
		t.Fatalf("wait_text: %v", err)
	}
	if resp.Reason != "matched" {
		t.Fatalf("reason = %q", resp.Reason)
	}

	cap, err := Do(sock, Request{Op: "capture"})
	if err != nil || cap.Frame == nil {
		t.Fatalf("capture: resp=%+v err=%v", cap, err)
	}
	if !strings.Contains(cap.Frame.Text(), "hello-sock") {
		t.Fatalf("capture missing text:\n%s", cap.Frame.Text())
	}

	st, err := Do(sock, Request{Op: "status"})
	if err != nil || st.Status == nil || !st.Status.Running {
		t.Fatalf("status: resp=%+v err=%v", st, err)
	}

	if _, err := Do(sock, Request{Op: "stop"}); err != nil {
		t.Fatalf("stop: %v", err)
	}

	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("Serve returned: %v", err)
		}
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("Serve did not return after stop")
	}
	cancel()
}

func waitSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if Ping(path) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("socket never came up")
}
