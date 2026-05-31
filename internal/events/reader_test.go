package events

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReaderRoundTripWithWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	want := []Event{
		{TimeMS: 0, Type: TypeOutput, Data: "hello\r\n"},
		{TimeMS: 5, Type: TypeInput, Data: "ls\n"},
		{TimeMS: 12, Type: TypeResize, Cols: 120, Rows: 40},
		{TimeMS: 20, Type: TypeMarker, Data: "checkpoint"},
	}
	for _, e := range want {
		if err := w.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	got, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("event count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestReaderEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	r, err := NewReader(path)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	defer r.Close()

	e, ok, err := r.Next()
	if err != nil {
		t.Fatalf("Next err: %v", err)
	}
	if ok {
		t.Fatalf("Next ok=true on empty, event=%#v", e)
	}
}

func TestReaderMissingFile(t *testing.T) {
	_, err := NewReader(filepath.Join(t.TempDir(), "nope.jsonl"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want fs.ErrNotExist", err)
	}
}

func TestReaderMalformedLineReportsLineNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	content := `{"timeMs":1,"type":"output","data":"ok"}` + "\n" +
		`{not json` + "\n" +
		`{"timeMs":3,"type":"output","data":"after"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	r, err := NewReader(path)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	defer r.Close()

	if _, ok, err := r.Next(); err != nil || !ok {
		t.Fatalf("first Next ok=%v err=%v, want ok err=nil", ok, err)
	}

	_, _, err = r.Next()
	if err == nil {
		t.Fatal("second Next err = nil, want malformed error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "line 2") {
		t.Fatalf("error missing line number: %v", err)
	}
	if !strings.Contains(msg, "not json") {
		t.Fatalf("error missing content excerpt: %v", err)
	}
}

func TestReaderLargeLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	big := strings.Repeat("x", 256<<10)
	want := Event{TimeMS: 1, Type: TypeOutput, Data: big}
	if err := w.Append(want); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	got, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("round-trip mismatch: got %d events, first=%v...", len(got), got[0].TimeMS)
	}
}

func TestReaderSkipsTrailingEmptyLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	content := `{"timeMs":1,"type":"output","data":"a"}` + "\n" +
		`{"timeMs":2,"type":"output","data":"b"}` + "\n\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("count = %d, want 2", len(got))
	}
}

func TestReadAllMatchesIterator(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	events := []Event{
		{TimeMS: 1, Type: TypeOutput, Data: "a"},
		{TimeMS: 2, Type: TypeInput, Data: "b"},
		{TimeMS: 3, Type: TypeResize, Cols: 80, Rows: 24},
	}
	for _, e := range events {
		if err := w.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close writer: %v", err)
	}

	bulk, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	r, err := NewReader(path)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	defer r.Close()
	var iter []Event
	for {
		e, ok, err := r.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if !ok {
			break
		}
		iter = append(iter, e)
	}

	if len(bulk) != len(iter) {
		t.Fatalf("len bulk=%d iter=%d", len(bulk), len(iter))
	}
	for i := range bulk {
		if bulk[i] != iter[i] {
			t.Fatalf("event[%d] bulk=%#v iter=%#v", i, bulk[i], iter[i])
		}
	}
}

func TestReaderCloseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	r, err := NewReader(path)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, _, err := r.Next(); !errors.Is(err, ErrReaderClosed) {
		t.Fatalf("Next after Close err = %v, want ErrReaderClosed", err)
	}
}
