package export

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Omotolani98/rekord/internal/commands"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/session"
)

func toolMissing(name string) bool {
	_, err := exec.LookPath(name)
	return err != nil
}

func renderCast(ctx context.Context, m session.Metadata, evs []events.Event, cmds []commands.Command, dir string) (string, error) {
	if err := os.MkdirAll(dir, castExportDirPerm); err != nil {
		return "", fmt.Errorf("create export directory: %w", err)
	}
	f, err := os.CreateTemp(dir, "rekord-*.cast")
	if err != nil {
		return "", fmt.Errorf("create temp cast: %w", err)
	}
	path := f.Name()
	_ = f.Close()

	if err := (CastExporter{}).Export(ctx, m, evs, cmds, path); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func runRender(ctx context.Context, name string, args ...string) error {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, string(out))
	}
	return nil
}

func tempPath(dir, ext string) (string, error) {
	f, err := os.CreateTemp(dir, "rekord-*."+ext)
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	return path, nil
}
