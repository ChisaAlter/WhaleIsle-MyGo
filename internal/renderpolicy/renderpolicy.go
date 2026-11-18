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

// Wallpaper reports the backdrop image the chrome paints. Minimal mode paints
// none, including a wallpaper the active theme family would supply.
func Wallpaper(path string) string {
	if Minimal() {
		return ""
	}
	return path
}

// Background returns the preferences the window chrome paints with. Under
// minimal mode the effects it pauses are neutralised in the copy: no ambient
// gradient, no family wallpaper, no transparent theme, fully solid materials
// and no pointer decoration. The stored preferences are left untouched.
//
// A caller keeps the user's own preferences for anything the user edits — the
// appearance page shows and saves those — and paints the backdrop through this
// projection only.
func Background(p wallpapers.Preferences) wallpapers.Preferences {
	if !Minimal() {
		return p
	}
	p.Effect, p.EffectExplicit = "none", true
	p.Preset = ""
	p.SkinWallpaperEnabled = false
	p.Transparent = false
	p.Glass, p.Terminal = 100, 100
	p.CursorEnabled = false
	return p
}
