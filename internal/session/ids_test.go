package session

import (
	"testing"
	"time"
)

func TestNewIDFormat(t *testing.T) {
	now := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	got := NewID("monocron-demo", now)
	want := "20260530-080000-monocron-demo"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNewIDNormalizesToUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("tz data unavailable")
	}
	now := time.Date(2026, 5, 30, 1, 0, 0, 0, loc)
	got := NewID("demo", now)
	want := "20260530-080000-demo"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSlugCases(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Hello World!", "hello-world"},
		{"   ", fallbackSlug},
		{"", fallbackSlug},
		{"foo--bar", "foo-bar"},
		{`a/b\c`, "a-b-c"},
		{"--leading", "leading"},
		{"trailing--", "trailing"},
		{"Mixed_Case_123", "mixed-case-123"},
	}
	for _, c := range cases {
		if got := slug(c.in); got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidateName(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		if err := ValidateName(in); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", in)
		}
	}
	if err := ValidateName("demo"); err != nil {
		t.Errorf("ValidateName(\"demo\") = %v, want nil", err)
	}
}
