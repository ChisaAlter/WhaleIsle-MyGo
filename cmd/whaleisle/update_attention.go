package main

import (
	"fmt"
	"runtime"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/startuprec"
)

// The caller holds s.mu. Query the live supervisor as well as the catalog epoch:
// neither a queued UI callback nor an old native notification can adopt a restart.
func (s *mainState) startupUpdateScopeLocked(win *mygo.Window, snapshot nativeReleaseState, vendorReady bool) (startuprec.UpdateAttentionScope, update.NativeRelease) {
	empty := startuprec.UpdateAttentionScope{}
	route := s.launcherDownloadRoute
	if route == "" && s.cfgStore != nil {
		route = s.cfgStore.Get().DownloadRoute
	}
	current := &s.nativeReleases
	if win == nil || s.win != win || s.quitting || s.launcherStartBusy || s.launcherImportBusy || snapshot.busy || snapshot.err != "" || snapshot.catalogRoute != snapshot.route || route != snapshot.route || current.busy || current.err != "" || current.route != snapshot.route || current.catalogRoute != snapshot.catalogRoute || current.attentionGeneration != snapshot.attentionGeneration {
		return empty, update.NativeRelease{}
	}
	for _, release := range snapshot.rows {
		if release.Prerelease || release.Validate(runtime.GOOS, runtime.GOARCH, !vendorReady) != nil || update.CompareVersions(release.Version, buildVersion) <= 0 {
			continue
		}
		if snapshot.stagedTag == release.Tag {
			return empty, update.NativeRelease{}
		}
		generation := s.bootStatus.Generation
		if s.sup != nil {
			generation = s.sup.Status().Generation
		}
		return startuprec.UpdateAttentionScope{Window: fmt.Sprintf("%p", win), Host: fmt.Sprintf("%p", s.sup), HostGeneration: generation, CatalogGeneration: snapshot.attentionGeneration, Route: snapshot.catalogRoute, Tag: release.Tag, Version: release.Version}, release
	}
	return empty, update.NativeRelease{}
}

func (s *mainState) startupUpdateCurrentLocked(win *mygo.Window, ticket startuprec.UpdateAttentionTicket) bool {
	r := &s.startupRecovery
	scope, _ := s.startupUpdateScopeLocked(win, s.nativeReleases, r.updateVendorReady)
	return r.updateAttention.Current(ticket) && ticket.Scope == scope && s.cfgStore != nil && s.cfgStore.Get().AskOnUpdate
}

// Called through Window.Update even when the launcher is hidden. This reads the
// existing catalog only; it never schedules a second check or delays startup.
func (s *mainState) refreshStartupUpdateAttention(win *mygo.Window) {
	s.mu.Lock()
	snapshot := s.nativeReleases
	snapshot.rows = append([]update.NativeRelease(nil), snapshot.rows...)
	vendorReady := s.startupRecovery.updateVendorReady
	s.mu.Unlock()
	s.maybeAskStartupUpdate(win, snapshot, vendorReady)
}

// Native reminder disposal does not consume the visible pending confirmation.
func (s *mainState) clearStartupUpdateNotice() {
	s.mu.Lock()
	r := &s.startupRecovery
	n, win := r.updateNotice, r.updateNoticeWindow
	r.updateNotice, r.updateNoticeWindow = nil, nil
	r.updateNoticeTicket = startuprec.UpdateAttentionTicket{}
	s.mu.Unlock()
	if n != nil {
		n.Close()
	}
	if win != nil && !win.IsDestroyed() {
		win.FlashFrame(false)
	}
}

func (s *mainState) watchStartupUpdateWindow(win *mygo.Window) {
	s.mu.Lock()
	r := &s.startupRecovery
	if r.updateWatchWindow == win {
		s.mu.Unlock()
		return
	}
	off := r.updateWatchOff
	r.updateWatchOff = nil
	r.updateWatchWindow = win
	r.updateAttention.Clear()
	s.mu.Unlock()
	for _, stop := range off {
		stop()
	}
	s.clearStartupUpdateNotice()
	focus := win.OnFocus(func() {
		s.clearStartupUpdateNotice()
		s.nativeReleaseNotify(win)
	})
	closed := win.OnClosed(func() {
		s.mu.Lock()
		r := &s.startupRecovery
		current := r.updateWatchWindow == win
		if current {
			r.updateAttention.Clear()
			r.updateWatchWindow = nil
			r.updateWatchOff = nil
		}
		s.mu.Unlock()
		if current {
			s.clearStartupUpdateNotice()
		}
	})
	s.mu.Lock()
	s.startupRecovery.updateWatchOff = []func(){focus, closed}
	s.mu.Unlock()
}

