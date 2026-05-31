package cli

import (
	"bytes"
	"testing"

	"github.com/Omotolani98/rekord/internal/session"
)

func TestStylerPaint(t *testing.T) {
	off := styler{on: false}
	if got := off.green("x"); got != "x" {
		t.Errorf("off.green(x) = %q, want %q", got, "x")
	}
	if got := off.statusColor(session.StatusCompleted); got != "completed" {
		t.Errorf("off.statusColor = %q, want %q", got, "completed")
	}

	on := styler{on: true}
	if got := on.green("x"); got != cGreen+"x"+cReset {
		t.Errorf("on.green(x) = %q, want wrapped", got)
	}
	if got := on.green(""); got != "" {
		t.Errorf("on.green(empty) = %q, want empty (no escapes)", got)
	}
	if got := on.statusColor(session.StatusFailed); got != cRed+"failed"+cReset {
		t.Errorf("on.statusColor(failed) = %q, want red-wrapped", got)
	}
}

func TestColorDisabledForBuffer(t *testing.T) {
	if newStyler(&bytes.Buffer{}).on {
		t.Fatal("color must be off for a non-terminal writer")
	}
}
