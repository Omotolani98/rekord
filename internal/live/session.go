package live

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	vt "github.com/charmbracelet/x/vt"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/frame"
	"github.com/Omotolani98/rekord/internal/ptyx"
	"github.com/Omotolani98/rekord/internal/session"
)

const (
	readBufSize          = 4096
	waitPollInterval     = 25 * time.Millisecond
	defaultMaxTranscript = 1 << 20
)

type Session struct {
	name      string
	id        string
	startedAt time.Time

	store  *session.FileStore
	pty    ptyx.PTY
	writer *events.Writer

	emuMu sync.Mutex
	emu   *vt.Emulator

	maxTranscript int

	mu           sync.Mutex
	meta         session.Metadata
	lastOutputAt time.Time
	transcript   []byte

	exited   chan struct{}
	exitOnce sync.Once
	exitCode int
	stopOnce sync.Once
}

func (s *Session) Name() string { return s.name }
func (s *Session) ID() string   { return s.id }

func (s *Session) elapsed() int64 {
	return time.Since(s.startedAt).Milliseconds()
}

func (s *Session) pump() {
	buf := make([]byte, readBufSize)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			s.emuMu.Lock()
			_, _ = s.emu.Write(chunk)
			s.emuMu.Unlock()
			_ = s.writer.Append(events.Event{
				TimeMS: s.elapsed(),
				Type:   events.TypeOutput,
				Data:   string(chunk),
			})
			s.mu.Lock()
			s.lastOutputAt = time.Now()
			s.transcript = appendCapped(s.transcript, chunk, s.maxTranscript)
			s.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
	code, _ := s.pty.Wait()
	s.finish(code)
}

func (s *Session) finish(code int) {
	s.exitOnce.Do(func() {
		s.exitCode = code
		s.mu.Lock()
		end := time.Now().UTC()
		s.meta.EndedAt = &end
		s.meta.DurationMS = end.Sub(s.startedAt).Milliseconds()
		s.meta.Status = session.StatusCompleted
		meta := s.meta
		s.mu.Unlock()
		_ = s.writer.Flush()
		_ = s.writer.Close()
		_ = s.store.WriteMetadata(context.Background(), meta)
		close(s.exited)
	})
}

func (s *Session) running() bool {
	select {
	case <-s.exited:
		return false
	default:
		return true
	}
}

func (s *Session) SendInput(b []byte) error {
	if !s.running() {
		return errors.New("session has exited")
	}
	if _, err := s.pty.Write(b); err != nil {
		return err
	}
	return s.writer.Append(events.Event{
		TimeMS: s.elapsed(),
		Type:   events.TypeInput,
		Data:   string(b),
	})
}

func (s *Session) Capture() frame.Frame {
	s.emuMu.Lock()
	defer s.emuMu.Unlock()
	return frame.FromScreen(s.emu)
}

func (s *Session) WaitForText(ctx context.Context, sub string) (string, frame.Frame, error) {
	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()
	for {
		f := s.Capture()
		if strings.Contains(f.Text(), sub) {
			return "matched", f, nil
		}
		select {
		case <-ctx.Done():
			return "deadline", f, nil
		case <-s.exited:
			f = s.Capture()
			if strings.Contains(f.Text(), sub) {
				return "matched", f, nil
			}
			return "exited", f, nil
		case <-ticker.C:
		}
	}
}

func (s *Session) WaitForIdle(ctx context.Context, quiet time.Duration) (string, frame.Frame, error) {
	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		last := s.lastOutputAt
		s.mu.Unlock()
		if time.Since(last) >= quiet {
			return "idle", s.Capture(), nil
		}
		select {
		case <-ctx.Done():
			return "deadline", s.Capture(), nil
		case <-s.exited:
			return "exited", s.Capture(), nil
		case <-ticker.C:
		}
	}
}

func (s *Session) WaitForExit(ctx context.Context) (int, string, error) {
	select {
	case <-s.exited:
		return s.exitCode, "exited", nil
	case <-ctx.Done():
		return 0, "deadline", nil
	}
}

func (s *Session) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return errors.New("cols and rows must be positive")
	}
	s.emuMu.Lock()
	s.emu.Resize(cols, rows)
	s.emuMu.Unlock()
	if err := s.pty.Resize(cols, rows); err != nil {
		return err
	}
	s.mu.Lock()
	s.meta.Cols = cols
	s.meta.Rows = rows
	s.mu.Unlock()
	return s.writer.Append(events.Event{
		TimeMS: s.elapsed(),
		Type:   events.TypeResize,
		Cols:   cols,
		Rows:   rows,
	})
}

func (s *Session) Logs(maxBytes int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.transcript
	if maxBytes > 0 && len(t) > maxBytes {
		t = t[len(t)-maxBytes:]
	}
	return string(t)
}

func (s *Session) Stop() {
	s.stopOnce.Do(func() { _ = s.pty.Kill() })
	<-s.exited
}

type Status struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`
	Running  bool   `json:"running"`
	ExitCode *int   `json:"exitCode,omitempty"`
}

func (s *Session) Status() Status {
	s.mu.Lock()
	cols, rows := s.meta.Cols, s.meta.Rows
	s.mu.Unlock()
	st := Status{
		Name:    s.name,
		ID:      s.id,
		Cols:    cols,
		Rows:    rows,
		Running: s.running(),
	}
	if !st.Running {
		code := s.exitCode
		st.ExitCode = &code
	}
	return st
}

func appendCapped(dst, src []byte, max int) []byte {
	dst = append(dst, src...)
	if max > 0 && len(dst) > max {
		dst = dst[len(dst)-max:]
	}
	return dst
}
