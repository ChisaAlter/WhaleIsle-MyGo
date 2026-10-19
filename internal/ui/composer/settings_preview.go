package composer

import (
	"encoding/json"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// SettingsEffectPreview uses the shipped native painters and an isolated draft.
// It has no session, submission, attachment, or Host callbacks.
func SettingsEffectPreview(c *ui.Context, style json.RawMessage, typing bool) {
	box := ui.Column(c.Key("settings-effect-preview")).FillWidth().MinWidth(0).Padding(12).Radius(16).Background(c.Theme().Surface)
	reducedMotion := c.Preferences().ReduceMotion
	if typing {
		fx := typingEffects{enabled: true}
		fx.style = readTypingStyle(style)
		clock := ui.Local(box, "typing-preview-clock", func() typingPreviewClock { return typingPreviewClock{} })
		box.MinHeight(86).DrawOver(func(p *ui.Painter, r ui.Rect) { clock.paint(p, r, fx, reducedMotion) })
		return
	}
	beam := readBeamStyle(style)
	beamState := ui.Local(box, "preview-beam-timeline", func() beamTimeline { return beamTimeline{} })
	box.Radius(22).MinHeight(86).Draw(func(p *ui.Painter, rect ui.Rect) { paintComposerBeam(p, rect, beam, beamState, true, reducedMotion) })
	box.Children(func() {
		ui.Text(c, composerText("智能体正在思考", "Agent is thinking")).FontSize(14).Padding(16).TextColor(c.Theme().Text)
	})
}

type typingPreviewClock struct {
	born, effectAt, caretAt time.Time
	sample, effect, cursor  string
	blink                   bool
}

func (s *typingPreviewClock) paint(p *ui.Painter, r ui.Rect, fx typingEffects, reduced bool) {
	now := p.Now()
	sample := composerText("输入特效", "Typing effects")
	if s.born.IsZero() || s.sample != sample {
		s.born = now
		s.sample = sample
	}
	if s.effect != fx.style.Effect {
		s.effect = fx.style.Effect
		s.effectAt = now
	}
	if s.cursor != fx.style.Cursor || s.blink != fx.style.CursorBlink {
		s.cursor = fx.style.Cursor
		s.blink = fx.style.CursorBlink
		s.caretAt = now
	}
	const step = 150 * time.Millisecond
	chars := []rune(sample)
	cycle := step * time.Duration(len(chars)+1)
	elapsed := now.Sub(s.born)
	phase := elapsed % cycle
	count := int(phase / step)
	if reduced {
		count = len(chars)
	} else {
		p.After(step - phase%step)
	}
	font := ui.Font{Family: theme.TextFamily(theme.CurrentPreferences().Interface.Family), Size: 14}
	echoInk, caretInk, textInk := fx.colors(p.Theme().Text, p.Theme().Accent)
	x, y := r.X+12, r.Y+(r.H-22)/2
	cycleAt := now.Add(-phase)
	for i, char := range chars[:count] {
		glyph := string(char)
		layout := ui.ShapeText(glyph, font, 0)
		w, h := layout.Size()
		p.TextLayout(layout, x, y, textInk)
		if !reduced {
			born := cycleAt.Add(time.Duration(i+1) * step)
			if s.effectAt.After(born) {
				born = s.effectAt
			}
			progress := float64(now.Sub(born)) / float64(typingDuration(fx.style.Speed))
			if progress >= 0 && progress < 1 {
				alpha, dy, scale := typingKeyframe(fx.style.Effect, progress)
				scaledFont := font
				scaledFont.Size *= scale
				echo := ui.ShapeText(glyph, scaledFont, 0)
				ew, eh := echo.Size()
				p.TextLayout(echo, x+(w-ew)/2, y+(h-eh)/2+dy*font.Size, echoInk.Alpha(alpha))
				p.AnimationFrame()
			}
		}
		x += w
	}
	if fx.style.Cursor == "native" {
		return
	}
	if fx.style.CursorBlink && !reduced {
		visible, next := typingCaretBlink(now.Sub(s.caretAt))
		p.After(next)
		if !visible {
			return
		}
	}
	caret := ui.Rect{X: x + 1, Y: y + (22-font.Size*1.1)/2, W: font.Size * .6, H: font.Size * 1.1}
	radius := float32(1)
	if fx.style.Cursor == "underline" {
		caret.Y = y + 11
		caret.H = 2
		radius = 0
	}
	p.Fill(caret, caretInk, radius)
}
