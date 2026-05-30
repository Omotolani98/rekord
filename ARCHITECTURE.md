# Rekord Architecture

Rekord is an open-source CLI application for recording terminal workflows and exporting them into useful formats such as terminal casts, Markdown documentation, JSON, GIF/video, and AI-ready context bundles.

The core principle is simple:

> Record terminal sessions as structured data first, then export that data into many formats.

This makes Rekord more than a screen recorder. It becomes a developer workflow capture tool for demos, documentation, debugging, DevRel, tutorials, and AI agent handoff.

---

## 1. Product Goals

Rekord should help developers capture what they are building with minimal friction.

Primary goals:

1. Record an interactive terminal session.
2. Store terminal activity as replayable structured data.
3. Extract commands and useful metadata from the session.
4. Export recordings to useful formats.
5. Protect users from leaking secrets.
6. Support tmux-based workflows.
7. Generate AI-ready context from a recorded session.
8. Support reusable recording recipes called skills.
9. Keep features small and independently buildable so Codex or other AI coding agents can implement one feature at a time.

Non-goals for the first version:

1. Building a full GUI editor.
2. Building a custom video renderer from scratch.
3. Cloud syncing or hosted storage.
4. Real-time collaboration.
5. Full shell parsing correctness for every shell edge case.

---

## 2. Core User Experience

### Basic usage

```bash
rekord start --name monocron-demo
```

The user runs commands normally inside the recorded shell.

```bash
go test ./...
docker compose up -d
curl localhost:8080/healthz
exit
```

Then the user exports the session.

```bash
rekord export monocron-demo --to markdown
rekord export monocron-demo --to cast
rekord handoff monocron-demo
```

### One-shot timer mode

```bash
rekord start --name quick-demo --timer 40s
rekord start --name long-demo --timer 5m
```

### Record a single command

```bash
rekord run --name tests -- go test ./...
```

### Tmux mode

```bash
rekord tmux start --session monocron-demo
rekord tmux export monocron-demo --to markdown
```

### AI handoff

```bash
rekord handoff monocron-demo --include-git --include-tree --include-logs
```

### Skills

```bash
rekord skills list
rekord skills run go-project-demo
```

---

## 3. Design Principles

1. **Data first**: Record terminal output and metadata as structured session data before exporting.
2. **Small feature slices**: Every feature should be independently buildable and testable.
3. **Local first**: Store everything locally under `.rekord/` or a user-level Rekord directory.
4. **Safe by default**: Redaction should be available early and enabled for sensitive exports.
5. **Composable**: Use external tools like `ffmpeg` for video export instead of reinventing everything.
6. **Scriptable**: Every feature should work from CLI commands without requiring UI interaction.
7. **Open-source friendly**: Clear project structure, simple config, good docs, and predictable behavior.
8. **AI-agent friendly**: Tasks should be easy for Codex to pick up feature-by-feature.

---

## 4. Recommended Technology Stack

### Language

Use **Go**.

Reasons:

1. Strong CLI ecosystem.
2. Good concurrency primitives.
3. Easy single-binary distribution.
4. Good fit for terminal, process, filesystem, and PTY work.
5. Good cross-platform build support.

### CLI framework

Recommended:

```text
github.com/spf13/cobra
```

Use Cobra for command organization:

```text
rekord start
rekord run
rekord list
rekord replay
rekord export
rekord scan
rekord handoff
rekord tmux
rekord skills
```

### Configuration

Recommended:

```text
github.com/spf13/viper
```

Use Viper only if configuration grows beyond a simple YAML loader. For the MVP, a direct YAML parser is acceptable.

YAML parser:

```text
gopkg.in/yaml.v3
```

### PTY support

Recommended:

```text
github.com/creack/pty
```

Used for spawning and interacting with the user shell inside a pseudo-terminal.

### Terminal handling

Recommended:

```text
golang.org/x/term
```

Used for raw terminal mode, terminal size detection, and terminal state restoration.

### SQLite or file storage

For MVP, use plain files.

Later, SQLite can be introduced if session indexing becomes more advanced.

Recommended initial storage:

```text
.rekord/
  sessions/
    <session-id>/
      metadata.json
      events.jsonl
      commands.json
      stdout.log
      stderr.log
      session.cast
```

### Video and GIF export

Use external tools first:

```text
ffmpeg
agg
asciinema
```

