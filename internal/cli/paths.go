package cli

import (
	"os"
	"path/filepath"
)

func defaultSessionsRoot() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".rekord", "sessions")
	}
	return filepath.Join(".rekord", "sessions")
}
