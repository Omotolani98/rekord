package skills

import "strings"

func RenderScript(s Skill) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	for _, step := range s.Steps {
		b.WriteString("echo " + singleQuote("$ "+step.Run) + "\n")
		b.WriteString(step.Run + "\n")
	}
	return b.String()
}

func singleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
