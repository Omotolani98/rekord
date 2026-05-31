package cli

import (
	"io"
	"os"

	"github.com/Omotolani98/rekord/internal/session"
	"golang.org/x/term"
)

const (
	cReset  = "\x1b[0m"
	cBold   = "\x1b[1m"
	cDim    = "\x1b[2m"
	cRed    = "\x1b[31m"
	cGreen  = "\x1b[32m"
	cYellow = "\x1b[33m"
)

type styler struct{ on bool }

func newStyler(w io.Writer) styler { return styler{on: colorEnabled(w)} }

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (s styler) paint(code, t string) string {
	if !s.on || t == "" {
		return t
	}
	return code + t + cReset
}

func (s styler) bold(t string) string   { return s.paint(cBold, t) }
func (s styler) dim(t string) string    { return s.paint(cDim, t) }
func (s styler) red(t string) string    { return s.paint(cRed, t) }
func (s styler) green(t string) string  { return s.paint(cGreen, t) }
func (s styler) yellow(t string) string { return s.paint(cYellow, t) }

func (s styler) statusColor(st session.Status) string {
	switch st {
	case session.StatusCompleted:
		return s.green(string(st))
	case session.StatusRecording:
		return s.yellow(string(st))
	case session.StatusFailed:
		return s.red(string(st))
	}
	return string(st)
}
