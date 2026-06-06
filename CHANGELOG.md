# Changelog

All notable changes to Rekord are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## v0.3.1 — 2026-06-06

### Fixed
- **Cross-tool memory now resolves to the same project.** Memory was keyed by the
  hash of whatever directory launched `rekord mcp`, so two tools (e.g. Claude Code
  and opencode) launched from different working directories wrote to different
  `~/.rekord/projects/<hash>/` folders and never saw each other's memory. Project
  identity now canonicalizes to the enclosing git repository root, so any working
  directory or subdirectory inside a repo maps to one stable project key. Paths
  outside a git repo are unchanged.
- **`--from-agent` / `--to-agent` no longer hide memories.** `--from-agent` was a
  hard filter, so `rekord resume --from-agent claude` dropped every memory not
  written by `claude` and returned "No Rekord memory found". Handoff fields are now
  labels only: `resume`, `recall`, and `memory list/search` return all of a
  project's memory regardless of which agent wrote it. Use `--agent <name>` to
  filter by writer explicitly.

### Added
- `rekord memory projects` lists every project with stored memory as
  `storage-key → project path`, so scattered or legacy memory folders are
  discoverable. New `memory_projects` MCP tool exposes the same listing to agents.
- Each project folder now records a `project.json` (`{path, key}`), and
  `rekord resume` prints the resolved `Storage key` so you can see exactly which
  folder a session reads and writes.

### Notes
- Memory folders written under the old (directory-based) keys are not migrated
  automatically. Run `rekord memory projects` to find them and move them to the
  git-root key if needed.

## v0.3.0 — 2026-06-06

### Added
- **Rekord Memory**, a user-local shared memory layer for humans and coding agents.
  Memories are stored per project under `~/.rekord/projects/<project-hash>/memories.jsonl`
  and can be created, searched, resolved, and used to resume work across terminal sessions.
- New memory commands:
  - `rekord remember <text>` stores a durable project memory.
  - `rekord recall [query]` searches project memory.
  - `rekord resume` generates agent-ready continuation context.
  - `rekord snapshot [note]` captures a resumable project checkpoint.
  - `rekord memory add/list/search/show/resolve` provides full memory management.
- **Agent-to-agent handoff** through memory filters. `--agent`, `--from-agent`,
  `--to-agent`, and `--session` let users resume work from a specific agent or named
  session, for example `rekord resume --from-agent claude --to-agent codex`.
- **Named session linkage** for memory and snapshots. Agents can attach memories and
  snapshots to a human-readable Rekord session name so users can resume later with
  `rekord resume --session <name>`.
- **Full git patch snapshots**. `rekord snapshot` captures git branch, HEAD, dirty state,
  changed files, and full binary-safe unstaged/staged patches under
  `~/.rekord/projects/<project-hash>/patches/`.
- **MCP memory tools** for coding agents:
  `memory_write`, `memory_search`, `memory_list`, `memory_get`, `memory_resolve`,
  `snapshot_create`, and `resume_context`.

### Notes
- Rekord Memory is user-local by default. It does not write `.rekord/` files into the
  current repository unless users choose to export or copy memory artifacts themselves.
- `resume_context` is the primary MCP handoff primitive: agents can ask Rekord what
  happened, what changed, what failed, and where to continue.

## v0.2.0 — 2026-06-04

### Added
- **Live agent-driven terminal control via MCP** (`rekord mcp`). Runs a Model Context
  Protocol server over stdio so an AI agent (Claude Code, Cursor, …) can drive real terminal
  programs instead of guessing at output. Tools: `launch`, `send`, `capture`, `wait_text`,
  `wait_idle`, `wait_exit`, `logs`, `resize`, `stop`, `list`, `status`. `capture` returns a
  deterministic screen frame (character grid + cursor), parsed with a built-in VT emulator.
  Captures and logs are redacted by default; pass `raw: true` to opt out.
