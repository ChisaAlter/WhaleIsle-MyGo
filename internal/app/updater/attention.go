package updater

import (
	"fmt"
	"runtime"

	"whaleisle/internal/desktop/update"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/i18n"
)

type attentionState struct {
	hold         bool
	tracker      startuprec.UpdateAttention
	vendorReady  bool
	notice       NoticeHandle
	noticeWindow WindowRef
	noticeTicket startuprec.UpdateAttentionTicket
	watchWindow  WindowRef
	watchOff     func()
}

func (c *Center) windowContext(win WindowRef) (owned bool, winKey string) {
	if c == nil || c.bridge == nil || win == nil {
		return false, ""
	}
	return c.bridge.OwnedWindow(win), c.bridge.WindowKey(win)
}

// Caller holds c.mu and has already captured env, ownedByShell, winKey without c.mu held.
func (c *Center) startupUpdateScopeLocked(env ShellEnvironment, ownedByShell bool, win WindowRef, winKey string, snapshot ReleaseState, vendorReady bool) (startuprec.UpdateAttentionScope, update.NativeRelease) {
	empty := startuprec.UpdateAttentionScope{}
	owned := ownedByShell || (c.releases.ModalTemporary && c.releases.ModalOwner == win)
	if win == nil || !owned || env.Quitting || env.LauncherStartBusy || env.LauncherImportBusy {
		return empty, update.NativeRelease{}
	}
	route := env.DownloadRoute
	current := &c.releases
	if snapshot.Busy || snapshot.Paused || snapshot.Err != "" || snapshot.CatalogRoute != snapshot.Route || route != snapshot.Route ||
		current.Busy || current.Paused || current.Err != "" || current.Route != snapshot.Route || current.CatalogRoute != snapshot.CatalogRoute ||
		current.AttentionGeneration != snapshot.AttentionGeneration {
		return empty, update.NativeRelease{}
	}
	for _, release := range snapshot.Rows {
		if release.UnavailableReason != "" || release.Prerelease || release.Validate(runtime.GOOS, runtime.GOARCH, !vendorReady) != nil || update.CompareVersions(release.Version, env.BuildVersion) <= 0 {
			continue
		}
		if snapshot.StagedTag == release.Tag {
			return empty, update.NativeRelease{}
		}
		return startuprec.UpdateAttentionScope{
			Window:            winKey,
			Host:              env.HostID,
			HostGeneration:    env.HostGeneration,
			CatalogGeneration: snapshot.AttentionGeneration,
			Route:             snapshot.CatalogRoute,
			Tag:               release.Tag,
			Version:           release.Version,
		}, release
	}
	return empty, update.NativeRelease{}
}

func (c *Center) startupUpdateCurrentLocked(env ShellEnvironment, ownedByShell bool, win WindowRef, winKey string, ticket startuprec.UpdateAttentionTicket) bool {
	scope, _ := c.startupUpdateScopeLocked(env, ownedByShell, win, winKey, c.releases, c.attention.vendorReady)
	return c.attention.tracker.Current(ticket) && ticket.Scope == scope && env.AskOnUpdate
}

// RefreshAttention re-evaluates startup update confirmation against the cached catalog.
func (c *Center) RefreshAttention(win WindowRef) {
	if c == nil {
		return
	}
	c.mu.Lock()
	snapshot := c.releases
	snapshot.Rows = append([]update.NativeRelease(nil), snapshot.Rows...)
	vendorReady := c.attention.vendorReady
	c.mu.Unlock()
	c.MaybeAskStartupUpdate(win, snapshot, vendorReady)
}

// ClearAttentionNotice closes any transient startup update notification without
// consuming the visible pending confirmation state.
func (c *Center) ClearAttentionNotice() {
	if c == nil {
		return
	}
	c.mu.Lock()
	n, win := c.attention.notice, c.attention.noticeWindow
	c.attention.notice, c.attention.noticeWindow = nil, nil
	c.attention.noticeTicket = startuprec.UpdateAttentionTicket{}
	c.mu.Unlock()
	if n != nil {
		n.Close()
	}
	if c.bridge != nil && win != nil && c.bridge.WindowAlive(win) {
		c.bridge.FlashFrame(win, false)
	}
}

