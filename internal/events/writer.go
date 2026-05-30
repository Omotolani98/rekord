package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

const eventsFilePerm = 0o600

var ErrWriterClosed = errors.New("events writer closed")

type Writer struct {
	f      *os.File
	bw     *bufio.Writer
	mu     sync.Mutex
	closed bool
}

func NewWriter(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, eventsFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open events file: %w", err)
	}
	return &Writer{f: f, bw: bufio.NewWriter(f)}, nil
}

func (w *Writer) Append(e Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrWriterClosed
	}

	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	payload = append(payload, '\n')

	if _, err := w.bw.Write(payload); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrWriterClosed
	}
	return w.bw.Flush()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true

	flushErr := w.bw.Flush()
	syncErr := w.f.Sync()
	closeErr := w.f.Close()

	if flushErr != nil {
		return fmt.Errorf("flush events: %w", flushErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync events: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close events: %w", closeErr)
	}
	return nil
}
