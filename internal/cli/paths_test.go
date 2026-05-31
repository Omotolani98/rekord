package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(work)

	wantHome := filepath.Join(home, ".rekord", "rekord.yaml")
	if got := defaultConfigPath(); got != wantHome {
		t.Fatalf("no cwd config: got %q, want %q", got, wantHome)
	}

	if err := os.WriteFile(filepath.Join(work, "rekord.yaml"), []byte("recording: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := defaultConfigPath(); got != "rekord.yaml" {
		t.Fatalf("cwd config present: got %q, want %q", got, "rekord.yaml")
	}
}