func (s *mainState) showStartupUpdateNotice(win *mygo.Window, ticket startuprec.UpdateAttentionTicket) {
	s.mu.Lock()
	if !s.startupUpdateCurrentLocked(win, ticket) {
		s.mu.Unlock()
		return
	}
	r := &s.startupRecovery
	r.updateNoticeWindow, r.updateNoticeTicket = win, ticket
	s.mu.Unlock()
	win.FlashFrame(true)
	// A failed/unsupported system notification still leaves the launcher notice
	// and focus-to-confirm path intact, with no repeated native notification.
	if !mygo.NotificationsSupported() {
		return
	}
	n := mygo.NewNotification(mygo.NotificationOptions{
		Title: resumeText("MyGo 更新待确认", "MyGo update awaiting confirmation"),
		Body:  fmt.Sprintf(resumeText("%s 已可用。点击返回更新确认；下载和安装仍需明确确认。", "%s is available. Return to confirmation; downloading and installation still need explicit confirmation."), ticket.Scope.Version), Silent: true,
	})
	s.mu.Lock()
	if !s.startupUpdateCurrentLocked(win, ticket) || s.startupRecovery.updateNoticeTicket != ticket {
		s.mu.Unlock()
		return
	}
	s.startupRecovery.updateNotice = n
	s.mu.Unlock()
	n.OnClick(func() {
		// Check the exact notification identity before clearing it: a delayed
		// click on a replaced notification cannot open any newer confirmation.
		s.mu.Lock()
		valid := s.startupRecovery.updateNotice == n && s.startupRecovery.updateNoticeTicket == ticket && s.startupUpdateCurrentLocked(win, ticket)
		s.mu.Unlock()
		if !valid {
			return
		}
		s.clearStartupUpdateNotice()
		// This is navigation only. maybeAskStartupUpdate owns the actual dialog;
		// downloadNativeRelease is never called by a notification callback.
		s.openDesktopTrayPage(deeplink.ActionLauncher)
		s.nativeReleaseNotify(win)
	})
	err := n.Show()
	s.mu.Lock()
	current := s.startupRecovery.updateNotice == n && s.startupUpdateCurrentLocked(win, ticket)
	s.mu.Unlock()
	if !current {
		// Show may finish after a focus/restart cleared the owned reminder.
		n.Close()
	} else if err != nil {
		s.clearStartupUpdateNotice()
	}
}

// Foreground confirmation consumes the same pinned release as the reminder.
// Neither background discovery nor clicking a notification authorizes download.
func (s *mainState) maybeAskStartupUpdate(win *mygo.Window, snapshot nativeReleaseState, vendorReady bool) {
	if win == nil || win.IsDestroyed() {
		return
	}
	s.mu.Lock()
	currentWindow := s.win == win && !s.quitting
	clearOwner := !currentWindow && s.startupRecovery.updateWatchWindow == win
	if clearOwner {
		s.startupRecovery.updateAttention.Clear()
	}
	s.mu.Unlock()
	if !currentWindow {
		if clearOwner {
			s.clearStartupUpdateNotice()
		}
		return
	}
	s.watchStartupUpdateWindow(win)
	visible := win.IsVisible() && !win.IsMinimized()
	focused := visible && win.IsFocused()
	s.mu.Lock()
	r := &s.startupRecovery
	r.updateVendorReady = vendorReady
	scope, release := s.startupUpdateScopeLocked(win, snapshot, vendorReady)
	ask := s.cfgStore != nil && s.cfgStore.Get().AskOnUpdate
	ticket, remind := r.updateAttention.Observe(scope, ask, focused)
	obsolete := r.updateNoticeWindow != nil && r.updateNoticeTicket != ticket
	can := startuprec.CanPromptUpdate(ask, r.updateAttention.Current(ticket), focused, s.launcherOpen, s.launcherStartBusy, r.updateAttention.Prompting(), s.quitting)
	if can {
		can = r.updateAttention.BeginPrompt(ticket)
	}
	s.mu.Unlock()
	if obsolete {
		s.clearStartupUpdateNotice()
	}
	if remind {
		s.showStartupUpdateNotice(win, ticket)
	}
	if !can {
		return
	}
	win.Update(func() {
		visible := !win.IsDestroyed() && win.IsVisible() && !win.IsMinimized() && win.IsFocused()
		s.mu.Lock()
		r := &s.startupRecovery
		can := visible && s.launcherOpen && r.updateAttention.Prompting() && s.startupUpdateCurrentLocked(win, ticket)
		if !can {
			r.updateAttention.CancelPrompt(ticket)
		}
		s.mu.Unlock()
		if !can {
			return
		}
		s.clearStartupUpdateNotice()
		choice, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageInfo,
			Title:   resumeText("MyGo 更新可用", "MyGo update available"),
			Message: fmt.Sprintf(resumeText("发现 %s。现在下载并校验该版本？安装仍需单独确认，可稍后重试。", "%s is available. Download and verify this version now? Installation needs a separate confirmation; you can retry later."), release.Version),
			Buttons: []string{resumeText("稍后", "Later"), resumeText("下载更新", "Download update")}, DefaultButton: 0, CancelButton: 0})
		visible = !win.IsDestroyed() && win.IsVisible() && !win.IsMinimized()
		s.mu.Lock()
		r = &s.startupRecovery
		current := s.startupUpdateCurrentLocked(win, ticket)
		resolved := false
		if current {
			resolved = r.updateAttention.Resolve(ticket)
		} else if r.updateAttention.Current(ticket) {
			r.updateAttention.Clear()
		}
		allowed := current && resolved && err == nil && choice.Button == 1 && visible && s.launcherOpen
		if err != nil && current {
			s.launcherNote = startuprec.RedactLine(err.Error())
		}
		s.mu.Unlock()
		if allowed {
			s.downloadNativeRelease(win, release.Tag, !vendorReady, ticket.Scope)
		}
	})
}
