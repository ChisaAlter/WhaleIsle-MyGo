package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestAppearanceFrameMapsNativeInteractiveTokens(t *testing.T) {
	previous := theme.CurrentPreferences()
	defer theme.SetPreferences(previous)
	custom := theme.Defaults()
	custom.Interface.Family = "Microsoft YaHei"
	theme.SetPreferences(custom)
	for _, id := range []string{theme.IDDeepseekLight, theme.IDDeepseekDark} {
		t.Run(id, func(t *testing.T) {
			colors := theme.Resolve(id).Active(id == theme.IDDeepseekDark)
			var native ui.Theme
			ui.NewTester(func(c *ui.Context) {
				applyAppearanceTheme(c, custom, colors, false)
				native = *c.Theme()
			}, 100, 100)
			if native.SurfacePressed != chrome.Color(colors.Pressed) || native.AccentHover != chrome.Color(colors.ButtonInfoHover) || native.AccentPressed != native.AccentHover {
				t.Fatalf("native interaction tokens did not follow %s: %+v", id, native)
			}
			if native.Font != theme.TextFamily(custom.Interface.Family) {
				t.Fatalf("custom font overwritten: %q", native.Font)
			}
			if native.ScrollbarWidth != 5 {
				t.Fatalf("native scrollbar width=%v, source requires5 DIP", native.ScrollbarWidth)
			}
		})
	}
}
