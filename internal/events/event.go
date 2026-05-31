package events

type Type string

const (
	TypeOutput Type = "output"
	TypeInput  Type = "input"
	TypeResize Type = "resize"
	TypeMarker Type = "marker"
)

type Event struct {
	TimeMS int64  `json:"timeMs"`
	Type   Type   `json:"type"`
	Data   string `json:"data,omitempty"`
	Cols   int    `json:"cols,omitempty"`
	Rows   int    `json:"rows,omitempty"`
}
