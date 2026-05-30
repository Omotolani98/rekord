# Rekord

[![CI](https://github.com/Omotolani98/rekord/actions/workflows/ci.yml/badge.svg)](https://github.com/Omotolani98/rekord/actions/workflows/ci.yml)

Rekord is a Go CLI that records terminal workflows as structured session data, then
exports them to Markdown, JSON, asciinema casts, GIF/MP4, and AI-ready handoff bundles.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design.

## Features

- **Record** interactive shells in a PTY (raw mode, terminal resize, `--timer`
  auto-stop) or a single command (`run`).
- **tmux** support: pane capture, `pipe-pane` streaming, and a managed session that
  records while you stay attached.
- **Replay** sessions with original timing (`--speed`).
- **Export** to `cast`, `json`, `markdown`, shell `script`, `gif`, and `mp4`.
- **Command extraction** from recorded output, with configurable prompt patterns.
- **Safety**: scan sessions for secrets and redact them on export — raw files are
  never modified.
- **AI handoff** bundles: session context plus optional git state, project tree, and
  clipboard copy.
- **Skills**: reusable YAML recording recipes, with built-in starters.
- **`doctor`** checks for optional external tools.

## Install

Homebrew (available after the first tagged release):

```bash
brew install Omotolani98/rekord/rekord
```

With Go:

```bash
go install github.com/Omotolani98/rekord/cmd/rekord@latest
```

Or download a prebuilt archive for your platform from the
[Releases](https://github.com/Omotolani98/rekord/releases) page.

## Quickstart

```bash
rekord run --name demo -- go test ./...     # record a command
rekord list                                 # list sessions
rekord replay demo                          # replay with original timing
rekord commands demo --json                 # commands extracted from output
rekord export demo --to markdown            # generate docs
rekord export demo --to cast                # asciinema cast (play with: asciinema play)
rekord handoff demo --include-git --copy    # AI context bundle, copied to clipboard
rekord scan demo --strict                   # fail if secrets are present
```

## Usage

```text
rekord
  start                 # record an interactive shell (--timer to auto-stop)
  run -- <cmd>          # record a single command
  list                  # list recorded sessions
  replay <session>      # replay a session with original timing (--speed)
  commands <session>    # extract commands run (--json)
  export <session>      # export --to cast|json|markdown|script|gif|mp4 (--redact)
  scan <session>        # report possible secrets (--strict)
  handoff <session>     # AI context bundle (--include-git/--include-tree/--copy)
  doctor                # check for optional external tools
  version
  tmux
    status              # is the shell inside tmux?
    capture             # snapshot a pane into a session
    record              # stream a pane via pipe-pane
    start               # create a tmux session, record, and attach
  skills
    list                # list reusable recording recipes
    run <skill>         # run a skill and record it
```

GIF/MP4 export needs `agg` (and `ffmpeg` for MP4); run `rekord doctor` to check what
is installed.

## Configuration

Optional `rekord.yaml` in the working directory tunes command extraction and
redaction. Values are merged with the built-in defaults:

```yaml
commands:
  promptPatterns:
    - "^❯\\s+(.+)$"
privacy:
  redact: true
  redactPatterns:
    - "mytoken-[0-9]+"
```

## Session storage

Each recording is a self-contained directory under `.rekord/`:

```text
.rekord/sessions/<id>/
  metadata.json     # session metadata
  events.jsonl      # append-only event log (output/input/resize)
  exports/          # generated cast/json/markdown/script/gif/mp4
  handoff/          # context.md, git.diff, tree.txt, logs.txt
```

Recordings stay local under `.rekord/` and must not be committed; treat recorded
output as sensitive (it may contain secrets).

## Development

```bash
make build      # build bin/rekord
make test       # go test -race ./...
make lint       # go vet + golangci-lint
make fmt        # gofmt
make run ARGS="--help"
```

## Releases

Releases are built with GoReleaser on a `v*` tag (see `.github/workflows/release.yml`):

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Tag with a semantic version such as `v0.1.0`. Homebrew formulas are published to
`Omotolani98/homebrew-rekord` (requires a `REKORD_TOKEN` secret — a PAT with push
access to the tap repo, since the default `GITHUB_TOKEN` cannot write to other repos).
