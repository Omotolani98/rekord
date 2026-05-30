# Rekord

Rekord is a Go CLI that records terminal workflows as structured session data, then
exports them to Markdown, JSON, asciinema casts, GIF/MP4, and AI-ready handoff bundles.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design.

## Install

Homebrew (available after the first tagged release):

```bash
brew install Omotolani98/tap/rekord
```

With Go:

```bash
go install github.com/Omotolani98/rekord/cmd/rekord@latest
```

Or download a prebuilt archive for your platform from the
[Releases](https://github.com/Omotolani98/rekord/releases) page.

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

Example:

```bash
rekord run --name tests -- go test ./...
rekord export tests --to markdown
```

Configuration lives in `rekord.yaml` (prompt patterns, redaction). GIF/MP4 export
needs `agg` (and `ffmpeg` for MP4); run `rekord doctor` to check.

## Development

```bash
make build      # build bin/rekord
make test       # go test -race ./...
make lint       # go vet + golangci-lint
make fmt        # gofmt
make run ARGS="--help"
```

Generated recordings stay local under `.rekord/` and must not be committed; treat
recorded output as sensitive (it may contain secrets).

## Releases

Releases are built with GoReleaser on a `v*` tag (see `.github/workflows/release.yml`):

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Tag with a semantic version such as `v0.1.0`. Homebrew formulas are published to
`Omotolani98/homebrew-tap`.
