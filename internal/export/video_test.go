package export

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func sampleVideoEvents() []events.Event {
	return []events.Event{
		{TimeMS: 0, Type: events.TypeOutput, Data: "$ echo hi\r\n"},
		{TimeMS: 100, Type: events.TypeOutput, Data: "hi\r\n"},
	}
}

func hasTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func TestGifExportMissingAgg(t *testing.T) {
	if hasTool("agg") {
		t.Skip("agg installed; skipping missing-tool assertion")
	}
	out := filepath.Join(t.TempDir(), "exports", "demo.gif")
	err := (GifExporter{}).Export(context.Background(), session.Metadata{Name: "demo"}, sampleVideoEvents(), nil, out)
	if err == nil || !strings.Contains(err.Error(), "agg") {
		t.Fatalf("err = %v, want error mentioning agg", err)
	}
}

func TestGifExportRenders(t *testing.T) {
	if !hasTool("agg") {
		t.Skip("agg not installed")
	}
	out := filepath.Join(t.TempDir(), "exports", "demo.gif")
	if err := (GifExporter{}).Export(context.Background(), session.Metadata{Name: "demo", Cols: 80, Rows: 24}, sampleVideoEvents(), nil, out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() == 0 {
		t.Fatalf("gif not produced: %v", err)
	}
}

func TestMP4ExportRenders(t *testing.T) {
	if !hasTool("agg") || !hasTool("ffmpeg") {
		t.Skip("agg or ffmpeg not installed")
	}
	exp, err := newMP4Exporter("720p")
	if err != nil {
		t.Fatalf("newMP4Exporter: %v", err)
	}
	out := filepath.Join(t.TempDir(), "exports", "demo.mp4")
	if err := exp.Export(context.Background(), session.Metadata{Name: "demo", Cols: 80, Rows: 24}, sampleVideoEvents(), nil, out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() == 0 {
		t.Fatalf("mp4 not produced: %v", err)
	}
}
