package chat

import (
	"math"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

// Conversation row styling. Owns the 24px seat, chevron turn, hover fill,
// focus ring and body animation using design tokens and coss icons.

const (
	disclosureHeight     float32 = 24
	disclosureLeading    float32 = 16
	disclosureGlyph              = design.IconXS
	disclosureDot        float32 = 2
	disclosureBodyIndent float32 = 22
)

// disclosureStyle carries the surface palette for row chrome.
type disclosureStyle struct {
	FG     ui.Color
	Muted  ui.Color
	Hover  ui.Color
	Accent ui.Color
	Size   float32
}

func (s disclosureStyle) size() float32 {
	if s.Size <= 0 {
		return 13
	}
	return s.Size
}

// rowStatus is the five-state status language shared by tool, turn and goal
// rows.
type rowStatus int

const (
	statusNone rowStatus = iota
	statusIdle
	statusRunning
	statusDone
	statusWarning
	statusError
)

func statusLabel(status rowStatus) string {
	switch status {
	case statusRunning:
		return "running"
	case statusDone:
		return "done"
	case statusWarning:
		return "warning"
	case statusError:
		return "error"
	default:
		return "idle"
	}
}

// disclosureRow paints one interactive row. The whole row is the hit target
// and the caller owns the open state.
type disclosureRow struct {
	Key        string
	Icon       *ui.SVG
	IconColor  ui.Color
	Title      string
	TitleColor ui.Color
	Summary    string
	Trailing   string
	// KeepSummary keeps the line readable while open, as tool rows do.
	KeepSummary bool
	Status      rowStatus
	Open        bool
	Expandable  bool
	Running     bool
	OnToggle    func()
}

func paintDisclosureRow(c *ui.Context, d disclosureRow, st disclosureStyle) {
	t := design.For(c)
	size := st.size()
	titleColor := d.TitleColor
	if titleColor.A == 0 {
		titleColor = t.Foreground
	}
	summaryColor := t.MutedForeground
	name := d.Title
	if d.Summary != "" {
		// The whole line reads as one announcement: title then summary.
		name = d.Title + " " + d.Summary
	}
	var root ui.Element
	if d.Key != "" {
		root = ui.Column(c.Key("row:" + d.Key)).FillWidth().MinWidth(0)
	} else {
		root = ui.Column(c).FillWidth().MinWidth(0)
	}
	root.Children(func() {
		interactive := d.Expandable && d.OnToggle != nil
		var row ui.Element
		var progress float32
		var hovered bool
		open := d.Open
		if !interactive {
			row = ui.Row(c).FillWidth().MinWidth(0).Height(disclosureHeight).AlignItems(ui.Center)
		} else {
			parts := ui.CollapsibleBase(c, &open)
			row = parts.Trigger.FillWidth().MinWidth(0).Height(disclosureHeight).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer)
			progress = parts.Progress()
			hovered = row.Hovered()
			row.Draw(func(p *ui.Painter, r ui.Rect) {
				if hovered {
					p.Fill(r, t.Accent.Alpha(0.5), design.RadiusSM)
				}
			})
			row.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if row.FocusVisible() {
					p.Stroke(ui.Rect{X: r.X + 1, Y: r.Y + 2, W: r.W - 2, H: r.H - 4}, t.Ring, design.RadiusSM, design.RingWidth)
				}
			})
			if row.Changed() {
				d.OnToggle()
			}
		}
		if name != "" {
			row.Label(name)
		}
		if d.Status != statusNone {
			row.Description(statusLabel(d.Status))
		}
		leadingColor := titleColor
		lineSummaryColor := summaryColor
		if hovered {
			leadingColor = leadingColor.Mix(t.Foreground, 0.35)
			lineSummaryColor = lineSummaryColor.Mix(t.Foreground, 0.35)
		}
		showsSummary := d.Summary != "" && (!interactive || !open || d.KeepSummary)
		row.Children(func() {
			leading := ui.Box(c).Size(disclosureLeading, disclosureLeading).Shrink(0).Center().Margin(0, design.Sp(1.5), 0, 0)
			leading.Children(func() {
				if d.Status != statusNone {
					chatStateDot(c, d.Status)
					return
				}
				if !interactive {
					if d.Icon != nil {
						ui.Icon(c, d.Icon).Size(disclosureGlyph, disclosureGlyph).TextColor(leadingColor)
					}
					return
				}
				fade := float32(1)
				if d.Icon != nil {
					target := float32(0)
					if progress > 0.05 || hovered {
						target = 1
					}
					fade = row.AnimateWith("coss.chevron", target, design.MotionFast, design.EaseInOut)
					ui.Icon(c, d.Icon).Size(disclosureGlyph, disclosureGlyph).TextColor(leadingColor).Opacity(1 - fade)
				}
				angle := 180 * progress
				leading.DrawOver(func(p *ui.Painter, r ui.Rect) {
					if fade <= 0.01 {
						return
					}
					paintDisclosureChevron(p, r, angle, leadingColor.Alpha(fade))
				})
			})
			if d.Title != "" {
				if d.Running {
					chatTextShimmer(c, textShimmerOptions{Text: d.Title, Active: true, Size: size, Color: leadingColor}).Shrink(0)
				} else {
					ui.Text(c, d.Title).FontSize(size).LineHeight(disclosureHeight / size).TextColor(leadingColor).SingleLine().Shrink(0)
				}
			}
			if showsSummary {
				ui.Box(c).Size(disclosureDot, disclosureDot).Radius(design.RadiusFull).
					Background(t.MutedForeground.Alpha(0.5)).Shrink(0).
					Margin(0, design.Sp(2), 0, design.Sp(2))
				if d.Running {
					chatTextShimmer(c, textShimmerOptions{Text: d.Summary, Active: true, Size: size, Color: lineSummaryColor}).Grow(1).MinWidth(0)
				} else {
					ui.Text(c, d.Summary).FontSize(size).LineHeight(disclosureHeight / size).TextColor(lineSummaryColor).SingleLine().Grow(1).MinWidth(0)
				}
			}
			if d.Trailing != "" {
				ui.Box(c).Shrink(0).Center().Margin(design.Sp(2), 0, 0, 0).Children(func() {
					ui.Text(c, d.Trailing).FontSize(size - 1).TextColor(lineSummaryColor).NoWrap()
				})
			}
		})
	})
}

func paintDisclosureChevron(p *ui.Painter, r ui.Rect, deg float32, color ui.Color) {
	_ = coss.IconChevronDown
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
