package session

import "time"

type Status string

const (
	StatusRecording Status = "recording"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Metadata struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	CreatedAt     time.Time  `json:"createdAt"`
	EndedAt       *time.Time `json:"endedAt,omitempty"`
	DurationMS    int64      `json:"durationMs"`
	Shell         string     `json:"shell"`
	Command       []string   `json:"command,omitempty"`
	CWD           string     `json:"cwd"`
	Cols          int        `json:"cols"`
	Rows          int        `json:"rows"`
	Status        Status     `json:"status"`
	RekordVersion string     `json:"rekordVersion"`
}
