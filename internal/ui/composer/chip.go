package composer

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// Permission shield icons and folder icon keep their bespoke SVG paths while
// standard glyphs (chevron, check, plus, send) come from coss.
var (
	shieldIcon               = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.59624 2.14853C7.50155 1.80917 8.49914 1.80919 9.40444 2.14859L13.9245 3.84317V7.11961C13.9245 11.6089 10.5565 13.5975 8.00035 14.5779C5.44423 13.5975 2.07544 11.6089 2.07544 7.11961V3.84317L6.59624 2.14853Z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/><path d="M8 4.39209V9.89209" stroke="currentColor" stroke-width="1"/><path d="M8 10.8081V11.8081" stroke="currentColor" stroke-width="1"/></svg>`))
	permissionReadOnlyIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M5.08545 8.13775L7.18455 10.2368C7.26636 10.3187 7.4003 10.3142 7.47649 10.2271L11.5148 5.61194" stroke="currentColor" stroke-width="1"/><path d="M6.59624 2.14853C7.50155 1.80917 8.49914 1.80919 9.40444 2.14859L13.9245 3.84317V7.11961C13.9245 11.6089 10.5565 13.5975 8.00035 14.5779C5.44423 13.5975 2.07544 11.6089 2.07544 7.11961V3.84317L6.59624 2.14853Z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/></svg>`))
	permissionWorkspaceIcon  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.4209 1.68067C7.43922 1.299 8.56177 1.29898 9.58008 1.68067L14.0996 3.375C14.2946 3.44811 14.4236 3.63455 14.4238 3.84278V6.89063C14.1115 6.71853 13.7761 6.58312 13.4238 6.48926V4.18946L9.22852 2.61621C8.43657 2.31947 7.56341 2.31939 6.77148 2.61621L2.5752 4.18946V7.11914C2.5752 11.1796 5.52369 13.056 8 14.0391C8.27653 13.9293 8.55827 13.8067 8.8418 13.6729C9.07101 13.9468 9.33228 14.1929 9.62012 14.4053C9.12409 14.6579 8.63578 14.8696 8.17871 15.0449C8.0637 15.0889 7.93628 15.0889 7.82129 15.0449C5.22011 14.0472 1.5752 11.9381 1.5752 7.11914V3.84278C1.57541 3.63469 1.70463 3.44821 1.89941 3.375L6.4209 1.68067Z" fill="currentColor"/><path d="M5.26392 6.60339H10.7361" stroke="currentColor" stroke-width="1"/><path d="M5.26392 9.86902H8.32833" stroke="currentColor" stroke-width="1"/><path d="M10.0317 13.2229C10.263 13.3929 10.4943 13.563 10.7256 13.733C10.7932 13.6482 10.8608 13.5634 10.9284 13.4786C12.1455 11.9522 13.3626 10.4258 14.5798 8.89935C14.6474 8.81455 14.715 8.72975 14.7826 8.64495C14.4143 8.37419 14.046 8.10344 13.6777 7.83268C13.6169 7.92252 13.5562 8.01236 13.4954 8.10219C12.4016 9.71926 11.3078 11.3363 10.214 12.9534C10.1532 13.0432 10.0924 13.1331 10.0317 13.2229Z" fill="currentColor"/><path d="M12.6516 12.6696C12.6516 12.925 12.6516 13.1804 12.6516 13.4359C12.6952 13.4378 12.7387 13.4398 12.7823 13.4417C13.5663 13.4768 14.3504 13.5118 15.1345 13.5469C15.178 13.5488 15.2216 13.5508 15.2651 13.5527C15.2651 13.2194 15.2651 12.8861 15.2651 12.5527C15.2216 12.5547 15.178 12.5566 15.1345 12.5586C14.3504 12.5936 13.5663 12.6287 12.7823 12.6637C12.7387 12.6657 12.6952 12.6676 12.6516 12.6696Z" fill="currentColor"/></svg>`))
	permissionAutoReviewIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M6.59624 2.14853C7.50155 1.80917 8.49914 1.80919 9.40444 2.14859L13.9245 3.84317V7.11961C13.9245 11.6089 10.5565 13.5975 8.00035 14.5779C5.44423 13.5975 2.07544 11.6089 2.07544 7.11961V3.84317L6.59624 2.14853Z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/><path d="M5.75 10.15L8 4.95L10.25 10.15M6.43 8.59H9.57" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	stopIcon                 = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><rect x="3" y="3" width="10" height="10" rx="3" fill="currentColor"/></svg>`))
	folderIcon               = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"><path d="M4 6.5A2 2 0 0 1 6 4.5h3.6c.5 0 1 .2 1.4.6l1 1c.4.4.9.6 1.4.6H18a2 2 0 0 1 2 2v7.8a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/></svg>`))
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
const menuRowHeight = float32(34)

func applyShadows(e ui.Element, sc design.Shadows, col ui.Color, alpha float32) {
	for i := len(sc) - 1; i >= 0; i-- {
		l := sc[i]
		e.Shadow(l.X, l.Y, l.Blur, l.Spread, col.Alpha(alpha))
	}
}

func composerCardFill(t *design.Tokens) ui.Color {
	if t.Glass {
		return t.Card
	}
	if t.IsDark() {
		return t.Input.Alpha(0.32)
	}
	return t.Background
}

