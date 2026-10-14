package terminal

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Sessions keeps one PTY per workspace. Switching or hiding never kills a shell.
// The owner calls CloseWorkspace for an explicit close/restart and Close at exit.
// Access is serialized by the main state's lock.
type Sessions struct {
	panels map[string]*Panel
	open   func(string, bool) *Panel
}

func workspaceKey(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	root = filepath.Clean(root)
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
	}
	return root
}
func (s *Sessions) Get(root string, dark bool) *Panel {
	key := workspaceKey(root)
	if s.panels == nil {
		s.panels = make(map[string]*Panel)
	}
	if p := s.panels[key]; p != nil {
		return p
	}
	open := s.open
	if open == nil {
		open = Open
	}
	p := open(strings.TrimSpace(root), dark)
	s.panels[key] = p
	return p
}
func (s *Sessions) CloseWorkspace(root string) {
	key := workspaceKey(root)
	if p := s.panels[key]; p != nil {
		p.Close()
		delete(s.panels, key)
	}
}
func (s *Sessions) Close() {
	for key, p := range s.panels {
		p.Close()
		delete(s.panels, key)
	}
}
