package chrome

import (
	"testing"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

// Toolkit-drawn menus (ui.Theme.Menu) must take the same look MenuSurface
// gives the chrome's own popovers: the raised fill, the border-l1 hairline
// and the menu's inks — opaque where no backdrop is live.
func TestMenuStyleIsOpaqueWithoutBackdrop(t *testing.T) {
	ConfigureBackdrop("", wallpapers.Defaults())
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	colors := theme.Resolve("deepseek").Light
	st := MenuStyle(colors, false)
	if st.Material != nil {
		t.Fatalf("expected no material without a backdrop, got %T", st.Material)
	}
	if st.Fill.A != 255 {
		t.Fatalf("expected opaque fill without a backdrop, got alpha %d", st.Fill.A)
	}
	if got, want := st.Fill, MenuBaseFill(colors, false); got != want {
		t.Fatalf("fill = %v, want MenuBaseFill %v", got, want)
	}
	if st.Stroke != Color(colors.BorderL1) {
		t.Fatalf("stroke = %v, want BorderL1 %v", st.Stroke, Color(colors.BorderL1))
	}
	if st.Hover != Color(colors.Hover) {
		t.Fatalf("hover = %v, want %v", st.Hover, Color(colors.Hover))
	}
	if st.Caption != Color(colors.Caption) {
		t.Fatalf("caption = %v, want %v", st.Caption, Color(colors.Caption))
	}
	if st.Separator != Color(colors.Border) {
		t.Fatalf("separator = %v, want %v", st.Separator, Color(colors.Border))
	}
	if st.Icon != Color(colors.FG) {
		t.Fatalf("icon = %v, want the primary ink %v", st.Icon, Color(colors.FG))
	}
}

// Under a wallpaper the menu fill drops to MaterialMenu's solidity share and
// the panel paints the menu frost — light and dark alike.
func TestMenuStyleSharesTheWallpaper(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	for _, tc := range []struct {
		name     string
		dark     bool
		colors   theme.Colors
		solidity float32
	}{
		{"light", false, theme.Resolve("deepseek").Light, menuSolidityLight},
		{"dark", true, theme.Resolve("deepseek").Dark, menuSolidityDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ConfigureBackdrop("wall.png", wallpapers.Defaults())
			st := MenuStyle(tc.colors, tc.dark)
			if st.Material == nil {
				t.Fatal("expected the menu frost material under a wallpaper")
			}
			base := MenuBaseFill(tc.colors, tc.dark)
			if st.Fill.R != base.R || st.Fill.G != base.G || st.Fill.B != base.B {
				t.Fatalf("fill rgb = %v, want MenuBaseFill rgb %v", st.Fill, base)
			}
			if got := float32(st.Fill.A) / 255; got < tc.solidity-0.01 || got > tc.solidity+0.01 {
				t.Fatalf("fill alpha = %.3f, want menu solidity %.2f", got, tc.solidity)
			}
			if wantStroke := MenuBorderColor(tc.colors, tc.dark); st.Stroke != wantStroke {
				t.Fatalf("stroke = %v, want %v", st.Stroke, wantStroke)
			}
		})
	}
}
