package launcherctl

import (
	"context"
	"fmt"
	"time"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

const launcherUpdateProbeTimeout = 30 * time.Second

// UpdateProbeSnapshot describes the launcher's non-blocking update check status.
type UpdateProbeSnapshot struct {
	CheckBusy   bool
	InstallBusy bool
	Available   bool
	Message     string
}

// UpdateProbeState returns a safe value snapshot of launcher update probe state.
func (c *Controller) UpdateProbeState() UpdateProbeSnapshot {
	if c == nil {
		return UpdateProbeSnapshot{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return UpdateProbeSnapshot{
		CheckBusy:   c.updateCheckBusy,
		InstallBusy: c.installUpdateBusy,
		Available:   c.updateAvailable,
		Message:     c.updateStatus,
	}
}

// SetUpdateAvailableForTest configures update availability (test helper).
func (c *Controller) SetUpdateAvailableForTest(avail bool) {
	c.mu.Lock()
	c.updateAvailable = avail
	c.mu.Unlock()
}

// CheckUpdate triggers a non-blocking update probe against buildVersion.
func (c *Controller) CheckUpdate(win WindowRef, buildVersion string) {
	c.runUpdateProbe(win, buildVersion, true)
}

// InstallUpdate opens the updater's interactive check/center and updates launcher probe status.
func (c *Controller) InstallUpdate(win WindowRef, version string) {
	if u := c.updaterCenter(); u != nil {
		u.ShowUpdateCheck(win)
	}
	c.runUpdateProbe(win, version, false)
}

// RunUpdateProbe launches a non-blocking update check unless one of that kind is already active.
func (c *Controller) RunUpdateProbe(win WindowRef, version string, reportAllStatuses bool) {
	c.runUpdateProbe(win, version, reportAllStatuses)
}

func (c *Controller) runUpdateProbe(win WindowRef, version string, reportAllStatuses bool) {
	env := c.hostEnv()
	if env.Quitting {
		return
	}
	c.mu.Lock()
	var gen uint64
	if reportAllStatuses {
		if c.updateCheckBusy {
			c.mu.Unlock()
			return
		}
		c.updateCheckBusy = true
		c.updateCheckGen++
		gen = c.updateCheckGen
		c.updateStatus = i18n.T(i18n.KeyCmdCheckingForUpdates)
	} else {
		if c.installUpdateBusy {
			c.mu.Unlock()
			return
		}
		c.installUpdateBusy = true
		c.installUpdateGen++
		gen = c.installUpdateGen
	}
	c.mu.Unlock()

	baseCtx := c.lifecycleContext()
	route := env.DownloadRoute
	go func() {
		ctx, cancel := context.WithTimeout(baseCtx, launcherUpdateProbeTimeout)
		defer cancel()
		res, err := desktopupdate.CheckForUpdatesVersionRoute(ctx, version, route)
		apply := func() {
			curEnv := c.hostEnv()
			c.mu.Lock()
			defer c.mu.Unlock()
			if reportAllStatuses {
				if c.updateCheckGen != gen {
					return
				}
				c.updateCheckBusy = false
			} else {
				if c.installUpdateGen != gen {
					return
				}
				c.installUpdateBusy = false
			}
			if curEnv.Quitting {
				return
			}
			if err != nil {
				c.updateAvailable = false
				if reportAllStatuses {
					c.updateStatus = err.Error()
				}
				return
			}
			c.updateAvailable = res.Status == desktopupdate.StatusUpdateAvailable
			if c.updateAvailable {
				c.updateStatus = fmt.Sprintf(i18n.T(i18n.KeyLauncherUpdateAvailable), res.Version, version)
			} else if reportAllStatuses {
				c.updateStatus = string(res.Status) + ": " + res.Message
				if res.ReleaseURL != "" {
					c.updateStatus += " · " + res.ReleaseURL
				}
			}
		}
		if win != nil && c.deps != nil {
			c.deps.UpdateWindow(win, apply)
		} else {
			apply()
		}
	}()
}
