package memory

import "time"

const (
	TypeNote     = "note"
	TypeFact     = "fact"
	TypeDecision = "decision"
	TypeTodo     = "todo"
	TypeBlocker  = "blocker"
	TypeWarning  = "warning"

	StatusOpen     = "open"
	StatusResolved = "resolved"

	SourceCLI      = "cli"
	SourceMCP      = "mcp"
	SourceSnapshot = "snapshot"
)

type Memory struct {
	ID               string    `json:"id"`
	Project          string    `json:"project"`
	Agent            string    `json:"agent,omitempty"`
	Actor            string    `json:"actor,omitempty"`
	Source           string    `json:"source,omitempty"`
	SessionID        string    `json:"session_id,omitempty"`
	SessionName      string    `json:"session_name,omitempty"`
	Type             string    `json:"type"`
	Status           string    `json:"status"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	Tags             []string  `json:"tags,omitempty"`
	RelatedFiles     []string  `json:"related_files,omitempty"`
	RelatedSessions  []string  `json:"related_sessions,omitempty"`
	RelatedSnapshots []string  `json:"related_snapshots,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Snapshot struct {
	ID          string     `json:"id"`
	Project     string     `json:"project"`
	Agent       string     `json:"agent,omitempty"`
	Actor       string     `json:"actor,omitempty"`
	Source      string     `json:"source,omitempty"`
	SessionID   string     `json:"session_id,omitempty"`
	SessionName string     `json:"session_name,omitempty"`
	Title       string     `json:"title"`
	Note        string     `json:"note,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	Git         GitState   `json:"git"`
	Patches     []PatchRef `json:"patches,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type GitState struct {
	Branch       string   `json:"branch,omitempty"`
	Head         string   `json:"head,omitempty"`
	IsDirty      bool     `json:"is_dirty"`
	ChangedFiles []string `json:"changed_files,omitempty"`
}

type PatchRef struct {
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type ResumeContext struct {
	Project        string    `json:"project"`
	Agent          string    `json:"agent,omitempty"`
	FromAgent      string    `json:"from_agent,omitempty"`
	ToAgent        string    `json:"to_agent,omitempty"`
	SessionID      string    `json:"session_id,omitempty"`
	SessionName    string    `json:"session_name,omitempty"`
	LatestSnapshot *Snapshot `json:"latest_snapshot,omitempty"`
	OpenMemories   []Memory  `json:"open_memories,omitempty"`
	RecentMemories []Memory  `json:"recent_memories,omitempty"`
	Summary        string    `json:"summary"`
}

type ProjectInfo struct {
	Path string `json:"path"`
	Key  string `json:"key"`
}

type Filter struct {
	Project string
	Agent   string
	Session string
	Status  string
	Limit   int
}
