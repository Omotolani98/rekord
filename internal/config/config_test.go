package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPromptPatterns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rekord.yaml")
	body := "commands:\n  promptPatterns:\n    - \"^PROMPT> (.+)$\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Commands.PromptPatterns) != 1 || cfg.Commands.PromptPatterns[0] != "^PROMPT> (.+)$" {
		t.Fatalf("patterns = %v", cfg.Commands.PromptPatterns)
	}
}

func TestLoadMissingFileDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(cfg.Commands.PromptPatterns) != 0 {
		t.Fatalf("patterns = %v, want empty default", cfg.Commands.PromptPatterns)
	}
}

func TestLoadMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("commands: [unterminated\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load err = nil, want error for malformed YAML")
	}
}
