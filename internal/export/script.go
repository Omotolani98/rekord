package export

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

const scriptExportFilePerm = 0o755

type ScriptExporter struct{}

func (ScriptExporter) Format() string { return "script" }
func (ScriptExporter) Ext() string    { return "sh" }

func (ScriptExporter) Export(_ context.Context, _ session.Metadata, _ []events.Event, cmds []commands.Command, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), castExportDirPerm); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, scriptExportFilePerm)
	if err != nil {
		return fmt.Errorf("create script export: %w", err)
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "#!/usr/bin/env bash")
	fmt.Fprintln(w, "set -e")
	fmt.Fprintln(w)
	for _, c := range cmds {
		fmt.Fprintln(w, c.Command)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush script export: %w", err)
	}
	return nil
}
