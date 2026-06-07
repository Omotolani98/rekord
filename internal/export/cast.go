package export

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

const (
	castExportDirPerm  = 0o700
	castExportFilePerm = 0o600
	defaultTerm        = "xterm-256color"
	defaultCols        = 80
	defaultRows        = 24
)

type CastExporter struct{}

func (CastExporter) Format() string { return "cast" }
func (CastExporter) Ext() string    { return "cast" }

type castHeader struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Env       map[string]string `json:"env"`
}

func (CastExporter) Export(ctx context.Context, m session.Metadata, evs []events.Event, _ []commands.Command, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), castExportDirPerm); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	env := map[string]string{"TERM": defaultTerm}
	if m.Shell != "" {
		env["SHELL"] = m.Shell
	}
	cols, rows := m.Cols, m.Rows
	if cols <= 0 {
		cols = defaultCols
	}
	if rows <= 0 {
		rows = defaultRows
	}
	header := castHeader{
		Version:   2,
		Width:     cols,
		Height:    rows,
		Timestamp: m.CreatedAt.Unix(),
		Env:       env,
	}

	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, castExportFilePerm)
	if err != nil {
		return fmt.Errorf("create cast file: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)

	if err := enc.Encode(header); err != nil {
		return fmt.Errorf("encode cast header: %w", err)
	}

	for _, e := range evs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.Type != events.TypeOutput {
			continue
		}
		row := []any{float64(e.TimeMS) / 1000.0, "o", e.Data}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("encode cast row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush cast file: %w", err)
	}
	return nil
}
