package events

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOutputEventJSONRoundTrip(t *testing.T) {
	event := Event{
		TimeMS: 132,
		Type:   TypeOutput,
		Data:   "ok github.com/example/app 0.231s\r\n",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	jsonText := string(data)
	for _, field := range []string{`"timeMs"`, `"type"`, `"data"`} {
		if !strings.Contains(jsonText, field) {
			t.Fatalf("event JSON missing field %s: %s", field, jsonText)
		}
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded != event {
		t.Fatalf("decoded event = %#v, want %#v", decoded, event)
	}
}

func TestResizeEventJSONRoundTrip(t *testing.T) {
	event := Event{
		TimeMS: 700,
		Type:   TypeResize,
		Cols:   120,
		Rows:   40,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	jsonText := string(data)
	for _, field := range []string{`"timeMs"`, `"type"`, `"cols"`, `"rows"`} {
		if !strings.Contains(jsonText, field) {
			t.Fatalf("event JSON missing field %s: %s", field, jsonText)
		}
	}
	if strings.Contains(jsonText, `"data"`) {
		t.Fatalf("resize event should omit empty data: %s", jsonText)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded != event {
		t.Fatalf("decoded event = %#v, want %#v", decoded, event)
	}
}
