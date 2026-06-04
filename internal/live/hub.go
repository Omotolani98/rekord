package live

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	vt "github.com/charmbracelet/x/vt"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/ptyx"
	"github.com/Omotolani98/rekord/internal/session"
)

const (
	defaultCols = 80
	defaultRows = 24
)

type Hub struct {
	root    string
	version string

	mu       sync.Mutex
	sessions map[string]*Session
}

func NewHub(root, version string) *Hub {
	return &Hub{
		root:     root,
		version:  version,
		sessions: make(map[string]*Session),
	}
}

type LaunchOptions struct {
	Name    string
	Command []string
	CWD     string
	Env     []string
	Cols    int
	Rows    int
}

func (h *Hub) Launch(ctx context.Context, opts LaunchOptions) (*Session, error) {
	if err := session.ValidateName(opts.Name); err != nil {
		return nil, err
	}
	if len(opts.Command) == 0 {
		return nil, errors.New("command is required")
	}

	h.mu.Lock()
	if _, exists := h.sessions[opts.Name]; exists {
		h.mu.Unlock()
		return nil, fmt.Errorf("session already exists: %q", opts.Name)
	}
	h.mu.Unlock()

	cols, rows := opts.Cols, opts.Rows
	if cols <= 0 {
		cols = defaultCols
	}
	if rows <= 0 {
		rows = defaultRows
	}

	cwd := opts.CWD
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}

	now := time.Now().UTC()
	id := session.NewID(opts.Name, now)
	store := session.NewFileStore(h.root)

	meta := session.Metadata{
		ID:            id,
		Name:          opts.Name,
		CreatedAt:     now,
		Command:       opts.Command,
		CWD:           cwd,
		Cols:          cols,
		Rows:          rows,
		Status:        session.StatusRecording,
		RekordVersion: h.version,
	}
	if err := store.Create(ctx, meta); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	writer, err := events.NewWriter(filepath.Join(store.SessionDir(id), "events.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("open events writer: %w", err)
	}

	handle, err := ptyx.Start(ptyx.Options{
		Command: opts.Command,
		CWD:     cwd,
		Env:     opts.Env,
		Cols:    cols,
		Rows:    rows,
	})
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("start pty: %w", err)
	}

	s := &Session{
		name:          opts.Name,
		id:            id,
		startedAt:     time.Now(),
		store:         store,
		pty:           handle,
		emu:           vt.NewEmulator(cols, rows),
		writer:        writer,
		maxTranscript: defaultMaxTranscript,
		meta:          meta,
		exited:        make(chan struct{}),
	}
	s.lastOutputAt = s.startedAt
	_ = writer.Append(events.Event{TimeMS: 0, Type: events.TypeResize, Cols: cols, Rows: rows})

	h.mu.Lock()
	h.sessions[opts.Name] = s
	h.mu.Unlock()

	go s.pump()

	return s, nil
}

func (h *Hub) Get(name string) (*Session, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[name]
	return s, ok
}

func (h *Hub) List() []Status {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Status, 0, len(h.sessions))
	for _, s := range h.sessions {
		out = append(out, s.Status())
	}
	return out
}

func (h *Hub) Stop(name string) error {
	h.mu.Lock()
	s, ok := h.sessions[name]
	if ok {
		delete(h.sessions, name)
	}
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found: %q", name)
	}
	s.Stop()
	return nil
}

func (h *Hub) Shutdown() {
	h.mu.Lock()
	all := make([]*Session, 0, len(h.sessions))
	for _, s := range h.sessions {
		all = append(all, s)
	}
	h.sessions = make(map[string]*Session)
	h.mu.Unlock()
	for _, s := range all {
		s.Stop()
	}
}
