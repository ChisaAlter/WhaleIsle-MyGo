package composer

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// Provider-anchored occupancy and heuristic composition are deliberately separate.
// The gauge keeps its seat (a "—%" placeholder) while the Host has not reported
// a context window yet, so the row does not jump when the first frame lands.
func paintContextMeter(c *ui.Context, snapshot session.ViewState, height float32) {
	t := design.For(c)
	muted := t.MutedForeground
	owner := ui.Box(c)
	state := ui.Local(owner, "context-panel", func() contextMeterPanel { return contextMeterPanel{} })
	binding := fmt.Sprintf("%d:%s:%s", snapshot.ComposerHostGeneration, snapshot.PlanHistoryBinding, snapshot.SelectedID)
	if state.Binding != binding {
		*state = contextMeterPanel{Binding: binding}
	}
	value := snapshot.ContextOccupancy
	if value == nil {
		state.Open = false
		// The reference omits the occupancy control until the Host reports a
		// context window; a "—%" placeholder reads as a broken metric.
		return
	}
	reading := fmt.Sprintf("%d%%", value.Percent)
	trigger := ui.ButtonBase(c).Label("Context occupancy").
		Tooltip(fmt.Sprintf(i18n.T(i18n.KeyComposerContextUsed), reading)+"  ·  "+i18n.T(i18n.KeyComposerShortcutEsc)).
		Height(height).Padding(0, design.Sp(2)).Gap(design.Sp(1.5)).Radius(design.RadiusMD).
		BorderWidth(0).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer).
		TextColor(muted)
	if trigger.Hovered() || state.Open {
		trigger = trigger.Background(t.Accent).TextColor(t.AccentForeground)
	}
	track := t.Border
	filled := t.SecondaryForeground
	trigger.Children(func() {
		contextRing(c, value.Percent, track, filled)
		ui.Text(c, reading).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(muted)
	})
	if trigger.Clicked() {
		state.Open = !state.Open
	}
	if !state.Open {
		return
	}
	ui.PopoverBase(c, trigger, &state.Open, func(panel ui.Element) {
		panel = menuSurface(c, panel.Width(264).Padding(design.Sp(2.5), design.Sp(3)).Gap(design.Sp(2)).AttachTo(trigger, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, design.Sp(2), 0))
		panel.Children(func() {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerContextUsed), reading)).FontSize(design.TextSM.Size).TextColor(t.PopoverForeground)
			ui.Text(c, fmt.Sprintf("~%s / %s", compactCount(value.UsedTokens), compactCount(value.ContextWindow))).
				FontSize(design.TextXS.Size).Selectable().TextColor(muted)
			// The bar length follows pressure; estimates only divide that length.
			segments := contextSegments(value, t)
			ui.Row(c).Width(240).Height(6).Background(t.Muted).Radius(design.RadiusFull).Children(func() {
				for _, segment := range segments {
					if segment.width <= 0 {
						continue
					}
					ui.Box(c).Width(segment.width).Height(6).Background(segment.color)
				}
			})
			if value.Breakdown != nil {
				ui.Text(c, i18n.T(i18n.KeyComposerContextBreakdown)).FontSize(design.TextXS.Size).TextColor(muted)
				for _, row := range segments {
					ui.Row(c).FillWidth().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Width(8).Height(8).Radius(design.RadiusXS).Background(row.color)
						ui.Text(c, row.label).Grow(1).FontSize(design.TextXS.Size).TextColor(muted)
						ui.Text(c, "~"+compactCount(row.tokens)).FontSize(design.TextXS.Size).Selectable().TextColor(muted)
					})
				}
			}
		})
	})
}

type contextMeterPanel struct {
	Binding string
	Open    bool
}

type contextSegment struct {
	label  string
	color  ui.Color
	width  float32
	tokens float64
}

// contextSegments splits the gauge's filled length into the heuristic parts, so
// the bar length stays provider-exact while its colors explain the composition.
func contextSegments(value *session.ContextOccupancy, t *design.Tokens) []contextSegment {
	const barWidth = float32(240)
	fill := func(tokens, total float64) float32 {
		if total <= 0 {
			return 0
		}
		return barWidth * float32(value.Percent) / 100 * float32(tokens/total)
	}
	b := value.Breakdown
	if b == nil {
		return []contextSegment{{label: fmt.Sprintf(i18n.T(i18n.KeyComposerContextUsed), fmt.Sprintf("%d%%", value.Percent)), color: t.Primary, width: barWidth * float32(value.Percent) / 100, tokens: value.UsedTokens}}
	}
	total := b.SystemTokens + b.ToolsTokens + b.MessageTokens
	if total <= 0 {
		return []contextSegment{{color: t.Primary, width: barWidth * float32(value.Percent) / 100, tokens: value.UsedTokens}}
	}
	return []contextSegment{
		{label: i18n.T(i18n.KeyComposerContextSystem), color: t.Primary, width: fill(b.SystemTokens, total), tokens: b.SystemTokens},
		{label: i18n.T(i18n.KeyComposerContextTools), color: t.Info, width: fill(b.ToolsTokens, total), tokens: b.ToolsTokens},
		{label: i18n.T(i18n.KeyComposerContextMessages), color: t.MutedForeground, width: fill(b.MessageTokens, total), tokens: b.MessageTokens},
	}
}

// contextRing paints the 14px gauge: a 1.75px track with the occupied arc turned a
// quarter counter-clockwise so it starts at the top.
func contextRing(c *ui.Context, percent int, track, filled ui.Color) ui.Element {
	ring := ui.Box(c).Size(14, 14).Shrink(0)
	ring.Draw(func(p *ui.Painter, r ui.Rect) {
		const strokeWidth = float32(1.75)
		radius := (min(r.W, r.H) - strokeWidth) / 2
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		box := ui.Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius}
		p.Stroke(box, track, radius, strokeWidth)
		if percent <= 0 || radius <= 0 {
			return
		}
		if percent > 100 {
			percent = 100
		}
		steps := max(2, int(float64(percent)*0.36))
		path := &ui.Path{}
		for i := 0; i <= steps; i++ {
			angle := -math.Pi/2 + 2*math.Pi*float64(percent)/100*float64(i)/float64(steps)
			x := cx + radius*float32(math.Cos(angle))
			y := cy + radius*float32(math.Sin(angle))
			if i == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
		}
		p.StrokePath(path, strokeWidth, filled)
	})
	return ring
}
