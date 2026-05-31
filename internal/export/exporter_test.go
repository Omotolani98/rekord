package export

import "testing"

func TestGetFormats(t *testing.T) {
	cases := []struct{ format, ext string }{
		{"cast", "cast"},
		{"json", "json"},
		{"markdown", "md"},
		{"script", "sh"},
		{"gif", "gif"},
		{"mp4", "mp4"},
	}
	for _, c := range cases {
		exp, err := Get(c.format, "")
		if err != nil {
			t.Fatalf("Get(%s): %v", c.format, err)
		}
		if exp.Format() != c.format {
			t.Fatalf("Format = %q, want %q", exp.Format(), c.format)
		}
		if exp.Ext() != c.ext {
			t.Fatalf("Ext = %q, want %q", exp.Ext(), c.ext)
		}
	}
}

func TestGetMP4Size(t *testing.T) {
	if _, err := Get("mp4", "1080p"); err != nil {
		t.Fatalf("Get(mp4, 1080p): %v", err)
	}
	if _, err := Get("mp4", "bogus"); err == nil {
		t.Fatal("Get(mp4, bogus) err = nil, want error")
	}
}

func TestGetUnknown(t *testing.T) {
	if _, err := Get("bogus", ""); err == nil {
		t.Fatal("Get(bogus) err = nil, want error")
	}
}
