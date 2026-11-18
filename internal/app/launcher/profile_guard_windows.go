//go:build windows

package launcherctl

import (
	"fmt"
	"path/filepath"

	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

// ClaimDesktopProfile acquires Chromium's Windows lockfile inside the profile
// root, falling back to the native sub-profile lock when running side-by-side
// with Electron whaleisle.
func ClaimDesktopProfile() (func(), error) {
	release, err := ClaimDesktopProfileAt(paths.UserDataDir())
	if err == nil {
		return release, nil
	}
	return ClaimDesktopProfileAt(filepath.Join(paths.UserDataDir(), "native"))
}

// ClaimDesktopProfileAt exclusively locks <profile>/lockfile without truncating it.
func ClaimDesktopProfileAt(profile string) (func(), error) {
	release, err := credentials.ClaimDesktopProfileWriter(profile)
	if err != nil {
		msg := "此共享数据目录正由另一个鲸屿 MyGo / 鲸屿桌面占用，或无法取得独占访问。请先关闭其他桌面实例，或使用独立数据目录。"
		if i18n.Locale() == i18n.LocaleEN {
			msg = "This shared profile is in use by another Whale Isle MyGo / Whale Isle desktop, or exclusive access is unavailable. Close the other desktop or choose an independent data directory."
		}
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return release, nil
}
