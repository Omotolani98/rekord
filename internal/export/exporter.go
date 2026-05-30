package export

import (
	"context"
	"fmt"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

type Exporter interface {
	Format() string
	Ext() string
	Export(ctx context.Context, m session.Metadata, evs []events.Event, cmds []commands.Command, outPath string) error
}

func Get(format string) (Exporter, error) {
	switch format {
	case "cast":
		return CastExporter{}, nil
	case "json":
		return JSONExporter{}, nil
	case "markdown":
		return MarkdownExporter{}, nil
	case "script":
		return ScriptExporter{}, nil
	default:
		return nil, fmt.Errorf("unknown export format %q", format)
	}
}
