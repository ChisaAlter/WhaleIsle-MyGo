package terminal

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// sameColor compares what a color paints, not how the parser stored it.
func sameColor(a, b ui.Color) bool { return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A }

// TestTerminalPaletteIsTheEnginesDefault pins the sixteen program colors. The
// reference passes no palette to Ghostty ("Ghostty owns ... the engine ANSI
// palette"), so these are Ghostty's built-in defaults and both theme halves
// share them.
func TestTerminalPaletteIsTheEnginesDefault(t *testing.T) {
	want := [16]string{
		"#1d1f21", "#cc6566", "#b6bd68", "#f0c674", "#82a2be", "#b294bb", "#8abeb7", "#c4c8c6",
		"#666666", "#d54e53", "#b9ca4b", "#e7c547", "#7aa6da", "#c397d8", "#70c0b1", "#eaeaea",
	}
	for i, hex := range want {
		if got := ansiColors[i]; !sameColor(got, ui.Hex(hex)) {
			t.Fatalf("ANSI %d = #%02x%02x%02x, want %s", i, got.R, got.G, got.B, hex)
		}
	}
	// The named colors a program is most likely to use, and their bright kinds.
	for _, c := range []struct {
		name  string
		index int
		hex   string
	}{
		{"black", 0, "#1d1f21"}, {"red", 1, "#cc6566"}, {"green", 2, "#b6bd68"},
		{"yellow", 3, "#f0c674"}, {"blue", 4, "#82a2be"}, {"magenta", 5, "#b294bb"},
		{"cyan", 6, "#8abeb7"}, {"white", 7, "#c4c8c6"},
		{"bright red", 9, "#d54e53"}, {"bright green", 10, "#b9ca4b"},
		{"bright yellow", 11, "#e7c547"}, {"bright blue", 12, "#7aa6da"},
	} {
		if got := ansiColors[c.index]; !sameColor(got, ui.Hex(c.hex)) {
			t.Fatalf("%s (ANSI %d) = #%02x%02x%02x, want %s", c.name, c.index, got.R, got.G, got.B, c.hex)
		}
	}
}

// TestTerminalThemeFollowsTheTokens checks the parts terminalThemeFromApp takes
// from the sheet: the pane's background and text are the theme's, the cursor and
// selection are the reference's fixed pair, and the palette does not move with
// the scheme.
func TestTerminalThemeFollowsTheTokens(t *testing.T) {
	light := theme.Resolve(theme.IDDeepseekLight).Active(false)
	dark := theme.Resolve(theme.IDDeepseekDark).Active(true)
	for _, c := range []struct {
		name   string
		colors theme.Colors
		dark   bool
		bg, fg string
	}{
		{"light", light, false, "#ffffff", "#0f1115"},
		{"dark", dark, true, "#151517", "#f9fafb"},
	} {
		th := terminalTheme(c.colors, c.dark)
		if !sameColor(th.Background, ui.Hex(c.bg)) {
			t.Errorf("%s background = %v, want %s (--dsw-alias-terminal-pane -> --dsw-alias-bg-base)", c.name, th.Background, c.bg)
		}
		if !sameColor(th.Foreground, ui.Hex(c.fg)) {
			t.Errorf("%s foreground = %v, want %s (--dsw-alias-label-primary)", c.name, th.Foreground, c.fg)
		}
		if !sameColor(th.Cursor, terminalCursor(c.dark)) || th.Cursor.A != 255 {
			t.Errorf("%s cursor = %v", c.name, th.Cursor)
		}
		if !sameColor(th.Selection, terminalSelection(c.dark)) {
			t.Errorf("%s selection = %v", c.name, th.Selection)
		}
		if th.Palette != ansiColors {
			t.Errorf("%s palette differs from the engine default", c.name)
		}
		// A program's colors survive a scheme flip, as they do in the reference.
		if th.Palette != terminalTheme(dark, c.dark).Palette {
			t.Errorf("%s palette follows the scheme", c.name)
		}
	}
	// The reference's constants, spelled out.
	if got := terminalCursor(true); !sameColor(got, ui.RGB(180, 203, 255)) {
		t.Fatalf("dark cursor = %v, want rgb(180, 203, 255)", got)
	}
	if got := terminalCursor(false); !sameColor(got, ui.RGB(38, 56, 78)) {
		t.Fatalf("light cursor = %v, want rgb(38, 56, 78)", got)
	}
	if got := terminalSelection(true); !sameColor(got, ui.RGBA(180, 203, 255, .25)) {
		t.Fatalf("dark selection = %v, want rgba(180, 203, 255, 0.25)", got)
	}
	if got := terminalSelection(false); !sameColor(got, ui.RGBA(37, 63, 99, .2)) {
		t.Fatalf("light selection = %v, want rgba(37, 63, 99, 0.2)", got)
	}
}