- **Persistent named sessions over a local socket** (`rekord session`). `session start
  --name <name> -- <command>` launches a detached background session reachable by other
  processes through an owner-only unix socket (`<root>/<name>.sock`, mode 0600).
  `session send`, `show`, `wait`, `status`, `list`, and `stop` drive it; the session retains
  its final screen after the program exits until you `stop` it.
- Agent- and socket-driven sessions are ordinary recordings: they write `metadata.json` and
  `events.jsonl` (now including `input` events) under the sessions root, so `rekord export`,
  `replay`, `commands`, and `handoff` work on them unchanged.

### Notes
- The MCP server holds sessions in-process for its lifetime; the `rekord session` socket
  layer is a separate surface. Bridging `rekord mcp` onto persistent sockets is planned.

## v0.1.7 — 2026-06-02

### Added
- Interactive recording (`start`, `run`, `skills`) now works on Windows via ConPTY
  (`github.com/aymanbagabas/go-pty`), removing the v0.1.6 limitation. Default shell is
  `powershell.exe`; window resizes are tracked by polling the console size.

## v0.1.6 — 2026-06-02

### Added
- Windows builds (`rekord.exe` + `rk.exe`, amd64 and arm64), shipped as zip archives.
- Chocolatey distribution: `choco install rekord` on Windows once the package clears
  community-feed moderation.

### Notes
- Interactive recording (`start`, `run`, `skills`) is unix-only; on Windows it exits with
  `interactive recording is not supported on windows`. Export, replay, list, commands, and
  handoff work cross-platform on existing sessions.

## v0.1.5 — 2026-06-02

### Fixed
- tmux recordings now capture the pane size, so exports get valid terminal dimensions
  instead of `0x0`. Previously `agg` rejected gif/mp4 exports with
  `invalid terminal size: 0x0`. The cast exporter also falls back to 80x24 for older
  sessions recorded with no size, so they export without re-recording.

## v0.1.4 — 2026-05-31

### Added
- `rk` short alias for the `rekord` command, shipped across all install channels
  (`go install …/cmd/rk@latest`, release archives, and Homebrew). Help/usage reflects
  whichever name is invoked.

## v0.1.3 — 2026-05-31

### Added
- `rk` short alias for the `rekord` command, shipped across all install channels
  (`go install …/cmd/rk@latest`, release archives, and Homebrew). Help/usage reflects
  whichever name is invoked.

### Changed
- The config file now defaults to `~/.rekord/rekord.yaml` (alongside the sessions store),
  falling back to a `./rekord.yaml` in the current directory when present. `--config <path>`
  still overrides, and `rekord config set` creates `~/.rekord` if needed.

## v0.1.2 — 2026-05-31

### Added
- Colorized command output on a terminal: green `✓` on success, red `●`/`✗` for active
  recording and detected secrets, dim `·` secondary notes, and a colored session `STATUS`
  column in `list`. Color is disabled automatically for pipes/non-terminals and when
  `NO_COLOR` is set, so scripted output stays plain.

## v0.1.1 — 2026-05-31

### Added
- Stop a `rekord start` recording with a hotkey (default `Ctrl-]`), configurable per run
  with `--stop-key ctrl-x` or persistently via the `recording.stopKey` config field.
- `rekord config` command (`get`, `set`, `view`, `path`) to read and edit `rekord.yaml`.

### Fixed
- `export -o <path>` now appends the format extension when the path has none
  (e.g. `-o ~/Downloads/demo --to gif` → `~/Downloads/demo.gif`).
- Recordings default to a global `~/.rekord/sessions` store instead of creating a `.rekord/`
  directory in the current working directory (`--root` still overrides).

## v0.1.0 — 2026-05-30

- Initial release: PTY/command recording, tmux capture, replay, exports
  (cast/json/markdown/script/gif/mp4), command extraction, secret scan + redaction,
  AI handoff bundles, skills, and `doctor`.
