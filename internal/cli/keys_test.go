package cli

import "testing"

func TestParseStopKey(t *testing.T) {
	cases := []struct {
		in    string
		want  byte
		label string
	}{
		{"", 0x1d, "Ctrl-]"},
		{"ctrl-]", 0x1d, "Ctrl-]"},
		{"CTRL-X", 0x18, "Ctrl-X"},
		{" ctrl-c ", 0x03, "Ctrl-C"},
	}
	for _, c := range cases {
		b, label, err := parseStopKey(c.in)
		if err != nil {
			t.Fatalf("parseStopKey(%q) error: %v", c.in, err)
		}
		if b != c.want {
			t.Errorf("parseStopKey(%q) byte = %#x, want %#x", c.in, b, c.want)
		}
		if label != c.label {
			t.Errorf("parseStopKey(%q) label = %q, want %q", c.in, label, c.label)
		}
	}

	for _, bad := range []string{"x", "ctrl-", "ctrl-ab", "ctrl-[", "alt-x"} {
		if _, _, err := parseStopKey(bad); err == nil {
			t.Errorf("parseStopKey(%q) expected error, got nil", bad)
		}
	}
}
