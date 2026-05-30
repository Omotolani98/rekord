package skills

import (
	"strings"
	"testing"
)

func TestRenderScript(t *testing.T) {
	s := Skill{
		Name:  "demo",
		Steps: []Step{{Run: "go version"}, {Run: "go test ./..."}},
	}
	script := RenderScript(s)

	if !strings.HasPrefix(script, "set -e\n") {
		t.Fatalf("missing set -e:\n%s", script)
	}
	for _, want := range []string{
		"echo '$ go version'",
		"go version",
		"echo '$ go test ./...'",
		"go test ./...",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q:\n%s", want, script)
		}
	}
	if strings.Index(script, "go version") > strings.Index(script, "go test ./...") {
		t.Fatalf("steps out of order:\n%s", script)
	}
}
