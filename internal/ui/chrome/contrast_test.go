package chrome

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/theme"
)

// guardWallpapers are the backdrop ends the shell must stay readable over: a
// bright photo and a dark one. The guard adds no third case: any image between
// them composites to a surface between these two, and the guard only raises the
// material's share of the backdrop.
var guardWallpapers = []struct {
	name string
	c    ui.Color
}{
	{"light-wallpaper", ui.RGB(232, 230, 226)},
	{"dark-wallpaper", ui.RGB(16, 20, 24)},
}

// guardSurface is one surface the shell paints text on: the theme color it is
// painted with and the material that shares it with the backdrop.
type guardSurface struct {
	name string
	base ui.Color
	m    Material
}

// guardSurfaces lists the surfaces from the shipped theme half.
func guardSurfaces(dark bool) []guardSurface {
	colors := guardColors(dark)
	return []guardSurface{
		{"panel on bg", Color(colors.BG), MaterialPanel},
		{"panel on field", Color(colors.Field), MaterialPanel},
		{"panel on input-major", Color(colors.InputMajor), MaterialPanel},
		{"menu on field", Color(colors.Field), MaterialMenu},
	}
}

// guardColors is the palette a snapshot renders: the shipped deepseek half.
func guardColors(dark bool) theme.Colors {
	if dark {
		return theme.Resolve(theme.IDDeepseekDark).Active(true)
	}
	return theme.Resolve(theme.IDDeepseekLight).Active(false)
}

// guardedSurface is the color material m leaves over backdrop, at the share the
// contrast guard picks for ink.
func guardedSurface(backdrop, base ui.Color, m Material, p wallpapers.Preferences, dark bool, ink ui.Color) (ui.Color, float32) {
	alpha := MaterialAlphaFor(backdrop, base, m, p, dark, ink)
	return MaterialSurface(backdrop, MaterialFill(base, alpha)), alpha
}

// pixel reads a rendered pixel as a ui color.
func pixel(c color.RGBA) ui.Color { return ui.RGBA(c.R, c.G, c.B, float32(c.A)/255) }

// TestGlassTextKeepsItsContrast is the material guard: body text stays at 4.5:1
// and secondary text at 3:1 on every panel and menu, in four combinations of
// theme half and wallpaper brightness, with and without a wallpaper.
func TestGlassTextKeepsItsContrast(t *testing.T) {
	prefs := wallpapers.Defaults()
	for _, dark := range []bool{false, true} {
		colors := guardColors(dark)
		body := Color(theme.PrimaryInk(dark))
		secondary := Color(theme.SecondaryInk(dark))
		mask := MaskAlpha(dark)
		half := map[bool]string{true: "dark", false: "light"}[dark]
		for _, wall := range guardWallpapers {
			backdrop := wall.c.Mix(ui.RGB(0, 0, 0), mask)
			for _, surface := range guardSurfaces(dark) {
				fill, alpha := guardedSurface(backdrop, surface.base, surface.m, prefs, dark, secondary)
				bodyRatio := ContrastRatio(body, fill)
				secondaryRatio := ContrastRatio(secondary, fill)
				t.Logf("%s/%s %-18s alpha=%.2f fill=%3d,%3d,%3d body=%.2f secondary=%.2f",
					half, wall.name, surface.name, alpha, fill.R, fill.G, fill.B, bodyRatio, secondaryRatio)
				if bodyRatio < TextContrast {
					t.Errorf("%s half over %s, %s: body text %.2f:1, want %.1f:1",
						half, wall.name, surface.name, bodyRatio, TextContrast)
				}
				if secondaryRatio < SecondaryContrast {
					t.Errorf("%s half over %s, %s: secondary text %.2f:1, want %.1f:1",
						half, wall.name, surface.name, secondaryRatio, SecondaryContrast)
				}
				if alpha < MaterialAlpha(surface.m, prefs, dark) {
					t.Errorf("%s/%s %s: guard lowered the material to %.2f", half, wall.name, surface.name, alpha)
				}
			}
		}
		// Without a wallpaper the surface sits on the theme's own background.
		page := Color(colors.BG)
		for _, surface := range guardSurfaces(dark) {
			fill := MaterialSurface(page, MaterialFill(surface.base, MaterialAlpha(surface.m, prefs, dark)))
			bodyRatio, secondaryRatio := ContrastRatio(body, fill), ContrastRatio(secondary, fill)
			t.Logf("%s/none      %-18s fill=%3d,%3d,%3d body=%.2f secondary=%.2f",
				half, surface.name, fill.R, fill.G, fill.B, bodyRatio, secondaryRatio)
			if bodyRatio < TextContrast || secondaryRatio < SecondaryContrast {
				t.Errorf("%s half without a wallpaper, %s: body %.2f:1 secondary %.2f:1",
					half, surface.name, bodyRatio, secondaryRatio)
			}
		}
	}
}

