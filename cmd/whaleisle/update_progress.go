package main

import (
	"fmt"
	"strings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/chrome"
)

// updateProgressPhases mirror the reference shell update surface: checking,
// downloading, then the staged hand-off that keeps the install decision with the
// user (reference/electron/src/main/update-dialog.js).
const (
	updatePhaseChecking    = "checking"
	updatePhaseDownloading = "downloading"
	updatePhaseStaged      = "staged"
)

// updateProgressState is the shell-owned view of one update operation. Percent
// is 0..100; a negative value means the length is unknown.
type updateProgressState struct {
	Phase   string
	Title   string
	Detail  string
	Percent float64
}

// setUpdateProgress replaces the visible update state and repaints.
func (s *mainState) setUpdateProgress(win *mygo.Window, state *updateProgressState) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.updateProgress = state
	s.mu.Unlock()
	if win != nil && !win.IsDestroyed() {
		win.Update(func() {})
	}
}

// clearUpdateProgress removes the row; the settled result stays in its dialog.
func (s *mainState) clearUpdateProgress(win *mygo.Window) {
	s.setUpdateProgress(win, nil)
}

// setLauncherUpdate keeps the launcher's existing notice bar in step with the
// update operation, so the product-level status and the row agree.
func (s *mainState) setLauncherUpdate(text string) {
	text = strings.TrimSpace(text)
	if s == nil || text == "" {
		return
	}
	s.mu.Lock()
	s.launcherUpdate = text
	s.mu.Unlock()
}

// paintUpdateProgress draws the row in the title-bar area, like the Git
// progress row, so the update phases never own a separate view.
func (s *mainState) paintUpdateProgress(c *ui.Context, colors chrome.TitleBarColors) {
	s.mu.Lock()
	state := s.updateProgress
	win := s.win
	s.mu.Unlock()
	if state == nil || strings.TrimSpace(state.Title) == "" {
		return
	}
	chrome.DrawUpdateProgress(c, colors, true, chrome.UpdateProgress{
		Title:        state.Title,
		Detail:       state.Detail,
		Percent:      state.Percent,
		DismissLabel: i18n.T(i18n.KeyCmdDismiss),
	}, func() { s.clearUpdateProgress(win) })
}

// updateDownloadDetail keeps the download evidence short and locale-stable.
func updateDownloadDetail(received, total int64) string {
	switch {
	case total > 0:
		return formatUpdateBytes(received) + " / " + formatUpdateBytes(total)
	case received > 0:
		return formatUpdateBytes(received)
	default:
		return ""
	}
}

// updateDownloadPercent reports 0..100 progress, or a negative value when the
// expected size is unknown.
func updateDownloadPercent(received, total int64) float64 {
	if total <= 0 {
		return -1
	}
	percent := float64(received) / float64(total) * 100
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

// formatUpdateBytes renders whole-unit sizes without locale-specific decimals.
func formatUpdateBytes(n int64) string {
	const (
		kib = 1024
		mib = kib * 1024
		gib = mib * 1024
	)
	switch {
	case n >= gib:
		return fmt.Sprintf("%.2f GB", float64(n)/gib)
	case n >= mib:
		return fmt.Sprintf("%.1f MB", float64(n)/mib)
	case n >= kib:
		return fmt.Sprintf("%.0f KB", float64(n)/kib)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// updateAvailableDetail is the bounded technical evidence attached to the
// update confirmation (port of update-dialog.js technicalDetails): version,
// download route, expected size and checksum, never the download URL.
func updateAvailableDetail(res desktopupdate.Result, route string) string {
	var lines []string
	if res.Version != "" {
		lines = append(lines, i18n.T(i18n.KeyCmdNewVersion)+res.Version)
	}
	if route = strings.ToLower(strings.TrimSpace(route)); route != "" {
		lines = append(lines, i18n.T(i18n.KeyCmdDownloadRoute)+route)
	}
	if res.Size > 0 {
		lines = append(lines, i18n.T(i18n.KeyCmdInstallerSize)+formatUpdateBytes(res.Size))
	}
	if res.SHA256 != "" {
		lines = append(lines, "SHA-256: "+res.SHA256)
	}
	if res.IncrementalPossible {
		lines = append(lines, i18n.T(i18n.KeyCmdAnIncrementalPatchPathIsAvailable))
	}
	return strings.Join(lines, "\n")
}

// nativeReleaseDetail is the bounded evidence attached to a native release
// prompt (port of the update dialog's technicalDetails): identity, platform,
// route, size and checksum, never a download URL.
func nativeReleaseDetail(release desktopupdate.NativeRelease) string {
	lines := []string{
		i18n.T(i18n.KeyCmdVersion) + release.Version,
		"Tag: " + release.Tag,
		i18n.T(i18n.KeyCmdPlatform) + release.Platform + "/" + release.Arch,
	}
	if release.DownloadRoute != "" {
		lines = append(lines, i18n.T(i18n.KeyCmdDownloadRoute)+strings.ToLower(release.DownloadRoute))
	}
	if release.RuntimeIncluded {
		lines = append(lines, i18n.T(i18n.KeyCmdIncludesTheRuntime))
	}
	if release.Size > 0 {
		lines = append(lines, i18n.T(i18n.KeyCmdInstallerSize)+formatUpdateBytes(release.Size))
	}
	if release.SHA256 != "" {
		lines = append(lines, "SHA-256: "+release.SHA256)
	}
	return strings.Join(lines, "\n")
}
