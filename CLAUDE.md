# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Rekord — Go CLI that records terminal workflows as structured session data, then exports to Markdown, JSON, asciinema casts, video, and AI handoff bundles. Module: `github.com/Omotolani98/rekord`. Go 1.26. Cobra-based.

Project is early; many CLI subcommands are scaffolded but return `notImplemented`. See `ARCHITECTURE.md` for the full target design — it is the authoritative spec for unbuilt features.

## Commands

- `go test ./...` — run tests
- `go test -race ./...` — race detection (required for recorder/PTY work once added)
- `go test ./internal/session -run TestFileStore` — run single package / test
- `go vet ./...`
- `gofmt -w <files>` before commit
- `go run ./cmd/rekord --help` / `go run ./cmd/rekord version`
- `goreleaser check` / `goreleaser release --snapshot --clean`

PRs target `dev` branch: `gh pr create --base dev --head <branch>`. Use conventional commits (`feat:`, `fix:`, `docs:`, `chore:`).

## Architecture

Entry: `cmd/rekord/main.go` → `internal/cli.Execute`. All command wiring lives in `internal/cli/` (`root.go` registers; `commands.go` holds stub subcommands; `version.go` real). New commands attach via `NewRootCommand` and follow the `start.go` / `export.go` / `handoff.go` per-command-file convention.

Planned package split (keep under `internal/` — no public API surface):

- `internal/session` — session metadata + `Store` interface. `FileStore` persists to `<root>/<sessionID>/metadata.json`. Layout per session: `metadata.json`, `events.jsonl`, `commands.json`, `stdout.log`, `stderr.log`.
- `internal/events` — `Event` (timeMs, type, data, cols, rows) for JSONL append log. Types: `output`, `input`, `resize`, `marker`.
- `internal/recorder` — PTY spawn (`github.com/creack/pty`), raw mode via `golang.org/x/term`, emits events.
- `internal/export` — Markdown/JSON/cast/video renderers; shell out to `ffmpeg` for video.
- `internal/redact` — secret filtering applied before export, not at record time.
- `internal/handoff` — AI context bundle (git state, tree, logs, transcript).
- `internal/transcript` — read-through bridge over other coding agents' native session logs for cross-agent context transfer. Pluggable `Source` (Claude `~/.claude/projects/**.jsonl`, Codex `~/.codex/sessions/**/rollout-*.jsonl`); each carries a `cwd` matched to the rekord project via `memory.NormalizeProject`/`ProjectKey`. Surfaced as `transcript_{sources,list,read,search}` MCP tools, `rekord transcript` CLI, and an opt-in `include_transcript`/`--include-transcript` on resume. Redacted by default.
- `internal/tmux`, `internal/skills` — tmux capture-pane workflow and reusable recording recipes.

Data-first principle: recorder writes structured events; exporters are pure transforms over session dir. Don't couple exporters to live recording.

Local runtime under `.rekord/` — never commit. Treat recorded output as sensitive (may contain secrets).

## Conventions

- **No code comments.** Don't add `//` explanations. Names + types carry meaning. Exception: short godoc on exported symbols when non-obvious.
- Packages: short, lowercase, singular (`session`, `recorder`, `redact`).
- CLI files named after command (`start.go`, `export.go`).
- Tests use stdlib `testing`. Fixtures under `testdata/`. Skip PTY/tmux tests with message when tools missing.
- Stub commands return `notImplemented("<name>")` from `internal/cli/root.go`.
