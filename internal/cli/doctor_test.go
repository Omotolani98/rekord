package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorListsTools(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, tool := range []string{"ffmpeg", "agg", "asciinema", "tmux", "git"} {
		if !strings.Contains(out, tool) {
			t.Fatalf("doctor output missing %q:\n%s", tool, out)
		}
	}
}
