package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"whaleisle/internal/paths"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/launcher"
	"whaleisle/internal/ui/theme"
)

// The update center is an overlay of the window that opened it, not navigation
// to the launcher. Download state survives closing and reopening the overlay.
func (s *mainState) syncNativeUpdateModal(c *ui.Context, owner *mygo.Window) {
	s.mu.Lock()
	r := s.nativeReleases
	themeID := s.themeID
	vendorReady := s.startupRecovery.updateVendorReady
	s.mu.Unlock()
	if !r.modalOpen || r.modalOwner != owner {
		return
	}
	open := true
	binJS := filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")
	vendorReady = launcherVendorProbe.get(binJS, time.Second, func() bool { info, err := os.Stat(binJS); return err == nil && !info.IsDir() }, s.launcherSurfaceInvalidator(owner))
	resolved := theme.ResolveWithSystem(themeID, mygo.Theme.IsDark())
	dark := resolved.Scheme == theme.SchemeDark
	colors := resolved.Active(dark)
	m := s.withNativeReleases(owner, launcher.Model{Version: buildVersion, VendorReady: vendorReady, SystemDark: dark})
	ui.DialogBase(c.Key("native-update-center"), &open, func(backdrop, panel ui.Element) {
		mask := ui.RGBA(0, 0, 0, .24)
		if dark {
			mask = ui.RGBA(0, 0, 0, .5)
		}
		chrome.RevealOverlay(c, backdrop, panel, mask)
		w, h := c.Size()
		for _, release := range m.Releases {
			if release.Notes != "" || release.NotesZH != "" || release.NotesEN != "" {
				panel.Height(min(680, max(0, h-48)))
				break
			}
		}
		panel.Width(min(600, max(0, w-48))).MaxHeight(max(0, h-48)).MinHeight(0).
			Background(chrome.Color(colors.BG)).Border(1, chrome.Color(colors.Border)).Radius(16).Clip().Children(func() {
			ui.Row(c).FillWidth().Shrink(0).Padding(20, 24).AlignItems(ui.Center).Gap(12).Children(func() {
				ui.Text(c, lifecycleText("更新中心", "Update center")).FontSize(18).FontWeight(600).TextColor(chrome.Color(colors.FG)).Grow(1)
				if ui.Button(c, "×").Size(28, 28).Label(lifecycleText("关闭", "Close")).AutoFocus().Clicked() {
					open = false
				}
			})
			ui.Scroll(c).FillWidth().MinHeight(0).Shrink(1).Children(func() {
				ui.Column(c).FillWidth().Padding(0, 24, 20, 24).Gap(12).Children(func() {
					launcher.UpdateCenterBody(c, m)
				})
			})
			ui.Column(c).FillWidth().Shrink(0).Padding(0, 24, 20, 24).Gap(8).Children(func() { launcher.UpdateCenterFooter(c, m) })
		})
	})
	if !open {
		s.closeNativeUpdateModal(owner)
	}
}

// Publish one ready transition per exact verified package. Clicking the native
// reminder opens the center; it never installs or authorizes task interruption.
func (s *mainState) notifyNativeUpdateReady(owner *mygo.Window) {
	route := s.updateDownloadRoute()
	s.mu.Lock()
	r := &s.nativeReleases
	key := r.active.PackageKey() + "|" + r.staged.SHA256
	ready := r.phase == "ready" && r.staged.Path != "" && !r.busy && !s.quitting && r.route == route
	if !ready {
		s.mu.Unlock()
		s.clearNativeUpdateReadyNotice()
		return
	}
	if r.readyNotified == key {
		s.mu.Unlock()
		return
	}
	r.readyNotified = key
	generation, stagedPath := r.attentionGeneration, r.staged.Path
	version := r.stagedTitle
	if version == "" {
		version = r.stagedTag
	}
	s.mu.Unlock()
	s.clearNativeUpdateReadyNotice()
	n := mygo.NewNotification(mygo.NotificationOptions{Title: lifecycleText("更新已就绪", "Update ready"), Body: fmt.Sprintf(lifecycleText("%s 已下载并校验，打开更新中心选择安装。", "%s is downloaded and verified. Open the update center to install."), version), Silent: true})
	s.mu.Lock()
	s.nativeReleases.readyNotice = n
	s.mu.Unlock()
	if owner != nil {
		off := owner.OnClosed(func() { s.clearNativeUpdateReadyNotice() })
		s.mu.Lock()
		s.nativeReleases.readyNoticeOff = off
		s.mu.Unlock()
	}
	n.OnClick(func() {
		currentRoute := s.updateDownloadRoute()
		s.mu.Lock()
		r := &s.nativeReleases
		valid := r.readyNotice == n && r.attentionGeneration == generation && r.route == route && currentRoute == route && r.active.PackageKey()+"|"+r.staged.SHA256 == key && r.staged.Path == stagedPath && r.phase == "ready" && !r.busy && !s.quitting && s.lifecycleWindowOwnedLocked(owner)
		s.mu.Unlock()
		if !valid || owner == nil || owner.IsDestroyed() {
			return
		}
		s.clearNativeUpdateReadyNotice()
		owner.Restore()
		owner.Show()
		owner.Focus()
		s.openNativeUpdateCenter(owner)
	})
	if systemNotificationsSupported() {
		_ = n.Show()
	}
}

