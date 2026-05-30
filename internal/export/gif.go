package export

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

type GifExporter struct{}

func (GifExporter) Format() string { return "gif" }
func (GifExporter) Ext() string    { return "gif" }

func (GifExporter) Export(ctx context.Context, m session.Metadata, evs []events.Event, cmds []commands.Command, outPath string) error {
	if toolMissing("agg") {
		return errors.New("gif export requires 'agg' (asciinema gif generator); install it and retry")
	}

	dir := filepath.Dir(outPath)
	castPath, err := renderCast(ctx, m, evs, cmds, dir)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(castPath) }()

	return runRender(ctx, "agg", castPath, outPath)
}
