package live

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"time"
)

const dialProbeTimeout = 500 * time.Millisecond

func Serve(ctx context.Context, path string, s *Session) error {
	if err := removeStale(path); err != nil {
		return err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	defer func() {
		_ = ln.Close()
		_ = os.Remove(path)
	}()

	stop := make(chan struct{})
	var stopOnce sync.Once
	triggerStop := func() { stopOnce.Do(func() { close(stop) }) }

	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
		}
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		go handleConn(ctx, conn, s, triggerStop)
	}

	s.Stop()
	return nil
}

func handleConn(ctx context.Context, conn net.Conn, s *Session, triggerStop func()) {
	defer func() { _ = conn.Close() }()
	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(Response{Error: err.Error()})
		return
	}
	resp := dispatch(ctx, s, req, triggerStop)
	_ = json.NewEncoder(conn).Encode(resp)
}

func dispatch(ctx context.Context, s *Session, req Request, triggerStop func()) Response {
	switch req.Op {
	case "status":
		st := s.Status()
		return Response{OK: true, Status: &st}

	case "send":
		buf, err := BuildInput(req.Text, req.Keys)
		if err != nil {
			return Response{Error: err.Error()}
		}
		if err := s.SendInput(buf); err != nil {
			return Response{Error: err.Error()}
		}
		return Response{OK: true, Sent: len(buf)}

	case "capture":
		f := s.Capture()
		return Response{OK: true, Frame: &f}

	case "wait_text":
		wctx, cancel := withTimeout(ctx, req.TimeoutMs)
		defer cancel()
		reason, f, _ := s.WaitForText(wctx, req.Sub)
		return Response{OK: true, Reason: reason, Frame: &f}

	case "wait_idle":
		quiet := 500 * time.Millisecond
		if req.QuietMs > 0 {
			quiet = time.Duration(req.QuietMs) * time.Millisecond
		}
		wctx, cancel := withTimeout(ctx, req.TimeoutMs)
		defer cancel()
		reason, f, _ := s.WaitForIdle(wctx, quiet)
		return Response{OK: true, Reason: reason, Frame: &f}

	case "wait_exit":
		wctx, cancel := withTimeout(ctx, req.TimeoutMs)
		defer cancel()
		code, reason, _ := s.WaitForExit(wctx)
		return Response{OK: true, Reason: reason, ExitCode: &code}

	case "logs":
		return Response{OK: true, Logs: s.Logs(req.MaxBytes)}

	case "resize":
		if err := s.Resize(req.Cols, req.Rows); err != nil {
			return Response{Error: err.Error()}
		}
		st := s.Status()
		return Response{OK: true, Status: &st}

	case "stop":
		triggerStop()
		return Response{OK: true}

	default:
		return Response{Error: "unknown op: " + req.Op}
	}
}

func withTimeout(ctx context.Context, ms int) (context.Context, context.CancelFunc) {
	d := 10 * time.Second
	if ms > 0 {
		d = time.Duration(ms) * time.Millisecond
	}
	return context.WithTimeout(ctx, d)
}

func removeStale(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	conn, err := net.DialTimeout("unix", path, dialProbeTimeout)
	if err == nil {
		_ = conn.Close()
		return errors.New("session socket already in use: " + path)
	}
	return os.Remove(path)
}
