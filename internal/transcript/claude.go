package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	mem "github.com/Omotolani98/rekord/internal/memory"
)

type claudeSource struct{}

func (claudeSource) Name() string { return "claude" }

func claudeRoot() string {
	if root := os.Getenv("REKORD_CLAUDE_PROJECTS"); root != "" {
		return root
	}
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".claude", "projects")
	}
	return filepath.Join(".claude", "projects")
}

func (claudeSource) Available() bool {
	info, err := os.Stat(claudeRoot())
	return err == nil && info.IsDir()
}

func (c claudeSource) List(project string) ([]Summary, error) {
	files, err := claudeProjectFiles(project)
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, f := range files {
		t, _, ok, err := parseClaudeFile(f)
		if err != nil || !ok {
			continue
		}
		out = append(out, t.Summary)
	}
	return out, nil
}

func (c claudeSource) Read(project, id string) (Transcript, error) {
	files, err := claudeProjectFiles(project)
	if err != nil {
		return Transcript{}, err
	}
	for _, f := range files {
		if strings.TrimSuffix(filepath.Base(f), ".jsonl") != id {
			continue
		}
		t, _, ok, err := parseClaudeFile(f)
		if err != nil {
			return Transcript{}, err
		}
		if ok {
			return t, nil
		}
	}
	return Transcript{}, os.ErrNotExist
}

func claudeProjectFiles(project string) ([]string, error) {
	norm, err := mem.NormalizeProject(project)
	if err != nil {
		return nil, err
	}
	key := mem.ProjectKey(norm)
	root := claudeRoot()
	dirs, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	enc := encodePath(norm)
	var files []string
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		name := d.Name()
		if name != enc && !strings.HasPrefix(name, enc+"-") {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, name))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(root, name, e.Name())
			if cwd := claudeFileCWD(path); cwd != "" && matchProject(cwd, key) {
				files = append(files, path)
			}
		}
	}
	return files, nil
}

func encodePath(p string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(p)
}

type claudeLine struct {
	Type      string         `json:"type"`
	CWD       string         `json:"cwd"`
	GitBranch string         `json:"gitBranch"`
	Timestamp string         `json:"timestamp"`
	Slug      string         `json:"slug"`
	SessionID string         `json:"sessionId"`
	Message   *claudeMessage `json:"message"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func claudeFileCWD(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var l claudeLine
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			continue
		}
		if l.CWD != "" {
			return l.CWD
		}
	}
	return ""
}

func parseClaudeFile(path string) (Transcript, string, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return Transcript{}, "", false, err
	}
	defer f.Close()

	t := Transcript{}
	t.Source = "claude"
	t.SessionID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	cwd := ""

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var l claudeLine
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			continue
		}
		if l.CWD != "" {
			cwd = l.CWD
		}
		if l.GitBranch != "" {
			t.Branch = l.GitBranch
		}
		if l.Slug != "" {
			t.Title = l.Slug
		}
		ts := parseTime(l.Timestamp)
		if !ts.IsZero() {
			if t.StartedAt.IsZero() {
				t.StartedAt = ts
			}
			t.EndedAt = ts
		}
		if l.Type != "user" && l.Type != "assistant" {
			continue
		}
		if l.Message == nil {
			continue
		}
		text, tools, thinking := flattenClaudeContent(l.Message.Content)
		if text == "" && len(tools) == 0 {
			continue
		}
		t.Entries = append(t.Entries, Entry{
			Role:            l.Message.Role,
			Text:            text,
			Tools:           tools,
			ThinkingOmitted: thinking,
			Time:            ts,
		})
		t.Messages++
		if l.Message.Role == "user" && t.FirstPrompt == "" && text != "" {
			t.FirstPrompt = firstLine(text)
		}
	}
	if err := sc.Err(); err != nil {
		return Transcript{}, "", false, err
	}
	t.CWD = cwd
	return t, cwd, len(t.Entries) > 0, nil
}

type claudeBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thinking string `json:"thinking"`
	Name     string `json:"name"`
}

func flattenClaudeContent(raw json.RawMessage) (string, []string, bool) {
	if len(raw) == 0 {
		return "", nil, false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s), nil, false
	}
	var blocks []claudeBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", nil, false
	}
	var texts []string
	var tools []string
	thinking := false
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if t := strings.TrimSpace(b.Text); t != "" {
				texts = append(texts, t)
			}
		case "thinking":
			thinking = true
		case "tool_use":
			if b.Name != "" {
				tools = append(tools, b.Name)
			}
		}
	}
	return strings.Join(texts, "\n"), tools, thinking
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
