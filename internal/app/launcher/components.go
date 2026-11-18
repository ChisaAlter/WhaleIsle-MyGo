package launcherctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
)

// RefreshComponents scans desktop components and bundled harness version off the
// UI thread while coalescing overlapping requests via componentsFresh.
func (c *Controller) RefreshComponents() {
	if c == nil {
		return
	}
	env := c.hostEnv()
	if env.Quitting {
		return
	}
	c.mu.Lock()
	if c.componentsRefreshBusy {
		c.componentsFresh = false
		c.mu.Unlock()
		return
	}
	c.componentsRefreshBusy = true
	c.componentsFresh = true
	c.mu.Unlock()

	go func() {
		if c.deps != nil {
			c.deps.ConfigureGenericComponents()
		}
		seen := map[string]struct{}{}
		var rows []desktopcomponents.Row
		for _, root := range DesktopComponentRoots() {
			for _, row := range desktopcomponents.ListUserData(root) {
				if _, ok := seen[row.ID]; ok {
					continue
				}
				seen[row.ID] = struct{}{}
				rows = append(rows, row)
			}
		}
		harnessVersion := ReadHarnessVersion()
		publish := func() {
			curEnv := c.hostEnv()
			c.mu.Lock()
			c.componentsRefreshBusy = false
			quitting := curEnv.Quitting
			if !quitting {
				c.components = rows
				c.harnessVersion = harnessVersion
			}
			c.mu.Unlock()
			if !quitting && c.deps != nil {
				c.deps.ObserveBridgeAlerts()
			}
		}
		curEnv := c.hostEnv()
		if curEnv.MainWindow != nil && c.deps != nil {
			c.deps.UpdateWindow(curEnv.MainWindow, publish)
		} else {
			publish()
		}
	}()
}

// MarkComponentsStale triggers a fresh scan on the next launcher render pass.
func (c *Controller) MarkComponentsStale() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.componentsFresh = false
	c.mu.Unlock()
}

// Components returns a shallow snapshot of discovered desktop component rows.
func (c *Controller) Components() []desktopcomponents.Row {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]desktopcomponents.Row(nil), c.components...)
}

// HarnessVersionSnapshot returns the last scanned bundled DSH package version.
func (c *Controller) HarnessVersionSnapshot() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.harnessVersion
}

// VendorReady returns whether the bundled harness CLI entry exists on disk,
// using a 1s stale-while-revalidate probe off the UI thread.
func (c *Controller) VendorReady(binJS string, notify func()) bool {
	if strings.TrimSpace(binJS) == "" {
		binJS = filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")
	}
	return c.vendorProbe.get(binJS, time.Second, func() bool {
		info, err := os.Stat(binJS)
		return err == nil && !info.IsDir()
	}, notify)
}

// ReadHarnessVersion reads package.json version from paths.VendorDir().
func ReadHarnessVersion() string {
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
