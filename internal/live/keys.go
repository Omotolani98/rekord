package live

import (
	"fmt"
	"strings"
)

var keyBytes = map[string][]byte{
	"enter":     []byte("\r"),
	"return":    []byte("\r"),
	"tab":       []byte("\t"),
	"escape":    []byte("\x1b"),
	"esc":       []byte("\x1b"),
	"space":     []byte(" "),
	"backspace": []byte("\x7f"),
	"delete":    []byte("\x1b[3~"),
	"up":        []byte("\x1b[A"),
	"down":      []byte("\x1b[B"),
	"right":     []byte("\x1b[C"),
	"left":      []byte("\x1b[D"),
	"home":      []byte("\x1b[H"),
	"end":       []byte("\x1b[F"),
	"pageup":    []byte("\x1b[5~"),
	"pagedown":  []byte("\x1b[6~"),
	"ctrl-a":    {0x01},
	"ctrl-b":    {0x02},
	"ctrl-c":    {0x03},
	"ctrl-d":    {0x04},
	"ctrl-e":    {0x05},
	"ctrl-k":    {0x0b},
	"ctrl-l":    {0x0c},
	"ctrl-r":    {0x12},
	"ctrl-u":    {0x15},
	"ctrl-w":    {0x17},
	"ctrl-z":    {0x1a},
}

func TranslateKey(name string) ([]byte, error) {
	b, ok := keyBytes[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("unknown key: %q", name)
	}
	return b, nil
}

func BuildInput(text string, keys []string) ([]byte, error) {
	buf := []byte(text)
	for _, k := range keys {
		b, err := TranslateKey(k)
		if err != nil {
			return nil, err
		}
		buf = append(buf, b...)
	}
	return buf, nil
}