func (s *mainState) clearNativeUpdateReadyNotice() {
	s.mu.Lock()
	n, off := s.nativeReleases.readyNotice, s.nativeReleases.readyNoticeOff
	s.nativeReleases.readyNotice, s.nativeReleases.readyNoticeOff = nil, nil
	s.mu.Unlock()
	if off != nil {
		off()
	}
	if n != nil {
		n.Close()
	}
}

// URL-backed component windows cannot paint Content overlays. Their center is
// hosted by an owned native modal; closing either surface retires that seat.
func (s *mainState) showNativeUpdateModal(parent *mygo.Window) {
	s.mu.Lock()
	current := s.nativeReleases
	s.mu.Unlock()
	if current.modalOpen && current.modalParent == parent && current.modalOwner != nil && !current.modalOwner.IsDestroyed() {
		current.modalOwner.Focus()
		return
	}
	s.closeNativeUpdateModal(nil)
	owner := parent
	temporary := parent.Page() != nil
	if temporary {
		var dialog *mygo.Window
		dialog = mygo.NewWindow(mygo.WindowOptions{Parent: parent, Modal: true, Title: lifecycleText("更新中心", "Update center"), Width: 648, Height: 720, MinWidth: 300, MinHeight: 260, Frameless: true, Transparent: true, AutoHideMenuBar: true, BackgroundColor: "#00000000",
			Content: ui.View(func(c *ui.Context) {
				ui.Box(c).Fill().Background(ui.Transparent)
				s.syncNativeUpdateModal(c, dialog)
				s.syncConfirmAskFor(c, dialog)
			}),
		})
		owner = dialog
	}
	if owner == nil || owner.IsDestroyed() {
		return
	}
	s.mu.Lock()
	r := &s.nativeReleases
	r.modalOwner, r.modalParent, r.modalOpen, r.modalTemporary = owner, parent, true, temporary
	s.startupRecovery.updateAttention.DismissCurrent()
	s.mu.Unlock()
	offClose := parent.OnClosed(func() { s.closeNativeUpdateModal(owner) })
	offHide := parent.OnHide(func() { s.closeNativeUpdateModal(owner) })
	offOwner := owner.OnClosed(func() { s.closeNativeUpdateModal(owner) })
	s.mu.Lock()
	s.nativeReleases.modalOff = func() { offClose(); offHide(); offOwner() }
	s.mu.Unlock()
	owner.Invalidate()
}

func (s *mainState) closeNativeUpdateModal(expected *mygo.Window) {
	s.mu.Lock()
	r := &s.nativeReleases
	if expected != nil && r.modalOwner != expected {
		s.mu.Unlock()
		return
	}
	owner, temporary, off := r.modalOwner, r.modalTemporary, r.modalOff
	r.modalOwner, r.modalParent, r.modalOpen, r.modalTemporary, r.modalOff = nil, nil, false, false, nil
	s.mu.Unlock()
	if off != nil {
		off()
	}
	if temporary && owner != nil && !owner.IsDestroyed() {
		owner.Destroy()
	}
	if owner != nil && !owner.IsDestroyed() {
		owner.Invalidate()
	}
}
