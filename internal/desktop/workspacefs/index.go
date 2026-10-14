package workspacefs

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IndexRelPaths walks root and returns slash-relative file paths (dirs included).
// Hidden names (dot prefix) are skipped. When maxFiles is positive, walking stops after that many paths.
func IndexRelPaths(root string, maxFiles int) ([]string, error) {
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, nil
	}
	var out []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := RelPath(root, path)
		if err != nil {
			return nil
		}
		out = append(out, rel)
		if maxFiles > 0 && len(out) >= maxFiles {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out, nil
}
