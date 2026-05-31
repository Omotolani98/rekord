package commands

type Command struct {
	Index         int    `json:"index"`
	Command       string `json:"command"`
	StartedAtMs   int64  `json:"startedAtMs"`
	EndedAtMs     int64  `json:"endedAtMs"`
	ExitCode      *int   `json:"exitCode"`
	OutputPreview string `json:"outputPreview"`
}
