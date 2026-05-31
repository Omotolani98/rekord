package recorder

import (
	"context"
	"io"
	"os"
	"time"
)

const defaultShell = "/bin/sh"

type Options struct {
	Shell      string
	Command    []string
	CWD        string
	Env        []string
	EventsPath string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	KillGrace  time.Duration
	StopKey    byte
}

type Result struct {
	Shell      string
	StartedAt  time.Time
	EndedAt    time.Time
	DurationMS int64
	ExitCode   int
}

type Recorder interface {
	Record(ctx context.Context, opts Options) (Result, error)
}

func resolveShell(override string) string {
	if override != "" {
		return override
	}
	if env := os.Getenv("SHELL"); env != "" {
		return env
	}
	return defaultShell
}
