package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Omotolani98/rekord/internal/frame"
	"github.com/Omotolani98/rekord/internal/live"
	mem "github.com/Omotolani98/rekord/internal/memory"
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

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "memory_write",
		Description: "Write a durable project memory that can be recalled by future agents.",
	}, d.memoryWrite)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "memory_search",
		Description: "Search durable project memory by query, agent, or session.",
	}, d.memorySearch)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "memory_list",
		Description: "List durable project memories, newest first.",
	}, d.memoryList)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "memory_get",
		Description: "Return a single project memory by id.",
	}, d.memoryGet)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "memory_resolve",
		Description: "Mark a project memory as resolved.",
	}, d.memoryResolve)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "snapshot_create",
		Description: "Create a resumable project snapshot with git patch files.",
	}, d.snapshotCreate)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "resume_context",
		Description: "Return agent-ready context for resuming project work from memory.",
	}, d.resumeContext)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "memory_projects",
		Description: "List projects that have stored memory, mapping each storage key to its project path.",
	}, d.memoryProjects)
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

type MemoryWriteInput struct {
	Project      string   `json:"project,omitempty" jsonschema:"project directory (default current working directory)"`
	Agent        string   `json:"agent,omitempty" jsonschema:"agent that produced this memory, e.g. claude or codex"`
	Actor        string   `json:"actor,omitempty" jsonschema:"human or agent"`
	Session      string   `json:"session,omitempty" jsonschema:"session name or id to link"`
	Title        string   `json:"title,omitempty"`
	Body         string   `json:"body"`
	Type         string   `json:"type,omitempty" jsonschema:"note, fact, decision, todo, blocker, warning"`
	Tags         []string `json:"tags,omitempty"`
	RelatedFiles []string `json:"related_files,omitempty"`
}

func (d *deps) memoryWrite(ctx context.Context, _ *mcp.CallToolRequest, in MemoryWriteInput) (*mcp.CallToolResult, mem.Memory, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, mem.Memory{}, err
	}
	now := time.Now()
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSpace(in.Body)
	}
	sessionID, sessionName := agentSessionParts(in.Session)
	m := mem.Memory{
		ID:           mem.NewID("mem", title, now),
		Project:      project,
		Agent:        strings.TrimSpace(in.Agent),
		Actor:        defaultAgentString(in.Actor, "agent"),
		Source:       mem.SourceMCP,
		SessionID:    sessionID,
		SessionName:  sessionName,
		Type:         defaultAgentString(in.Type, mem.TypeNote),
		Status:       mem.StatusOpen,
		Title:        title,
		Body:         strings.TrimSpace(in.Body),
		Tags:         in.Tags,
		RelatedFiles: in.RelatedFiles,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := store.AddMemory(ctx, m); err != nil {
		return nil, mem.Memory{}, err
	}
	return text(fmt.Sprintf("wrote memory %s", m.ID)), m, nil
}