Initial MVP should not require video export. MP4/GIF can be a later phase.

### Testing

Use Go standard testing:

```text
testing
```

Optional:

```text
github.com/stretchr/testify
```

### Linting and quality

Recommended:

```text
gofmt
go vet
golangci-lint
```

### Release tooling

Recommended later:

```text
goreleaser
```

---

## 5. High-Level Architecture

```text
             ┌──────────────┐
             │  rekord CLI   │
             └──────┬───────┘
                    │
                    v
             ┌──────────────┐
             │ Command Layer │
             └──────┬───────┘
                    │
       ┌────────────┼────────────┐
       v            v            v
┌────────────┐ ┌───────────┐ ┌────────────┐
│  Recorder  │ │  Exporter │ │   Handoff  │
└─────┬──────┘ └─────┬─────┘ └─────┬──────┘
      │              │             │
      v              v             v
┌────────────┐ ┌───────────┐ ┌────────────┐
│ PTY/Tmux   │ │ Formats   │ │ AI Context │
└─────┬──────┘ └─────┬─────┘ └─────┬──────┘
      │              │             │
      └──────────────┼─────────────┘
                     v
              ┌─────────────┐
              │ SessionStore │
              └─────────────┘
```

---

## 6. Module Layout

Recommended repository structure:

```text
rekord/
  go.mod
  README.md
  ARCHITECTURE.md
  rekord.yaml.example

  cmd/
    rekord/
      main.go

  internal/
    cli/
      root.go
      start.go
      run.go
      list.go
      replay.go
      export.go
      scan.go
      handoff.go
      tmux.go
      skills.go

    recorder/
      recorder.go
      pty_recorder.go
      command_recorder.go
      timer.go

    session/
      model.go
      store.go
      file_store.go
      ids.go

    events/
      event.go
      writer.go
      reader.go

    export/
      exporter.go
      cast.go
      markdown.go
      json.go
      script.go
      mp4.go
      gif.go

    commands/
      extractor.go
      shell.go

    redact/
      redactor.go
      patterns.go
      scanner.go

    handoff/
      generator.go
      git.go
      tree.go
      logs.go

    tmux/
      tmux.go
      capture.go
      session.go

    skills/
      skill.go
      loader.go
      runner.go

    config/
      config.go
      loader.go

    platform/
      paths.go
      terminal.go
      exec.go

  testdata/
    sessions/
    skills/
```

---

## 7. Core Data Model

### Session metadata

`metadata.json`

```json
{
  "id": "20260530-080000-monocron-demo",
  "name": "monocron-demo",
  "createdAt": "2026-05-30T08:00:00Z",
  "endedAt": "2026-05-30T08:01:14Z",
  "durationMs": 74000,
  "shell": "/bin/zsh",
  "cwd": "/Users/tolani/projects/monocron",
  "cols": 120,
  "rows": 40,
  "status": "completed",
  "rekordVersion": "0.1.0"
}
```

### Terminal event

`events.jsonl`

Each line is a JSON object.

```json
{"timeMs":0,"type":"output","data":"$ go test ./...\r\n"}
{"timeMs":132,"type":"output","data":"ok github.com/example/app 0.231s\r\n"}
{"timeMs":700,"type":"resize","cols":120,"rows":40}
```

Event types:

```text
output
input
resize
marker
```

For MVP, only `output` and `resize` are required.

### Command model

`commands.json`

```json
[
  {
    "index": 1,
    "command": "go test ./...",
    "startedAtMs": 0,
    "endedAtMs": 900,
    "exitCode": null,
    "outputPreview": "ok github.com/example/app 0.231s"
  }
]
```

Command extraction can be basic in the MVP. Perfect shell parsing is not required.

### Asciinema-compatible cast

`session.cast`

```json
{"version":2,"width":120,"height":40,"timestamp":1780128000,"env":{"SHELL":"/bin/zsh","TERM":"xterm-256color"}}
[0.000,"o","$ go test ./...\r\n"]
[0.132,"o","ok github.com/example/app 0.231s\r\n"]
```

---

## 8. Storage Design

Default local project storage:

```text
.rekord/
  sessions/
    <session-id>/
      metadata.json
      events.jsonl
      commands.json
      session.cast
      exports/
        demo.md
        demo.json
        demo.sh
        demo.mp4
        demo.gif
      handoff/
        context.md
        commands.json
        tree.txt
        git.diff
        logs.txt
```

