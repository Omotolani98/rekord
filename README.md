# Rekord

Rekord is a Go CLI for recording terminal workflows as structured session data, then exporting those sessions to formats such as Markdown, JSON, asciinema casts, and AI-ready handoff bundles.

The project is currently bootstrapped with a minimal CLI shell. See [ARCHITECTURE.md](ARCHITECTURE.md) for the planned module layout and product direction.

## Development

```bash
go test ./...
go run ./cmd/rekord --help
go run ./cmd/rekord version
```

Generated recordings should stay local under `.rekord/` and must not be committed.

## Releases

Releases are built with GoReleaser:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Tag releases with semantic versions such as `v0.1.0`, then run GoReleaser from `main`.
