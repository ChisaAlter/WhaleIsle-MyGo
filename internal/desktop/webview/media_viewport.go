package webview

import (
	"errors"
	"slices"

	"github.com/egoist/mygo"
)

type mediaPlacement struct {
	bounds mygo.Rectangle
	holes  []mygo.Rectangle
}

func clippedSurfaceHoles(bounds, visible mygo.Rectangle, occlusions []mygo.Rectangle) []mygo.Rectangle {
	left, top := visible.X-bounds.X, visible.Y-bounds.Y
	right, bottom := left+visible.Width, top+visible.Height
	holes := make([]mygo.Rectangle, 0, len(occlusions)+4)
	for _, r := range []mygo.Rectangle{{Width: bounds.Width, Height: top}, {Y: bottom, Width: bounds.Width, Height: bounds.Height - bottom}, {Y: top, Width: left, Height: visible.Height}, {X: right, Y: top, Width: bounds.Width - right, Height: visible.Height}} {
		if r.Width > 0 && r.Height > 0 {
			holes = append(holes, r)
		}
	}
	for _, r := range occlusions {
		r = intersectMediaRect(r, bounds)
		if r.Width > 0 && r.Height > 0 {
			r.X -= bounds.X
			r.Y -= bounds.Y
			holes = append(holes, r)
		}
	}
	return holes
}

func intersectMediaRect(a, b mygo.Rectangle) mygo.Rectangle {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	right, bottom := min(a.X+a.Width, b.X+b.Width), min(a.Y+a.Height, b.Y+b.Height)
	return mygo.Rectangle{X: x, Y: y, Width: max(0, right-x), Height: max(0, bottom-y)}
}

// PresentMedia is a narrow paint-owned alternative to whole-panel SetViewport.
// Bounds/clip/occlusions are trusted parent-content DIPs, never renderer data.
// Unlike SetViewport it preserves the unclipped player's size when an edge is
// outside the scroll viewport. All invisible pixels AND input are excluded.
// A platform unable to confirm clipping hides the page and reports failure.
// Ordinary Browser and document Hosts never opt into this path.
func (h *Host) PresentMedia(bounds, clip mygo.Rectangle, occlusions []mygo.Rectangle) (bool, error) {
	return h.presentClipped(bounds, clip, occlusions)
}

func (h *Host) presentClipped(bounds, clip mygo.Rectangle, occlusions []mygo.Rectangle) (bool, error) {
	if h == nil {
		return false, errors.New("media: native browser is unavailable")
	}
	shown, result := false, errors.New("media: application UI loop is unavailable")
	mygo.RunOnMain(func() {
		h.mu.Lock()
		w, parent := h.win, h.parent
		private := h.transient && h.gateway == nil && h.localPDFURL == "" && h.localPDFViewerURL == "" && h.localHTMLURL == "" && h.localMediaURL == ""
		if private {
			h.mediaSeat = true
		}
		previous := h.mediaPlacement
		h.mu.Unlock()
		if !private || w == nil || parent == nil || w.IsDestroyed() || parent.IsDestroyed() {
			result = errors.New("media: a live private native browser is required")
			return
		}
		origin := parent.ContentBounds()
		visible := intersectMediaRect(intersectMediaRect(bounds, clip), mygo.Rectangle{Width: origin.Width, Height: origin.Height})
		if !parent.IsVisible() || parent.IsMinimized() || bounds.Width < 1 || bounds.Height < 1 || visible.Width < 1 || visible.Height < 1 {
			w.Hide()
			result = nil
			return
		}
		if bounds.Width > 8192 || bounds.Height > 8192 || bounds.X < -1e7 || bounds.X > 1e7 || bounds.Y < -1e7 || bounds.Y > 1e7 {
			w.Hide()
			result = errors.New("media: native paint bounds exceed the surface range")
			return
		}
		holes := clippedSurfaceHoles(bounds, visible, occlusions)
		screen := bounds
		screen.X += origin.X
		screen.Y += origin.Y
		if previous == nil || previous.bounds != screen || !slices.Equal(previous.holes, holes) {
			// Apply the new clip region before moving/resizing so an already-visible
			// surface never flashes a stale seat region or drops mouse capture.
			if !w.IsVisible() {
				w.Hide()
			}
			if err := clipMediaNativeWindow(w, bounds.Width, bounds.Height, holes); err != nil {
				w.Hide()
				result = err
				return
			}
			w.SetBounds(screen)
			h.mu.Lock()
			h.mediaPlacement = &mediaPlacement{bounds: screen, holes: append([]mygo.Rectangle(nil), holes...)}
			h.mu.Unlock()
		}
		if !w.IsVisible() {
			w.ShowInactive()
		}
		shown, result = true, nil
	})
	return shown, result
}
