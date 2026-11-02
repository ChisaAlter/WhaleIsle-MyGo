package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Chevrons and the permission shield are drawn as SVG so the 12/14px sizes and
// the 1.5px stroke match the reference instead of a text glyph's metrics.
var (
	chevronDown              = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M3 4.5L6 7.5L9 4.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	chevronUp                = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M3 7.5L6 4.5L9 7.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	shieldIcon               = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.59624 2.14853C7.50155 1.80917 8.49914 1.80919 9.40444 2.14859L13.9245 3.84317V7.11961C13.9245 11.6089 10.5565 13.5975 8.00035 14.5779C5.44423 13.5975 2.07544 11.6089 2.07544 7.11961V3.84317L6.59624 2.14853Z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/><path d="M8 4.39209V9.89209" stroke="currentColor" stroke-width="1"/><path d="M8 10.8081V11.8081" stroke="currentColor" stroke-width="1"/></svg>`))
	permissionReadOnlyIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M5.08545 8.13775L7.18455 10.2368C7.26636 10.3187 7.4003 10.3142 7.47649 10.2271L11.5148 5.61194" stroke="currentColor" stroke-width="1"/><path d="M6.59624 2.14853C7.50155 1.80917 8.49914 1.80919 9.40444 2.14859L13.9245 3.84317V7.11961C13.9245 11.6089 10.5565 13.5975 8.00035 14.5779C5.44423 13.5975 2.07544 11.6089 2.07544 7.11961V3.84317L6.59624 2.14853Z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/></svg>`))
	permissionWorkspaceIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.4209 1.68067C7.43922 1.299 8.56177 1.29898 9.58008 1.68067L14.0996 3.375C14.2946 3.44811 14.4236 3.63455 14.4238 3.84278V6.89063C14.1115 6.71853 13.7761 6.58312 13.4238 6.48926V4.18946L9.22852 2.61621C8.43657 2.31947 7.56341 2.31939 6.77148 2.61621L2.5752 4.18946V7.11914C2.5752 11.1796 5.52369 13.056 8 14.0391C8.27653 13.9293 8.55827 13.8067 8.8418 13.6729C9.07101 13.9468 9.33228 14.1929 9.62012 14.4053C9.12409 14.6579 8.63578 14.8696 8.17871 15.0449C8.0637 15.0889 7.93628 15.0889 7.82129 15.0449C5.22011 14.0472 1.5752 11.9381 1.5752 7.11914V3.84278C1.57541 3.63469 1.70463 3.44821 1.89941 3.375L6.4209 1.68067Z" fill="currentColor"/><path d="M5.26392 6.60339H10.7361" stroke="currentColor" stroke-width="1"/><path d="M5.26392 9.86902H8.32833" stroke="currentColor" stroke-width="1"/><path d="M10.0317 13.2229C10.263 13.3929 10.4943 13.563 10.7256 13.733C10.7932 13.6482 10.8608 13.5634 10.9284 13.4786C12.1455 11.9522 13.3626 10.4258 14.5798 8.89935C14.6474 8.81455 14.715 8.72975 14.7826 8.64495C14.4143 8.37419 14.046 8.10344 13.6777 7.83268C13.6169 7.92252 13.5562 8.01236 13.4954 8.10219C12.4016 9.71926 11.3078 11.3363 10.214 12.9534C10.1532 13.0432 10.0924 13.1331 10.0317 13.2229Z" fill="currentColor"/><path d="M12.6516 12.6696C12.6516 12.925 12.6516 13.1804 12.6516 13.4359C12.6952 13.4378 12.7387 13.4398 12.7823 13.4417C13.5663 13.4768 14.3504 13.5118 15.1345 13.5469C15.178 13.5488 15.2216 13.5508 15.2651 13.5527C15.2651 13.2194 15.2651 12.8861 15.2651 12.5527C15.2216 12.5547 15.178 12.5566 15.1345 12.5586C14.3504 12.5936 13.5663 13.6287 12.7823 12.6637C12.7387 12.6657 12.6952 12.6676 12.6516 12.6696Z" fill="currentColor"/></svg>`))
	permissionAutoReviewIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.59624 2.14853C7.50155 1.80917 8.49914 1.80919 9.40444 2.14859L13.9245 3.84317V7.11961C13.9245 11.6089 10.5565 13.5975 8.00035 14.5779C5.44423 13.5975 2.07544 11.6089 2.07544 7.11961V3.84317L6.59624 2.14853Z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/><path d="M5.75 10.15L8 4.95L10.25 10.15M6.43 8.59H9.57" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	sendIcon                 = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8.3125 0.980183C8.66767 1.0531 8.97902 1.20418 9.2627 1.43233C9.48724 1.61297 9.73029 1.85793 9.97949 2.10714L14.707 6.83468L13.293 8.24874L9 3.95577V15.0417H7V3.95577L2.70703 8.24874L1.29297 6.83468L6.02051 2.10714C6.26971 1.85793 6.51277 1.61297 6.7373 1.43233C6.97662 1.23986 7.28445 1.04402 7.6875 0.980183C7.8973 0.947006 8.1031 0.95516 8.3125 0.980183Z" fill="currentColor"/></svg>`))
	stopIcon                 = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><rect x="3" y="3" width="10" height="10" rx="3" fill="currentColor"/></svg>`))
	checkIcon                = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M3 8.5L6.2 11.7L13 4.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	folderIcon               = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M4 6.5A2 2 0 0 1 6 4.5h3.6c.5 0 1 .2 1.4.6l1 1c.4.4.9.6 1.4.6H18a2 2 0 0 1 2 2v7.8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/></svg>`))
	plusIcon                 = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 3V13M3 8H13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`))
)

func permissionIconFor(value string) *ui.SVG {
	switch value {
	case "read-only", "plan":
		return permissionReadOnlyIcon
	case "workspace-write", "default", "acceptEdits":
		return permissionWorkspaceIcon
	case "auto-review", "auto":
		return permissionAutoReviewIcon
	default:
		return shieldIcon
	}
}

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
	row = row.Transition(ui.ElementTransition{Duration: chrome.PopoverDuration, Colors: true})
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
	return ui.Icon(c, icon).Size(12, 12).Shrink(0).Opacity(0.72)
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

func isDark(colors theme.Colors) bool {
	if colors.BG == "" {
		return false
	}
	bg := chrome.Color(colors.BG)
	return bg.R < 128 && bg.G < 128 && bg.B < 128
}

// paintInnerBloom stands in for the reference's
// `inset 0 0 12px 0 rgba(255,255,255,.25)` card shadow: concentric hairlines
// just inside the edge, each fainter than the last. Over white it stays
// invisible; in dark themes it lights the rim the way the original does.
func paintInnerBloom(p *ui.Painter, r ui.Rect, radius float32) {
	white := ui.RGB(255, 255, 255)
	p.Clip(r, radius, func() {
		const steps = 16
		for i := 0; i <= steps; i++ {
			d := float32(i)*0.75 + 0.4
			band := ui.Rect{X: r.X + d, Y: r.Y + d, W: r.W - 2*d, H: r.H - 2*d}
			if band.W <= 0 || band.H <= 0 {
				return
			}
			t := float32(i) / float32(steps)
			falloff := (1 - t) * (1 - t)
			alpha := float32(0.004) + float32(0.052)*falloff
			p.Stroke(band, white.Alpha(alpha), max(0, radius-d), 0.85)
		}
	})
}
