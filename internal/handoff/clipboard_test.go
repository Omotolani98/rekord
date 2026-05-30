package handoff

import (
	"os/exec"
	"testing"
)

func TestCopy(t *testing.T) {
	available := false
	for _, t := range clipboardTools {
		if _, err := exec.LookPath(t.name); err == nil {
			available = true
			break
		}
	}
	if !available {
		t.Skip("no clipboard tool available")
	}
	if err := Copy("rekord-clipboard-test"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
}
