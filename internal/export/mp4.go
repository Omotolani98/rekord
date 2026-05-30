package export

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

const defaultMP4Size = "720p"

var mp4Heights = map[string]int{
	"720p":  720,
	"1080p": 1080,
}

type MP4Exporter struct {
	height int
}

func newMP4Exporter(size string) (MP4Exporter, error) {
	if size == "" {
		size = defaultMP4Size
	}
	h, ok := mp4Heights[size]
	if !ok {
		return MP4Exporter{}, fmt.Errorf("unsupported mp4 size %q (use 720p or 1080p)", size)
	}
	return MP4Exporter{height: h}, nil
}

func (MP4Exporter) Format() string { return "mp4" }
func (MP4Exporter) Ext() string    { return "mp4" }

func (e MP4Exporter) Export(ctx context.Context, m session.Metadata, evs []events.Event, cmds []commands.Command, outPath string) error {
	if toolMissing("agg") {
		return errors.New("mp4 export requires 'agg' (asciinema gif generator); install it and retry")
	}
	if toolMissing("ffmpeg") {
		return errors.New("mp4 export requires 'ffmpeg'; install it and retry")
	}

	dir := filepath.Dir(outPath)
	castPath, err := renderCast(ctx, m, evs, cmds, dir)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(castPath) }()

	gifPath, err := tempPath(dir, "gif")
	if err != nil {
		return fmt.Errorf("create temp gif: %w", err)
	}
	defer func() { _ = os.Remove(gifPath) }()

	if err := runRender(ctx, "agg", castPath, gifPath); err != nil {
		return err
	}

	scale := fmt.Sprintf("scale=-2:%d,format=yuv420p", e.height)
	return runRender(ctx, "ffmpeg", "-y", "-i", gifPath, "-vf", scale, outPath)
}
