package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/rekord/internal/session"
)

func TestListCommandEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"list", "--root", t.TempDir()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Execute returned %d, want 0; stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "No sessions recorded yet.\n" {
		t.Fatalf("stdout = %q, want %q", got, "No sessions recorded yet.\n")
	}
}

func TestListCommandRendersTable(t *testing.T) {
	root := t.TempDir()
	store := session.NewFileStore(root)
	created := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	ended := created.Add(42 * time.Second)
	m := session.Metadata{
		ID:            "20260530-080000-monocron-demo",
		Name:          "monocron-demo",
		CreatedAt:     created,
		EndedAt:       &ended,
		DurationMS:    42000,
		Shell:         "/bin/zsh",
		CWD:           "/tmp",
		Cols:          120,
		Rows:          40,
		Status:        session.StatusCompleted,
		RekordVersion: "0.1.0",
	}
	if err := store.Create(context.Background(), m); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Execute([]string{"list", "--root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Execute returned %d, want 0; stderr=%q", code, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{"NAME", "DURATION", "STATUS", "CREATED", "monocron-demo", "42s", "completed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}
