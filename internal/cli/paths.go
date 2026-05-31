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

func defaultConfigPath() string {
	const name = "rekord.yaml"
	if _, err := os.Stat(name); err == nil {
		return name
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".rekord", name)
	}
	return name
}
