package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mem "github.com/Omotolani98/rekord/internal/memory"
	"github.com/Omotolani98/rekord/internal/redact"
)

func encode(p string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(p)
}

func canonical(t *testing.T, p string) string {
	t.Helper()
	norm, err := mem.NormalizeProject(p)
	if err != nil {
		t.Fatalf("normalize %q: %v", p, err)
	}
	return norm
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func isolate(t *testing.T) (claudeRootDir, codexRootDir string) {
	t.Helper()
	claudeRootDir = t.TempDir()
	codexRootDir = t.TempDir()
	t.Setenv("REKORD_CLAUDE_PROJECTS", claudeRootDir)
	t.Setenv("REKORD_CODEX_SESSIONS", codexRootDir)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	return
}

func TestClaudeListAndRead(t *testing.T) {
	claudeRootDir, _ := isolate(t)
	project := t.TempDir()
	canon := canonical(t, project)

	dir := filepath.Join(claudeRootDir, encode(canon))
	lines := []string{
		`{"type":"summary","leafUuid":"x"}`,
		`{"type":"user","cwd":"` + project + `","gitBranch":"main","slug":"my-feature","timestamp":"2026-06-01T10:00:00.000Z","message":{"role":"user","content":"build the thing"}}`,
		`{"type":"assistant","cwd":"` + project + `","timestamp":"2026-06-01T10:00:05.000Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hmm"},{"type":"text","text":"on it"},{"type":"tool_use","name":"Edit","input":{}}]}}`,
		`{"type":"user","cwd":"` + project + `","timestamp":"2026-06-01T10:00:09.000Z","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}`,
	}
	writeFile(t, filepath.Join(dir, "sess-1.jsonl"), strings.Join(lines, "\n")+"\n")

	summaries, err := claudeSource{}.List(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("want 1 summary, got %d", len(summaries))
	}
	s := summaries[0]
	if s.SessionID != "sess-1" || s.Source != "claude" {
		t.Fatalf("bad summary: %+v", s)
	}
	if s.Branch != "main" || s.Title != "my-feature" {
		t.Fatalf("missing branch/title: %+v", s)
	}
	if s.FirstPrompt != "build the thing" {
		t.Fatalf("first prompt: %q", s.FirstPrompt)
	}

	tr, err := claudeSource{}.Read(project, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Entries) != 2 {
		t.Fatalf("want 2 entries (tool_result skipped), got %d: %+v", len(tr.Entries), tr.Entries)
	}
	asst := tr.Entries[1]
	if asst.Role != "assistant" || asst.Text != "on it" {
		t.Fatalf("assistant entry: %+v", asst)
	}
	if !asst.ThinkingOmitted {
		t.Fatalf("thinking should be marked omitted")
	}
	if len(asst.Tools) != 1 || asst.Tools[0] != "Edit" {
		t.Fatalf("tools: %+v", asst.Tools)
	}
}

func TestClaudeSubdirMatchesGitRoot(t *testing.T) {
	claudeRootDir, _ := isolate(t)
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	canon := canonical(t, project)
	sub := filepath.Join(canon, "internal")

	dir := filepath.Join(claudeRootDir, encode(sub))
	line := `{"type":"user","cwd":"` + sub + `","timestamp":"2026-06-01T10:00:00.000Z","message":{"role":"user","content":"hi from subdir"}}`
	writeFile(t, filepath.Join(dir, "sub.jsonl"), line+"\n")

	summaries, err := claudeSource{}.List(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("subdir cwd should map to git root; got %d summaries", len(summaries))
	}
}

func TestClaudeIgnoresOtherProject(t *testing.T) {
	claudeRootDir, _ := isolate(t)
	project := t.TempDir()
	other := t.TempDir()
	canonOther := canonical(t, other)

	dir := filepath.Join(claudeRootDir, encode(canonOther))
	line := `{"type":"user","cwd":"` + other + `","timestamp":"2026-06-01T10:00:00.000Z","message":{"role":"user","content":"elsewhere"}}`
	writeFile(t, filepath.Join(dir, "x.jsonl"), line+"\n")

	summaries, err := claudeSource{}.List(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 0 {
		t.Fatalf("should not match other project, got %d", len(summaries))
	}
}

func TestCodexListAndRead(t *testing.T) {
	_, codexRootDir := isolate(t)
	project := t.TempDir()

	dir := filepath.Join(codexRootDir, "2026", "06", "01")
	lines := []string{
		`{"timestamp":"2026-06-01T09:00:00.000Z","type":"session_meta","payload":{"id":"cdx-1","timestamp":"2026-06-01T09:00:00.000Z","cwd":"` + project + `"}}`,
		`{"timestamp":"2026-06-01T09:00:01.000Z","type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"<env>"}]}}`,
		`{"timestamp":"2026-06-01T09:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"refactor parser"}]}}`,
		`{"timestamp":"2026-06-01T09:00:03.000Z","type":"response_item","payload":{"type":"reasoning","summary":[]}}`,
		`{"timestamp":"2026-06-01T09:00:04.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"sure"}]}}`,
		`{"timestamp":"2026-06-01T09:00:05.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{}","call_id":"c1"}}`,
		`{"timestamp":"2026-06-01T09:00:06.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"done"}}`,
	}
	writeFile(t, filepath.Join(dir, "rollout-2026-06-01T09-00-00-cdx-1.jsonl"), strings.Join(lines, "\n")+"\n")

	summaries, err := codexSource{}.List(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("want 1 codex summary, got %d", len(summaries))
	}
	if summaries[0].SessionID != "cdx-1" || summaries[0].FirstPrompt != "refactor parser" {
		t.Fatalf("bad codex summary: %+v", summaries[0])
	}

	tr, err := codexSource{}.Read(project, "cdx-1")
	if err != nil {
		t.Fatal(err)
	}
	roles := make([]string, len(tr.Entries))
	for i, e := range tr.Entries {
		roles[i] = e.Role
	}
	if len(tr.Entries) != 3 {
		t.Fatalf("want 3 entries (developer/reasoning/output skipped), got %d: %v", len(tr.Entries), roles)
	}
	last := tr.Entries[2]
	if last.Role != "assistant" || len(last.Tools) != 1 || last.Tools[0] != "exec_command" {
		t.Fatalf("tool entry: %+v", last)
	}
}

func TestMergeOrderingNewestFirst(t *testing.T) {
	claudeRootDir, codexRootDir := isolate(t)
	project := t.TempDir()
	canon := canonical(t, project)

	cl := filepath.Join(claudeRootDir, encode(canon))
	clLine := `{"type":"user","cwd":"` + project + `","timestamp":"2026-06-05T10:00:00.000Z","message":{"role":"user","content":"newer claude"}}`
	writeFile(t, filepath.Join(cl, "c.jsonl"), clLine+"\n")

	cx := filepath.Join(codexRootDir, "2026", "06", "01")
	cxLines := []string{
		`{"timestamp":"2026-06-01T09:00:00.000Z","type":"session_meta","payload":{"id":"old","timestamp":"2026-06-01T09:00:00.000Z","cwd":"` + project + `"}}`,
		`{"timestamp":"2026-06-01T09:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"older codex"}]}}`,
	}
	writeFile(t, filepath.Join(cx, "rollout-old.jsonl"), strings.Join(cxLines, "\n")+"\n")

	all, err := List(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 merged, got %d", len(all))
	}
	if all[0].Source != "claude" {
		t.Fatalf("newest (claude) should be first, got %s", all[0].Source)
	}
}

func TestRedactTranscript(t *testing.T) {
	tr := Transcript{Entries: []Entry{{Role: "user", Text: "key is sk-ABCDEFGHIJKLMNOPQRSTUVWX"}}}
	out := tr.Redact(redact.NewDefault())
	if strings.Contains(out.Entries[0].Text, "sk-ABCDEFGHIJKLMNOPQRSTUVWX") {
		t.Fatalf("secret not redacted: %q", out.Entries[0].Text)
	}
}

func TestDigest(t *testing.T) {
	tr := Transcript{
		Summary: Summary{Source: "claude", SessionID: "s1", Branch: "main"},
		Entries: []Entry{
			{Role: "user", Text: "do x"},
			{Role: "assistant", Text: "ok", Tools: []string{"Edit", "Edit"}},
		},
	}
	d := Digest(tr, 10, 8000)
	if !strings.Contains(d, "user: do x") {
		t.Fatalf("digest missing user line:\n%s", d)
	}
	if !strings.Contains(d, "assistant: ok [Edit]") {
		t.Fatalf("digest missing assistant/tool line:\n%s", d)
	}
	if len(Digest(tr, 10, 20)) > 40 {
		t.Fatalf("digest not truncated by maxBytes")
	}
}
