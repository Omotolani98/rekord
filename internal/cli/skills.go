package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/Omotolani98/rekord/internal/recorder"
	"github.com/Omotolani98/rekord/internal/session"
	"github.com/Omotolani98/rekord/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Run reusable recording recipes",
	}
	cmd.AddCommand(newSkillsListCommand(), newSkillsRunCommand())
	return cmd
}

func skillDirs(local string) []string {
	dirs := []string{local}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".rekord", "skills"))
	}
	return dirs
}

func newSkillsListCommand() *cobra.Command {
	var skillsDir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSkillsList(cmd, skillsDir)
		},
	}
	cmd.Flags().StringVar(&skillsDir, "skills-dir", filepath.Join(".rekord", "skills"), "local skills directory")
	return cmd
}

func runSkillsList(cmd *cobra.Command, skillsDir string) error {
	dirs := skillDirs(skillsDir)
	type row struct{ name, source, desc string }
	seen := make(map[string]struct{})
	var rows []row

	sources := []string{"local", "global"}
	for i, dir := range dirs {
		loaded, err := skills.LoadDir(dir)
		if err != nil {
			return err
		}
		src := "local"
		if i < len(sources) {
			src = sources[i]
		}
		for _, s := range loaded {
			if _, ok := seen[s.Name]; ok {
				continue
			}
			seen[s.Name] = struct{}{}
			rows = append(rows, row{s.Name, src, s.Description})
		}
	}
	for _, s := range skills.Builtins() {
		if _, ok := seen[s.Name]; ok {
			continue
		}
		seen[s.Name] = struct{}{}
		rows = append(rows, row{s.Name, "builtin", s.Description})
	}

	if len(rows) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No skills found.")
		return err
	}

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSOURCE\tDESCRIPTION")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.name, r.source, r.desc)
	}
	return tw.Flush()
}

func newSkillsRunCommand() *cobra.Command {
	var name, root, skillsDir string
	cmd := &cobra.Command{
		Use:   "run <skill>",
		Short: "Run a skill and record it as a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillsRun(cmd, args[0], name, root, skillsDir)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "recording name (defaults to the skill name)")
	cmd.Flags().StringVar(&root, "root", defaultSessionsRoot(), "sessions root directory")
	cmd.Flags().StringVar(&skillsDir, "skills-dir", filepath.Join(".rekord", "skills"), "local skills directory")
	return cmd
}

func runSkillsRun(cmd *cobra.Command, skillName, name, root, skillsDir string) error {
	skill, err := skills.Find(skillDirs(skillsDir), skillName)
	if err != nil {
		return err
	}
	if name == "" {
		name = skill.Name
	}
	if err := session.ValidateName(name); err != nil {
		return fmt.Errorf("--name is required: %w", err)
	}

	ctx := cmd.Context()
	now := time.Now().UTC()
	id := session.NewID(name, now)
	m := session.Metadata{
		ID:            id,
		Name:          name,
		CreatedAt:     now,
		Status:        session.StatusRecording,
		RekordVersion: Version(),
	}
	store := session.NewFileStore(root)
	if err := store.Create(ctx, m); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	script := skills.RenderScript(skill)
	eventsPath := filepath.Join(store.SessionDir(id), "events.jsonl")
	rec := recorder.NewPTYRecorder()
	res, recErr := rec.Record(ctx, recorder.Options{
		Command:    []string{"/bin/sh", "-c", script},
		EventsPath: eventsPath,
		Stdin:      cmd.InOrStdin(),
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
	})

	ended := res.EndedAt
	if ended.IsZero() {
		ended = time.Now()
	}
	ended = ended.UTC()
	m.EndedAt = &ended
	m.DurationMS = res.DurationMS
	if recErr != nil {
		m.Status = session.StatusFailed
	} else {
		m.Status = session.StatusCompleted
	}

	if err := store.WriteMetadata(context.Background(), m); err != nil {
		if recErr != nil {
			return fmt.Errorf("recorder failed: %w; also failed to update metadata: %v", recErr, err)
		}
		return fmt.Errorf("update metadata: %w", err)
	}

	if recErr != nil {
		return recErr
	}
	if res.ExitCode != 0 {
		return &exitCodeError{code: res.ExitCode}
	}
	return nil
}
