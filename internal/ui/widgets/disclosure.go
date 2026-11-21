package widgets

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// The metrics of a DisclosureRow: a 24 DIP line holding a 16 DIP leading slot
// with a 14 DIP glyph, the title, a 2 DIP separator dot and a summary that
// takes what is left, as the design system's flow rows do.
const (
	disclosureHeight     = 24
	disclosureLeading    = 16
	disclosureGlyph      = 14
	disclosureGap        = 6
	disclosureDot        = 2
	disclosureDotMargin  = 8
	disclosureBodyIndent = 22
)

// DisclosureRowOptions describes a DisclosureRow.
type DisclosureRowOptions struct {
	// Icon is the leading glyph, drawn at 14 DIPs in the 16 DIP slot. It
	// gives way to the chevron while the row is hovered, and while it is
	// open, as the design system's flow rows do. Nil draws the chevron.
	Icon *ui.SVG
	// Leading builds leading content of your own instead of Icon, and keeps
	// the chevron out of the way, as a StateDot there needs.
	Leading func()
	// Title is the row's label, as "Thinking" or a tool's name. The caller
	// owns the words.
	Title string
	// Summary is the rest of the line, truncated with an ellipsis when it
	// does not fit. It hides once the row opens unless KeepSummaryWhenOpen.
	Summary string
	// Status describes the row for assistive technology, as "Failed" or
	// "Running", when the title does not say it.
	Status string
	// Open is the row's open state, owned by the caller; the row pretends
	// to be a plain line while it is nil. A click, Enter or Space toggles
	// it.
	Open *bool
	// Running sweeps a highlight across the title and the summary, for work
	// that is still going on.
	Running bool
	// KeepSummaryWhenOpen keeps the summary beside the title while the row
	// is open, as a tool row does.
	KeepSummaryWhenOpen bool
	// Size is the text's size in DIPs, 13 when 0.
	Size float32
	// Color is the title's color, the theme's muted text when zero, which
	// the theme's secondary text is: flow rows are quiet until hovered.
	Color ui.Color
	// SummaryColor is the summary's color, Color when zero.
	SummaryColor ui.Color
	// Body builds what the row reveals, indented under its title and
	// animated open and shut.
	Body func()
	// Indent is how far Body is indented, 22 DIPs (the leading slot and the
	// gap) when 0.
	Indent float32
	// Label names the row for assistive technology instead of Title, for a
	// row whose visible text does not say what it does.
	Label string
	// Trailing builds what follows the summary without being cut with it,
	// as a change's "+3 −1" counts, or a chevron of your own.
	Trailing func()
	// OnToggle is told the new state once a click or a key toggles the row.
	OnToggle func(open bool)
}

// DisclosureRow is one line that opens and shuts what it summarizes, as a
// tool call, a reasoning block or a turn's process does: a leading glyph, a
// title, a 2 DIP dot, a summary truncated to the room left and a chevron that
// turns as the row opens. The whole 24 DIP line is the target — a click, or
// Enter or Space while it has the keyboard focus — and it takes a quiet fill
// while hovered.
//
// The body opens below the title, indented under it, growing and clipping as
// it goes so that nothing pops:
//
//	open := app.openTools[id]
//	widgets.DisclosureRow(c, widgets.DisclosureRowOptions{
//		Icon:     icons.Terminal,
//		Title:    "bash",
//		Summary:  "go test ./internal/ui/widgets",
//		Open:     &open,
//		Running:  running,
//		Body:     func() { terminalBlock(c, output) },
//		OnToggle: func(open bool) { app.openTools[id] = open },
//	})
//
// Without motion — the desktop's reduce-motion setting — the chevron turns at
// once and the body takes its room at once.
func DisclosureRow(c *ui.Context, o DisclosureRowOptions) ui.Element {
	t := c.Theme()
	size := o.Size
	if size <= 0 {
		size = 13
	}
	titleColor := o.Color
	if titleColor.A == 0 {
		titleColor = t.TextMuted
	}
	summaryColor := o.SummaryColor
	if summaryColor.A == 0 {
		summaryColor = titleColor
	}
	name := o.Label
	if name == "" {
		name = o.Title
	}
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(0)
	if o.Open == nil {
		root.Children(func() {
			row := ui.Row(c).FillWidth().MinWidth(0).Height(disclosureHeight).AlignItems(ui.Center)
			if name != "" {
				row.Label(name)
			}
			if o.Status != "" {
				row.Description(o.Status)
			}
			disclosureLine(c, o, row, t, size, titleColor, summaryColor, 0, false, false)
		})
		return root
	}
	root.Children(func() {
		parts := ui.CollapsibleBase(c, o.Open)
		row := parts.Trigger.FillWidth().MinWidth(0).Height(disclosureHeight).AlignItems(ui.Center).FocusRing(false)
		if name != "" {
			row.Label(name)
		}
		if o.Status != "" {
			row.Description(o.Status)
		}
		progress := parts.Progress()
		// Asked while building, which is what tells the window to track the
		// pointer over such a line at all.
		hovered := row.Hovered()
		disclosureLine(c, o, row, t, size, titleColor, summaryColor, progress, hovered, true)
		row.Draw(func(p *ui.Painter, r ui.Rect) {
			// Under the glyphs, which are children.
			if hovered {
				p.Fill(r, t.SurfaceHover.Alpha(0.5), 4)
			}
		})
		row.DrawOver(func(p *ui.Painter, r ui.Rect) {
			// Inside the line, where a scrolling transcript cannot clip it.
			if row.FocusVisible() {
				p.Stroke(ui.Rect{X: r.X + 1, Y: r.Y + 2, W: r.W - 2, H: r.H - 4}, t.Focus, 4, 2)
			}
		})
		if o.Body != nil {
			if panel := parts.Panel(o.Body); panel.Valid() {
				indent := o.Indent
				if indent <= 0 {
					indent = disclosureBodyIndent
				}
				panel.Padding(4, 0, 4, indent)
			}
		}
		if o.OnToggle != nil && row.Changed() {
			o.OnToggle(*o.Open)
		}
	})
	return root
}

