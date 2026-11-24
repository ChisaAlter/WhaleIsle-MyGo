package openui

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func init() {
	Register("Flashcard", flashcardView)
	Register("Celebration", celebrationView)
}

// flashcardView flips between front and back on a click; the flipped state
// persists with the card's state (viewer component state "flipped").
func flashcardView(r *Render, el *lang.Element) ui.Element {
	rt := r.opts.Runtime
	id := el.Identity()
	flipped := lang.Truthy(rt.ComponentState(id, "flipped"))
	t := r.T
	b := ui.ButtonBase(r.C).FillWidth().MinHeight(design.Sp(32)).Padding(design.Sp(5)).Radius(design.RadiusXL).
		Border(design.BorderWidth, t.Border).Background(t.Card).Cursor(ui.CursorPointer).Justify(ui.Center).
		Label(i18n.Pick("翻转卡片", "Flip card"))
	if b.Hovered() {
		b.Background(t.Accent)
	}
	b.Children(func() {
		ui.Column(r.C).FillWidth().AlignItems(ui.Center).Gap(design.Sp(2)).Children(func() {
			side := i18n.Pick("正面", "Front")
			content := el.Prop("front")
			if flipped {
				side, content = i18n.Pick("背面", "Back"), el.Prop("back")
			}
			r.text(side, design.TextXS, design.WeightMedium, t.MutedForeground)
			if s, ok := content.(string); ok {
				r.paragraph(s, design.TextLG, design.WeightSemibold, t.Foreground).TextAlign(ui.Center)
			} else {
				r.Content(content)
			}
			if hint := str(el, "hint"); hint != "" && !flipped {
				r.paragraph(hint, design.TextXS, design.WeightNormal, t.MutedForeground).TextAlign(ui.Center)
			}
		})
	})
	if b.Clicked() {
		rt.SetComponentState(id, "flipped", !flipped)
	}
	return b
}

type celebrationMemo struct {
	was   bool
	start time.Time
	seen  bool
}

const celebrationLength = 1400 * time.Millisecond

// celebrationView bursts confetti over the card when `when` turns true.
// With reduced motion it draws nothing.
func celebrationView(r *Render, el *lang.Element) ui.Element {
	box := ui.Box(r.C).FillWidth().Height(0).PassThrough()
	m := ui.Local(box, "openui.celebration", func() celebrationMemo { return celebrationMemo{} })
	on := boolean(el, "when")
	now := r.C.Now()
	if on && !m.was && m.seen && !r.C.Preferences().ReduceMotion {
		m.start = now
	}
	m.was, m.seen = on, true
	elapsed := now.Sub(m.start)
	if m.start.IsZero() || elapsed >= celebrationLength {
		return box
	}
	r.C.AnimationFrame()
	progress := float32(elapsed) / float32(celebrationLength)
	colors := []ui.Color{r.T.Chart1, r.T.Chart2, r.T.Chart3, r.T.Chart4, r.T.Chart5, r.T.Primary}
	return box.DrawOver(func(p *ui.Painter, rect ui.Rect) {
		const n = 36
		for i := 0; i < n; i++ {
			// Deterministic spread: angle and speed from the index.
			a := float64(i)*2.399963 + 0.3
			speed := float32(80 + (i*37)%120)
			dx := float32(math.Cos(a)) * speed * progress
			dy := -float32(math.Abs(math.Sin(a)))*speed*progress + 260*progress*progress
			x := rect.X + rect.W/2 + dx*1.6
			y := rect.Y + dy - 40
			col := colors[i%len(colors)].Alpha(1 - progress)
			s := float32(4 + i%4)
			p.Fill(ui.Rect{X: x, Y: y, W: s, H: s * 0.6}, col, 1)
		}
	})
}
