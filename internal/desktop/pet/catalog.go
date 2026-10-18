// Package pet owns the opt-in sprite/THA4 companion. It does not implement Cubism.
package pet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/image/webp"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Skin struct {
	ID, Name string
	Rows     int
	Sheet    []byte
}

func DefaultRoot() string {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		h, _ := os.UserHomeDir()
		home = filepath.Join(h, ".codex")
	}
	return filepath.Join(home, "pets")
}

// Discover reads only pet.json and its confined WebP sprite. os.Root rejects
// symlink escapes as well as traversal. No user package is executed or changed.
func Discover(dir string) ([]Skin, error) {
	root, err := os.OpenRoot(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	var out []Skin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sub, err := root.OpenRoot(entry.Name())
		if err != nil {
			continue
		}
		skin, err := readSkin(sub, entry.Name())
		sub.Close()
		if err == nil {
			out = append(out, skin)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func limited(root *os.Root, name string, max int64) ([]byte, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > max {
		return nil, fmt.Errorf("pet file is not a bounded regular file")
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if int64(len(b)) > max {
		return nil, fmt.Errorf("pet file too large")
	}
	return b, err
}
func readSkin(root *os.Root, id string) (Skin, error) {
	var m struct {
		DisplayName string `json:"displayName"`
		Path        string `json:"spritesheetPath"`
	}
	b, err := limited(root, "pet.json", 64<<10)
	if err != nil {
		return Skin{}, err
	}
	if err = json.Unmarshal(b, &m); err != nil {
		return Skin{}, err
	}
	if m.Path == "" || strings.ContainsAny(m.Path, "\\:") || !filepath.IsLocal(m.Path) {
		return Skin{}, fmt.Errorf("unsafe sheet path")
	}
	b, err = limited(root, m.Path, 16<<20)
	if err != nil {
		return Skin{}, err
	}
	cfg, err := webp.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return Skin{}, err
	}
	if cfg.Width != 1536 || (cfg.Height != 1872 && cfg.Height != 2288) {
		return Skin{}, fmt.Errorf("unsupported atlas")
	}
	if m.DisplayName == "" {
		m.DisplayName = id
	}
	return Skin{ID: id, Name: m.DisplayName, Rows: cfg.Height / 208, Sheet: b}, nil
}
