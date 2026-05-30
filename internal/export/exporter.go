package export

import (
	"context"
	"fmt"

	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

type Exporter interface {
	Format() string
	Export(ctx context.Context, m session.Metadata, evs []events.Event, outPath string) error
}

func Get(format string) (Exporter, error) {
	switch format {
	case "cast":
		return CastExporter{}, nil
	default:
		return nil, fmt.Errorf("unknown export format %q", format)
	}
}
