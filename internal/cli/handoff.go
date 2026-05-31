package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Omotolani98/rekord/internal/config"
	"github.com/Omotolani98/rekord/internal/events"
	"github.com/Omotolani98/rekord/internal/handoff"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/spf13/cobra"
)

const (
	handoffDirPerm    = 0o700
	handoffFilePerm   = 0o600
	handoffOutputMax  = 8192
	handoffGitDiffMax = 20000
	handoffTreeDepth  = 4
	handoffTreeFiles  = 500
	handoffMaxErrors  = 20
)

var errorLineRe = regexp.MustCompile(`(?i)(error|fail|panic|fatal)`)

func newHandoffCommand() *cobra.Command {
	var root, cfgPath string
	var includeGit, includeTree, includeLogs, copyClip bool

	cmd := &cobra.Command{
		Use:   "handoff <session>",
		Short: "Generate AI-ready context from a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHandoff(cmd, args[0], root, cfgPath, includeGit, includeTree, includeLogs, copyClip)
		},
	}

	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&cfgPath, "config", "rekord.yaml", "config file with prompt and redaction patterns")
	cmd.Flags().BoolVar(&includeGit, "include-git", false, "include git status and diff context")
	cmd.Flags().BoolVar(&includeTree, "include-tree", false, "include a repository tree snapshot")
	cmd.Flags().BoolVar(&includeLogs, "include-logs", false, "include captured session logs")
	cmd.Flags().BoolVar(&copyClip, "copy", false, "copy the context to the clipboard")

	return cmd
}

func runHandoff(cmd *cobra.Command, ref, root, cfgPath string, includeGit, includeTree, includeLogs, copyClip bool) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	red, err := buildRedactor(cfg)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	store := session.NewFileStore(root)
	m, err := store.Resolve(ctx, ref)
	if err != nil {
		return err
	}

	evs, err := events.ReadAll(filepath.Join(store.SessionDir(m.ID), "events.jsonl"))
	if err != nil {
		return fmt.Errorf("read events: %w", err)
	}

	cmds, err := extractWithConfig(cfg, evs)
	if err != nil {
		return err
	}

	fullOutput := red.Redact(joinOutput(evs))
	excerpt := fullOutput
	if len(excerpt) > handoffOutputMax {
		excerpt = excerpt[len(excerpt)-handoffOutputMax:]
	}

	in := handoff.Input{
		Metadata: redactMetadata(red, m),
		Commands: redactCommands(red, cmds),
		Output:   excerpt,
		Errors:   errorLines(fullOutput),
	}

	handoffDir := filepath.Join(store.SessionDir(m.ID), "handoff")
	if err := os.MkdirAll(handoffDir, handoffDirPerm); err != nil {
		return fmt.Errorf("create handoff directory: %w", err)
	}

	if includeGit {
		gc, ok := handoff.GatherGit(ctx, m.CWD, handoffGitDiffMax)
		if ok {
			gc.Status = red.Redact(gc.Status)
			gc.Diff = red.Redact(gc.Diff)
			in.Git = &gc
			if err := os.WriteFile(filepath.Join(handoffDir, "git.diff"), []byte(gc.Diff), handoffFilePerm); err != nil {
				return fmt.Errorf("write git.diff: %w", err)
			}
		} else {
			fmt.Fprintln(cmd.ErrOrStderr(), "handoff: not a git repository, skipping git context")
		}
	}

	if includeTree {
		tree, terr := handoff.BuildTree(m.CWD, handoffTreeDepth, handoffTreeFiles)
		if terr != nil {
			return fmt.Errorf("build tree: %w", terr)
		}
		in.Tree = tree
		if err := os.WriteFile(filepath.Join(handoffDir, "tree.txt"), []byte(tree), handoffFilePerm); err != nil {
			return fmt.Errorf("write tree.txt: %w", err)
		}
	}

	if includeLogs {
		if err := os.WriteFile(filepath.Join(handoffDir, "logs.txt"), []byte(fullOutput), handoffFilePerm); err != nil {
			return fmt.Errorf("write logs.txt: %w", err)
		}
	}

	contextMD := handoff.Generate(in)
	contextPath := filepath.Join(handoffDir, "context.md")
	if err := os.WriteFile(contextPath, []byte(contextMD), handoffFilePerm); err != nil {
		return fmt.Errorf("write context.md: %w", err)
	}

	if copyClip {
		if err := handoff.Copy(contextMD); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "handoff: clipboard copy failed: %v\n", err)
		}
	}

	out := cmd.OutOrStdout()
	st := newStyler(out)
	if st.on {
		_, err = fmt.Fprintln(out, st.green("✓ ")+contextPath+st.dim(" · ready for your agent"))
	} else {
		_, err = fmt.Fprintln(out, contextPath)
	}
	return err
}

func joinOutput(evs []events.Event) string {
	var b strings.Builder
	for _, e := range evs {
		if e.Type == events.TypeOutput {
			b.WriteString(strings.ReplaceAll(e.Data, "\r", ""))
		}
	}
	return b.String()
}

func errorLines(output string) []string {
	var out []string
	for _, line := range strings.Split(output, "\n") {
		if errorLineRe.MatchString(line) {
			out = append(out, strings.TrimSpace(line))
			if len(out) >= handoffMaxErrors {
				break
			}
		}
	}
	return out
}
