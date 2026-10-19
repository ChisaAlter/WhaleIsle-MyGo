package composer

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

type presetIntroduction struct {
	Binding  string
	Started  time.Time
	Finished bool
	Glyphs   []ui.Glyph
}

// Creator's staged choice announces itself once. Glyphs retain their full
// layout width throughout the reveal; only their ink moves and fades.
func paintPresetIntroduction(c *ui.Context, m Model, caption string, fg ui.Color) {
	root := ui.Row(c.Key("preset-introduction")).Gap(4).Grow(1).MinWidth(0).AlignItems(ui.Center)
	state := ui.Local(root, "timeline", func() presetIntroduction { return presetIntroduction{} })
	font := ui.Font{Size: 13, Weight: 500}
	sequence := m.AgentPresetSelection.Introduction
	binding := fmt.Sprintf("%d:%s:%d:%s", m.Snapshot.ComposerHostGeneration, m.Snapshot.SelectedID, sequence, caption)
	if state.Binding != binding {
		*state = presetIntroduction{Binding: binding, Started: c.Now(), Glyphs: ui.Shape(caption, font)}
	}
	count := len([]rune(caption))
	stagger := time.Duration(0)
	if count > 1 {
		stagger = min(40*time.Millisecond, 200*time.Millisecond/time.Duration(count-1))
	}
	elapsed := c.Now().Sub(state.Started)
	duration := 150*time.Millisecond + time.Duration(max(0, count-1))*stagger + 400*time.Millisecond
	animating := sequence != 0 && !state.Finished && count > 0 && !c.Preferences().ReduceMotion && elapsed < duration
	if sequence != 0 && !state.Finished && !animating {
		state.Finished = true
		if m.OnAgentPresetIntroduced != nil {
			m.OnAgentPresetIntroduced(m.Snapshot.ComposerHostGeneration, m.Snapshot.SelectedID, sequence)
		}
	}
	if animating {
		c.AnimationFrame()
	}
	root.Children(func() {
		icon := ui.Text(c, "⌂").Size(16, 16).FontSize(13).TextColor(fg)
		if animating {
			progress := cssBezier(float64(elapsed)/float64(150*time.Millisecond), .16, 1, .3, 1)
			icon.TextColor(ui.Transparent).DrawOver(func(p *ui.Painter, r ui.Rect) {
				size := float32(.5+.5*progress) * 13
				p.Text(r.X+(r.W-size)/2, r.Y+(r.H-size)/2, "⌂", size, fg.Alpha(float32(progress)))
			})
		}
		label := ui.Text(c, caption).Grow(1).MinWidth(0).SingleLine().FontSize(13).FontWeight(500).TextColor(fg)
		if animating {
			label.TextColor(ui.Transparent).DrawOver(func(p *ui.Painter, r ui.Rect) {
				metrics := font.Metrics()
				baseline := r.Y + (r.H-metrics.Ascent-metrics.Descent)/2 + metrics.Ascent
				for _, glyph := range state.Glyphs {
					delay := 150*time.Millisecond + time.Duration(glyph.Cluster)*stagger
					progress := cssBezier(float64(elapsed-delay)/float64(400*time.Millisecond), 0, 0, .58, 1)
					p.Glyphs([]ui.Glyph{glyph}, r.X, baseline+4*float32(1-progress), fg.Alpha(float32(progress)))
				}
			})
		}
	})
}
