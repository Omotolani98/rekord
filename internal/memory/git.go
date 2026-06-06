package memory

import (
	"context"
	"os/exec"
	"strings"
)

func GatherGit(ctx context.Context, project string) (GitState, []byte, []byte, bool) {
	branch, err := gitOutput(ctx, project, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return GitState{}, nil, nil, false
	}
	head, _ := gitOutput(ctx, project, "rev-parse", "HEAD")
	status, _ := gitOutput(ctx, project, "status", "--short")
	unstaged, _ := gitBytes(ctx, project, "diff", "--binary")
	staged, _ := gitBytes(ctx, project, "diff", "--binary", "--staged")

	state := GitState{
		Branch:       strings.TrimSpace(branch),
		Head:         strings.TrimSpace(head),
		IsDirty:      strings.TrimSpace(status) != "",
		ChangedFiles: changedFiles(status),
	}
	return state, unstaged, staged, true
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := gitBytes(ctx, dir, args...)
	return string(out), err
}

func gitBytes(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.Output()
}

func changedFiles(status string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(status, "\n") {
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		file := strings.TrimSpace(line[3:])
		if idx := strings.Index(file, " -> "); idx >= 0 {
			file = strings.TrimSpace(file[idx+4:])
		}
		if file != "" && !seen[file] {
			seen[file] = true
			out = append(out, file)
		}
	}
	return out
}
