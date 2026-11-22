package main

import (
	"log"
	"os"
	"strings"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
)

// ensureLaunchWorkspace resolves the workspace a cold start hands the Host the
// way the reference shell does (config.js defaultWorkspace + app-ready.js
// mkdirSync/saveConfig): an empty setting adopts the default Documents folder,
// and a saved folder that vanished is recreated — only a recreate that fails
// falls back to the default. A folder that is adopted is persisted so the
// launch and every later one agree; a start that still cannot be provisioned
// keeps the picker's decline path.
func (s *mainState) ensureLaunchWorkspace(cfg config.Config) config.Config {
	ws := strings.TrimSpace(cfg.Workspace)
	if ws != "" {
		if info, err := os.Stat(ws); err == nil && info.IsDir() {
			return cfg
		}
		if err := os.MkdirAll(ws, 0o755); err == nil {
			log.Printf("workspace: recreated %q", ws)
			return cfg
		}
		log.Printf("workspace: saved folder %q cannot be recreated; adopting the default", ws)
	}
	ws = paths.DefaultWorkspace()
	if err := os.MkdirAll(ws, 0o755); err != nil {
		log.Printf("workspace: cannot create %q: %v", ws, err)
		return cfg
	}
	s.mu.Lock()
	store := s.cfgStore
	s.mu.Unlock()
	if store != nil {
		if err := store.Update(func(c *config.Config) { c.Workspace = ws }); err != nil {
			log.Printf("workspace: save %q: %v", ws, err)
		}
	}
	cfg.Workspace = ws
	return cfg
}
