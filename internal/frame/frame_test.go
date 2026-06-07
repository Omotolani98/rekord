package frame

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Omotolani98/rekord/internal/redact"
	vt "github.com/charmbracelet/x/vt"
)

func TestFromScreenPlainText(t *testing.T) {
	e := vt.NewEmulator(10, 3)
	e.WriteString("hi")

	f := FromScreen(e)

	if f.Cols != 10 || f.Rows != 3 {
		t.Fatalf("dims = %dx%d, want 10x3", f.Cols, f.Rows)
	}
	if f.Lines[0] != "hi" {
		t.Fatalf("line 0 = %q, want %q", f.Lines[0], "hi")
	}
	if f.Cursor.X != 2 || f.Cursor.Y != 0 {
		t.Fatalf("cursor = (%d,%d), want (2,0)", f.Cursor.X, f.Cursor.Y)
	}
}

func TestFromScreenCursorMoveAndColor(t *testing.T) {
	e := vt.NewEmulator(10, 3)
	e.WriteString("\x1b[2;3H\x1b[31mok\x1b[0m")

	f := FromScreen(e)

	if f.Lines[1] != "  ok" {
		t.Fatalf("line 1 = %q, want %q", f.Lines[1], "  ok")
	}
	if strings.Contains(f.Text(), "\x1b") {
		t.Fatalf("text retained escape sequences: %q", f.Text())
	}
}

func TestFromScreenWideRune(t *testing.T) {
	e := vt.NewEmulator(10, 1)
	e.WriteString("世x")

	f := FromScreen(e)

	if f.Lines[0] != "世x" {
		t.Fatalf("line 0 = %q, want %q", f.Lines[0], "世x")
	}
}

func TestRedact(t *testing.T) {
	e := vt.NewEmulator(40, 1)
	e.WriteString("token=abcdef123456")

	f := FromScreen(e).Redact(redact.NewDefault())

	if strings.Contains(f.Text(), "abcdef123456") {
		t.Fatalf("secret not redacted: %q", f.Text())
	}
	if !strings.Contains(f.Text(), "[REDACTED]") {
		t.Fatalf("missing redaction marker: %q", f.Text())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	e := vt.NewEmulator(5, 2)
	e.WriteString("ab")

	data, err := json.Marshal(FromScreen(e))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Frame
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Lines[0] != "ab" || got.Cursor.X != 2 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}