type MemorySearchInput struct {
	Project string `json:"project,omitempty"`
	Query   string `json:"query,omitempty"`
	Agent   string `json:"agent,omitempty"`
	Session string `json:"session,omitempty"`
	Status  string `json:"status,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type MemoriesOutput struct {
	Memories []mem.Memory `json:"memories"`
}

func (d *deps) memorySearch(ctx context.Context, _ *mcp.CallToolRequest, in MemorySearchInput) (*mcp.CallToolResult, MemoriesOutput, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, MemoriesOutput{}, err
	}
	items, err := store.SearchMemories(ctx, in.Query, mem.Filter{Project: project, Agent: in.Agent, Session: in.Session, Status: in.Status, Limit: in.Limit})
	if err != nil {
		return nil, MemoriesOutput{}, err
	}
	return text(fmt.Sprintf("%d memory(s)", len(items))), MemoriesOutput{Memories: items}, nil
}

func (d *deps) memoryList(ctx context.Context, _ *mcp.CallToolRequest, in MemorySearchInput) (*mcp.CallToolResult, MemoriesOutput, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, MemoriesOutput{}, err
	}
	items, err := store.ListMemories(ctx, mem.Filter{Project: project, Agent: in.Agent, Session: in.Session, Status: in.Status, Limit: in.Limit})
	if err != nil {
		return nil, MemoriesOutput{}, err
	}
	return text(fmt.Sprintf("%d memory(s)", len(items))), MemoriesOutput{Memories: items}, nil
}

type MemoryGetInput struct {
	Project string `json:"project,omitempty"`
	ID      string `json:"id"`
}

func (d *deps) memoryGet(ctx context.Context, _ *mcp.CallToolRequest, in MemoryGetInput) (*mcp.CallToolResult, mem.Memory, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, mem.Memory{}, err
	}
	m, err := store.GetMemory(ctx, project, in.ID)
	if err != nil {
		return nil, mem.Memory{}, err
	}
	return text(m.Body), m, nil
}

func (d *deps) memoryResolve(ctx context.Context, _ *mcp.CallToolRequest, in MemoryGetInput) (*mcp.CallToolResult, mem.Memory, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, mem.Memory{}, err
	}
	m, err := store.GetMemory(ctx, project, in.ID)
	if err != nil {
		return nil, mem.Memory{}, err
	}
	m.Status = mem.StatusResolved
	m.UpdatedAt = time.Now()
	if err := store.UpdateMemory(ctx, m); err != nil {
		return nil, mem.Memory{}, err
	}
	return text(fmt.Sprintf("resolved memory %s", m.ID)), m, nil
}

type SnapshotCreateInput struct {
	Project string `json:"project,omitempty"`
	Agent   string `json:"agent,omitempty"`
	Actor   string `json:"actor,omitempty"`
	Session string `json:"session,omitempty"`
	Title   string `json:"title,omitempty"`
	Note    string `json:"note,omitempty"`
}

func (d *deps) snapshotCreate(ctx context.Context, _ *mcp.CallToolRequest, in SnapshotCreateInput) (*mcp.CallToolResult, mem.Snapshot, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, mem.Snapshot{}, err
	}
	snap, err := mem.CreateSnapshot(ctx, store, mem.SnapshotOptions{
		Project: project,
		Agent:   in.Agent,
		Actor:   defaultAgentString(in.Actor, "agent"),
		Source:  mem.SourceMCP,
		Session: in.Session,
		Title:   in.Title,
		Note:    in.Note,
	})
	if err != nil {
		return nil, mem.Snapshot{}, err
	}
	return text(fmt.Sprintf("created snapshot %s", snap.ID)), snap, nil
}

type ResumeContextInput struct {
	Project   string `json:"project,omitempty"`
	Agent     string `json:"agent,omitempty"`
	FromAgent string `json:"from_agent,omitempty"`
	ToAgent   string `json:"to_agent,omitempty"`
	Session   string `json:"session,omitempty"`
	Query     string `json:"query,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (d *deps) resumeContext(ctx context.Context, _ *mcp.CallToolRequest, in ResumeContextInput) (*mcp.CallToolResult, mem.ResumeContext, error) {
	project, store, err := agentMemoryStore(in.Project)
	if err != nil {
		return nil, mem.ResumeContext{}, err
	}
	rc, err := mem.BuildResumeContext(ctx, store, mem.ResumeOptions{Project: project, Agent: in.Agent, FromAgent: in.FromAgent, ToAgent: in.ToAgent, Session: in.Session, Query: in.Query, Limit: in.Limit})
	if err != nil {
		return nil, mem.ResumeContext{}, err
	}
	return text(rc.Summary), rc, nil
}

type ProjectsOutput struct {
	Projects []mem.ProjectInfo `json:"projects"`
}

func (d *deps) memoryProjects(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ProjectsOutput, error) {
	store := mem.NewFileStore(mem.DefaultRoot())
	projects, err := store.ListProjects(ctx)
	if err != nil {
		return nil, ProjectsOutput{}, err
	}
	return text(fmt.Sprintf("%d project(s)", len(projects))), ProjectsOutput{Projects: projects}, nil
}

func agentMemoryStore(project string) (string, *mem.FileStore, error) {
	project, err := mem.NormalizeProject(project)
	if err != nil {
		return "", nil, err
	}
	return project, mem.NewFileStore(mem.DefaultRoot()), nil
}

func agentSessionParts(session string) (string, string) {
	if strings.HasPrefix(session, "sess_") || strings.Contains(session, "_") {
		return session, ""
	}
	return "", session
}

func defaultAgentString(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
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