Optional global storage later:

```text
~/.rekord/sessions/
```

MVP should use project-local `.rekord/` by default.

---

## 9. Configuration

Example `rekord.yaml`:

```yaml
name: monocron-demo

recording:
  shell: zsh
  timer: 40s
  cols: 120
  rows: 40

exports:
  default:
    - cast
    - markdown
    - json

privacy:
  redact: true
  redactPatterns:
    - "sk-[a-zA-Z0-9]+"
    - "postgres://.*"
    - "ghp_[a-zA-Z0-9]+"

handoff:
  includeGit: true
  includeTree: true
  includeLogs: true
  maxOutputBytes: 200000

ui:
  theme: catppuccin
  fontSize: 16
```

MVP config fields:

```text
recording.shell
recording.timer
privacy.redact
privacy.redactPatterns
exports.default
```

---

## 10. Feature Roadmap

The roadmap is intentionally split into small features that Codex can implement one by one.

Each feature should have:

1. A clear scope.
2. Files to touch.
3. Acceptance criteria.
4. Tests where reasonable.
5. No dependency on future features unless explicitly stated.

---

# Phase 0: Repository Foundation

## Feature 0.1: Initialize Go module

### Goal

Create the initial Go project structure.

### Scope

- Create `go.mod`.
- Create `cmd/rekord/main.go`.
- Create basic `internal/cli/root.go`.
- Add `rekord --help`.

### Suggested packages

- `github.com/spf13/cobra`

### Acceptance criteria

- `go run ./cmd/rekord --help` works.
- `go test ./...` passes.
- Project has a basic README.

### Codex instruction

Implement only the project skeleton and root command. Do not implement recording yet.

---

## Feature 0.2: Add version command

### Goal

Add a version command.

### CLI

```bash
rekord version
```

### Acceptance criteria

- Prints version, commit, and build date.
- Defaults to `dev` values when not set by build flags.

### Codex instruction

Implement version plumbing only. Do not add release automation yet.

---

## Feature 0.3: Add project-local paths

### Goal

Create a platform path helper for `.rekord/` storage.

### Scope

- Add `internal/platform/paths.go`.
- Add helper for session root path.
- Ensure `.rekord/sessions` is created when needed.

### Acceptance criteria

- Unit tests validate generated paths.
- Paths are relative to current working directory by default.

---

# Phase 1: Session Storage

## Feature 1.1: Define session models

### Goal

Create core session and event models.

### Files

```text
internal/session/model.go
internal/events/event.go
```

### Acceptance criteria

- Session metadata model exists.
- Terminal event model exists.
- JSON marshaling works in tests.

### Codex instruction

Only define data models and tests. Do not implement file writing yet.

---

## Feature 1.2: Implement file-based session store

### Goal

Create sessions on disk.

### Files

```text
internal/session/store.go
internal/session/file_store.go
```

### Acceptance criteria

- Can create a session directory.
- Can write `metadata.json`.
- Can read `metadata.json` back.
- Tests use temporary directories.

---

## Feature 1.3: Implement event writer

### Goal

Write terminal events to `events.jsonl`.

### Files

```text
internal/events/writer.go
```

### Acceptance criteria

- Appends events as JSON lines.
- Flushes and closes cleanly.
- Unit test verifies JSONL output.

---

## Feature 1.4: Implement event reader

### Goal

Read terminal events from `events.jsonl`.

### Files

```text
internal/events/reader.go
```

### Acceptance criteria

- Reads events in order.
- Handles empty files.
- Returns useful errors for malformed JSON.

---

## Feature 1.5: Add `rekord list`

### Goal

List recorded sessions.

### CLI

```bash
rekord list
```

### Output

```text
NAME              DURATION   STATUS      CREATED
monocron-demo     42s        completed   2026-05-30 08:00
```

### Acceptance criteria

- Lists sessions from `.rekord/sessions`.
- Handles no sessions gracefully.
- Does not require recording to be implemented.

---

# Phase 2: Basic Terminal Recording

## Feature 2.1: PTY shell recorder

### Goal

Record an interactive shell session using a pseudo-terminal.

### Files

```text
internal/recorder/recorder.go
internal/recorder/pty_recorder.go
```

### Suggested packages

```text
github.com/creack/pty
golang.org/x/term
```

### Behavior

