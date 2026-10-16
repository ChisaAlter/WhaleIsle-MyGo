package previewprocess

import (
	"github.com/egoist/mygo/ui"
	"math"
	"sync"
	"time"
)

// A pixel seat contains only validated decoded images and trusted native input.
// Windows sessions leave this nil and retain their existing HWND presentation.
type pixelSurface struct {
	mu        sync.Mutex
	bitmap    *ui.Bitmap
	width     int
	lastPaint time.Time
	enqueue   func(packet) bool
}

func pixelInput(e ui.InputEvent, width int, seatWidth, seatHeight float32) (interactiveInput, bool) {
	v := interactiveInput{}
	if width < 240 || seatWidth <= 0 || seatHeight <= 0 || math.IsNaN(float64(seatWidth)) || math.IsNaN(float64(seatHeight)) {
		return v, false
	}
	switch e.Kind {
	case ui.InputPointerDown, ui.InputPointerUp, ui.InputPointerMove, ui.InputScroll:
		scale := min(seatWidth/float32(width), seatHeight/interactiveHeight)
		left, top := (seatWidth-float32(width)*scale)/2, (seatHeight-interactiveHeight*scale)/2
		x, y := (e.X-left)/scale, (e.Y-top)/scale
		if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) {
			return v, false
		}
		outside := x < 0 || y < 0 || x >= float32(width) || y >= interactiveHeight
		if outside && e.Kind != ui.InputPointerUp && !(e.Kind == ui.InputPointerMove && e.Button >= 0) {
			return v, false
		}
		v.Kind = "pointer"
		v.X = max(0, min(width-1, int(x)))
		v.Y = max(0, min(interactiveHeight-1, int(y)))
		if e.Kind == ui.InputPointerDown || e.Kind == ui.InputPointerUp {
			if e.Button < 0 || e.Button > 2 {
				return v, false
			}
			v.Button = []int{1, 3, 2}[e.Button]
			v.Down = e.Kind == ui.InputPointerDown
		}
		if e.Kind == ui.InputScroll {
			if e.DY == 0 {
				return v, false
			}
			v.Button = 5
			if e.DY < 0 {
				v.Button = 4
			}
		}
	case ui.InputText:
		v.Kind = "text"
		v.Text = e.Text
	case ui.InputKeyDown:
		if e.Mods != 0 {
			return v, false
		}
		v.Kind = "key"
		v.Key = map[ui.Key]string{ui.KeyEnter: "Return", ui.KeyEscape: "Escape", ui.KeyBackspace: "BackSpace", ui.KeyTab: "Tab", ui.KeyDelete: "Delete", ui.KeyHome: "Home", ui.KeyEnd: "End", ui.KeyPageUp: "Prior", ui.KeyPageDown: "Next", ui.KeyLeft: "Left", ui.KeyRight: "Right", ui.KeyUp: "Up", ui.KeyDown: "Down"}[e.Key]
	default:
		return v, false
	}
	return v, validateInteractiveInput(v, width) == nil
}

// RenderPixels is an optional native seat, detected structurally by the card UI.
// No parent display handle is ever sent to the helper. Contain-fit coordinates
// share precisely the same geometry as Painter.Image, including letterboxing.
func (s *Session) RenderPixels(c *ui.Context, height float32, appearance string, variables map[string]string) bool {
	ps := s.pixels
	if ps == nil {
		return false
	}
	select {
	case <-s.Done():
		return true
	default:
	}
	if err := s.SetTheme(appearance, variables); err != nil {
		go s.Close()
		return true
	}
	c.After(100 * time.Millisecond)
	ps.mu.Lock()
	bitmap := ps.bitmap
	ps.mu.Unlock()
	var seatWidth, seatHeight float32
	ui.Box(c).FillWidth().Height(height).Focusable().TextCaret(ui.Rect{W: 1, H: 20}).Draw(func(p *ui.Painter, r ui.Rect) {
		seatWidth, seatHeight = r.W, r.H
		clip := p.NativeSurfaceClip()
		if min(r.X+r.W, clip.X+clip.W) > max(r.X, clip.X) && min(r.Y+r.H, clip.Y+clip.H) > max(r.Y, clip.Y) {
			ps.mu.Lock()
			ps.lastPaint = time.Now()
			ps.mu.Unlock()
		}
		if bitmap != nil {
			p.Image(bitmap, r, ui.Contain)
		}
	}).HandleInput(func(e ui.InputEvent) bool {
		select {
		case <-s.Done():
			return false
		default:
		}
		ps.mu.Lock()
		active := time.Since(ps.lastPaint) < 300*time.Millisecond
		ps.mu.Unlock()
		if !active {
			return false
		}
		v, ok := pixelInput(e, ps.width, seatWidth, seatHeight)
		if !ok {
			return false
		}
		return ps.enqueue(packet{Kind: "poll", Input: &v})
	})
	return true
}

// Recheck the paint lease at dispatch, not just when the UI queued input.
// Otherwise a queued press can run after the hidden-seat release and leave the
// renderer dragging indefinitely while idle polls are suspended.
func inlinePixelCommand(command packet, active bool, suspended *bool) (packet, bool) {
	if command.Kind != "poll" {
		return command, true
	}
	if active {
		*suspended = false
		return command, true
	}
	if *suspended {
		return packet{}, false
	}
	*suspended = true
	return packet{Kind: "release"}, true
}
