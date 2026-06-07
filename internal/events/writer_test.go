package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestWriterAppendsJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	want := []Event{
		{TimeMS: 0, Type: TypeOutput, Data: "hello\r\n"},
		{TimeMS: 12, Type: TypeInput, Data: "ls\n"},
		{TimeMS: 50, Type: TypeResize, Cols: 120, Rows: 40},
	}
	for _, e := range want {
		if err := w.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got := readEvents(t, path)
	if len(got) != len(want) {
		t.Fatalf("event count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestWriterOmitsEmptyFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	if err := w.Append(Event{TimeMS: 1, Type: TypeOutput, Data: "x"}); err != nil {
		t.Fatalf("Append output: %v", err)
	}
	if err := w.Append(Event{TimeMS: 2, Type: TypeResize, Cols: 80, Rows: 24}); err != nil {
		t.Fatalf("Append resize: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, path)
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	if strings.Contains(lines[0], `"cols"`) || strings.Contains(lines[0], `"rows"`) {
		t.Fatalf("output line should omit cols/rows: %s", lines[0])
	}
	if strings.Contains(lines[1], `"data"`) {
		t.Fatalf("resize line should omit data: %s", lines[1])
	}
}

func TestWriterAppendsToExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")

	w1, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter 1: %v", err)
	}
	for _, e := range []Event{
		{TimeMS: 1, Type: TypeOutput, Data: "a"},
		{TimeMS: 2, Type: TypeOutput, Data: "b"},
	} {
		if err := w1.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := w1.Close(); err != nil {
		t.Fatalf("Close 1: %v", err)
	}

	w2, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter 2: %v", err)
	}
	if err := w2.Append(Event{TimeMS: 3, Type: TypeOutput, Data: "c"}); err != nil {
		t.Fatalf("Append 2: %v", err)
	}
	if err := w2.Close(); err != nil {
		t.Fatalf("Close 2: %v", err)
	}

	got := readEvents(t, path)
	if len(got) != 3 {
		t.Fatalf("event count = %d, want 3", len(got))
	}
	for i, data := range []string{"a", "b", "c"} {
		if got[i].Data != data {
			t.Fatalf("event[%d].Data = %q, want %q", i, got[i].Data, data)
		}
	}
}

func TestWriterCloseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if err := w.Append(Event{TimeMS: 1, Type: TypeOutput, Data: "x"}); !errors.Is(err, ErrWriterClosed) {
		t.Fatalf("Append after Close error = %v, want ErrWriterClosed", err)
	}
	if err := w.Flush(); !errors.Is(err, ErrWriterClosed) {
		t.Fatalf("Flush after Close error = %v, want ErrWriterClosed", err)
	}
}

func TestWriterFilePerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions are not enforced on windows")
	}
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 0600", perm)
	}
}

func TestWriterConcurrentAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			if err := w.Append(Event{TimeMS: int64(i), Type: TypeOutput, Data: "x"}); err != nil {
				t.Errorf("Append: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got := readEvents(t, path)
	if len(got) != n {
		t.Fatalf("event count = %d, want %d", len(got), n)
	}
	for i, e := range got {
		if e.Type != TypeOutput || e.Data != "x" {
			t.Fatalf("event[%d] = %#v, want type=output data=x", i, e)
		}
	}
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return lines
}

func readEvents(t *testing.T, path string) []Event {
	t.Helper()
	lines := readLines(t, path)
	events := make([]Event, 0, len(lines))
	for i, line := range lines {
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("decode line %d (%q): %v", i, line, err)
		}
		events = append(events, e)
	}
	return events
}
