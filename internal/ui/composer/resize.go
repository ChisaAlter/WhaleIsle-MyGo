package composer

import "github.com/egoist/mygo/ui"

// ResizeState keeps the current gesture local; only a released axis is persisted.
type ResizeState struct {
	Binding                     string
	Height, Width               float32
	savedHeight, savedWidth     float32
	heightPending, widthPending bool
	dragging                    string
	origin, originSize          float32
}

func (s *ResizeState) Adopt(binding string, height, width float32) {
	if s.Binding != binding {
		*s = ResizeState{Binding: binding, Height: height, Width: width, savedHeight: height, savedWidth: width}
		return
	}
	if height != s.savedHeight {
		s.savedHeight, s.heightPending = height, false
		if s.dragging != "height" {
			s.Height = height
		}
	} else if s.dragging != "height" && !s.heightPending {
		s.Height = height
	}
	if width != s.savedWidth {
		s.savedWidth, s.widthPending = width, false
		if s.dragging != "left" && s.dragging != "right" {
			s.Width = width
		}
	} else if s.dragging == "" && !s.widthPending {
		s.Width = width
	}
}

func resizeInputHeight(c *ui.Context, m Model, natural float32) float32 {
	if !m.ResizeEnabled || m.Resize == nil || m.Resize.Height <= 0 {
		return natural
	}
	return min(c.Root().Bounds().H*.7, max(float32(52), m.Resize.Height))
}

func paintResizeHandles(c *ui.Context, m Model, card ui.Element, muted ui.Color) {
	if !m.ResizeEnabled || m.Resize == nil {
		return
	}
	s := m.Resize
	for _, axis := range []string{"height", "left", "right"} {
		axis := axis
		handle := ui.Box(c.Key("composer-resize-" + axis)).Absolute().Focusable().Role(ui.RoleSplitter).
			Label("Resize composer " + axis)
		if axis == "height" {
			handle.Left(28).Right(28).Top(-4).Height(8).Cursor(ui.CursorResizeNS)
		} else {
			handle.Top(24).Bottom(24).Width(8).Cursor(ui.CursorResizeEW)
			if axis == "left" {
				handle.Left(-4)
			} else {
				handle.Right(-4)
			}
		}
		handle.Draw(func(p *ui.Painter, r ui.Rect) {
			if !handle.Hovered() && s.dragging != axis {
				return
			}
			grip := ui.Rect{X: r.X + 3, Y: r.Y + r.H/2 - 12, W: 2, H: 24}
			if axis == "height" {
				grip = ui.Rect{X: r.X + r.W/2 - 16, Y: r.Y + 3, W: 32, H: 2}
			}
			p.Fill(grip, muted, 1)
		}).HandleInput(func(ev ui.InputEvent) bool {
			bounds := handle.Bounds()
			pointer := bounds.X + ev.X
			if axis == "height" {
				pointer = bounds.Y + ev.Y
			}
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button != 0 {
					return false
				}
				handle.Focus()
				s.dragging, s.origin = axis, pointer
				s.originSize = card.Bounds().W
				if axis == "height" {
					s.originSize = float32(36)
					if m.Hero {
						s.originSize = 52
					}
					if m.TextInput != nil {
						s.originSize = m.TextInput.viewportHeight
					}
					if s.Height > 0 {
						s.originSize = s.Height
					}
				}
				return true
			case ui.InputPointerMove:
				if s.dragging != axis {
					return false
				}
				if axis == "height" {
					s.Height = min(c.Root().Bounds().H*.7, max(float32(52), s.originSize+s.origin-pointer))
				} else {
					delta := pointer - s.origin
					if axis == "left" {
						delta = -delta
					}
					s.Width = min(m.ResizeMaxWidth, max(min(float32(280), m.ResizeMaxWidth), s.originSize+delta))
				}
				return true
			case ui.InputPointerUp:
				if s.dragging != axis {
					return false
				}
				s.dragging = ""
				if axis == "height" {
					s.heightPending = true
					if m.OnResizeEnd != nil {
						m.OnResizeEnd("height", s.Height)
					}
				} else {
					s.widthPending = true
					if m.OnResizeEnd != nil {
						m.OnResizeEnd("width", s.Width)
					}
				}
				return true
			case ui.InputKeyDown:
				delta := float32(0)
				if axis == "height" {
					if ev.Key == ui.KeyUp {
						delta = 10
					}
					if ev.Key == ui.KeyDown {
						delta = -10
					}
					if delta == 0 {
						return false
					}
					current := s.Height
					if current <= 0 && m.TextInput != nil {
						current = m.TextInput.viewportHeight
					}
					s.Height = min(c.Root().Bounds().H*.7, max(float32(52), current+delta))
					s.heightPending = true
					if m.OnResizeEnd != nil {
						m.OnResizeEnd("height", s.Height)
					}
				} else {
					if ev.Key == ui.KeyRight {
						delta = 10
					}
					if ev.Key == ui.KeyLeft {
						delta = -10
					}
					if delta == 0 {
						return false
					}
					if axis == "left" {
						delta = -delta
					}
					current := s.Width
					if current <= 0 {
						current = card.Bounds().W
					}
					s.Width = min(m.ResizeMaxWidth, max(min(float32(280), m.ResizeMaxWidth), current+delta))
					s.widthPending = true
					if m.OnResizeEnd != nil {
						m.OnResizeEnd("width", s.Width)
					}
				}
				return true
			}
			return false
		})
	}
}
