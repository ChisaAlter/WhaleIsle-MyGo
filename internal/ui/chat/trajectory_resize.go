package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

type trajectoryResize struct {
	width, base, origin float32
	dragging            bool
	handle              ui.Handle
}

func trajectoryResizeBounds(pane float32) (low, high float32) {
	high = max(float32(0), pane-280)
	return min(float32(260), high), high
}

func buildTrajectoryRows(messages []session.Message, query string, turns, calls bool) []trajRow {
	s := trajectoryFoldState{}
	return s.rows(trajectoryAllRows(messages, ""), query, turns, calls)
}
func paintTrajectorySplitter(c *ui.Context, s *trajectoryResize, pane, width float32, border ui.Color) {
	low, high := trajectoryResizeBounds(pane)
	sep := ui.Box(c.Key("trajectory-detail-splitter")).Absolute().Right(width-2.5).Top(0).Bottom(0).Width(5).Background(border).Cursor(ui.CursorResizeEW).Focusable().Role(ui.RoleSplitter).Label("Resize trajectory details").Range(float64(low), float64(high), float64(width)).Step(20)
	sep.Bind(&s.handle)
	sep.HandleInput(func(ev ui.InputEvent) bool {
		pointer := s.handle.Bounds(c).X + ev.X
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button == 0 {
				s.handle.Focus(c)
				s.dragging = true
				s.origin = pointer
				s.base = width
				return true
			}
		case ui.InputPointerMove:
			if s.dragging {
				s.width = min(max(low, s.base+s.origin-pointer), high)
				return true
			}
		case ui.InputPointerUp:
			if s.dragging {
				s.dragging = false
				return true
			}
		case ui.InputKeyDown:
			switch ev.Key {
			case ui.KeyLeft:
				s.width = min(width+20, high)
				return true
			case ui.KeyRight:
				s.width = max(low, width-20)
				return true
			case ui.KeyHome:
				s.width = 0
				return true
			case ui.KeyEscape:
				if s.dragging {
					s.width = s.base
					s.dragging = false
					return true
				}
			}
		}
		return false
	})
	if sep.DoubleClicked() {
		s.width = 0
		s.dragging = false
	}
}
