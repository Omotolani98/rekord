//go:build !windows

package agent

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Omotolani98/rekord/internal/live"
	"github.com/Omotolani98/rekord/internal/redact"
)

func defaultRedactorForTest() *redact.Redactor {
	return redact.NewDefault()
}

func TestMCPMemoryToolsSupportAgentHandoff(t *testing.T) {
	t.Setenv("REKORD_MEMORY_ROOT", t.TempDir())
	project := t.TempDir()
	d := &deps{}
	ctx := context.Background()

	_, written, err := d.memoryWrite(ctx, nil, MemoryWriteInput{
		Project: project,
		Agent:   "claude",
		Session: "memory-claude",
		Body:    "Claude stopped at the failing parser test",
		Tags:    []string{"parser"},
	})
	if err != nil {
		t.Fatalf("memoryWrite: %v", err)
	}
	if written.Agent != "claude" || written.SessionName != "memory-claude" {
		t.Fatalf("written memory = %+v", written)
	}

	_, found, err := d.memorySearch(ctx, nil, MemorySearchInput{Project: project, Query: "parser", Agent: "claude"})
	if err != nil {
		t.Fatalf("memorySearch: %v", err)
	}
	if len(found.Memories) != 1 || found.Memories[0].ID != written.ID {
		t.Fatalf("found = %+v, want %s", found.Memories, written.ID)
	}

	_, rc, err := d.resumeContext(ctx, nil, ResumeContextInput{Project: project, FromAgent: "claude", ToAgent: "codex"})
	if err != nil {
		t.Fatalf("resumeContext: %v", err)
	}
	if !strings.Contains(rc.Summary, "Claude stopped") || !strings.Contains(rc.Summary, "Intended next agent: codex") {
		t.Fatalf("resume summary:\n%s", rc.Summary)
	}

	_, resolved, err := d.memoryResolve(ctx, nil, MemoryGetInput{Project: project, ID: written.ID})
	if err != nil {
		t.Fatalf("memoryResolve: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("resolved status = %q", resolved.Status)
	}
}

func TestMCPSnapshotCreateWritesPatch(t *testing.T) {
	requireCmd(t, "git")
	t.Setenv("REKORD_MEMORY_ROOT", t.TempDir())
	project := t.TempDir()
	runAgentGit(t, project, "init")
	runAgentGit(t, project, "config", "user.email", "test@example.com")
	runAgentGit(t, project, "config", "user.name", "Test")
	writeAgentFile(t, filepath.Join(project, "README.md"), "hello\n")
	runAgentGit(t, project, "add", "README.md")
	runAgentGit(t, project, "commit", "-m", "init")
	writeAgentFile(t, filepath.Join(project, "README.md"), "hello\nagent\n")

	d := &deps{}
	_, snap, err := d.snapshotCreate(context.Background(), nil, SnapshotCreateInput{Project: project, Agent: "claude", Session: "memory-claude", Note: "handoff point"})
	if err != nil {
		t.Fatalf("snapshotCreate: %v", err)
	}
	if snap.Agent != "claude" || len(snap.Patches) != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
	data, err := os.ReadFile(snap.Patches[0].Path)
	if err != nil {
		t.Fatalf("read patch: %v", err)
	}
	if !strings.Contains(string(data), "+agent") {
		t.Fatalf("patch missing agent change:\n%s", data)
	}
}

func runAgentGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeAgentFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireCmd(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available", name)
	}
}

func newTestClient(t *testing.T, hub *live.Hub) *mcp.ClientSession {
	t.Helper()
	serverT, clientT := mcp.NewInMemoryTransports()
	srv := NewServer(hub, nil, "test")

	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %s", name, textOf(res))
	}
	if out != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("marshal structured: %v", err)
		}
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("unmarshal structured into %T: %v", out, err)
		}
	}
	return res
}

func textOf(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestMCPDriveSession(t *testing.T) {
	requireCmd(t, "cat")

	hub := live.NewHub(t.TempDir(), "test")
	defer hub.Shutdown()
	cs := newTestClient(t, hub)

	var st live.Status
	call(t, cs, "launch", map[string]any{
		"name":    "demo",
		"command": []string{"cat"},
		"cols":    40,
		"rows":    10,
	}, &st)
	if st.Name != "demo" || !st.Running {
		t.Fatalf("launch status = %+v", st)
	}

	var sent SendOutput
	call(t, cs, "send", map[string]any{"name": "demo", "text": "hello-mcp\n"}, &sent)
	if sent.Sent == 0 {
		t.Fatal("send reported 0 bytes")
	}

	var wt WaitOutput
	call(t, cs, "wait_text", map[string]any{"name": "demo", "text": "hello-mcp", "timeoutMs": 3000}, &wt)
	if wt.Reason != "matched" {
		t.Fatalf("wait_text reason = %q, frame:\n%s", wt.Reason, wt.Frame.Text())
	}

	var f struct {
		Lines []string `json:"lines"`
	}
	cres := call(t, cs, "capture", map[string]any{"name": "demo"}, &f)
	if !strings.Contains(textOf(cres), "hello-mcp") {
		t.Fatalf("capture text missing marker:\n%s", textOf(cres))
	}

	var list ListOutput
	call(t, cs, "list", map[string]any{}, &list)
	if len(list.Sessions) != 1 {
		t.Fatalf("list = %d sessions, want 1", len(list.Sessions))
	}

	var stopped StopOutput
	call(t, cs, "stop", map[string]any{"name": "demo"}, &stopped)
	if !stopped.Stopped {
		t.Fatal("stop did not report stopped")
	}
}

func TestMCPCaptureRedaction(t *testing.T) {
	requireCmd(t, "cat")

	hub := live.NewHub(t.TempDir(), "test")
	defer hub.Shutdown()

	serverT, clientT := mcp.NewInMemoryTransports()
	srv := NewServer(hub, defaultRedactorForTest(), "test")
	ctx := context.Background()
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	call(t, cs, "launch", map[string]any{"name": "r", "command": []string{"cat"}, "cols": 60, "rows": 5}, nil)
	call(t, cs, "send", map[string]any{"name": "r", "text": "token=supersecretvalue\n"}, nil)
	var wt WaitOutput
	call(t, cs, "wait_text", map[string]any{"name": "r", "text": "token=", "timeoutMs": 3000}, &wt)

	cres := call(t, cs, "capture", map[string]any{"name": "r"}, nil)
	if strings.Contains(textOf(cres), "supersecretvalue") {
		t.Fatalf("secret leaked in redacted capture:\n%s", textOf(cres))
	}

	raw := call(t, cs, "capture", map[string]any{"name": "r", "raw": true}, nil)
	if !strings.Contains(textOf(raw), "supersecretvalue") {
		t.Fatalf("raw capture should contain secret:\n%s", textOf(raw))
	}
}