// TestTerminalFontFollowsTheAppearance checks the font the pane sets: family and
// size from the rendered preferences, the reference's clamps, and rows at
// round(size * 1.35).
func TestTerminalFontFollowsTheAppearance(t *testing.T) {
	for _, c := range []struct{ size, want float32 }{
		{0, 12}, {13, 13}, {13.4, 13}, {13.6, 14}, {3, 6}, {100, 32}, {-4, 12},
	} {
		if got := terminalFontSize(c.size); got != c.want {
			t.Errorf("terminalFontSize(%v) = %v, want %v", c.size, got, c.want)
		}
	}
	for _, c := range []struct{ size, want float32 }{
		{12, 16}, {13, 18}, {6, 8}, {32, 43},
	} {
		if got := terminalCellHeight(c.size); got != c.want {
			t.Errorf("row height at %vpx = %v, want %v", c.size, got, c.want)
		}
	}
	font := terminalFont(theme.FontPreference{Family: "JetBrains Mono", Size: 13})
	if font.Family != "JetBrains Mono" || font.Size != 13 {
		t.Fatalf("font = %+v, want the preference's family at 13", font)
	}
	if font.LineHeight <= 0 {
		t.Fatal("the pane kept default rows; the reference's cell height was not applied")
	}
	metrics := ui.Font{Family: "JetBrains Mono", Size: 13, Weight: 400}.Metrics()
	natural := metrics.Ascent + metrics.Descent + metrics.LineGap
	if natural <= 0 {
		t.Skip("no font metrics in this environment")
	}
	if row := font.LineHeight * natural; math.Abs(float64(row-terminalCellHeight(13))) > 0.5 {
		t.Fatalf("rows land on %.2f DIP, want %.1f", row, terminalCellHeight(13))
	}
	// The rendered preferences supply what the pane reads.
	rendered := theme.RenderPreferences().Terminal
	if rendered.Family == "" {
		t.Fatal("RenderPreferences left the terminal family empty")
	}
}

// TestTerminalRowsSitOnTheReferencePitch renders two rows the way a pane stacks
// them and checks the pitch in the tester: 18 DIP at the default 13px font.
func TestTerminalRowsSitOnTheReferencePitch(t *testing.T) {
	font := terminalFont(theme.RenderPreferences().Terminal)
	row := terminalCellHeight(font.Size)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			for _, label := range []string{"row one", "row two"} {
				ui.Box(c).FillWidth().Height(row).Children(func() {
					ui.Text(c, label).Font(font.Family).FontSize(font.Size)
				})
			}
		})
	}, 200, 100)
	first, ok := tt.Find("row one")
	if !ok {
		t.Fatalf("first row missing")
	}
	second, ok := tt.Find("row two")
	if !ok {
		t.Fatalf("second row missing")
	}
	if pitch := second.Y - first.Y; math.Abs(float64(pitch-row)) > 0.5 {
		t.Fatalf("rows sit %.1f DIP apart, want %.1f", pitch, row)
	}
}
