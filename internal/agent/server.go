package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Omotolani98/rekord/internal/frame"
	"github.com/Omotolani98/rekord/internal/live"
	"github.com/Omotolani98/rekord/internal/redact"
)

const (
	defaultTimeout   = 10 * time.Second
	defaultQuiet     = 500 * time.Millisecond
	defaultLogsBytes = 65536
)

type deps struct {
	hub      *live.Hub
	redactor *redact.Redactor
}

func NewServer(hub *live.Hub, redactor *redact.Redactor, version string) *mcp.Server {
	d := &deps{hub: hub, redactor: redactor}
	srv := mcp.NewServer(&mcp.Implementation{Name: "rekord", Version: version}, nil)
	d.register(srv)
	return srv
}

func (d *deps) register(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "launch",
		Description: "Launch a terminal program in a new recorded session driven by this server.",
	}, d.launch)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "send",
		Description: "Send input to a session: literal text and/or named keys (enter, tab, esc, up, down, left, right, ctrl-c, ...).",
	}, d.send)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "capture",
		Description: "Capture the current deterministic screen frame (character grid + cursor). Redacted unless raw is true.",
	}, d.capture)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "wait_text",
		Description: "Wait until the screen contains the given text, the process exits, or the timeout elapses.",
	}, d.waitText)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "wait_idle",
		Description: "Wait until output has been quiet for quietMs, the process exits, or the timeout elapses.",
	}, d.waitIdle)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "wait_exit",
		Description: "Wait until the session process exits or the timeout elapses.",
	}, d.waitExit)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "logs",
		Description: "Return the retained output transcript for a session. Redacted unless raw is true.",
	}, d.logs)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "resize",
		Description: "Resize a session's terminal viewport.",
	}, d.resize)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "stop",
		Description: "Terminate a session and finalize its recording.",
	}, d.stop)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list",
		Description: "List active and finished sessions managed by this server.",
	}, d.list)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "status",
		Description: "Report the status of a single session.",
	}, d.status)
}

type LaunchInput struct {
	Name    string   `json:"name" jsonschema:"unique session name"`
	Command []string `json:"command" jsonschema:"program and arguments to run"`
	Cols    int      `json:"cols,omitempty" jsonschema:"terminal columns (default 80)"`
	Rows    int      `json:"rows,omitempty" jsonschema:"terminal rows (default 24)"`
	CWD     string   `json:"cwd,omitempty" jsonschema:"working directory"`
	Env     []string `json:"env,omitempty" jsonschema:"environment variables as KEY=VALUE"`
}

func (d *deps) launch(ctx context.Context, _ *mcp.CallToolRequest, in LaunchInput) (*mcp.CallToolResult, live.Status, error) {
	s, err := d.hub.Launch(ctx, live.LaunchOptions{
		Name:    in.Name,
		Command: in.Command,
		CWD:     in.CWD,
		Env:     in.Env,
		Cols:    in.Cols,
		Rows:    in.Rows,
	})
	if err != nil {
		return nil, live.Status{}, err
	}
	st := s.Status()
	return text(fmt.Sprintf("launched %q (id %s)", st.Name, st.ID)), st, nil
}

type SendInput struct {
	Name string   `json:"name"`
	Text string   `json:"text,omitempty" jsonschema:"literal text to type"`
	Keys []string `json:"keys,omitempty" jsonschema:"named keys to send after text"`
}

type SendOutput struct {
	Sent int `json:"sent" jsonschema:"number of bytes sent"`
}

func (d *deps) send(_ context.Context, _ *mcp.CallToolRequest, in SendInput) (*mcp.CallToolResult, SendOutput, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, SendOutput{}, notFound(in.Name)
	}
	buf, err := live.BuildInput(in.Text, in.Keys)
	if err != nil {
		return nil, SendOutput{}, err
	}
	if err := s.SendInput(buf); err != nil {
		return nil, SendOutput{}, err
	}
	return text(fmt.Sprintf("sent %d bytes", len(buf))), SendOutput{Sent: len(buf)}, nil
}

type CaptureInput struct {
	Name string `json:"name"`
	Raw  bool   `json:"raw,omitempty" jsonschema:"return unredacted text (sensitive)"`
}

func (d *deps) capture(_ context.Context, _ *mcp.CallToolRequest, in CaptureInput) (*mcp.CallToolResult, frame.Frame, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, frame.Frame{}, notFound(in.Name)
	}
	f := d.maybeRedact(s.Capture(), in.Raw)
	return text(f.Text()), f, nil
}

type WaitTextInput struct {
	Name      string `json:"name"`
	Text      string `json:"text" jsonschema:"substring to wait for"`
	TimeoutMs int    `json:"timeoutMs,omitempty" jsonschema:"timeout in milliseconds (default 10000)"`
	Raw       bool   `json:"raw,omitempty"`
}

type WaitOutput struct {
	Reason string      `json:"reason" jsonschema:"matched, idle, exited, or deadline"`
	Frame  frame.Frame `json:"frame"`
}

