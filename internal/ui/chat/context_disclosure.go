package chat

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type contextPresence struct {
	open   bool
	until  time.Time
	height float32
}

type contextDisclosure struct {
	Icon                     *ui.SVG
	Title, Producer, Summary string
	Single                   bool
	Body                     func()
}

// contextBezier follows the reference's CSS cubic timing functions.
func contextBezier(t, x1, x2 float32) float32 {
	lo, hi := float32(0), float32(1)
	for i := 0; i < 18; i++ {
		u := (lo + hi) / 2
		x := 3*(1-u)*(1-u)*u*x1 + 3*(1-u)*u*u*x2 + u*u*u
		if x < t {
			lo = u
		} else {
			hi = u
		}
	}
	u := (lo + hi) / 2
	return 3*(1-u)*u*u + u*u*u
}

func contextFadeEase(t float32) float32 { return contextBezier(min(t*2, 1), .4, .2) }
func contextColorEase(t float32) float32 {
	// CSS ease has y1=.1, unlike the shared fade's y1=0.
	lo, hi := float32(0), float32(1)
	for i := 0; i < 18; i++ {
		u := (lo + hi) / 2
		x := 3*(1-u)*(1-u)*u*.25 + 3*(1-u)*u*u*.25 + u*u*u
		if x < t {
			lo = u
		} else {
			hi = u
		}
	}
	u := (lo + hi) / 2
	return 3*(1-u)*(1-u)*u*.1 + 3*(1-u)*u*u + u*u*u
}

func contextBodyPalette(palette theme.Colors) contextBodyStyle {
	return contextBodyStyle{Font: theme.RenderPreferences().Code.Family, Size: 11, LineHeight: 16,
		Secondary: chrome.Color(palette.Secondary), Tertiary: chrome.Color(palette.Muted),
		Caption: chrome.Color(palette.Caption), Border: chrome.Color(palette.Border),
		Primary: chrome.Color(palette.FG), BorderL1: chrome.Color(palette.BorderL1),
		CodeBlock: chrome.Color(palette.CodeBlock), Hover: chrome.Color(palette.Hover)}
}

// paintContextDisclosure shares the reference's 24px flow row and code well.
// The toolkit retains an inert paint copy on exit; a separate empty seat keeps
// its layout for the reference's 200ms hold, without rebuilding old controls.
func paintContextDisclosure(c *ui.Context, root ui.Element, open *bool, o contextDisclosure, palette theme.Colors) {
	delta := theme.RenderPreferences().Chat.Size - 14
	size := secondaryChatSize()
	muted, secondary, caption := chrome.Color(palette.Muted), chrome.Color(palette.Secondary), chrome.Color(palette.Caption)
	parts := []string{o.Title}
	for _, part := range []string{o.Producer, o.Summary} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	name := strings.Join(parts, " ")
	var row ui.Element
	if o.Single {
		row = ui.Row(c).Label(name)
	} else {
		row = ui.Button(c, "").Label(name).Role(ui.RoleDisclosure).Expanded(*open).
			Padding(0).BorderWidth(0).Background(ui.Transparent).Radius(0)
		if row.Clicked() {
			*open = !*open
			c.Invalidate()
		}
	}
	row.FillWidth().MinWidth(0).Height(24 + delta).AlignItems(ui.Center).Justify(ui.Start).Clip()
	hovered := row.Hovered()
	target := float32(0)
	if hovered {
		target = 1
	}
	ink := muted.Mix(secondary, row.AnimateWith("context-hover", target, 100*time.Millisecond, contextColorEase))
	row.Children(func() {
		ui.Box(c).Size(16+delta, 16+delta).Margin(0, 6, 0, 0).Shrink(0).Center().Children(func() {
			if !o.Single && *open {
				ui.Icon(c, contextChevronIcon).Size(14+delta, 14+delta).Rotate(180).TextColor(secondary).Role(ui.RoleNone)
				return
			}
			if o.Single {
				ui.Icon(c, o.Icon).Size(14+delta, 14+delta).TextColor(ink).Role(ui.RoleNone)
				return
			}
			fade := row.AnimateWith("context-icon-hover", target, 100*time.Millisecond, func(t float32) float32 { return contextBezier(t, .4, .2) })
			ui.Icon(c, o.Icon).Size(14+delta, 14+delta).TextColor(ink).Opacity(1 - fade).Role(ui.RoleNone)
			ui.Icon(c, contextChevronIcon).Size(14+delta, 14+delta).Absolute().TextColor(secondary).Opacity(fade).Role(ui.RoleNone)
		})
		ui.Text(c, o.Title).Font(theme.RenderPreferences().Chat.Family).FontSize(size).FixedLineHeight(24 + delta).TextColor(ink).SingleLine().Shrink(0)
		part := func(text string, grow bool) {
			if text == "" {
				return
			}
			ui.Box(c).Size(2, 2).Radius(1).Background(caption).Margin(0, 8, 0, 8).Shrink(0).Role(ui.RoleNone)
			e := ui.Text(c, text).Font(theme.RenderPreferences().Chat.Family).FontSize(size).FixedLineHeight(24 + delta).TextColor(muted).SingleLine().MinWidth(0)
			if grow {
				e.Grow(1)
			} else {
				e.Shrink(0)
			}
		}
		part(o.Producer, false)
		part(o.Summary, true)
	})
	shown := !o.Single && *open
	presence := ui.Local(root, "context-presence", func() contextPresence { return contextPresence{} })
	if presence.open != shown {
		presence.open = shown
		if !shown {
			presence.until = c.Now().Add(200 * time.Millisecond)
		}
	}
	if !shown {
		if !c.Preferences().ReduceMotion && c.Now().Before(presence.until) {
			ui.Box(c).Key("context-exit-space").FillWidth().Height(presence.height + 8).PassThrough()
			c.After(presence.until.Sub(c.Now()))
		}
		return
	}
	root.Padding(0, 0, 4, 0)
	panel := ui.Column(c).Key("context-body").MinWidth(0).Margin(4, 0, 0, 22+delta).
		Transition(ui.ElementTransition{Duration: 200 * time.Millisecond, Ease: contextFadeEase, Colors: true,
			Enter: &ui.Motion{Opacity: 0}, Exit: &ui.Motion{Opacity: 0}})
	panel.DrawOver(func(_ *ui.Painter, r ui.Rect) { presence.height = r.H })
	panel.Children(func() {
		ui.Scroll(c).ScrollOverflowX().FillWidth().MinWidth(0).MaxHeight(141).Padding(10, 16, 12, 12).
			Background(chrome.Color(palette.CodeBlock)).Radius(12).Children(o.Body)
	})
}