- Start user's shell inside a PTY.
- Pipe user stdin to PTY.
- Pipe PTY output to user stdout.
- Write PTY output to `events.jsonl`.
- Restore terminal state on exit.

### Acceptance criteria

- `rekord start --name demo` opens an interactive shell.
- Commands typed by user appear normally.
- Exiting the shell ends recording.
- `events.jsonl` contains output events.
- Terminal state is restored after exit.

### Codex instruction

Implement minimal PTY recording only. Do not implement command extraction, tmux, or exports yet.

---

## Feature 2.2: Add `rekord start`

### Goal

Expose PTY recording through CLI.

### CLI

```bash
rekord start --name demo
```

### Flags

```text
--name string
--shell string
--cwd string
```

### Acceptance criteria

- Creates a session directory.
- Writes metadata.
- Writes event stream.
- Updates metadata status to `completed` on normal exit.
- Updates metadata status to `failed` on recorder error.

---

## Feature 2.3: Terminal resize events

### Goal

Capture terminal resize events.

### Behavior

- Detect window size at start.
- Listen for `SIGWINCH`.
- Resize PTY.
- Write resize event to `events.jsonl`.

### Acceptance criteria

- Initial terminal size is stored in metadata.
- Resize events are recorded.
- PTY size updates when terminal is resized.

---

## Feature 2.4: Timer mode

### Goal

Stop recording automatically after a duration.

### CLI

```bash
rekord start --name demo --timer 40s
rekord start --name demo --timer 5m
```

### Acceptance criteria

- Parses Go duration strings.
- Stops session after timer expires.
- Metadata shows session as completed.

---

## Feature 2.5: Record single command

### Goal

Record a single non-interactive command.

### CLI

```bash
rekord run --name tests -- go test ./...
```

### Acceptance criteria

- Runs command inside a PTY or command process.
- Records output.
- Exits with same exit code as command.
- Stores command metadata.

---

# Phase 3: Replay and Cast Export

## Feature 3.1: Replay events locally

### Goal

Replay a recorded session in the terminal.

### CLI

```bash
rekord replay demo
```

### Behavior

- Reads `events.jsonl`.
- Replays output with original timing.
- Add `--speed` flag later.

### Acceptance criteria

- Replays output events in order.
- Respects event timing approximately.
- Handles missing session gracefully.

---

## Feature 3.2: Export to Asciinema cast

### Goal

Generate an asciinema-compatible `.cast` file.

### CLI

```bash
rekord export demo --to cast
```

### Output

```text
.rekord/sessions/demo/exports/demo.cast
```

### Acceptance criteria

- Writes cast v2 header.
- Converts output events to cast event rows.
- Uses metadata width, height, timestamp, and shell.

---

## Feature 3.3: Add generic export interface

### Goal

Create reusable export abstraction.

### Files

```text
internal/export/exporter.go
```

### Interface

```go
type Exporter interface {
    Format() string
    Export(ctx context.Context, session Session, events EventReader, outputPath string) error
}
```

### Acceptance criteria

- Cast exporter implements interface.
- CLI chooses exporter by `--to` value.
- Unknown format returns clear error.

---

# Phase 4: Command Extraction

## Feature 4.1: Basic command extraction

### Goal

Extract commands from terminal output using simple prompt heuristics.

### Files

```text
internal/commands/extractor.go
```

### MVP heuristic

Detect lines that look like:

```text
$ command
> command
❯ command
➜ command
```

### Acceptance criteria

- Extracts basic commands from event output.
- Ignores empty prompt lines.
- Writes `commands.json`.
- Has tests using fixture outputs.

### Codex instruction

Do not attempt perfect shell parsing. Implement simple heuristics only.

---

## Feature 4.2: Configurable prompt patterns

### Goal

Allow users to configure prompt patterns.

### Config

```yaml
commands:
  promptPatterns:
    - "^\\$\\s+(.+)$"
    - "^❯\\s+(.+)$"
```

### Acceptance criteria

- Loads prompt patterns from config.
- Falls back to defaults.
- Invalid regex returns a clear error.

---

## Feature 4.3: `rekord commands`

### Goal

Show extracted commands.

### CLI

```bash
rekord commands demo
```

### Acceptance criteria

- Prints command list in order.
- Handles sessions with no extracted commands.
- Supports `--json` output.

---

# Phase 5: Documentation Exports

## Feature 5.1: Export to JSON

### Goal

