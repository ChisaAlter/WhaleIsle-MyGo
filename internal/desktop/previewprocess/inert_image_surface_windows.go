//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"errors"
	"math"
)

var imageSurfaceEnable = surfaceUser.NewProc("EnableWindow")
var imageSurfaceRound = surfaceGDI.NewProc("CreateRoundRectRgn")
var imageSurfaceCombine = surfaceGDI.NewProc("CombineRgn")

// PresentImageSurface uses the same verified child as PresentSurface. Disable
// its input so native parent controls retain pointer/zoom/drag ownership, just
// as an inert <img> does inside the original Files/lightbox parent.
func (s *Session) PresentImageSurface(parent uintptr, x, y, w, h, cx, cy, cw, ch int) error {
	return s.PresentImageSurfaceClipped(parent, x, y, w, h, cx, cy, cw, ch, ImageSurfaceClip{})
}

// PresentImageSurfaceClipped keeps the helper out of rounded corners and native
// overlay controls. The region belongs to this verified child only; no general
// window, input or document capability is exposed through the helper pipe.
func (s *Session) PresentImageSurfaceClipped(parent uintptr, x, y, w, h, cx, cy, cw, ch int, shape ImageSurfaceClip) error {
	if err := s.PresentSurface(parent, x, y, w, h, cx, cy, cw, ch); err != nil {
		return err
	}
	s.surfaceMu.Lock()
	defer s.surfaceMu.Unlock()
	if !validSurfaceWindow(s.surfaceHandle, s.surfacePID) {
		return errors.New("SVG image surface owner changed")
	}
	imageSurfaceEnable.Call(s.surfaceHandle, 0)
	if parent == 0 || w <= 0 || h <= 0 || cw <= 0 || ch <= 0 {
		return nil
	}
	dpi, _, _ := surfaceDPI.Call(parent)
	if dpi == 0 {
		dpi = 96
	}
	scale := float64(dpi) / 96
	px := func(v float32) uintptr { return uintptr(int32(math.Round(float64(v) * scale))) }
	left, top := max(0, cx-x), max(0, cy-y)
	right, bottom := min(w, cx+cw-x), min(h, cy+ch-y)
	region, _, _ := surfaceRect.Call(px(float32(left)), px(float32(top)), px(float32(right)), px(float32(bottom)))
	if region == 0 {
		return errors.New("could not create SVG image clip")
	}
	owned := true
	defer func() {
		if owned {
			surfaceDelete.Call(region)
		}
	}()
	makeShape := func(v ImageSurfaceShape) uintptr {
		l, t := v.Bounds.X-float32(x), v.Bounds.Y-float32(y)
		r, b := l+v.Bounds.W, t+v.Bounds.H
		var result uintptr
		if v.Radius > 0 {
			result, _, _ = imageSurfaceRound.Call(px(l), px(t), px(r)+1, px(b)+1, px(2*v.Radius), px(2*v.Radius))
		} else {
			result, _, _ = surfaceRect.Call(px(l), px(t), px(r), px(b))
		}
		return result
	}
	apply := func(v ImageSurfaceShape, mode uintptr) error {
		rgn := makeShape(v)
		if rgn == 0 {
			return errors.New("could not create SVG image shape")
		}
		defer surfaceDelete.Call(rgn)
		ok, _, _ := imageSurfaceCombine.Call(region, region, rgn, mode)
		if ok == 0 {
			return errors.New("could not combine SVG image shape")
		}
		return nil
	}
	if shape.Bounds.W > 0 && shape.Bounds.H > 0 {
		if err := apply(ImageSurfaceShape{shape.Bounds, shape.Radius}, 2); err != nil {
			return err
		}
	}
	for _, hole := range shape.Exclude {
		if err := apply(hole, 4); err != nil {
			return err
		}
	}
	ok, _, _ := surfaceRegion.Call(s.surfaceHandle, region, 1)
	if ok == 0 {
		return errors.New("could not install SVG image shape")
	}
	owned = false
	return nil
}
