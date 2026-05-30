package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Omotolani98/rekord/internal/events"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")

func DefaultPatterns() []string {
	return []string{
		`^\$\s+(.+)$`,
		`^>\s+(.+)$`,
		`^❯\s+(.+)$`,
		`^➜\s+(.+)$`,
	}
}

func CompilePatterns(pats []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(pats))
	for _, p := range pats {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid prompt pattern %q: %w", p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

type Extractor struct {
	patterns []*regexp.Regexp
}

func NewExtractor(patterns []*regexp.Regexp) *Extractor {
	return &Extractor{patterns: patterns}
}

type lineRec struct {
	text   string
	timeMs int64
}

func (e *Extractor) Extract(evs []events.Event) []Command {
	lines := toLines(evs)

	var cmds []Command
	for i, ln := range lines {
		text, ok := e.matchPrompt(ln.text)
		if !ok {
			continue
		}
		c := Command{
			Index:       len(cmds) + 1,
			Command:     text,
			StartedAtMs: ln.timeMs,
		}
		for j := i + 1; j < len(lines); j++ {
			if _, isPrompt := e.matchPrompt(lines[j].text); isPrompt {
				break
			}
			if strings.TrimSpace(lines[j].text) == "" {
				continue
			}
			c.OutputPreview = strings.TrimSpace(lines[j].text)
			break
		}
		cmds = append(cmds, c)
	}

	for k := range cmds {
		if k+1 < len(cmds) {
			cmds[k].EndedAtMs = cmds[k+1].StartedAtMs
		} else if len(lines) > 0 {
			cmds[k].EndedAtMs = lines[len(lines)-1].timeMs
		}
	}

	return cmds
}

func (e *Extractor) matchPrompt(line string) (string, bool) {
	for _, re := range e.patterns {
		m := re.FindStringSubmatch(line)
		if len(m) >= 2 {
			cmd := strings.TrimSpace(m[1])
			if cmd != "" {
				return cmd, true
			}
		}
	}
	return "", false
}

func toLines(evs []events.Event) []lineRec {
	var lines []lineRec
	var buf strings.Builder
	var bufTime int64

	flush := func(timeMs int64) {
		s := buf.String()
		buf.Reset()
		s = strings.TrimSuffix(s, "\r")
		s = ansiRe.ReplaceAllString(s, "")
		lines = append(lines, lineRec{text: s, timeMs: timeMs})
	}

	for _, ev := range evs {
		if ev.Type != events.TypeOutput {
			continue
		}
		data := ev.Data
		for {
			idx := strings.IndexByte(data, '\n')
			if idx < 0 {
				if data != "" {
					if buf.Len() == 0 {
						bufTime = ev.TimeMS
					}
					buf.WriteString(data)
				}
				break
			}
			if buf.Len() == 0 {
				bufTime = ev.TimeMS
			}
			buf.WriteString(data[:idx])
			flush(bufTime)
			data = data[idx+1:]
		}
	}
	if buf.Len() > 0 {
		flush(bufTime)
	}

	return lines
}
