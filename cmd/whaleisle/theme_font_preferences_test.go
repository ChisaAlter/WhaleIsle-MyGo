package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
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
			dark := id == theme.IDDeepseekDark
			colors := theme.Resolve(id).Active(dark)
			var native ui.Theme
			ui.NewTester(func(c *ui.Context) {
				applyAppearanceTheme(c, custom, colors, dark)
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
			scheme := design.Light
			if dark {
				scheme = design.Dark
			}
			if wantFocus := design.AccessibleRing(scheme, native.Accent); native.Focus != wantFocus {
				t.Fatalf("native focus=%+v, want WCAG-compliant focus %+v", native.Focus, wantFocus)
			}
			if c := design.Contrast(native.Focus.Over(native.Background), native.Background); c < 3.0 {
				t.Fatalf("native focus %+v contrast on %+v = %.2f < 3.0", native.Focus, native.Background, c)
			}
		})
	}
}
