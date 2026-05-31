package handoff

import (
	"context"
	"os/exec"
	"strings"
)

type GitContext struct {
	Branch string
	Status string
	Diff   string
}

func GatherGit(ctx context.Context, dir string, maxDiffBytes int) (GitContext, bool) {
	branch, err := gitOutput(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return GitContext{}, false
	}

	status, _ := gitOutput(ctx, dir, "status", "--short")
	diff, _ := gitOutput(ctx, dir, "diff")
	if maxDiffBytes > 0 && len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n… (truncated)"
	}

	return GitContext{
		Branch: strings.TrimSpace(branch),
		Status: status,
		Diff:   diff,
	}, true
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
