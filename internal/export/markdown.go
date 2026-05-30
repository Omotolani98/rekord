package export

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

type MarkdownExporter struct{}

func (MarkdownExporter) Format() string { return "markdown" }
func (MarkdownExporter) Ext() string    { return "md" }

func (MarkdownExporter) Export(_ context.Context, m session.Metadata, _ []events.Event, cmds []commands.Command, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), castExportDirPerm); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, castExportFilePerm)
	if err != nil {
		return fmt.Errorf("create markdown export: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)

	shell := m.Shell
	if shell == "" {
		shell = "-"
	}

	fmt.Fprintf(w, "# Rekord Session: %s\n\n", m.Name)
	fmt.Fprintf(w, "## Summary\n\n")
	fmt.Fprintf(w, "- Duration: %s\n", formatDurationMS(m.DurationMS))
	fmt.Fprintf(w, "- Shell: %s\n", shell)
	fmt.Fprintf(w, "- Working directory: %s\n\n", m.CWD)
	fmt.Fprintf(w, "## Commands\n\n")

	if len(cmds) == 0 {
		fmt.Fprintf(w, "_No commands extracted._\n")
	}
	for _, c := range cmds {
		fmt.Fprintf(w, "### %d. %s\n\n", c.Index, c.Command)
		if c.OutputPreview != "" {
			fmt.Fprintf(w, "```text\n%s\n```\n\n", c.OutputPreview)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush markdown export: %w", err)
	}
	return nil
}

func formatDurationMS(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	d := time.Duration(ms) * time.Millisecond
	if d >= time.Second {
		d = d.Round(time.Second)
	}
	return d.String()
}
