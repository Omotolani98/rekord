package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	readerInitBuf = 64 << 10
	readerMaxBuf  = 1 << 20
	errSnippetMax = 200
)

var ErrReaderClosed = errors.New("events reader closed")

type Reader struct {
	f      *os.File
	sc     *bufio.Scanner
	line   int
	closed bool
}

func NewReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open events file: %w", err)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, readerInitBuf), readerMaxBuf)
	return &Reader{f: f, sc: sc}, nil
}

func (r *Reader) Next() (Event, bool, error) {
	if r.closed {
		return Event{}, false, ErrReaderClosed
	}

	for r.sc.Scan() {
		line := r.sc.Bytes()
		if len(line) == 0 {
			continue
		}
		r.line++

		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			return Event{}, false, fmt.Errorf("malformed event at line %d: %w (content: %q)", r.line, err, snippet(line))
		}
		return e, true, nil
	}

	if err := r.sc.Err(); err != nil {
		return Event{}, false, fmt.Errorf("scan events at line %d: %w", r.line+1, err)
	}
	return Event{}, false, nil
}

func (r *Reader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	if err := r.f.Close(); err != nil {
		return fmt.Errorf("close events file: %w", err)
	}
	return nil
}

func ReadAll(path string) ([]Event, error) {
	r, err := NewReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var out []Event
	for {
		e, ok, err := r.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return out, nil
		}
		out = append(out, e)
	}
}

func snippet(line []byte) string {
	if len(line) <= errSnippetMax {
		return string(line)
	}
	return string(line[:errSnippetMax]) + "..."
}