func (c *Center) watchAttentionWindow(win WindowRef) {
	if c == nil || c.bridge == nil || win == nil {
		return
	}
	c.mu.Lock()
	if c.attention.watchWindow == win {
		c.mu.Unlock()
		return
	}
	off := c.attention.watchOff
	c.attention.watchOff = nil
	c.attention.watchWindow = win
	c.attention.tracker.Clear()
	c.mu.Unlock()
	if off != nil {
		off()
	}
	c.ClearAttentionNotice()

	newOff := c.bridge.WatchFocusAndClose(win, func() {
		c.ClearAttentionNotice()
		c.Notify(win)
	}, func() {
		c.mu.Lock()
		current := c.attention.watchWindow == win
		if current {
			c.attention.tracker.Clear()
			c.attention.watchWindow = nil
			c.attention.watchOff = nil
		}
		c.mu.Unlock()
		if current {
			c.ClearAttentionNotice()
		}
	})
	c.mu.Lock()
	if c.attention.watchWindow == win {
		c.attention.watchOff = newOff
		newOff = nil
	}
	c.mu.Unlock()
	if newOff != nil {
		newOff()
	}
}

func (c *Center) showAttentionNotice(win WindowRef, ticket startuprec.UpdateAttentionTicket) {
	if c == nil || c.bridge == nil {
		return
	}
	env := c.env()
	ownedWin, winKey := c.windowContext(win)
	c.mu.Lock()
	if !c.startupUpdateCurrentLocked(env, ownedWin, win, winKey, ticket) {
		c.mu.Unlock()
		return
	}
	c.attention.noticeWindow, c.attention.noticeTicket = win, ticket
	c.mu.Unlock()

	c.bridge.FlashFrame(win, true)
	if !c.bridge.SystemNotificationsSupported() {
		return
	}
	n := c.bridge.NewNotification(
		i18n.T(i18n.KeyCmdMygoUpdateAwaitingConfirmation),
		fmt.Sprintf(i18n.T(i18n.KeyCmdIsAvailableReturnToConfirmationDownloa), ticket.Scope.Version),
		true,
	)
	if n == nil {
		return
	}
	env2 := c.env()
	ownedWin2, winKey2 := c.windowContext(win)
	c.mu.Lock()
	if !c.startupUpdateCurrentLocked(env2, ownedWin2, win, winKey2, ticket) || c.attention.noticeTicket != ticket {
		c.mu.Unlock()
		n.Close()
		return
	}
	c.attention.notice = n
	c.mu.Unlock()

	n.OnClick(func() {
		curEnv := c.env()
		owned, wk := c.windowContext(win)
		c.mu.Lock()
		valid := c.attention.notice == n && c.attention.noticeTicket == ticket && c.startupUpdateCurrentLocked(curEnv, owned, win, wk, ticket)
		c.mu.Unlock()
		if !valid {
			return
		}
		c.ClearAttentionNotice()
		c.bridge.RestoreShowFocus(win)
		c.OpenCenter(win)
		c.Notify(win)
	})
	err := n.Show()
	curEnv := c.env()
	owned3, winKey3 := c.windowContext(win)
	c.mu.Lock()
	current := c.attention.notice == n && c.startupUpdateCurrentLocked(curEnv, owned3, win, winKey3, ticket)
	c.mu.Unlock()
	if !current {
		n.Close()
	} else if err != nil {
		c.ClearAttentionNotice()
	}
}

