package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func DefaultRoot() string {
	if root := os.Getenv("REKORD_MEMORY_ROOT"); root != "" {
		return root
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".rekord", "projects")
	}
	return filepath.Join(".rekord", "projects")
}

func NormalizeProject(path string) (string, error) {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve project path: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = real
	}
	abs = filepath.Clean(abs)
	if root, ok := gitRoot(abs); ok {
		return root, nil
	}
	return abs, nil
}

func gitRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func ProjectKey(project string) string {
	sum := sha256.Sum256([]byte(project))
	return hex.EncodeToString(sum[:])[:16]
}
