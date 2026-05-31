package handoff

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var treeExcludes = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"vendor":       {},
	".rekord":      {},
	"dist":         {},
	"build":        {},
	"target":       {},
}

func BuildTree(root string, maxDepth, maxFiles int) (string, error) {
	var b strings.Builder
	count := 0
	truncated := false

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}

		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		depth := len(strings.Split(rel, string(filepath.Separator)))

		if d.IsDir() {
			if _, skip := treeExcludes[d.Name()]; skip {
				return fs.SkipDir
			}
			if depth >= maxDepth {
				return fs.SkipDir
			}
		}

		if depth > maxDepth {
			return nil
		}

		if count >= maxFiles {
			truncated = true
			return fs.SkipAll
		}
		count++

		indent := strings.Repeat("  ", depth-1)
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		fmt.Fprintf(&b, "%s%s\n", indent, name)
		return nil
	})
	if err != nil {
		return "", err
	}
	if truncated {
		b.WriteString("… (truncated)\n")
	}
	return b.String(), nil
}
