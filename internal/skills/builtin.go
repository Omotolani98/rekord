package skills

import (
	"embed"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

var (
	builtinOnce sync.Once
	builtinList []Skill
)

func Builtins() []Skill {
	builtinOnce.Do(func() {
		entries, err := builtinFS.ReadDir("builtin")
		if err != nil {
			return
		}
		for _, e := range entries {
			data, derr := builtinFS.ReadFile("builtin/" + e.Name())
			if derr != nil {
				continue
			}
			var s Skill
			if yaml.Unmarshal(data, &s) != nil {
				continue
			}
			builtinList = append(builtinList, s)
		}
		sort.Slice(builtinList, func(i, j int) bool {
			return builtinList[i].Name < builtinList[j].Name
		})
	})
	return builtinList
}