func (d *deps) waitText(ctx context.Context, _ *mcp.CallToolRequest, in WaitTextInput) (*mcp.CallToolResult, WaitOutput, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, WaitOutput{}, notFound(in.Name)
	}
	wctx, cancel := withTimeout(ctx, in.TimeoutMs)
	defer cancel()
	reason, f, err := s.WaitForText(wctx, in.Text)
	if err != nil {
		return nil, WaitOutput{}, err
	}
	f = d.maybeRedact(f, in.Raw)
	return text(reason), WaitOutput{Reason: reason, Frame: f}, nil
}

type WaitIdleInput struct {
	Name      string `json:"name"`
	QuietMs   int    `json:"quietMs,omitempty" jsonschema:"required quiet period in milliseconds (default 500)"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
	Raw       bool   `json:"raw,omitempty"`
}

func (d *deps) waitIdle(ctx context.Context, _ *mcp.CallToolRequest, in WaitIdleInput) (*mcp.CallToolResult, WaitOutput, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, WaitOutput{}, notFound(in.Name)
	}
	quiet := defaultQuiet
	if in.QuietMs > 0 {
		quiet = time.Duration(in.QuietMs) * time.Millisecond
	}
	wctx, cancel := withTimeout(ctx, in.TimeoutMs)
	defer cancel()
	reason, f, err := s.WaitForIdle(wctx, quiet)
	if err != nil {
		return nil, WaitOutput{}, err
	}
	f = d.maybeRedact(f, in.Raw)
	return text(reason), WaitOutput{Reason: reason, Frame: f}, nil
}

type WaitExitInput struct {
	Name      string `json:"name"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
}

type WaitExitOutput struct {
	Reason   string `json:"reason"`
	ExitCode int    `json:"exitCode"`
}

func (d *deps) waitExit(ctx context.Context, _ *mcp.CallToolRequest, in WaitExitInput) (*mcp.CallToolResult, WaitExitOutput, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, WaitExitOutput{}, notFound(in.Name)
	}
	wctx, cancel := withTimeout(ctx, in.TimeoutMs)
	defer cancel()
	code, reason, err := s.WaitForExit(wctx)
	if err != nil {
		return nil, WaitExitOutput{}, err
	}
	return text(fmt.Sprintf("%s (exit %d)", reason, code)), WaitExitOutput{Reason: reason, ExitCode: code}, nil
}

type LogsInput struct {
	Name     string `json:"name"`
	MaxBytes int    `json:"maxBytes,omitempty" jsonschema:"max trailing bytes to return (default 65536)"`
	Raw      bool   `json:"raw,omitempty"`
}

type LogsOutput struct {
	Logs string `json:"logs"`
}

func (d *deps) logs(_ context.Context, _ *mcp.CallToolRequest, in LogsInput) (*mcp.CallToolResult, LogsOutput, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, LogsOutput{}, notFound(in.Name)
	}
	max := in.MaxBytes
	if max <= 0 {
		max = defaultLogsBytes
	}
	out := s.Logs(max)
	if !in.Raw && d.redactor != nil {
		out = d.redactor.Redact(out)
	}
	return text(out), LogsOutput{Logs: out}, nil
}

type ResizeInput struct {
	Name string `json:"name"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func (d *deps) resize(_ context.Context, _ *mcp.CallToolRequest, in ResizeInput) (*mcp.CallToolResult, live.Status, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, live.Status{}, notFound(in.Name)
	}
	if err := s.Resize(in.Cols, in.Rows); err != nil {
		return nil, live.Status{}, err
	}
	st := s.Status()
	return text(fmt.Sprintf("resized to %dx%d", st.Cols, st.Rows)), st, nil
}

type StopInput struct {
	Name string `json:"name"`
}

type StopOutput struct {
	Stopped bool `json:"stopped"`
}

func (d *deps) stop(_ context.Context, _ *mcp.CallToolRequest, in StopInput) (*mcp.CallToolResult, StopOutput, error) {
	if err := d.hub.Stop(in.Name); err != nil {
		return nil, StopOutput{}, err
	}
	return text(fmt.Sprintf("stopped %q", in.Name)), StopOutput{Stopped: true}, nil
}

type ListOutput struct {
	Sessions []live.Status `json:"sessions"`
}

func (d *deps) list(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListOutput, error) {
	sessions := d.hub.List()
	return text(fmt.Sprintf("%d session(s)", len(sessions))), ListOutput{Sessions: sessions}, nil
}

type StatusInput struct {
	Name string `json:"name"`
}

func (d *deps) status(_ context.Context, _ *mcp.CallToolRequest, in StatusInput) (*mcp.CallToolResult, live.Status, error) {
	s, ok := d.hub.Get(in.Name)
	if !ok {
		return nil, live.Status{}, notFound(in.Name)
	}
	st := s.Status()
	return text(fmt.Sprintf("%q running=%v", st.Name, st.Running)), st, nil
}

func (d *deps) maybeRedact(f frame.Frame, raw bool) frame.Frame {
	if raw || d.redactor == nil {
		return f
	}
	return f.Redact(d.redactor)
}

func withTimeout(ctx context.Context, ms int) (context.Context, context.CancelFunc) {
	d := defaultTimeout
	if ms > 0 {
		d = time.Duration(ms) * time.Millisecond
	}
	return context.WithTimeout(ctx, d)
}

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

func notFound(name string) error {
	return fmt.Errorf("session not found: %q", name)
}
