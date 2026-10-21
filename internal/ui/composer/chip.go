package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Chevrons and the permission shield are drawn as SVG so the 12/14px sizes and
// the 1.5px stroke match the reference instead of a text glyph's metrics.
var (
	chevronDown = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M3 4.5L6 7.5L9 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	chevronUp   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M3 7.5L6 4.5L9 7.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	shieldIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 1.8L13.1 3.6V7.9C13.1 10.9 10.9 13.3 8 14.3C5.1 13.3 2.9 10.9 2.9 7.9V3.6L8 1.8Z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/></svg>`))
	sendIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8.3125 0.980183C8.66767 1.0531 8.97902 1.20418 9.2627 1.43233C9.48724 1.61297 9.73029 1.85793 9.97949 2.10714L14.707 6.83468L13.293 8.24874L9 3.95577V15.0417H7V3.95577L2.70703 8.24874L1.29297 6.83468L6.02051 2.10714C6.26971 1.85793 6.51277 1.61297 6.7373 1.43233C6.97662 1.23986 7.28445 1.04402 7.6875 0.980183C7.8973 0.947006 8.1031 0.95516 8.3125 0.980183Z" fill="currentColor"/></svg>`))
	stopIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><rect x="3" y="3" width="10" height="10" rx="3" fill="currentColor"/></svg>`))
	checkIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M3 8.5L6.2 11.7L13 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	folderIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M4 6.5A2 2 0 0 1 6 4.5h3.6c.5 0 1 .2 1.4.6l1 1c.4.4.9.6 1.4.6H18a2 2 0 0 1 2 2v7.8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/></svg>`))
	plusIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 3V13M3 8H13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`))
)

// The chip row shares one height with the toolbar's other controls.
const chipHeight = float32(28)

// Menu surfaces share one set of metrics so the permission/model picks, the
// context panel and the `/` `@` menus read as one family.
const (
	menuRowHeight = float32(36)
	menuRadius    = float32(12)
)

// menuSurface gives a popup the shared menu material: sampled glass fill, the
// 0.5 DIP hairline and the elevation shadow every floating menu uses.
func menuSurface(c *ui.Context, e ui.Element, colors theme.Colors) ui.Element {
	e = chrome.MenuSurface(e, chrome.Color(colors.BG), chrome.Color(colors.Border), menuRadius)
	return e.Opacity(chrome.Reveal(c, e, chrome.PopoverDuration))
}

// menuRow is one selectable row inside a popover menu: no frame of its own, a
// hover/pressed fill and the shared 36 DIP rhythm.
func menuRow(c *ui.Context, colors theme.Colors) ui.Element {
	row := ui.Button(c, "").FillWidth().MinHeight(menuRowHeight).Padding(6, 8).Radius(8).BorderWidth(0).
		Background(ui.Transparent).AlignItems(ui.Center)
	if row.Pressed() {
		return row.Background(chrome.Color(colors.Pressed))
	}
	if row.Hovered() {
		return row.Background(chrome.Color(colors.Hover))
	}
	return row
}

// chevron paints the closed/open marker of a picker chip: ChevronDown at rest,
// ChevronUp while its popup is open (the reference rotates it 180 degrees).
func chevron(c *ui.Context, open bool) ui.Element {
	icon := chevronDown
	if open {
		icon = chevronUp
	}
	return ui.Icon(c, icon).Size(12, 12).Shrink(0)
}

// chip paints the shared closed chrome of the composer's pickers: a transparent
// 28px capsule that only fills while hovered or open. A plain Button with an
// explicit background drops styleButton's own hover (third_party/mygo/ui
// widgets.go), so the fill is chosen here from the live hover state.
func chip(c *ui.Context, colors theme.Colors, open bool, textColor ui.Color) ui.Element {
	trigger := ui.Button(c, "").Height(chipHeight).Radius(8).Padding(0, 4, 0, 8).
		Gap(4).AlignItems(ui.Center).MinWidth(0).MaxWidth(220).BorderWidth(0).
		Background(ui.Transparent).TextColor(textColor)
	if trigger.Hovered() || open {
		trigger = trigger.Background(chrome.Color(colors.Hover))
	}
	return trigger
}

// paintInnerBloom stands in for the reference's
// `inset 0 0 12px 1px rgba(255,255,255,.25)` card shadow: concentric hairlines
// just inside the edge, each fainter than the last. Over white it stays
// invisible; in dark themes it lights the rim the way the original does.
func paintInnerBloom(p *ui.Painter, r ui.Rect, radius float32) {
	const steps = 8
	white := ui.RGB(255, 255, 255)
	for i := 0; i < steps; i++ {
		inset := float32(i) + 1
		band := ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}
		if band.W <= 0 || band.H <= 0 {
			return
		}
		falloff := 1 - float32(i)/steps
		p.Stroke(band, white.Alpha(0.25*falloff*falloff), max(0, radius-inset), 1)
	}
}
