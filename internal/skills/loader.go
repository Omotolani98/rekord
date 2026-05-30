package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, fmt.Errorf("read skill: %w", err)
	}
	var s Skill
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Skill{}, fmt.Errorf("parse skill %s: %w", path, err)
	}
	if err := validate(s); err != nil {
		return Skill{}, fmt.Errorf("invalid skill %s: %w", path, err)
	}
	s.SourcePath = path
	return s, nil
}

func validate(s Skill) error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("name is required")
	}
	if len(s.Steps) == 0 {
		return errors.New("at least one step is required")
	}
	for i, st := range s.Steps {
		if strings.TrimSpace(st.Run) == "" {
			return fmt.Errorf("step %d: run is required", i+1)
		}
	}
	return nil
}

func LoadDir(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	var out []Skill
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		s, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func Find(dirs []string, name string) (Skill, error) {
	for _, dir := range dirs {
		skills, err := LoadDir(dir)
		if err != nil {
			return Skill{}, err
		}
		for _, s := range skills {
			if s.Name == name {
				return s, nil
			}
		}
	}
	for _, s := range Builtins() {
		if s.Name == name {
			return s, nil
		}
	}
	return Skill{}, fmt.Errorf("skill %q not found", name)
}