Export full session summary as JSON.

### CLI

```bash
rekord export demo --to json
```

### Acceptance criteria

- Includes metadata.
- Includes commands.
- Includes optional output summary.
- Writes to exports directory.

---

## Feature 5.2: Export to Markdown

### Goal

Generate documentation from a session.

### CLI

```bash
rekord export demo --to markdown
```

### Output example

```markdown
# Rekord Session: demo

## Summary

- Duration: 42s
- Shell: zsh
- Working directory: /project

## Commands

### 1. go test ./...

```text
ok github.com/example/app 0.231s
```
```

### Acceptance criteria

- Generates readable Markdown.
- Includes metadata summary.
- Includes extracted commands.
- Includes output previews where available.
- Redaction is applied when enabled.

---

## Feature 5.3: Export to shell script

### Goal

Generate a replayable shell script from extracted commands.

### CLI

```bash
rekord export demo --to script
```

### Acceptance criteria

- Writes `demo.sh`.
- Includes shebang and `set -e`.
- Includes extracted commands in order.
- Does not include commands marked unsafe later.

---

# Phase 6: Redaction and Safety

## Feature 6.1: Built-in redaction patterns

### Goal

Add default secret redaction.

### Files

```text
internal/redact/patterns.go
internal/redact/redactor.go
```

### Default patterns

Detect common secrets:

```text
OPENAI_API_KEY
GITHUB_TOKEN
ghp_...
sk-...
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
DATABASE_URL
postgres://...
mysql://...
password=...
token=...
secret=...
```

### Acceptance criteria

- Redactor replaces secrets with `[REDACTED]`.
- Tests cover common secret patterns.
- Redactor can be used by exporters.

---

## Feature 6.2: `rekord scan`

### Goal

Scan a session for possible secrets.

### CLI

```bash
rekord scan demo
```

### Acceptance criteria

- Scans events and command files.
- Reports categories, not raw secret values.
- Returns non-zero exit code if secrets are found when `--strict` is used.

---

## Feature 6.3: Redacted export mode

### Goal

Apply redaction during export.

### CLI

```bash
rekord export demo --to markdown --redact
rekord export demo --to json --redact
```

### Acceptance criteria

- Markdown and JSON exports apply redaction.
- Raw event files are not modified.
- User can disable redaction explicitly with `--no-redact`.

---

# Phase 7: AI Handoff

## Feature 7.1: Generate basic AI context

### Goal

Create a Markdown context file for AI agents.

### CLI

```bash
rekord handoff demo
```

### Output

```text
.rekord/sessions/demo/handoff/context.md
```

### Contents

```markdown
# Rekord AI Context

## Session

## Commands Run

## Observed Output

## Possible Errors

## Suggested Summary
```

### Acceptance criteria

- Generates context.md.
- Includes metadata.
- Includes commands.
- Includes redacted output excerpts.

---

## Feature 7.2: Include git context

### Goal

Add git branch, status, and diff to handoff.

### CLI

```bash
rekord handoff demo --include-git
```

### Acceptance criteria

- Captures current branch.
- Captures `git status --short`.
- Captures `git diff` with size limit.
- Handles non-git directories gracefully.

---

## Feature 7.3: Include project tree

### Goal

Add a project tree snapshot to handoff.

### CLI

```bash
rekord handoff demo --include-tree
```

### Acceptance criteria

- Generates `tree.txt`.
- Excludes `.git`, `node_modules`, `vendor`, `.rekord`, build directories.
- Applies max depth and max file count.

---

## Feature 7.4: Clipboard support

### Goal

Copy handoff context to clipboard.

### CLI

```bash
rekord handoff demo --copy
```

### Acceptance criteria

- Works on macOS using `pbcopy`.
- Works on Linux when `xclip` or `wl-copy` is available.
- Fails gracefully when clipboard tool is missing.

---

# Phase 8: Tmux Support

## Feature 8.1: Detect tmux

### Goal

Detect if the user is inside tmux.

### CLI

```bash
rekord tmux status
```

### Acceptance criteria

- Detects `$TMUX`.
- Prints current tmux session name if available.
- Handles missing tmux binary gracefully.

---

## Feature 8.2: Capture tmux pane output

### Goal

Capture output from a tmux pane.

### CLI

```bash
rekord tmux capture --pane %1 --name pane-demo
```

### Approach

Use:

