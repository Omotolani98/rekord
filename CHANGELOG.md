# Changelog

All notable changes to Rekord are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and the project adheres to
[Semantic Versioning](https://semver.org/).

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