// MaybeAskStartupUpdate raises a foreground confirmation dialog (or background
// attention reminder) for an available update matching AskOnUpdate policy.
func (c *Center) MaybeAskStartupUpdate(win WindowRef, snapshot ReleaseState, vendorReady bool) {
	if c == nil || c.bridge == nil || win == nil || !c.bridge.WindowAlive(win) {
		return
	}
	env := c.env()
	ownedWin, winKey := c.windowContext(win)

	c.mu.Lock()
	currentWindow := (ownedWin || (c.releases.ModalTemporary && c.releases.ModalOwner == win)) && !env.Quitting
	clearOwner := !currentWindow && c.attention.watchWindow == win
	if clearOwner {
		c.attention.tracker.Clear()
	}
	c.mu.Unlock()
	if !currentWindow {
		if clearOwner {
			c.ClearAttentionNotice()
		}
		return
	}

	c.watchAttentionWindow(win)
	visible, focused := c.bridge.WindowVisibility(win)
	env = c.env()
	ownedWin, winKey = c.windowContext(win)

	c.mu.Lock()
	c.attention.vendorReady = vendorReady
	scope, release := c.startupUpdateScopeLocked(env, ownedWin, win, winKey, snapshot, vendorReady)
	ticket, remind := c.attention.tracker.Observe(scope, env.AskOnUpdate, focused)
	obsolete := c.attention.noticeWindow != nil && c.attention.noticeTicket != ticket
	can := startuprec.CanPromptUpdate(
		env.AskOnUpdate,
		c.attention.tracker.Current(ticket),
		visible && focused,
		env.WorkbenchCanPrompt && !c.releases.ModalOpen,
		env.LauncherStartBusy,
		c.attention.tracker.Prompting(),
		env.Quitting,
	)
	if can {
		can = c.attention.tracker.BeginPrompt(ticket)
	}
	c.mu.Unlock()
	if obsolete {
		c.ClearAttentionNotice()
	}
	if remind {
		c.showAttentionNotice(win, ticket)
	}
	if !can {
		return
	}
	c.bridge.UpdateWindow(win, func() {
		vis, foc := c.bridge.WindowVisibility(win)
		alive := c.bridge.WindowAlive(win)
		curEnv := c.env()
		curOwned, curWinKey := c.windowContext(win)
		visibleFocused := alive && vis && foc
		c.mu.Lock()
		stillCan := visibleFocused && curEnv.WorkbenchCanPrompt && !c.releases.ModalOpen && c.attention.tracker.Prompting() && c.startupUpdateCurrentLocked(curEnv, curOwned, win, curWinKey, ticket)
		if !stillCan {
			c.attention.tracker.CancelPrompt(ticket)
		}
		rows := append([]update.NativeRelease(nil), snapshot.Rows...)
		c.mu.Unlock()
		if !stillCan {
			return
		}
		c.ClearAttentionNotice()
		c.bridge.AskConfirm(ConfirmRequest{
			Owner:            win,
			Width:            640,
			Title:            i18n.T(i18n.KeyCmdMygoUpdateAvailable),
			Body:             update.NativeReleasePrompt(curEnv.BuildVersion, release, rows, i18n.Locale() != i18n.LocaleEN),
			ConfirmLabel:     i18n.T(i18n.KeyCmdReviewVersionsAndDownloads),
			CancelLabel:      i18n.T(i18n.KeyCmdLater),
			TechnicalDetails: NativeReleaseDetail(release),
			Run:              func() { c.finishStartupUpdatePrompt(win, ticket, release, true) },
			Cancel:           func() { c.finishStartupUpdatePrompt(win, ticket, release, false) },
		})
	})
}

func (c *Center) finishStartupUpdatePrompt(win WindowRef, ticket startuprec.UpdateAttentionTicket, release update.NativeRelease, review bool) {
	if c == nil || c.bridge == nil || win == nil {
		return
	}
	vis, _ := c.bridge.WindowVisibility(win)
	visible := c.bridge.WindowAlive(win) && vis
	env := c.env()
	ownedWin, winKey := c.windowContext(win)

	c.mu.Lock()
	current := c.startupUpdateCurrentLocked(env, ownedWin, win, winKey, ticket)
	resolved := false
	if current {
		resolved = c.attention.tracker.Resolve(ticket)
	} else if c.attention.tracker.Current(ticket) {
		c.attention.tracker.Clear()
	}
	allowed := current && resolved && review && visible
	if allowed {
		c.releases.Selected = release.PackageKey()
		c.attention.hold = true
	}
	c.mu.Unlock()
	if allowed {
		c.OpenCenter(win)
	}
}
