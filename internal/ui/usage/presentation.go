package usage

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// These are dsh-usage-panel/styles.ts's layer-1, border-l1 and secondary
// label roles. Keep them local to the statistics surface.
func usageCardPalette(c *ui.Context) (surface, border, secondary ui.Color) {
	t := c.Theme()
	if int(t.Background.R)+int(t.Background.G)+int(t.Background.B) < 384 {
		return t.Surface.Alpha(.8), ui.RGBA(255, 255, 255, .06), ui.Hex("#cfd3d6")
	}
	return t.Background.Alpha(.8), ui.RGBA(0, 0, 0, .04), ui.Hex("#61666b")
}

func usageHeatColor(level int, surface, secondary ui.Color) ui.Color {
	if level <= 0 {
		return secondary.Alpha(.14)
	}
	accent := ui.Hex("#4176e6") // --dsw-static-deepseek-500, in both schemes.
	weight := []float32{0, .26, .48, .74, 1}[min(level, 4)]
	channel := func(a, b uint8) uint8 { return uint8(float32(a)*weight + float32(b)*(1-weight) + .5) }
	return ui.RGB(channel(accent.R, surface.R), channel(accent.G, surface.G), channel(accent.B, surface.B))
}

func usageCard(c *ui.Context, key, title string, fg ui.Color, content func()) {
	surface, border, _ := usageCardPalette(c)
	ui.Column(c.Key(key)).FillWidth().MinWidth(0).Padding(16).Gap(12).Radius(8).Background(surface).Border(1, border).Children(func() {
		if title != "" {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontWeight(600).TextColor(fg)
		}
		content()
	})
}

func usageEmpty(c *ui.Context, title, hint string, fg, muted ui.Color) {
	surface, _, _ := usageCardPalette(c)
	ui.Column(c).FillWidth().MinWidth(0).Padding(32, 20).Gap(4).AlignItems(ui.Center).Radius(8).Background(surface).Border(1, c.Theme().Border).BorderStyle(ui.BorderDashed).Children(func() {
		if title != "" {
			ui.Text(c, title).FillWidth().TextAlign(ui.Center).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontWeight(600).TextColor(fg)
		}
		if hint != "" {
			ui.Text(c, hint).FillWidth().TextAlign(ui.Center).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		}
	})
}

var usagePageIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M3 13V9.5M8 13V5.5M13 13V3M2 13.5h12"/></svg>`))

// Disclosure chevrons for folded sections, from the original 16px icon set.
var usageChevronDownIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 6.25L8 9.75L11.5 6.25"/></svg>`))

var usageChevronRightIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><path d="M6.25 4.5L9.75 8L6.25 11.5"/></svg>`))
