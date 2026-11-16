package updater

import (
	"fmt"
)

// OpenCenter restores saved state if needed and displays the update center modal
// attached to the given parent window (or the main/launcher window if nil).
func (c *Center) OpenCenter(win WindowRef) {
	if c == nil {
		return
	}
	c.EnsureLoaded()
	if c.bridge == nil {
		return
	}
	c.bridge.RunOnMain(func() {
		owner := c.bridge.DialogOwner(win)
		if owner == nil {
			mainWin, launcherWin := c.bridge.PrimaryWindows()
			owner = mainWin
			if launcherWin != nil && c.bridge.WindowAlive(launcherWin) {
				owner = launcherWin
			}
			if owner == nil || !c.bridge.WindowAlive(owner) {
				return
			}
			c.bridge.RestoreShowFocus(owner)
		}
		c.ShowModal(owner)
	})
}

// ShowModal opens the update center modal anchored on parent (creating a
// dedicated modal window if parent hosts a web Page).
func (c *Center) ShowModal(parent WindowRef) {
	if c == nil || c.bridge == nil || parent == nil || !c.bridge.WindowAlive(parent) {
		return
	}
	c.mu.Lock()
	current := c.releases
	c.mu.Unlock()
	if current.ModalOpen && current.ModalParent == parent && current.ModalOwner != nil && c.bridge.WindowAlive(current.ModalOwner) {
		c.bridge.RestoreShowFocus(current.ModalOwner)
		return
	}
	c.CloseModal(nil)

	owner := parent
	temporary := c.bridge.WindowHasPage(parent)
	if temporary {
		owner = c.bridge.OpenModalWindow(parent, lifecycleText("更新中心", "Update center"))
	}
	if owner == nil || !c.bridge.WindowAlive(owner) {
		return
	}
	c.mu.Lock()
	r := &c.releases
	r.ModalOwner, r.ModalParent, r.ModalOpen, r.ModalTemporary = owner, parent, true, temporary
	c.attention.tracker.DismissCurrent()
	c.mu.Unlock()

	off := c.bridge.WatchModalWindows(parent, owner, func() { c.CloseModal(owner) })
	c.mu.Lock()
	if c.releases.ModalOwner == owner {
		c.releases.ModalOff = off
	} else if off != nil {
		c.mu.Unlock()
		off()
		goto notify
	}
	c.mu.Unlock()
notify:
	c.bridge.InvalidateWindow(owner)
}

// CloseModal closes the active update center modal if it matches expected (or any if nil).
func (c *Center) CloseModal(expected WindowRef) {
	if c == nil {
		return
	}
	c.mu.Lock()
	r := &c.releases
	if expected != nil && r.ModalOwner != expected {
		c.mu.Unlock()
		return
	}
	owner, temporary, off := r.ModalOwner, r.ModalTemporary, r.ModalOff
	r.ModalOwner, r.ModalParent, r.ModalOpen, r.ModalTemporary, r.ModalOff = nil, nil, false, false, nil
	c.mu.Unlock()
	if off != nil {
		off()
	}
	if c.bridge == nil {
		return
	}
	if temporary && owner != nil && c.bridge.WindowAlive(owner) {
		c.bridge.DestroyWindow(owner)
	}
	if owner != nil && c.bridge.WindowAlive(owner) {
		c.bridge.InvalidateWindow(owner)
	}
}

// ModalVisibleFor reports whether the update center modal is currently owned by win.
func (c *Center) ModalVisibleFor(win WindowRef) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.releases.ModalOpen && c.releases.ModalOwner == win
}

// LifecycleWindowOwnedLocked-equivalent helper exported for shell integration checks.
func (c *Center) IsTemporaryModalWindow(win WindowRef) bool {
	if c == nil || win == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.releases.ModalTemporary && c.releases.ModalOwner == win
}

// NotifyReady publishes one OS notification per verified staged release.
func (c *Center) NotifyReady(owner WindowRef) {
	if c == nil || c.bridge == nil {
		return
	}
	env := c.env()
	route := env.DownloadRoute

	c.mu.Lock()
	r := &c.releases
	key := r.Active.PackageKey() + "|" + r.Staged.SHA256
	ready := r.Phase == "ready" && r.Staged.Path != "" && !r.Busy && !env.Quitting && r.Route == route
	if !ready {
		c.mu.Unlock()
		c.ClearReadyNotice()
		return
	}
	if r.ReadyNotified == key {
		c.mu.Unlock()
		return
	}
	r.ReadyNotified = key
	generation, stagedPath := r.AttentionGeneration, r.Staged.Path
	version := r.StagedTitle
	if version == "" {
		version = r.StagedTag
	}
	c.mu.Unlock()

	c.ClearReadyNotice()
	n := c.bridge.NewNotification(
		lifecycleText("更新已就绪", "Update ready"),
		fmt.Sprintf(lifecycleText("%s 已下载并校验，打开更新中心选择安装。", "%s is downloaded and verified. Open the update center to install."), version),
		true,
	)
	if n == nil {
		return
	}
	c.mu.Lock()
	c.releases.ReadyNotice = n
	c.mu.Unlock()

	if owner != nil {
		off := c.bridge.WatchFocusAndClose(owner, nil, func() { c.ClearReadyNotice() })
		c.mu.Lock()
		if c.releases.ReadyNotice == n {
			c.releases.ReadyNoticeOff = off
			off = nil
		}
		c.mu.Unlock()
		if off != nil {
			off()
		}
	}
	n.OnClick(func() {
		curEnv := c.env()
		owned := c.windowOwned(owner)
		currentRoute := curEnv.DownloadRoute
		c.mu.Lock()
		r := &c.releases
		ownedOrTemp := owned || (r.ModalTemporary && r.ModalOwner == owner)
		valid := r.ReadyNotice == n &&
			r.AttentionGeneration == generation &&
			r.Route == route &&
			currentRoute == route &&
			r.Active.PackageKey()+"|"+r.Staged.SHA256 == key &&
			r.Staged.Path == stagedPath &&
			r.Phase == "ready" &&
			!r.Busy &&
			!curEnv.Quitting &&
			ownedOrTemp
		c.mu.Unlock()
		if !valid || owner == nil || !c.bridge.WindowAlive(owner) {
			return
		}
		c.ClearReadyNotice()
		c.bridge.RestoreShowFocus(owner)
		c.OpenCenter(owner)
	})
	if c.bridge.SystemNotificationsSupported() {
		_ = n.Show()
	}
}

// ClearReadyNotice dismisses any active update-ready desktop notification.
func (c *Center) ClearReadyNotice() {
	if c == nil {
		return
	}
	c.mu.Lock()
	n, off := c.releases.ReadyNotice, c.releases.ReadyNoticeOff
	c.releases.ReadyNotice, c.releases.ReadyNoticeOff = nil, nil
	c.mu.Unlock()
	if off != nil {
		off()
	}
	if n != nil {
		n.Close()
	}
}

func (c *Center) windowOwned(win WindowRef) bool {
	if c == nil || win == nil || c.bridge == nil {
		return false
	}
	return c.bridge.OwnedWindow(win)
}
