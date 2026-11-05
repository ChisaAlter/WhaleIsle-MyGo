package settings

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

func (g *WallpaperEditor) hasEffectiveWallpaper(p wallpapers.Preferences) bool {
	if g.EffectiveImage != nil {
		return strings.TrimSpace(g.EffectiveImage(p)) != ""
	}
	return g.ImageSource != nil && strings.TrimSpace(g.ImageSource()) != ""
}

// Immediate controls write only their selected fields. The source editor's
// unfinished name/URL remains a separate catalog draft.
func (g *WallpaperEditor) changePreference(change func(*wallpapers.Preferences)) bool {
	previous, next, draft := g.currentPreferences().Clone(), g.currentPreferences().Clone(), g.Prefs.Clone()
	change(&next)
	if next.Effect != previous.Effect {
		next.EffectExplicit = true
	}
	if next.Transparent && (!previous.Transparent || !g.hasEffectiveWallpaper(previous)) && g.hasEffectiveWallpaper(next) && next.Blur < 20 {
		next.Blur = 20
	}
	if g.Save == nil {
		g.Status = providerCopy("背景设置为只读。", "Background settings are read-only.")
		return false
	}
	if err := g.Save(next); err != nil {
		g.Status = err.Error()
		g.failure = g.Status
		return false
	}
	if g.Pending != nil && g.Pending() {
		pending := next.Clone()
		g.pendingPreferences = &pending
	} else {
		g.Saved = next.Clone()
		g.pendingPreferences = nil
	}
	g.Prefs = next.Clone()
	if !reflect.DeepEqual(draft.Sources, previous.Sources) {
		g.Prefs.Sources = draft.Sources
	}
	g.Status = ""
	return true
}

func wallpaperChange(c *ui.Context, g *WallpaperEditor, change func(*wallpapers.Preferences)) bool {
	if !g.changePreference(change) {
		c.AddToast(ui.Toast{ID: "appearance-preference", Title: g.Status, Type: "error"})
		return false
	}
	return true
}

func wallpaperColorsValid(c *ui.Context, colors []string) bool {
	for _, color := range colors {
		if color != "" && !effectHex(color) {
			c.AddToast(ui.Toast{ID: "appearance-color", Title: providerCopy("颜色请使用 #RRGGBB，或留空继承主题。", "Use #RRGGBB colors, or leave blank to inherit the theme."), Type: "error"})
			return false
		}
	}
	return true
}

// wallpaperImmediateSlider paints the reference's block slider (.field): a
// label row carrying the current percent, then a full-width slider that
// persists only its own field.
func wallpaperImmediateSlider(c *ui.Context, g *WallpaperEditor, key, label string, value, lo, hi, step float64, disabled bool, set func(*wallpapers.Preferences, float64), fg, muted ui.Color) {
	block := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(6)
	settingsSearchTarget(c, key, label, block)
	block.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted)
			ui.Text(c, fmt.Sprintf("%.0f%%", value)).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontFeatures("tnum").TextColor(muted)
		})
		if ui.StepSlider(c, &value, lo, hi, step).FillWidth().Label(label).Disabled(g.Save == nil || disabled).Changed() {
			wallpaperChange(c, g, func(p *wallpapers.Preferences) { set(p, value) })
		}
	})
}
