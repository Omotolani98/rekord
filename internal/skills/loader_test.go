package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, dir, file, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, file)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeSkill(t, t.TempDir(), "s.yaml", "name: demo\ndescription: d\nsteps:\n  - run: echo hi\n")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Name != "demo" || len(s.Steps) != 1 || s.Steps[0].Run != "echo hi" {
		t.Fatalf("skill = %+v", s)
	}
	if s.SourcePath != path {
		t.Fatalf("SourcePath = %q", s.SourcePath)
	}
}

func TestLoadInvalid(t *testing.T) {
	cases := map[string]string{
		"no-name":   "description: d\nsteps:\n  - run: echo hi\n",
		"no-steps":  "name: demo\n",
		"empty-run": "name: demo\nsteps:\n  - run: \"\"\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeSkill(t, t.TempDir(), "s.yaml", body)
			if _, err := Load(path); err == nil {
				t.Fatal("Load err = nil, want error")
			}
		})
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "a.yaml", "name: a\nsteps:\n  - run: echo a\n")
	writeSkill(t, dir, "b.yml", "name: b\nsteps:\n  - run: echo b\n")
	writeSkill(t, dir, "ignore.txt", "not a skill")

	list, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
}

func TestLoadDirMissing(t *testing.T) {
	list, err := LoadDir(filepath.Join(t.TempDir(), "nope"))
	if err != nil || list != nil {
		t.Fatalf("LoadDir missing = (%v, %v), want (nil, nil)", list, err)
	}
}

func TestFindBuiltin(t *testing.T) {
	s, err := Find(nil, "go-project-demo")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if s.Name != "go-project-demo" {
		t.Fatalf("name = %q", s.Name)
	}
	if _, err := Find(nil, "does-not-exist"); err == nil {
		t.Fatal("Find unknown err = nil, want error")
	}
}
