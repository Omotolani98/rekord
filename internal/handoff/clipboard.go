package handoff

import (
	"errors"
	"os/exec"
	"strings"
)

type clipboardTool struct {
	name string
	args []string
}

var clipboardTools = []clipboardTool{
	{"pbcopy", nil},
	{"wl-copy", nil},
	{"xclip", []string{"-selection", "clipboard"}},
}

func Copy(s string) error {
	for _, t := range clipboardTools {
		if _, err := exec.LookPath(t.name); err != nil {
			continue
		}
		cmd := exec.Command(t.name, t.args...)
		cmd.Stdin = strings.NewReader(s)
		return cmd.Run()
	}
	return errors.New("no clipboard tool found (install pbcopy, wl-copy, or xclip)")
}
