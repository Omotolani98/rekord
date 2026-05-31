package redact

import (
	"strings"
	"testing"
)

func TestRedactDefaults(t *testing.T) {
	r := NewDefault()
	cases := []struct {
		name   string
		in     string
		secret string
	}{
		{"openai", "key sk-abcdef0123456789ABCD here", "sk-abcdef0123456789ABCD"},
		{"github", "tok ghp_abcdefghijklmnopqrstuvwxyz0123 x", "ghp_abcdefghijklmnopqrstuvwxyz0123"},
		{"aws", "id AKIAABCDEFGHIJKLMNOP end", "AKIAABCDEFGHIJKLMNOP"},
		{"postgres", "url postgres://user:pass@host:5432/db now", "postgres://user:pass@host:5432/db"},
		{"password", "password=hunter2", "hunter2"},
		{"env", "OPENAI_API_KEY=sk-zzzzzzzzzzzzzzzzzzzz", "sk-zzzzzzzzzzzzzzzzzzzz"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r.Redact(c.in)
			if strings.Contains(got, c.secret) {
				t.Fatalf("secret leaked: %q", got)
			}
			if !strings.Contains(got, Placeholder) {
				t.Fatalf("no placeholder: %q", got)
			}
		})
	}
}

func TestRedactKeepsKey(t *testing.T) {
	got := NewDefault().Redact("password=hunter2")
	if got != "password="+Placeholder {
		t.Fatalf("got %q, want password=%s", got, Placeholder)
	}
}

func TestRedactPlainUnchanged(t *testing.T) {
	in := "just a normal line with no secrets"
	if got := NewDefault().Redact(in); got != in {
		t.Fatalf("got %q, want unchanged", got)
	}
}

func TestScanCategories(t *testing.T) {
	cats := NewDefault().Scan("password=hunter2 and postgres://u:p@h/db")
	joined := strings.Join(cats, ",")
	if !strings.Contains(joined, "password") || !strings.Contains(joined, "postgres-url") {
		t.Fatalf("categories = %v", cats)
	}
	for i := 1; i < len(cats); i++ {
		if cats[i-1] > cats[i] {
			t.Fatalf("categories not sorted: %v", cats)
		}
	}
}

func TestScanClean(t *testing.T) {
	if cats := NewDefault().Scan("nothing secret here"); len(cats) != 0 {
		t.Fatalf("cats = %v, want empty", cats)
	}
}