```bash
tmux capture-pane -p -t <pane>
```

### Acceptance criteria

- Captures pane text to session output.
- Stores metadata identifying tmux pane.
- Does not require interactive PTY recording.

---

## Feature 8.3: Record tmux pipe-pane

### Goal

Use tmux `pipe-pane` to stream pane output into Rekord.

### CLI

```bash
rekord tmux record --pane %1 --name demo
```

### Acceptance criteria

- Starts pipe-pane recording.
- Stops cleanly.
- Writes events to `events.jsonl`.
- Documents limitations.

---

## Feature 8.4: Create managed tmux session

### Goal

Let Rekord create a tmux session for recording.

### CLI

```bash
rekord tmux start --session monocron-demo
```

### Acceptance criteria

- Creates tmux session.
- Starts recording.
- Attaches user to session.
- Stops recording when session ends or command is issued.

---

# Phase 9: Skills System

## Feature 9.1: Define skill schema

### Goal

Create YAML schema for reusable recording recipes.

### Example

```yaml
name: go-project-demo
description: Record a basic Go project demo
steps:
  - run: go version
  - run: go test ./...
  - run: go run ./cmd/app
```

### Acceptance criteria

- Skill model exists.
- YAML loader exists.
- Tests load valid and invalid skills.

---

## Feature 9.2: `rekord skills list`

### Goal

List available skills.

### CLI

```bash
rekord skills list
```

### Skill locations

```text
.rekord/skills/
~/.rekord/skills/
```

### Acceptance criteria

- Lists local skills.
- Lists global skills later.
- Handles no skills gracefully.

---

## Feature 9.3: `rekord skills run`

### Goal

Run a skill and record its output.

### CLI

```bash
rekord skills run go-project-demo --name go-demo
```

### Acceptance criteria

- Executes skill steps in order.
- Records output as a session.
- Stops on failure by default.
- Supports `continueOnError` later.

---

## Feature 9.4: Built-in starter skills

### Goal

Ship useful example skills.

### Skills

```text
go-project-demo
docker-demo
kubernetes-demo
terraform-demo
github-actions-demo
```

### Acceptance criteria

- Built-in skills can be listed.
- Built-in skills are documented.
- Users can copy built-ins into `.rekord/skills`.

---

# Phase 10: Video and GIF Export

## Feature 10.1: Detect external render tools

### Goal

Detect whether required tools are installed.

### CLI

```bash
rekord doctor
```

### Checks

```text
ffmpeg
agg
asciinema
tmux
git
```

### Acceptance criteria

- Prints available and missing tools.
- Does not fail if optional tools are missing.

---

## Feature 10.2: GIF export

### Goal

Export session to GIF using external tools.

### CLI

```bash
rekord export demo --to gif
```

### Acceptance criteria

- Uses cast export as intermediate.
- Uses supported external tool if available.
- Clear error if dependency is missing.

---

## Feature 10.3: MP4 export

### Goal

Export session to MP4.

### CLI

```bash
rekord export demo --to mp4 --size 1080p
```

### Acceptance criteria

- Uses intermediate render path.
- Supports 720p and 1080p presets.
- Clear dependency error if missing.

---

# Phase 11: Developer Experience and Distribution

## Feature 11.1: Add Makefile

### Goal

Provide common development commands.

### Targets

```makefile
build
test
lint
fmt
run
clean
```

### Acceptance criteria

- `make build` creates a binary.
- `make test` runs all tests.
- `make fmt` formats code.

---

## Feature 11.2: Add GitHub Actions CI

### Goal

Run tests and linting on pull requests.

### Acceptance criteria

- CI runs on push and pull request.
- CI runs `go test ./...`.
- CI runs formatting check.

---

## Feature 11.3: Add GoReleaser

### Goal

Build cross-platform binaries.

### Acceptance criteria

- `.goreleaser.yaml` exists.
- Supports macOS, Linux, and Windows where possible.
- Produces archives with checksums.

---

## Feature 11.4: Homebrew formula support

### Goal

Prepare for Homebrew installation.

### Acceptance criteria

- Release artifacts are compatible with Homebrew.
- README documents future install flow.

---

# 12. CLI Command Reference

Target command tree:

```text
rekord
  start
  run
  list
  replay
  commands
  export
  scan
  handoff
  doctor
  version
  tmux
    status
    capture
    record
    start
  skills
    list
    run
    validate
```

