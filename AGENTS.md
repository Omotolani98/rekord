# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go CLI project for `rekord`, a terminal workflow recorder. The current checkout contains `go.mod` and `ARCHITECTURE.md`; use the architecture document as the implementation guide.

Expected layout:

- `cmd/rekord/`: CLI entry point (`main.go`).
- `internal/cli/`: Cobra command wiring for `start`, `run`, `export`, `handoff`, `tmux`, and related commands.
- `internal/recorder/`, `internal/session/`, `internal/events/`: recording, session storage, and event JSONL handling.
- `internal/export/`, `internal/redact/`, `internal/handoff/`: exports, secret filtering, and AI handoff generation.
- `testdata/`: sample sessions, skills, and fixtures for tests.
- `.rekord/`: local runtime output; do not commit generated sessions or exports.

## Build, Test, and Development Commands

- `go test ./...`: run all Go tests.
- `go test -race ./...`: run tests with race detection for recorder and PTY code.
- `go vet ./...`: catch common correctness issues.
- `gofmt -w <files>`: format edited Go files before committing.
- `go run ./cmd/rekord --help`: run the CLI locally once the entry point exists.

Add dependencies with `go get <module>` and keep `go.mod`/`go.sum` changes focused.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`) and idiomatic package names: short, lowercase, and singular where practical (`session`, `recorder`, `redact`). Keep implementation packages under `internal/` to avoid accidental public APIs.

Prefer explicit data models for session files (`metadata.json`, `events.jsonl`, `commands.json`). Name CLI command files after commands, for example `start.go`, `export.go`, and `handoff.go`.

## Testing Guidelines

Use Go’s standard `testing` package by default. Name test files `*_test.go` and test functions `TestXxx`. Place fixtures in `testdata/` so Go tooling ignores them during builds.

Focus tests on session serialization, redaction, command extraction, path handling, and exporter output. For PTY/tmux behavior, skip with a useful message when required tools are unavailable.

## Commit & Pull Request Guidelines

This checkout does not include Git history, so use clear conventional-style commits such as `feat: add session store`, `fix: redact token URLs`, or `docs: update architecture`.

Pull requests should include a short summary, test results (`go test ./...`), and any relevant CLI examples. Link issues when available. Include screenshots or terminal output only for user-visible CLI behavior.

## Security & Configuration Tips

Treat recorded terminal output as sensitive. Do not commit `.rekord/`, real credentials, generated handoff bundles, or local `rekord.yaml` files containing private paths or tokens. Keep example configuration in files such as `rekord.yaml.example`.
