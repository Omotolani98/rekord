# Changelog

All notable changes to Rekord are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and the project adheres to
[Semantic Versioning](https://semver.org/).

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
