package theme

import "strings"

// NormalizeWallpaperPath trims a local image path from config (goWallpaperPath).
// Empty or whitespace-only values mean no wallpaper. Electron keeps wallpaper
// in ui-theme host settings; the Go shell stores a single optional file path
// here until gallery/crop parity lands.
func NormalizeWallpaperPath(path string) string {
	return strings.TrimSpace(path)
}

// WindowWallpaperPath returns the path to use as a window backdrop when set.
//
// mygo.WindowOptions (v0.3.5) exposes BackgroundColor only — there is no
// BackgroundImage field yet. When a path is returned, cmd/whaleisle paints
// it in the root view via chrome.WallpaperBackdrop; BackgroundColor still
// fills the window before the first frame.
func WindowWallpaperPath(cfgPath string) (path string, ok bool) {
	path = NormalizeWallpaperPath(cfgPath)
	return path, path != ""
}
