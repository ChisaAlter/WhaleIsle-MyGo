// Package renderpolicy is the appearance page's 极简模式 (minimal mode) at
// render time: one switch that pauses the effects the desktop draws —
// wallpapers, the ambient background, transparent materials, pointer and typing
// decorations, the composer beam and interface transitions.
//
// It is a policy over the stored preferences, never a rewrite of them. The
// window chrome asks Wallpaper and Background what to paint, the composer asks
// Minimal whether to run its decorations, and every preference the user chose
// stays exactly as it was, so turning the switch off restores the stored look.
package renderpolicy

import (
	"sync/atomic"

	"whaleisle/internal/desktop/wallpapers"
)

// minimal is the switch for this process. Paint and view code on the UI thread
// read it on every frame and must not wait on a lock, so it is an atomic rather
// than a field of the application state.
var minimal atomic.Bool

// Minimal reports whether minimal mode is on.
func Minimal() bool { return minimal.Load() }

// SetMinimal records the switch and reports whether it changed.
func SetMinimal(on bool) bool { return minimal.Swap(on) != on }

// Wallpaper reports the backdrop image the chrome paints. Minimal mode and the
// glass theme being off paint none, including a family wallpaper.
func Wallpaper(path string) string {
	if NoBackdrop() {
		return ""
	}
	return path
}

// Background returns the preferences the window chrome paints with. Without a
// backdrop (minimal mode, or the glass theme off) the copy has no ambient
// gradient, no family wallpaper, no transparent theme and fully solid
// materials; minimal mode also drops the pointer decoration. The stored preferences are left untouched.
//
// A caller keeps the user's own preferences for anything the user edits — the
// appearance page shows and saves those — and paints the backdrop through this
// projection only.
func Background(p wallpapers.Preferences) wallpapers.Preferences {
	if NoBackdrop() {
		p.Effect, p.EffectExplicit = "none", true
		p.Preset = ""
		p.SkinWallpaperEnabled = false
		p.Transparent = false
		p.Glass, p.Terminal = 100, 100
	}
	if Minimal() {
		p.CursorEnabled = false
	}
	return p
}

// glass is the optional wallpaper-and-frosted-glass appearance theme. It is
// off by default: the window paints flat theme surfaces and none of the
// wallpaper, ambient-gradient or glass machinery runs.
var glass atomic.Bool

// Glass reports whether the wallpaper and frosted-glass theme is on.
func Glass() bool { return glass.Load() }

// SetGlass records the theme switch and reports whether it changed.
func SetGlass(on bool) bool { return glass.Swap(on) != on }

// NoBackdrop reports that the window paints no backdrop at all: minimal mode,
// or the glass theme being off. Callers then configure no wallpaper, no
// gradient and no translucent or desktop-sampling material.
func NoBackdrop() bool { return Minimal() || !Glass() }
