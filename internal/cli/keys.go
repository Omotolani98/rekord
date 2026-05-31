package cli

import (
	"fmt"
	"strings"
)

const defaultStopKey = "ctrl-]"

func parseStopKey(s string) (byte, string, error) {
	if s == "" {
		s = defaultStopKey
	}
	norm := strings.ToLower(strings.TrimSpace(s))
	rest, ok := strings.CutPrefix(norm, "ctrl-")
	if !ok || len(rest) != 1 {
		return 0, "", fmt.Errorf("stop key %q must be ctrl-<char> (e.g. ctrl-])", s)
	}
	b := rest[0] & 0x1f
	if b == 0 || b == 0x1b {
		return 0, "", fmt.Errorf("unsupported stop key %q", s)
	}
	return b, "Ctrl-" + strings.ToUpper(rest), nil
}
