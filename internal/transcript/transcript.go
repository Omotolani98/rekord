package transcript

import (
	"fmt"
	"sort"
	"strings"
	"time"

	mem "github.com/Omotolani98/rekord/internal/memory"
	"github.com/Omotolani98/rekord/internal/redact"
)

// Entry is one flattened turn from an agent transcript.
type Entry struct {
	Role            string    `json:"role"`
	Text            string    `json:"text,omitempty"`
	Tools           []string  `json:"tools,omitempty"`
	ThinkingOmitted bool      `json:"thinkingOmitted,omitempty"`
	Time            time.Time `json:"time,omitempty"`
}

// Summary is lightweight metadata for listing a transcript without parsing it fully.
type Summary struct {
	Source      string    `json:"source"`
	SessionID   string    `json:"sessionId"`
	Title       string    `json:"title,omitempty"`
	Branch      string    `json:"branch,omitempty"`
	CWD         string    `json:"cwd,omitempty"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	EndedAt     time.Time `json:"endedAt,omitempty"`
	Messages    int       `json:"messages"`
	FirstPrompt string    `json:"firstPrompt,omitempty"`
}

// Transcript is a full flattened agent session.
type Transcript struct {
	Summary
	Entries []Entry `json:"entries"`
}

// Source reads sessions recorded by one coding agent and maps them to rekord projects.
type Source interface {
	Name() string
	Available() bool
	List(project string) ([]Summary, error)
	Read(project, id string) (Transcript, error)
}

// DefaultSources returns the built-in agent adapters.
func DefaultSources() []Source {
	return []Source{claudeSource{}, codexSource{}}
}

// Sources returns the names of available agent transcript sources.
func Sources() []string {
	var out []string
	for _, s := range DefaultSources() {
		if s.Available() {
			out = append(out, s.Name())
		}
	}
	return out
}

// List merges transcript summaries across all available sources, newest first.
func List(project string) ([]Summary, error) {
	var all []Summary
	for _, s := range DefaultSources() {
		if !s.Available() {
			continue
		}
		items, err := s.List(project)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Name(), err)
		}
		all = append(all, items...)
	}
	sortSummaries(all)
	return all, nil
}

// Read returns one transcript from the named source.
func Read(project, source, id string) (Transcript, error) {
	for _, s := range DefaultSources() {
		if s.Name() != source {
			continue
		}
		if !s.Available() {
			return Transcript{}, fmt.Errorf("source %q is not available", source)
		}
		return s.Read(project, id)
	}
	return Transcript{}, fmt.Errorf("unknown source: %q", source)
}

// Search returns transcript summaries whose prompts or entry text match the query.
func Search(project, query string) ([]Summary, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return List(project)
	}
	summaries, err := List(project)
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, s := range summaries {
		if strings.Contains(strings.ToLower(s.Title), q) || strings.Contains(strings.ToLower(s.FirstPrompt), q) {
			out = append(out, s)
			continue
		}
		t, err := Read(project, s.Source, s.SessionID)
		if err != nil {
			continue
		}
		if t.matches(q) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (t Transcript) matches(lowerQuery string) bool {
	for _, e := range t.Entries {
		if strings.Contains(strings.ToLower(e.Text), lowerQuery) {
			return true
		}
	}
	return false
}

// Digest renders a compact, agent-ready summary of a transcript.
func Digest(t Transcript, lastN, maxBytes int) string {
	if lastN <= 0 {
		lastN = 20
	}
	if maxBytes <= 0 {
		maxBytes = 8000
	}
	var b strings.Builder
	header := fmt.Sprintf("Transcript from %s session %s", t.Source, t.SessionID)
	if t.Branch != "" {
		header += " (branch " + t.Branch + ")"
	}
	b.WriteString(header + "\n")
	if t.Title != "" {
		b.WriteString("Title: " + t.Title + "\n")
	}
	b.WriteString("\n")

	entries := t.Entries
	if len(entries) > lastN {
		entries = entries[len(entries)-lastN:]
	}
	for _, e := range entries {
		line := strings.TrimSpace(e.Text)
		var parts []string
		if line != "" {
			parts = append(parts, line)
		}
		if len(e.Tools) > 0 {
			parts = append(parts, "["+strings.Join(uniqueTools(e.Tools), ", ")+"]")
		}
		if len(parts) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", e.Role, strings.Join(parts, " "))
	}
	return truncateBytes(b.String(), maxBytes)
}

// Redact returns a copy of the transcript with secrets removed from text fields.
func (t Transcript) Redact(r *redact.Redactor) Transcript {
	if r == nil {
		return t
	}
	out := t
	out.Summary = t.Summary.Redact(r)
	out.Entries = make([]Entry, len(t.Entries))
	for i, e := range t.Entries {
		e.Text = r.Redact(e.Text)
		out.Entries[i] = e
	}
	return out
}

// Redact returns a copy of the summary with secrets removed from text fields.
func (s Summary) Redact(r *redact.Redactor) Summary {
	if r == nil {
		return s
	}
	s.Title = r.Redact(s.Title)
	s.FirstPrompt = r.Redact(s.FirstPrompt)
	return s
}

func matchProject(cwd, targetKey string) bool {
	if cwd == "" {
		return false
	}
	norm, err := mem.NormalizeProject(cwd)
	if err != nil {
		return false
	}
	return mem.ProjectKey(norm) == targetKey
}

func sortSummaries(s []Summary) {
	sort.SliceStable(s, func(i, j int) bool {
		return s[i].EndedAt.After(s[j].EndedAt)
	})
}

func uniqueTools(tools []string) []string {
	seen := make(map[string]struct{}, len(tools))
	var out []string
	for _, t := range tools {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func truncateBytes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	cut := s[:max]
	if i := strings.LastIndexByte(cut, '\n'); i > max/2 {
		cut = cut[:i]
	}
	return cut + "\n…(truncated)"
}
