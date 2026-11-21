package frame

import (
	"github.com/egoist/mygo/ui"
	"math"
)

// TranscriptWidth is the committed local preference. The gesture never writes
// pane dimensions or persists a viewport concession before release.
type conversationWidthGesture struct {
	handles                       [2]ui.Handle
	binding, side                 string
	origin, base, preview, column float32
}

func resolvedConversationWidth(column, preference float32) float32 {
	if preference <= 0 || math.IsNaN(float64(preference)) || math.IsInf(float64(preference), 0) {
		return conversationContentWidth(column)
	}
	return min(max(float32(640), column-176), max(float32(640), preference))
}
func (s *LayoutSizes) conversationWidth(column float32) float32 {
	preference := s.TranscriptWidth
	if s.transcriptGesture.side != "" {
		preference = s.transcriptGesture.preview
	}
	return resolvedConversationWidth(column, preference)
}
func (s *LayoutSizes) startConversationWidth(side string, x, column float32) {
	g := &s.transcriptGesture
	g.side, g.origin, g.column = side, x, column
	g.base = resolvedConversationWidth(column, s.TranscriptWidth)
	g.preview = g.base
}
func (s *LayoutSizes) dragConversationWidth(x float32) {
	g := &s.transcriptGesture
	delta := x - g.origin
	if g.side == "left" {
		delta = -delta
	}
	g.preview = resolvedConversationWidth(g.column, g.base+2*delta)
}
func (s *LayoutSizes) endConversationWidth(x float32, commit bool) {
	g := &s.transcriptGesture
	if g.side == "" {
		return
	}
	if commit && x != g.origin {
		s.dragConversationWidth(x)
		s.TranscriptWidth = g.preview
	}
	g.side = ""
}

// Width rails sit outside the message body and remain clear of pane dividers
// and the scrollbar. In a narrow pane they disappear instead of forcing width.
func conversationWidthRails(column, content float32) (left, right, width float32) {
	margin := (column - content) / 2
	width = max(float32(0), min(float32(10), margin-48))
	return max(float32(0), margin-24-width), min(column, margin+content+24), width
}
func paintConversationWidthControls(c *ui.Context, m Model, body ui.Element, content float32, active bool, scroll *ui.ScrollState) {
	if m.Sizes == nil {
		return
	}
	s := m.Sizes
	g := &s.transcriptGesture
	if g.binding != m.FileBindingID || !active {
		g.side = ""
		g.binding = m.FileBindingID
	}
	if !active {
		return
	}
	column := body.Bounds().W
	if column <= 0 {
		column = s.Center
	}
	left, right, width := conversationWidthRails(column, min(content, max(float32(0), column-64)))
	if width <= 0 {
		g.side = ""
		return
	}
	for index, side := range []string{"left", "right"} {
		ref := &g.handles[index]
		side := side
		x := left
		if side == "right" {
			x = right
		}
		handle := ui.Box(c.Key("transcript-width-" + side)).Absolute().Left(x).Top(0).Bottom(0).Width(width).Cursor(ui.CursorResizeEW).Focusable().Role(ui.RoleSplitter).Label("Resize transcript " + side)
		handle.Bind(ref)
		handle.Range(640, float64(max(float32(640), column-176)), float64(s.conversationWidth(column))).Step(20)
		handle.HandleInput(func(ev ui.InputEvent) bool {
			pointer := ref.Bounds(c).X + ev.X
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button != 0 {
					return false
				}
				ref.Focus(c)
				s.startConversationWidth(side, pointer, column)
				return true
			case ui.InputPointerMove:
				if g.side != side {
					return false
				}
				s.dragConversationWidth(pointer)
				return true
			case ui.InputPointerUp:
				if g.side != side {
					return false
				}
				s.endConversationWidth(pointer, ref.Focused(c))
				return true
			case ui.InputKeyDown:
				if ev.Key == ui.KeyEscape && g.side == side {
					s.endConversationWidth(pointer, false)
					return true
				}
			case ui.InputScroll:
				if ev.Mods&ui.Ctrl != 0 || ev.DY == 0 || m.ChatList == nil || scroll == nil {
					return false
				}
				scroll.Y += ev.DY
				return true
			}
			return false
		})
		for _, key := range []ui.Key{ui.KeyLeft, ui.KeyRight} {
			if handle.Shortcut(0, key) {
				delta := float32(20)
				if key == ui.KeyLeft {
					delta = -delta
				}
				if side == "left" {
					delta = -delta
				}
				s.TranscriptWidth = resolvedConversationWidth(column, s.conversationWidth(column)+delta)
			}
		}
		if handle.Shortcut(0, ui.KeyEscape) {
			s.endConversationWidth(g.origin, false)
		}
		handle.Draw(func(p *ui.Painter, r ui.Rect) {
			if !handle.Hovered() && g.side != side && !handle.FocusVisible() {
				return
			}
			_, y, over := handle.PointerPosition()
			if !over {
				y = r.H / 2
			}
			y = max(float32(12), min(r.H-12, y))
			p.Fill(ui.Rect{X: r.X + (r.W-2)/2, Y: r.Y + y - 12, W: 2, H: 24}, c.Theme().Accent, 1)
			if handle.FocusVisible() {
				p.FocusRing(r, [4]float32{})
			}
		})
	}
}
