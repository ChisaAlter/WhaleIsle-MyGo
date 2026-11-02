package main

import (
	"runtime"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Windows' Acrylic samples the live desktop in DWM. It is independent of the
// wallpaper sampler, which continues to own image and gradient materials.
// The existing main window is already a transparent DirectComposition window;
// switching its system backdrop needs no recreation or polling loop.
var desktopGlassWindows sync.Map // *mygo.Window -> mygo.Vibrancy

func desktopGlassMaterial(path string, p wallpapers.Preferences, dark, highContrast bool) mygo.Vibrancy {
	p = wallpapers.NormalizeEffect(path, p)
	if runtime.GOOS == "windows" && path == "" && p.Effect != "gradient" && dark && !highContrast {
		return mygo.VibrancyAcrylic
	}
	return mygo.VibrancyNone
}

func (s *mainState) desktopGlassPainter(path string, prefs wallpapers.Preferences, draw func(*ui.Painter, ui.Rect)) func(*ui.Painter, ui.Rect) {
	if runtime.GOOS != "windows" {
		return draw
	}
	flatChrome := s.activeSkin() == theme.SkinCodex
	return func(p *ui.Painter, r ui.Rect) {
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		if win != nil {
			// Theme changes rebuild the view; high contrast is also delivered by
			// the native preference event. Only transitions call SetVibrancy.
			want := desktopGlassMaterial(path, prefs, p.Theme().Dark, false)
			if flatChrome {
				want = mygo.VibrancyNone
			}
			old, known := desktopGlassWindows.Load(win)
			if !known || old != want {
				desktopGlassWindows.Store(win, want)
				if !known {
					win.OnClosed(func() { desktopGlassWindows.Delete(win) })
				}
				win.SetVibrancy(want)
			}
		}
		draw(p, r)
		if !flatChrome && desktopGlassMaterial(path, prefs, p.Theme().Dark, false) != mygo.VibrancyNone {
			chrome.DesktopGlassVignette(p, r, prefs)
		}
	}
}