// TestMaterialGuardRaisesTheShareWhereTheTokenFails pins why the guard exists:
// the light half over a dark wallpaper leaves secondary text under 3:1 at the
// token's own share, and the guard raises it.
func TestMaterialGuardRaisesTheShareWhereTheTokenFails(t *testing.T) {
	prefs := wallpapers.Defaults()
	secondary := Color(theme.SecondaryInk(false))
	backdrop := ui.RGB(16, 20, 24).Mix(ui.RGB(0, 0, 0), MaskAlpha(false))
	base := Color(guardColors(false).Field)

	token := MaterialAlpha(MaterialMenu, prefs, false)
	tokenFill := MaterialSurface(backdrop, MaterialFill(base, token))
	t.Logf("light menu over a dark wallpaper: token %.2f → fill %v, secondary %.2f:1", token, tokenFill, ContrastRatio(secondary, tokenFill))
	if got := ContrastRatio(secondary, tokenFill); got >= SecondaryContrast {
		t.Fatalf("the token share already reads at %.2f:1; the guard's case is gone", got)
	}
	guarded, alpha := guardedSurface(backdrop, base, MaterialMenu, prefs, false, secondary)
	t.Logf("guarded: alpha %.2f → fill %v, secondary %.2f:1", alpha, guarded, ContrastRatio(secondary, guarded))
	if alpha <= token {
		t.Fatalf("guard kept the failing share: %.2f", alpha)
	}
	if got := ContrastRatio(secondary, guarded); got < SecondaryContrast {
		t.Fatalf("guarded menu still reads at %.2f:1", got)
	}
}

// TestEnsureContrastReachesTheFloor covers the ink form of the guard.
func TestEnsureContrastReachesTheFloor(t *testing.T) {
	mid := ui.RGB(206, 207, 208)
	faint := ui.RGB(173, 178, 184) // label-caption over a light glass panel
	if got := ContrastRatio(faint, mid); got >= TextContrast {
		t.Fatalf("fixture ink already reads at %.2f:1", got)
	}
	fixed := EnsureContrast(faint, mid)
	if got := ContrastRatio(fixed, mid); got < TextContrast {
		t.Fatalf("EnsureContrast left %.2f:1", got)
	}
	if fixed == faint {
		t.Fatal("EnsureContrast returned the unreadable ink")
	}
	if fixed.R == 255 && fixed.G == 255 && fixed.B == 255 {
		t.Fatal("EnsureContrast on a light surface brightened the ink instead of darkening it")
	}
	if got := EnsureContrast(ui.RGB(15, 17, 21), ui.RGB(255, 255, 255)); got != (ui.RGB(15, 17, 21)) {
		t.Fatalf("readable ink changed: %v", got)
	}
	if got := EnsureContrastAt(faint, mid, 1); got != faint {
		t.Fatal("a floor of 1 should keep the ink as it is")
	}
}

// TestMaterialHairlineStaysVisibleInTheDark checks the 0.5px border MenuSurface
// paints: in the dark half it is the only separation between the menu and the
// wallpaper behind it.
func TestMaterialHairlineStaysVisibleInTheDark(t *testing.T) {
	colors := guardColors(true)
	field := Color(colors.Field)
	border := Color(colors.Border)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		// The page behind the menu: the same fill, so only the hairline shows.
		ui.Box(c).Fill().Background(field)
		MenuSurface(ui.Box(c).Absolute().Left(10).Top(10).Width(60).Height(40).Radius(10), field, border, 10)
	}, 80, 60)
	tt.Frame()
	image := tt.Image()
	inside := pixel(image.RGBAAt(40, 30))
	edge := pixel(image.RGBAAt(40, 10))
	line := ContrastRatio(edge, inside)
	t.Logf("menu fill=%v hairline=%v contrast=%.2f", inside, edge, line)
	if line < 1.2 {
		t.Fatalf("the dark menu hairline is invisible: %.2f:1 (%v on %v)", line, edge, inside)
	}
	if RelativeLuminance(edge) <= RelativeLuminance(inside) {
		t.Fatalf("the dark hairline should be lighter than the fill: %v on %v", edge, inside)
	}
}

// TestDarkWallpaperMaskAndMaterialShare checks the wallpaper layer of the sheet:
// the dim mask takes half the image in the dark half, and the material guard
// keeps a panel over it within a hair of its own surface color, so the text
// colors the theme designed keep working.
func TestDarkWallpaperMaskAndMaterialShare(t *testing.T) {
	t.Cleanup(func() { ConfigureBackdrop("", wallpapers.Defaults()) })
	const wall = 110
	url := pngDataURL(t, color.RGBA{R: wall, G: wall, B: wall, A: 255}, 8, 8)
	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	colors := guardColors(true)
	field := Color(colors.Field)
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		paint := WallpaperBackgroundPainter(url, prefs, c.Preferences().ReduceMotion)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paint(p, r) })
		ui.Box(c).Absolute().Left(0).Top(0).Width(44).Height(60).
			Draw(Glass(field, MaterialPanel, 0))
	}, 80, 60)
	tt.Frame()
	image := tt.Image()
	painted := pixel(image.RGBAAt(70, 50)) // outside the panel: the masked wallpaper
	want := float64(wall) * (1 - maskDark)
	if math.Abs(float64(painted.R)-want) > 2 {
		t.Fatalf("dark mask left the wallpaper at %d, want %.0f", painted.R, want)
	}
	panel := pixel(image.RGBAAt(22, 30))
	own := MaterialSurface(Color(colors.BG), MaterialFill(field, MaterialAlpha(MaterialPanel, prefs, true)))
	if got := ContrastRatio(panel, own); got > 1.45 {
		t.Fatalf("panel over the wallpaper moved to %.2f:1 from its own surface (%v vs %v)", got, panel, own)
	}
	secondary := Color(theme.SecondaryInk(true))
	if got := ContrastRatio(secondary, panel); got < SecondaryContrast {
		t.Fatalf("secondary text over the dark wallpaper reads at %.2f:1", got)
	}
	t.Logf("masked wallpaper=%v panel=%v own-surface=%v secondary=%.2f:1",
		painted, panel, own, ContrastRatio(secondary, panel))
}
