package updater

import (
	"fmt"
	"strings"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

// Progress phases mirror the host shell update indicator: checking,
// downloading, then the staged hand-off that leaves installation with the user.
const (
	ProgressPhaseChecking    = "checking"
	ProgressPhaseDownloading = "downloading"
	ProgressPhaseStaged      = "staged"
)

// ProgressState is the shell-owned view of one update operation. Percent is
// 0..100; a negative value indicates the content length is unknown.
type ProgressState struct {
	Phase   string
	Title   string
	Detail  string
	Percent float64
}

// ProgressSnapshot returns a shallow copy of the current progress bar state.
func (c *Center) ProgressSnapshot() *ProgressState {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.progress == nil {
		return nil
	}
	cp := *c.progress
	return &cp
}

// SetProgress replaces the visible update progress row and triggers a window update.
func (c *Center) SetProgress(win WindowRef, state *ProgressState) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if state != nil {
		cp := *state
		c.progress = &cp
	} else {
		c.progress = nil
	}
	c.mu.Unlock()
	if c.bridge != nil && win != nil && c.bridge.WindowAlive(win) {
		c.bridge.UpdateWindow(win, func() {})
	}
}

// ClearProgress removes the progress row.
func (c *Center) ClearProgress(win WindowRef) {
	c.SetProgress(win, nil)
}

func (c *Center) setLauncherNotice(text string) {
	text = strings.TrimSpace(text)
	if c == nil || text == "" || c.bridge == nil {
		return
	}
	c.bridge.SetLauncherUpdateNotice(text)
}

// DownloadDetail keeps the download progress evidence short and locale-stable.
func DownloadDetail(received, total int64) string {
	switch {
	case total > 0:
		return FormatBytes(received) + " / " + FormatBytes(total)
	case received > 0:
		return FormatBytes(received)
	default:
		return ""
	}
}

// DownloadPercent reports 0..100 progress, or a negative value when total <= 0.
func DownloadPercent(received, total int64) float64 {
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

// FormatBytes renders byte counts without locale-specific digit grouping.
func FormatBytes(n int64) string {
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

// AvailableDetail formats bounded technical metadata for an available feed update.
func AvailableDetail(res desktopupdate.Result, route string) string {
	var lines []string
	if res.Version != "" {
		lines = append(lines, i18n.T(i18n.KeyCmdNewVersion)+res.Version)
	}
	if route = strings.ToLower(strings.TrimSpace(route)); route != "" {
		lines = append(lines, i18n.T(i18n.KeyCmdDownloadRoute)+route)
	}
	if res.Size > 0 {
		lines = append(lines, i18n.T(i18n.KeyCmdInstallerSize)+FormatBytes(res.Size))
	}
	if res.SHA256 != "" {
		lines = append(lines, "SHA-256: "+res.SHA256)
	}
	if res.IncrementalPossible {
		lines = append(lines, i18n.T(i18n.KeyCmdAnIncrementalPatchPathIsAvailable))
	}
	return strings.Join(lines, "\n")
}

// NativeReleaseDetail formats bounded technical metadata for a native release prompt.
func NativeReleaseDetail(release desktopupdate.NativeRelease) string {
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
		lines = append(lines, i18n.T(i18n.KeyCmdInstallerSize)+FormatBytes(release.Size))
	}
	if release.SHA256 != "" {
		lines = append(lines, "SHA-256: "+release.SHA256)
	}
	return strings.Join(lines, "\n")
}