// disclosureLine builds the leading slot, title and summary of a row, which
// is open by progress and interactive when its keys and pointer work.
func disclosureLine(c *ui.Context, o DisclosureRowOptions, row ui.Element, t *ui.Theme, size float32, titleColor, summaryColor ui.Color, progress float32, hovered, interactive bool) {
	leadingColor := titleColor
	if hovered {
		// Hovered, the line reads one step louder, as the design system's
		// flow rows do, and clears the fill it takes.
		leadingColor = leadingColor.Mix(t.Text, 0.35)
		summaryColor = summaryColor.Mix(t.Text, 0.35)
	}
	row.Children(func() {
		leading := ui.Box(c).Size(disclosureLeading, disclosureLeading).Shrink(0).Center().Margin(0, disclosureGap, 0, 0)
		leading.Children(func() {
			if o.Leading != nil {
				o.Leading()
				return
			}
			if !interactive {
				if o.Icon != nil {
					ui.Icon(c, o.Icon).Width(disclosureGlyph).Height(disclosureGlyph).TextColor(leadingColor)
				}
				return
			}
			fade := float32(1)
			if o.Icon != nil {
				// The glyph gives way to the chevron on hover, and while open.
				fade = SwapMotion.Animate(row, "chevron", boolF32(progress > 0.05 || hovered))
				ui.Icon(c, o.Icon).Width(disclosureGlyph).Height(disclosureGlyph).TextColor(leadingColor).Opacity(1 - fade)
			}
			angle := 180 * progress
			leading.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if fade <= 0.01 {
					return
				}
				paintChevron(p, r, angle, leadingColor.Alpha(fade))
			})
		})
		if o.Title != "" {
			if o.Running {
				TextShimmer(c, TextShimmerOptions{Text: o.Title, Active: true, Size: size, Color: leadingColor}).Shrink(0)
			} else {
				ui.Text(c, o.Title).FontSize(size).LineHeight(disclosureHeight / size).TextColor(leadingColor).SingleLine().Shrink(0)
			}
		}
		if disclosureShowsSummary(o) {
			ui.Box(c).Size(disclosureDot, disclosureDot).Radius(disclosureDot/2).
				Background(t.TextMuted.Alpha(0.5)).Shrink(0).
				Margin(0, disclosureDotMargin, 0, disclosureDotMargin)
			if o.Running {
				TextShimmer(c, TextShimmerOptions{Text: o.Summary, Active: true, Size: size, Color: summaryColor}).Grow(1).MinWidth(0)
			} else {
				ui.Text(c, o.Summary).FontSize(size).LineHeight(disclosureHeight / size).TextColor(summaryColor).SingleLine().Grow(1).MinWidth(0)
			}
		}
		if o.Trailing != nil {
			// Past the summary, so that a count keeps its whole value however
			// narrow the line gets.
			trailing := ui.Box(c).Shrink(0).Center().Margin(8, 0, 0, 0)
			trailing.Children(o.Trailing)
		}
	})
}

// disclosureShowsSummary reports whether a row shows its summary, which a
// collapsed one always does.
func disclosureShowsSummary(o DisclosureRowOptions) bool {
	if o.Summary == "" {
		return false
	}
	if o.Open == nil {
		return true
	}
	return !*o.Open || o.KeepSummaryWhenOpen
}

// paintChevron paints a chevron pointing down at deg 0 and up at deg 180,
// centered in r.
func paintChevron(p *ui.Painter, r ui.Rect, deg float32, color ui.Color) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	sin, cos := math.Sincos(float64(deg) * math.Pi / 180)
	at := func(x, y float32) (float32, float32) {
		return cx + x*float32(cos) - y*float32(sin), cy + x*float32(sin) + y*float32(cos)
	}
	half, drop := float32(3.6), float32(2)
	x0, y0 := at(-half, -drop)
	x1, y1 := at(0, drop)
	x2, y2 := at(half, -drop)
	var path ui.Path
	path.MoveTo(x0, y0).LineTo(x1, y1).LineTo(x2, y2)
	p.StrokePath(&path, 1.5, color)
}

// boolF32 returns 1 for true and 0 for false.
func boolF32(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
