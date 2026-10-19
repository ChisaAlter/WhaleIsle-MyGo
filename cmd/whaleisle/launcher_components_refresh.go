package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
)

// A component worker may be waiting for a lifecycle operation. Never wait for
// its list response in the launcher's view/input thread. Fresh records whether
// this refresh has consumed the latest request; invalidations during the scan
// stay false so the next view requests one more scan.
func (s *mainState) refreshLauncherComponents() {
	s.mu.Lock()
	if s.quitting {
		s.mu.Unlock()
		return
	}
	if s.launcherComponentsRefreshBusy {
		s.launcherComponentsFresh = false
		s.mu.Unlock()
		return
	}
	s.launcherComponentsRefreshBusy = true
	s.launcherComponentsFresh = true
	s.mu.Unlock()
	go func() {
		s.configureGenericComponents()
		seen := map[string]struct{}{}
		var rows []desktopcomponents.Row
		for _, root := range desktopComponentRoots() {
			for _, row := range desktopcomponents.ListUserData(root) {
				if _, ok := seen[row.ID]; ok {
					continue
				}
				seen[row.ID] = struct{}{}
				rows = append(rows, row)
			}
		}
		harnessVersion := readLauncherHarnessVersion()
		publish := func() {
			s.mu.Lock()
			s.launcherComponentsRefreshBusy = false
			quitting := s.quitting
			if !quitting {
				s.launcherComponents = rows
				s.launcherHarnessVersion = harnessVersion
			}
			s.mu.Unlock()
			if !quitting {
				s.observeWhaleBridgeAlerts()
			}
		}
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Update(publish)
		} else {
			publish()
		}
	}()
}

func readLauncherHarnessVersion() string {
	raw, err := os.ReadFile(filepath.Join(paths.VendorDir(), "deepseek-harness", "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(raw, &pkg) != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Version)
}

func (s *mainState) harnessVersionSnapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.launcherHarnessVersion
}
