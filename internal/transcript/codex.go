package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	mem "github.com/Omotolani98/rekord/internal/memory"
)

type codexSource struct{}

func (codexSource) Name() string { return "codex" }

func codexRoot() string {
	if root := os.Getenv("REKORD_CODEX_SESSIONS"); root != "" {
		return root
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codex", "sessions")
	}
	return filepath.Join(".codex", "sessions")
}

func (codexSource) Available() bool {
	info, err := os.Stat(codexRoot())
	return err == nil && info.IsDir()
}

func (c codexSource) List(project string) ([]Summary, error) {
	key, err := projectKey(project)
	if err != nil {
		return nil, err
	}
	var out []Summary
	err = walkCodexFiles(func(path string) {
		t, ok := parseCodexFile(path)
		if !ok || !matchProject(t.CWD, key) {
			return
		}
		out = append(out, t.Summary)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c codexSource) Read(project, id string) (Transcript, error) {
	key, err := projectKey(project)
	if err != nil {
		return Transcript{}, err
	}
	var found Transcript
	ok := false
	err = walkCodexFiles(func(path string) {
		if ok {
			return
		}
		t, parsed := parseCodexFile(path)
		if !parsed || t.SessionID != id || !matchProject(t.CWD, key) {
			return
		}
		found = t
		ok = true
	})
	if err != nil {
		return Transcript{}, err
	}
	if !ok {
		return Transcript{}, os.ErrNotExist
	}
	return found, nil
}

func projectKey(project string) (string, error) {
	norm, err := mem.NormalizeProject(project)
	if err != nil {
		return "", err
	}
	return mem.ProjectKey(norm), nil
}

func walkCodexFiles(fn func(path string)) error {
	root := codexRoot()
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		fn(path)
		return nil
	})
}

type codexLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexMeta struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
}

type codexPayload struct {
	Type    string         `json:"type"`
	Role    string         `json:"role"`
	Name    string         `json:"name"`
	Content []codexContent `json:"content"`
}

type codexContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func parseCodexFile(path string) (Transcript, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Transcript{}, false
	}
	defer f.Close()

	t := Transcript{}
	t.Source = "codex"

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	for sc.Scan() {
		var l codexLine
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			continue
		}
		ts := parseTime(l.Timestamp)
		switch l.Type {
		case "session_meta":
			var m codexMeta
			if err := json.Unmarshal(l.Payload, &m); err != nil {
				continue
			}
			t.SessionID = m.ID
			t.CWD = m.CWD
			if mt := parseTime(m.Timestamp); !mt.IsZero() {
				t.StartedAt = mt
			}
		case "response_item":
			var p codexPayload
			if err := json.Unmarshal(l.Payload, &p); err != nil {
				continue
			}
			e, ok := codexEntry(p)
			if !ok {
				continue
			}
			e.Time = ts
			t.Entries = append(t.Entries, e)
			t.Messages++
			if !ts.IsZero() {
				t.EndedAt = ts
			}
			if e.Role == "user" && t.FirstPrompt == "" && e.Text != "" && !strings.HasPrefix(strings.TrimSpace(e.Text), "<") {
				t.FirstPrompt = firstLine(e.Text)
			}
		}
	}
	if t.SessionID == "" {
		return Transcript{}, false
	}
	if t.Title == "" {
		t.Title = t.FirstPrompt
	}
	return t, len(t.Entries) > 0
}

func codexEntry(p codexPayload) (Entry, bool) {
	switch p.Type {
	case "message":
		if p.Role != "user" && p.Role != "assistant" {
			return Entry{}, false
		}
		var texts []string
		for _, c := range p.Content {
			switch c.Type {
			case "input_text", "output_text", "text":
				if txt := strings.TrimSpace(c.Text); txt != "" {
					texts = append(texts, txt)
				}
			}
		}
		if len(texts) == 0 {
			return Entry{}, false
		}
		return Entry{Role: p.Role, Text: strings.Join(texts, "\n")}, true
	case "function_call", "custom_tool_call":
		if p.Name == "" {
			return Entry{}, false
		}
		return Entry{Role: "assistant", Tools: []string{p.Name}}, true
	}
	return Entry{}, false
}