func composerCardBorder(t *design.Tokens, focused bool) ui.Color {
	if focused {
		return t.Ring
	}
	return t.Input
}

func paintGlassInnerRim(p *ui.Painter, r ui.Rect, radius float32, dark bool, darkAlpha, lightAlpha float32) {
	inset := design.BorderWidth
	inner := ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}
	if inner.W <= 0 || inner.H <= 0 {
		return
	}
	alpha := lightAlpha
	if dark {
		alpha = darkAlpha
	}
	p.Stroke(inner, design.TW.White.Alpha(alpha), max(0, radius-inset), design.BorderWidth)
}

func composerCardElevation(e ui.Element, t *design.Tokens, focused, disabled bool) {
	if disabled || !focused {
		return
	}
	e.Shadow(0, 0, 0, design.InputRingWidth, t.Ring.Alpha(design.InputRingAlpha))
}

// menuPanelLook gives a popup the shared coss popover look: popover fill,
// border, and lv3 shadow without an enter/position transition.
func menuPanelLook(c *ui.Context, e ui.Element) ui.Element {
	return chrome.ApplyMenuSurface(e, design.For(c), theme.ContextRadii(c).MD)
}

// menuSurface gives a PopoverBase panel the shared coss popover look plus
// popover enter motion without breaking glass backdrop sampling.
func menuSurface(c *ui.Context, e ui.Element) ui.Element {
	menuPanelLook(c, e)
	return chrome.PopoverReveal(c, e)
}

// menuRow is one selectable row inside a popover menu: no frame of its own, a
// coss accent hover/pressed fill, and the shared 34 DIP rhythm.
func menuRow(c *ui.Context) ui.Element {
	t := design.For(c)
	row := ui.ButtonBase(c).FillWidth().MinHeight(menuRowHeight).Padding(design.Sp(1.5), design.Sp(2)).Gap(design.Sp(1.5)).
		Radius(design.RadiusLG).FocusRing(false).Cursor(ui.CursorPointer).Justify(ui.Start).AlignItems(ui.Center)
	tr := design.TransitionFor("button/button")
	row.Transition(ui.ElementTransition{Duration: tr.Duration, Ease: design.EaseInOut, Colors: true})
	if row.Pressed() || row.Hovered() {
		row.Background(t.Accent).TextColor(t.AccentForeground)
	} else {
		row.TextColor(t.PopoverForeground)
	}
	return row
}

// chevron paints the closed/open marker of a picker chip: rotates the single
// chevron 180deg over 100ms ease-in-out while its popup opens.
func chevron(c *ui.Context, open bool) ui.Element {
	t := design.For(c)
	icon := ui.Icon(c, coss.IconChevronDown.SVG()).Size(design.IconXS, design.IconXS).Shrink(0).TextColor(t.MutedForeground).Opacity(design.OpacityIcon)
	st := ui.Local(icon, "chevron-open", func() struct {
		open  bool
		since time.Time
	} {
		return struct {
			open  bool
			since time.Time
		}{open: open}
	})
	angle := float32(0)
	if open {
		angle = 180
	}
	if st.open != open {
		st.open, st.since = open, c.Now()
	}
	if !st.since.IsZero() && !c.Preferences().ReduceMotion {
		if elapsed := float32(c.Now().Sub(st.since)) / float32(100*time.Millisecond); elapsed < 1 {
			p := design.EaseInOut(elapsed) * 180
			if open {
				angle = p
			} else {
				angle = 180 - p
			}
			c.After(16 * time.Millisecond)
		}
	}
	if angle != 0 {
		icon = icon.Rotate(angle)
	}
	return icon
}

// chip paints the shared closed chrome of the composer's pickers: a transparent
// 28px capsule that fills with coss Accent while hovered or open.
func chip(c *ui.Context, open bool, textColor ui.Color) ui.Element {
	t := design.For(c)
	trigger := ui.ButtonBase(c).Height(chipHeight).Radius(design.RadiusMD).Padding(0, design.Sp(1), 0, design.Sp(2)).
		Gap(design.Sp(1)).AlignItems(ui.Center).MinWidth(0).MaxWidth(220).
		TextColor(textColor).FocusRing(false).Cursor(ui.CursorPointer)
	if trigger.Hovered() || trigger.Pressed() || open {
		trigger.Background(t.Accent).TextColor(t.AccentForeground)
	}
	return trigger
}

// paintInnerBloom stands in for the InputBar resting inner glow: concentric
// hairlines just inside the edge, fading smoothly across 12 DIPs.
func paintInnerBloom(p *ui.Painter, r ui.Rect, radius, peakAlpha float32) {
	if peakAlpha <= 0 {
		return
	}
	white := design.TW.White
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
			alpha := peakAlpha * falloff
			if alpha < 0.002 {
				continue
			}
			p.Stroke(band, white.Alpha(alpha), max(0, radius-d), 0.85)
		}
	})
}

func cossIcon(c *ui.Context, icon coss.Icon, size float32, color ui.Color) ui.Element {
	return ui.Icon(c, icon.SVG()).Size(size, size).Shrink(0).TextColor(color)
}
