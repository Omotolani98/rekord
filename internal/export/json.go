package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

const outputSummaryMax = 4096

type JSONExporter struct{}

func (JSONExporter) Format() string { return "json" }
func (JSONExporter) Ext() string    { return "json" }

type jsonSummary struct {
	Metadata      session.Metadata   `json:"metadata"`
	Commands      []commands.Command `json:"commands"`
	OutputSummary string             `json:"outputSummary"`
}

func (JSONExporter) Export(_ context.Context, m session.Metadata, evs []events.Event, cmds []commands.Command, outPath string) error {
	if cmds == nil {
		cmds = []commands.Command{}
	}
	summary := jsonSummary{
		Metadata:      m,
		Commands:      cmds,
		OutputSummary: outputSummary(evs),
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json export: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(outPath), castExportDirPerm); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}
	if err := os.WriteFile(outPath, data, castExportFilePerm); err != nil {
		return fmt.Errorf("write json export: %w", err)
	}
	return nil
}

func outputSummary(evs []events.Event) string {
	var b strings.Builder
	for _, e := range evs {
		if e.Type != events.TypeOutput {
			continue
		}
		b.WriteString(strings.ReplaceAll(e.Data, "\r", ""))
		if b.Len() >= outputSummaryMax {
			break
		}
	}
	s := b.String()
	if len(s) > outputSummaryMax {
		return s[:outputSummaryMax] + "…"
	}
	return s
}
