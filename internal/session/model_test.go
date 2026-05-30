package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMetadataJSONRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	endedAt := createdAt.Add(74 * time.Second)
	metadata := Metadata{
		ID:            "20260530-080000-monocron-demo",
		Name:          "monocron-demo",
		CreatedAt:     createdAt,
		EndedAt:       &endedAt,
		DurationMS:    74000,
		Shell:         "/bin/zsh",
		CWD:           "/Users/tolani/projects/monocron",
		Cols:          120,
		Rows:          40,
		Status:        StatusCompleted,
		RekordVersion: "0.1.0",
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	jsonText := string(data)
	for _, field := range []string{
		`"id"`,
		`"createdAt"`,
		`"endedAt"`,
		`"durationMs"`,
		`"rekordVersion"`,
	} {
		if !strings.Contains(jsonText, field) {
			t.Fatalf("metadata JSON missing field %s: %s", field, jsonText)
		}
	}

	var decoded Metadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.ID != metadata.ID || decoded.Name != metadata.Name {
		t.Fatalf("decoded identity = (%q, %q), want (%q, %q)", decoded.ID, decoded.Name, metadata.ID, metadata.Name)
	}
	if !decoded.CreatedAt.Equal(createdAt) {
		t.Fatalf("decoded CreatedAt = %s, want %s", decoded.CreatedAt, createdAt)
	}
	if decoded.EndedAt == nil || !decoded.EndedAt.Equal(endedAt) {
		t.Fatalf("decoded EndedAt = %v, want %s", decoded.EndedAt, endedAt)
	}
	if decoded.Status != StatusCompleted {
		t.Fatalf("decoded Status = %q, want %q", decoded.Status, StatusCompleted)
	}
}

func TestMetadataOmitsEmptyEndedAt(t *testing.T) {
	metadata := Metadata{
		ID:        "20260530-080000-demo",
		Name:      "demo",
		CreatedAt: time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC),
		Status:    StatusRecording,
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	if strings.Contains(string(data), "endedAt") {
		t.Fatalf("metadata JSON included empty endedAt: %s", data)
	}
}
