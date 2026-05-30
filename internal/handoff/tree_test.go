package handoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildTreeExcludes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"))
	mustWrite(t, filepath.Join(root, "pkg", "util.go"))
	mustWrite(t, filepath.Join(root, "node_modules", "dep", "index.js"))

	tree, err := BuildTree(root, 4, 500)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	if !strings.Contains(tree, "main.go") || !strings.Contains(tree, "pkg/") {
		t.Fatalf("tree missing expected entries:\n%s", tree)
	}
	if strings.Contains(tree, "node_modules") || strings.Contains(tree, "index.js") {
		t.Fatalf("tree should exclude node_modules:\n%s", tree)
	}
}

func TestBuildTreeMaxFiles(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 10; i++ {
		mustWrite(t, filepath.Join(root, "f"+string(rune('0'+i))+".txt"))
	}
	tree, err := BuildTree(root, 4, 3)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	if !strings.Contains(tree, "… (truncated)") {
		t.Fatalf("expected truncation:\n%s", tree)
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}
