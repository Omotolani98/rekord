package export

import "testing"

func TestGetFormats(t *testing.T) {
	cases := []struct{ format, ext string }{
		{"cast", "cast"},
		{"json", "json"},
		{"markdown", "md"},
		{"script", "sh"},
	}
	for _, c := range cases {
		exp, err := Get(c.format)
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

func TestGetUnknown(t *testing.T) {
	if _, err := Get("bogus"); err == nil {
		t.Fatal("Get(bogus) err = nil, want error")
	}
}
