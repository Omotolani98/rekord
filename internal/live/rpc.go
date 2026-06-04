package live

import "github.com/Omotolani98/rekord/internal/frame"

type Request struct {
	Op        string   `json:"op"`
	Text      string   `json:"text,omitempty"`
	Keys      []string `json:"keys,omitempty"`
	Sub       string   `json:"sub,omitempty"`
	QuietMs   int      `json:"quietMs,omitempty"`
	TimeoutMs int      `json:"timeoutMs,omitempty"`
	MaxBytes  int      `json:"maxBytes,omitempty"`
	Cols      int      `json:"cols,omitempty"`
	Rows      int      `json:"rows,omitempty"`
}

type Response struct {
	OK       bool         `json:"ok"`
	Error    string       `json:"error,omitempty"`
	Reason   string       `json:"reason,omitempty"`
	ExitCode *int         `json:"exitCode,omitempty"`
	Sent     int          `json:"sent,omitempty"`
	Logs     string       `json:"logs,omitempty"`
	Frame    *frame.Frame `json:"frame,omitempty"`
	Status   *Status      `json:"status,omitempty"`
}
