package frame

import (
	"strings"

	"github.com/Omotolani98/rekord/internal/redact"
	uv "github.com/charmbracelet/ultraviolet"
)

type Cursor struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Frame struct {
	Cols   int      `json:"cols"`
	Rows   int      `json:"rows"`
	Cursor Cursor   `json:"cursor"`
	Lines  []string `json:"lines"`
}

type Screen interface {
	Width() int
	Height() int
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
}

func FromScreen(s Screen) Frame {
	w, h := s.Width(), s.Height()
	lines := make([]string, h)
	for y := range h {
		lines[y] = lineText(s, y, w)
	}
	pos := s.CursorPosition()
	return Frame{
		Cols:   w,
		Rows:   h,
		Cursor: Cursor{X: pos.X, Y: pos.Y},
		Lines:  lines,
	}
}

func lineText(s Screen, y, w int) string {
	var b strings.Builder
	skip := 0
	for x := range w {
		if skip > 0 {
			skip--
			continue
		}
		c := s.CellAt(x, y)
		if c == nil {
			b.WriteByte(' ')
			continue
		}
		if c.Width >= 2 {
			skip = c.Width - 1
		}
		if c.Content == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(c.Content)
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func (f Frame) Text() string {
	return strings.Join(f.Lines, "\n")
}

func (f Frame) Redact(r *redact.Redactor) Frame {
	if r == nil {
		return f
	}
	lines := make([]string, len(f.Lines))
	for i, l := range f.Lines {
		lines[i] = r.Redact(l)
	}
	return Frame{Cols: f.Cols, Rows: f.Rows, Cursor: f.Cursor, Lines: lines}
}
