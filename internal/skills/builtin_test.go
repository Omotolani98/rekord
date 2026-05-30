package skills

import "testing"

func TestBuiltins(t *testing.T) {
	got := Builtins()
	want := map[string]bool{
		"go-project-demo":     false,
		"docker-demo":         false,
		"kubernetes-demo":     false,
		"terraform-demo":      false,
		"github-actions-demo": false,
	}
	for _, s := range got {
		if s.Name == "" || len(s.Steps) == 0 {
			t.Fatalf("builtin %q invalid: %+v", s.Name, s)
		}
		if _, ok := want[s.Name]; ok {
			want[s.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("builtin %q missing", name)
		}
	}
}
