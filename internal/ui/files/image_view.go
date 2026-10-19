package files

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// The view state belongs to the resource tab, independently of the decoded
// image cache: switching files may release pixels but must retain the view.
type imageViewState struct {
	scale    float32 // zero is fit-width; fixed scales are 25–400 percent
	scroll   ui.ScrollState
	viewport ui.Rect
}

func imageFitScale(width, intrinsic float32) float32 {
	if intrinsic <= 0 {
		return 1
	}
	return min(1, max(1, width-24)/intrinsic)
}
func imageZoomStep(scale float32, direction int) float32 {
	step := math.Floor(float64(scale)*4) + 1
	if direction < 0 {
		step = math.Ceil(float64(scale)*4) - 1
	}
	return float32(max(.25, min(4, step/4)))
}
func (v *imageViewState) zoom(previous, next float32) {
	v.zoomAt(previous, next, v.viewport.W/2, v.viewport.H/2)
}
func (v *imageViewState) zoomAt(previous, next, x, y float32) {
	v.scale = next
	if previous > 0 {
		ratio := next / previous
		// The image surface has the same 12-DIP inset at every scale.
		v.scroll.X = max(0, (v.scroll.X+x-12)*ratio-x+12)
		v.scroll.Y = max(0, (v.scroll.Y+y-12)*ratio-y+12)
	}
}

// MyGo normalizes wheel/trackpad deltas into DIPs. The original reader uses
// Ctrl+wheel (including browser pinch synthesis), capped at 40 pixels per event.
func (v *imageViewState) wheel(ev ui.InputEvent, intrinsicWidth float32) bool {
	if ev.Kind != ui.InputScroll || ev.Mods&ui.Ctrl == 0 {
		return false
	}
	previous := v.scale
	if previous == 0 {
		previous = imageFitScale(v.viewport.W, intrinsicWidth)
	}
	delta := max(-40, min(40, float64(ev.DY)))
	next := float32(max(.25, min(4, float64(previous)*math.Exp(-delta*.01))))
	v.zoomAt(previous, next, ev.X, ev.Y)
	return true
}

func paintImageView(c *ui.Context, preview *imagePreview, v *imageViewState) {
	var width, height float32
	if preview.svg != nil {
		width, height = preview.svg.Size()
	} else if preview.bitmap != nil {
		w, h := preview.bitmap.Size()
		width, height = float32(w), float32(h)
	} else if preview.vector != nil {
		width, height = preview.vector.Size()
	}
	if width <= 0 || height <= 0 {
		return
	}
	viewport := v.viewport.W
	if viewport <= 0 {
		viewport = 480
	}
	scale := v.scale
	if scale == 0 {
		scale = imageFitScale(viewport, width)
	}
	ui.Row(c).FillWidth().Gap(6).Padding(4, 12).Children(func() {
		if ui.Button(c, label("缩小", "Zoom out")).Disabled(scale <= .25).Clicked() {
			v.zoom(scale, imageZoomStep(scale, -1))
		}
		ui.MenuButton(c, fmt.Sprintf("%.0f%%", scale*100), func(menu *ui.Menu) {
			for _, preset := range []float32{.25, .5, 1, 1.5, 2} {
				if menu.Item(fmt.Sprintf("%.0f%%", preset*100)).Checked(v.scale == preset).Chosen() {
					v.zoom(scale, preset)
				}
			}
		}).Label(label("图片缩放", "Image zoom"))
		if ui.Button(c, label("放大", "Zoom in")).Disabled(scale >= 4).Clicked() {
			v.zoom(scale, imageZoomStep(scale, 1))
		}
		if ui.Button(c, label("适宽", "Fit width")).Clicked() {
			v.scale = 0
			v.scroll.X = 0
		}
		if ui.Button(c, "100%").Clicked() {
			v.zoom(scale, 1)
		}
	})
	scale = v.scale
	if scale == 0 {
		scale = imageFitScale(viewport, width)
	}
	scroll := ui.ScrollBoth(c).Fill().Grow(1).MinWidth(0).MinHeight(0).TrackScroll(&v.scroll).HandleInput(func(ev ui.InputEvent) bool { return v.wheel(ev, width) })
	if bounds := scroll.Bounds(); bounds.W > 0 {
		if v.viewport.W != bounds.W && v.scale == 0 {
			c.Invalidate()
		}
		v.viewport = bounds
	}
	scroll.Children(func() {
		surface := ui.Box(c).Key(v).Width(width*scale + 24).Height(height*scale + 24).Padding(12).Children(func() {
			if preview.svg != nil {
				ui.Box(c).Width(width * scale).Height(height * scale).DrawOver(func(p *ui.Painter, r ui.Rect) { preview.svg.Paint(p, r, ui.Contain) })
			} else if preview.bitmap != nil {
				ui.Image(c, preview.bitmap).Width(width * scale).Height(height * scale).Fit(ui.Contain)
			} else {
				ui.Image(c, preview.vector).Width(width * scale).Height(height * scale).Fit(ui.Contain)
			}
		})
		if dx, dy, held := surface.Dragged(); held {
			v.scroll.X = max(0, min(v.scroll.MaxX, v.scroll.X-dx))
			v.scroll.Y = max(0, min(v.scroll.MaxY, v.scroll.Y-dy))
		}
	})
}
