package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/app/updater"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/launcher"
	"whaleisle/internal/ui/theme"
)

func updaterWindow(w *mygo.Window) updater.WindowRef {
	if w == nil {
		return nil
	}
	return w
}

func (s *mainState) ensureNativeUpdateCenter() { s.updaterCenter().EnsureLoaded() }
func (s *mainState) ensureNativeUpdateCenterAsync(w *mygo.Window) {
	s.updaterCenter().EnsureLoadedAsync(updaterWindow(w))
}
func (s *mainState) startNativeUpdateCenter(w *mygo.Window) {
	u := s.updaterCenter()
	go func() {
		binJS := filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")
		info, err := os.Stat(binJS)
		u.SetVendorReady(err == nil && !info.IsDir())
		u.StartBackground(updaterWindow(w))
	}()
}
func (s *mainState) stopNativeUpdateCenter() { s.updaterCenter().StopBackground() }
func (s *mainState) openNativeUpdateCenter(w *mygo.Window) {
	s.updaterCenter().OpenCenter(updaterWindow(w))
}
func (s *mainState) refreshNativeReleases(w *mygo.Window) {
	s.updaterCenter().Refresh(updaterWindow(w))
}
func (s *mainState) showUpdateCheck(w *mygo.Window) {
	s.updaterCenter().ShowUpdateCheck(updaterWindow(w))
}
func (s *mainState) markNativeInstallerLaunched(staged desktopupdate.ApplyResult) {
	s.updaterCenter().MarkInstallerLaunched(staged)
}
func (s *mainState) recordUpdateAction(action string) { s.updaterCenter().RecordAction(action) }
func (s *mainState) refreshStartupUpdateAttention(w *mygo.Window) {
	s.updaterCenter().RefreshAttention(updaterWindow(w))
}

func (s *mainState) withNativeReleases(win *mygo.Window, m launcher.Model) launcher.Model {
	u := s.updaterCenter()
	winRef := updaterWindow(win)
	snap := u.PrepareLauncherModel(winRef, m.VendorReady)
	m.Releases = snap.Rows
	m.RouteUnavailable = snap.RouteUnavailable
	m.ReleaseBusy = snap.Busy
	m.ReleaseStatus = snap.Status
	m.ReleaseError = snap.Err
	m.ReleaseSelectedPackage = snap.Selected
	m.ReleaseReceived, m.ReleaseTotal = snap.Received, snap.Total
	m.ReleasePhase, m.ReleasePaused, m.ReleaseAutoDownload = snap.Phase, snap.Paused, snap.AutoDownload
	m.ReleaseStagedTag = snap.StagedTitle
	if m.ReleaseStagedTag == "" {
		m.ReleaseStagedTag = snap.StagedTag
	}
	m.OnPauseRelease = func() { u.Pause(winRef) }
	m.OnResumeRelease = func() { u.Resume(winRef) }
	m.OnSetAutoDownload = func(enabled bool) { u.SetAutoDownload(winRef, enabled) }
	m.OnRefreshReleases = func() { u.Refresh(winRef) }
	m.OnCheckUpdate = func() { u.OpenCenter(winRef); u.Refresh(winRef) }
	m.OnInstallRuntime = func() { u.Download(winRef, "", true) }
	m.OnInstallRelease = func(packageID string) { u.Download(winRef, packageID, !m.VendorReady) }
	m.OnInstallUpdate = func() { u.Download(winRef, "", !m.VendorReady) }
	m.OnSelectRelease = func(packageID string) { u.SelectRelease(winRef, packageID) }
	m.OnCancelRelease = func() { u.Cancel(winRef) }
	m.OnDismissRelease = func() { u.DismissRelease(winRef) }
	m.OnStartStagedRelease = func() { u.StartStaged(winRef) }
	m.UpdateAvailable = snap.UpdateAvailable
	if snap.UpdateStatus != "" {
		m.UpdateStatus = snap.UpdateStatus
	} else if snap.UpdateAvailable {
		m.UpdateStatus = ""
	}
	return m
}

func (s *mainState) syncNativeUpdateModal(c *ui.Context, owner *mygo.Window) {
	u := s.updaterCenter()
	ownerRef := updaterWindow(owner)
	modal := u.ModalSnapshot()
	if !modal.Open || modal.Owner != ownerRef {
		return
	}
	s.mu.Lock()
	themeID := s.themeID
	s.mu.Unlock()
	open := true
	binJS := filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")
	vendorReady := launcherVendorProbe.get(binJS, time.Second, func() bool {
		info, err := os.Stat(binJS)
		return err == nil && !info.IsDir()
	}, s.launcherSurfaceInvalidator(owner))
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
		chrome.FrostSurface(panel.Width(min(600, max(0, w-48))).MaxHeight(max(0, h-48)).MinHeight(0).
			Border(1, chrome.Color(colors.Border)).Clip(), colors, dark, chrome.Color(colors.BG), 16).Children(func() {
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
		u.CloseModal(ownerRef)
	}
}

func (s *mainState) paintUpdateProgress(c *ui.Context, colors chrome.TitleBarColors) {
	u := s.updaterCenter()
	state := u.ProgressSnapshot()
	if state == nil || state.Title == "" {
		return
	}
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	chrome.DrawUpdateProgress(c, colors, true, chrome.UpdateProgress{
		Title:        state.Title,
		Detail:       state.Detail,
		Percent:      state.Percent,
		DismissLabel: i18n.T(i18n.KeyCmdDismiss),
	}, func() { u.ClearProgress(updaterWindow(win)) })
}

var _ = fmt.Sprint
var _ = updater.ProgressPhaseChecking
