package export

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func TestCastExporterWritesHeaderAndRows(t *testing.T) {
	m := session.Metadata{
		Name:      "demo",
		Shell:     "/bin/zsh",
		Cols:      120,
		Rows:      40,
		CreatedAt: time.Unix(1780128000, 0).UTC(),
	}
	evs := []events.Event{
		{TimeMS: 0, Type: events.TypeResize, Cols: 120, Rows: 40},
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ go test ./...\r\n"},
		{TimeMS: 132, Type: events.TypeOutput, Data: "ok\r\n"},
	}

	out := filepath.Join(t.TempDir(), "exports", "demo.cast")
	if err := (CastExporter{}).Export(context.Background(), m, evs, nil, out); err != nil {
		t.Fatalf("Export: %v", err)
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("no header line")
	}
	var header struct {
		Version   int               `json:"version"`
		Width     int               `json:"width"`
		Height    int               `json:"height"`
		Timestamp int64             `json:"timestamp"`
		Env       map[string]string `json:"env"`
	}
	if err := json.Unmarshal(sc.Bytes(), &header); err != nil {
		t.Fatalf("header unmarshal: %v", err)
	}
	if header.Version != 2 {
		t.Fatalf("version = %d, want 2", header.Version)
	}
	if header.Width != 120 || header.Height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", header.Width, header.Height)
	}
	if header.Timestamp != 1780128000 {
		t.Fatalf("timestamp = %d, want 1780128000", header.Timestamp)
	}
	if header.Env["SHELL"] != "/bin/zsh" {
		t.Fatalf("env.SHELL = %q, want /bin/zsh", header.Env["SHELL"])
	}

	var rows []string
	for sc.Scan() {
		rows = append(rows, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (resize excluded)", len(rows))
	}

	var first []any
	if err := json.Unmarshal([]byte(rows[0]), &first); err != nil {
		t.Fatalf("row unmarshal: %v", err)
	}
	if first[1] != "o" {
		t.Fatalf("row code = %v, want o", first[1])
	}
	if !strings.Contains(rows[1], `"o"`) {
		t.Fatalf("second row missing output code: %s", rows[1])
	}
	if first[0].(float64) != 0.0 {
		t.Fatalf("first row time = %v, want 0", first[0])
	}
}
