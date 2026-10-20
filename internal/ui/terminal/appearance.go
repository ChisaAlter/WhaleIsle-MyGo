package terminal

import (
	"math"

	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// The reference terminal's appearance comes from
// packages/client/ui-user-terminal/lib/client.js:
//
//   - terminalThemeFromApp takes the pane's background and text from the
//     drawer surface's computed colors, which the sheet fills with
//     --dsw-alias-terminal-pane (--dsw-alias-bg-base) and colors with
//     --dsw-alias-label-primary;
//   - its cursor and selection are fixed per half;
//   - it passes no palette at all: "Ghostty owns fit, 150 ms PTY resize
//     debounce, bold-as-700, and the engine ANSI palette", so both halves share
//     Ghostty's built-in default sixteen;
//   - measureGhosttyCell sizes a cell as round(fontSize * 1.35), the font size
//     clamped to 6..32 and 12 when unset, with the family from
//     --dsw-font-family-terminal (falling back to the code family).

// ansiColors is Ghostty's built-in default palette, the sixteen colors the
// shipped "Ghostty Default Style Dark" theme file carries: black, red, green,
// yellow, blue, magenta, cyan and white, then their bright kinds. The reference
// leaves the palette to the engine, so this one serves both theme halves; a
// program's colors therefore stay the same when the scheme flips, exactly as
// they do in the reference.
var ansiColors = [16]ui.Color{
	ui.Hex("#1d1f21"), ui.Hex("#cc6566"), ui.Hex("#b6bd68"), ui.Hex("#f0c674"),
	ui.Hex("#82a2be"), ui.Hex("#b294bb"), ui.Hex("#8abeb7"), ui.Hex("#c4c8c6"),
	ui.Hex("#666666"), ui.Hex("#d54e53"), ui.Hex("#b9ca4b"), ui.Hex("#e7c547"),
	ui.Hex("#7aa6da"), ui.Hex("#c397d8"), ui.Hex("#70c0b1"), ui.Hex("#eaeaea"),
}

// Terminal sizes and metrics from the reference's canvas.
const (
	// terminalMinSize and terminalMaxSize are MIN_TERMINAL_FONT_SIZE and
	// MAX_TERMINAL_FONT_SIZE.
	terminalMinSize = 6
	terminalMaxSize = 32
	// terminalDefaultSize is the 12 terminalFontSize falls back to.
	terminalDefaultSize = 12
	// terminalCellRatio is the height of a cell as a multiple of the font size
	// (measureGhosttyCell: round(fontSize * 1.35)).
	terminalCellRatio = 1.35
)

// terminalTheme colors a pane the way terminalThemeFromApp does: the theme's own
// background and text, the reference's fixed cursor and selection, and the
// engine's sixteen colors.
func terminalTheme(colors theme.Colors, dark bool) *myterm.Theme {
	th := &myterm.Theme{
		Background: chrome.Color(colors.BG),
		Foreground: chrome.Color(colors.FG),
		Cursor:     terminalCursor(dark),
		Selection:  terminalSelection(dark),
		Palette:    ansiColors,
	}
	// A block cursor is drawn over the glyph it covers, so its text takes the
	// pane's background, as the reference leaves CursorText to the renderer.
	th.CursorText = th.Background
	return th
}

// terminalCursor is the reference's cursor: rgb(180, 203, 255) over the dark
// half and rgb(38, 56, 78) over the light one.
func terminalCursor(dark bool) ui.Color {
	if dark {
		return ui.RGB(180, 203, 255)
	}
	return ui.RGB(38, 56, 78)
}

// terminalSelection is the reference's selection: rgba(180, 203, 255, 0.25) over
// the dark half and rgba(37, 63, 99, 0.2) over the light one.
func terminalSelection(dark bool) ui.Color {
	if dark {
		return ui.RGBA(180, 203, 255, .25)
	}
	return ui.RGBA(37, 63, 99, .2)
}

// terminalFont is the pane's font: the family and size the appearance
// preferences render, with rows at the reference's cell height. MyGo scales a
// font's own line height instead of taking an absolute cell size, so the
// multiplier is the reference's cell height over this font's ascent, descent and
// gap; a font without metrics keeps the plugin's default rows.
func terminalFont(pref theme.FontPreference) myterm.Font {
	size := terminalFontSize(pref.Size)
	font := myterm.Font{Family: pref.Family, Size: size}
	metrics := ui.Font{Family: pref.Family, Size: size, Weight: 400}.Metrics()
	if natural := metrics.Ascent + metrics.Descent + metrics.LineGap; natural > 0 {
		font.LineHeight = terminalCellHeight(size) / natural
	}
	return font
}

// terminalFontSize rounds and clamps a preference the way terminalFontSize in
// the reference does: 12 when unset or not a number, else 6..32.
func terminalFontSize(size float32) float32 {
	if size <= 0 || math.IsNaN(float64(size)) || math.IsInf(float64(size), 0) {
		return terminalDefaultSize
	}
	return float32(min(max(math.Round(float64(size)), terminalMinSize), terminalMaxSize))
}

// terminalCellHeight is the height of a row in DIPs: the reference's
// round(fontSize * 1.35), never under one.
func terminalCellHeight(size float32) float32 {
	return max(1, float32(math.Round(float64(size)*terminalCellRatio)))
}