---

# 13. Error Handling Rules

All commands should return clear human-readable errors.

Examples:

```text
session not found: monocron-demo
unknown export format: pdf
ffmpeg is required for mp4 export but was not found in PATH
could not open PTY: <reason>
invalid timer value: use values like 40s, 5m, or 1h
```

Rules:

1. Do not panic for user-facing errors.
2. Wrap internal errors with context.
3. Use non-zero exit codes for failed commands.
4. Restore terminal state even when recording fails.
5. Never print raw secret values in scan output.

---

# 14. Testing Strategy

## Unit tests

Use unit tests for:

1. Session models.
2. File store.
3. Event writer and reader.
4. Exporters.
5. Redactor.
6. Command extractor.
7. Skill loader.
8. Config loader.

## Integration tests

Use integration tests for:

1. CLI command execution.
2. Recording a simple command.
3. Exporting a small fixture session.

## Fixture sessions

Store small fixture sessions in:

```text
testdata/sessions/
```

Example:

```text
testdata/sessions/basic-go-test/
  metadata.json
  events.jsonl
  commands.json
```

---

# 15. Privacy and Security

Rekord records terminal sessions. Terminal sessions may contain secrets.

Security rules:

1. Redaction should be supported before public export formats become advanced.
2. Raw event files should not be mutated by redaction.
3. Exports should support redaction.
4. `rekord scan` should report categories and locations, not secret values.
5. Handoff context should be redacted by default.
6. `.rekord/` should be added to `.gitignore` by documentation recommendation.

Recommended `.gitignore`:

```gitignore
.rekord/sessions/
.rekord/handoff/
```

Skills may be committed if desired:

```gitignore
!.rekord/skills/
```

---

# 16. Codex Working Rules

This project should be built feature-by-feature.

When asking Codex to work on the codebase, use this format:

```markdown
Implement Feature X.Y: <feature name>

Rules:
- Only implement this feature.
- Do not start future phases.
- Add or update tests.
- Keep public interfaces small.
- Run `go test ./...`.
- Update README only if needed for this feature.
```

Codex should not implement large batches unless explicitly requested.

Good Codex task example:

```markdown
Implement Feature 1.3: Event Writer

Add `internal/events/writer.go`.
The writer should append JSON lines to an `events.jsonl` file.
Add unit tests using a temporary directory.
Do not implement the event reader yet.
Run `go test ./...`.
```

Bad Codex task example:

```markdown
Build the whole Rekord app.
```

---

# 17. Suggested First 10 Codex Tasks

Use this exact order for manageable tracking.

1. Feature 0.1: Initialize Go module.
2. Feature 0.2: Add version command.
3. Feature 0.3: Add project-local paths.
4. Feature 1.1: Define session models.
5. Feature 1.2: Implement file-based session store.
6. Feature 1.3: Implement event writer.
7. Feature 1.4: Implement event reader.
8. Feature 1.5: Add `rekord list`.
9. Feature 2.1: PTY shell recorder.
10. Feature 2.2: Add `rekord start`.

After these tasks, Rekord should have a working foundation and basic recording.

---

# 18. MVP Definition

The first MVP is complete when the following commands work:

```bash
rekord start --name demo
rekord list
rekord replay demo
rekord export demo --to cast
rekord export demo --to markdown
rekord scan demo
rekord handoff demo
```

MVP includes:

1. PTY recording.
2. Session storage.
3. Event replay.
4. Cast export.
5. Markdown export.
6. Basic command extraction.
7. Basic redaction.
8. Basic AI handoff.

MVP excludes:

1. MP4 export.
2. GIF export.
3. Full tmux recording.
4. Skills execution.
5. Cloud sync.
6. GUI editor.

---

# 19. Future Ideas

Potential future features:

1. Web player for Rekord sessions.
2. Shareable HTML export.
3. Themeable terminal rendering.
4. Voiceover script generation.
5. Auto-generated blog post export.
6. AI-generated tutorial from a session.
7. GitHub Action to record CI demos.
8. VS Code/Cursor extension.
9. Rekord Cloud for optional hosted playback.
10. Team demo library.

---

# 20. Project Tagline

Possible taglines:

```text
Rekord — terminal recordings for builders.
```

```text
Record what you build. Export what you learned. Handoff context to AI.
```

```text
Turn terminal workflows into demos, docs, and AI-ready context.
```
