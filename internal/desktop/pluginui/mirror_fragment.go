package pluginui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// MirrorFragment retains the full source row and its drawable source CSS area.
// Viewport and Visible use the same main-frame CSS coordinates as Full.
type MirrorFragment struct {
	Full, Visible, Viewport MirrorRect
}

// MirrorNativeFragment maps a source fragment into a uniformly scaled row.
// Destination is the full native row; Clip is its drawable native area.
// SourceRect is the inverse of Clip in source main-frame CSS coordinates.
type MirrorNativeFragment struct {
	Destination ui.Rect
	Clip        ui.Rect
	SourceRect  MirrorRect
	Scale       float64
}

// NewMirrorFragment intersects a row with both the viewport and its ancestor
// clip. All rectangles must have finite coordinates and positive extents.
// A caller without an ancestor clip should pass viewport as ancestorClip.
func NewMirrorFragment(full, viewport, ancestorClip MirrorRect) (MirrorFragment, bool) {
	visible, ok := mirrorFragmentIntersection(full, viewport, ancestorClip)
	if !ok {
		return MirrorFragment{}, false
	}
	return MirrorFragment{Full: full, Visible: visible, Viewport: viewport}, true
}

// MapMirrorFragment uses one scale for both axes, derived from the full row's
// width. Destination height may differ from Full.H*Scale by at most 2e-4
// relative error, accommodating float32 layout rounding without stretching.
//
// A whole zero-origin source bitmap would be painted at
// (dst.X-Scale*Full.X, dst.Y-Scale*Full.Y), then clipped to Clip. If the bitmap's
// origin is Viewport.X/Y, add Scale*Viewport.X/Y to that paint position. No
// bitmap needs to be cropped or independently resized to the fragment's height.
func MapMirrorFragment(dst, nativeClip ui.Rect, fragment MirrorFragment) (MirrorNativeFragment, bool) {
	destination := mirrorFragmentFromNative(dst)
	clip := mirrorFragmentFromNative(nativeClip)
	if !mirrorFragmentNativeValid(dst) || !mirrorFragmentNativeValid(nativeClip) {
		return MirrorNativeFragment{}, false
	}
	visible, ok := mirrorFragmentIntersection(fragment.Full, fragment.Viewport, fragment.Visible)
	if !ok {
		return MirrorNativeFragment{}, false
	}
	scale := destination.W / fragment.Full.W
	height := fragment.Full.H * scale
	const relativeHeightTolerance = 2e-4
	if !mirrorFragmentFinite(scale) || scale <= 0 || !mirrorFragmentFinite(height) || height <= 0 ||
		math.Abs(destination.H-height) > relativeHeightTolerance*math.Max(destination.H, height) {
		return MirrorNativeFragment{}, false
	}
	mapped := MirrorRect{
		X: destination.X + scale*(visible.X-fragment.Full.X),
		Y: destination.Y + scale*(visible.Y-fragment.Full.Y),
		W: visible.W * scale,
		H: visible.H * scale,
	}
	clipped, ok := mirrorFragmentIntersection(destination, clip, mapped)
	if !ok {
		return MirrorNativeFragment{}, false
	}
	// Round inward so float32 clipping cannot expose source pixels outside the
	// actual intersection. Derive input geometry from the published clip.
	published, ok := mirrorFragmentToNative(clipped)
	if !ok {
		return MirrorNativeFragment{}, false
	}
	source := MirrorRect{
		X: fragment.Full.X + (float64(published.X)-destination.X)/scale,
		Y: fragment.Full.Y + (float64(published.Y)-destination.Y)/scale,
		W: float64(published.W) / scale,
		H: float64(published.H) / scale,
	}
	// The mathematical inverse is within visible. Intersect once more to
	// absorb float64 endpoint rounding at source viewport boundaries.
	source, ok = mirrorFragmentIntersection(source, visible)
	if !ok {
		return MirrorNativeFragment{}, false
	}
	return MirrorNativeFragment{Destination: dst, Clip: published, SourceRect: source, Scale: scale}, true
}

// MirrorSourceScrollDelta preserves the requested offset within a source row.
// rowTop may be negative; the relative offset and viewport target must be
// finite and nonnegative. The caller validates the offset against row height.
func MirrorSourceScrollDelta(rowTop, relativeCSSOffset, targetViewportY float64) (float64, bool) {
	if !mirrorFragmentFinite(rowTop) || !mirrorFragmentFinite(relativeCSSOffset) ||
		!mirrorFragmentFinite(targetViewportY) || relativeCSSOffset < 0 || targetViewportY < 0 {
		return 0, false
	}
	delta := rowTop + relativeCSSOffset - targetViewportY
	if !mirrorFragmentFinite(delta) {
		return 0, false
	}
	return delta, true
}

func mirrorFragmentFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func mirrorFragmentValid(rect MirrorRect) bool {
	return mirrorFragmentFinite(rect.X) && mirrorFragmentFinite(rect.Y) &&
		mirrorFragmentFinite(rect.W) && mirrorFragmentFinite(rect.H) && rect.W > 0 && rect.H > 0 &&
		mirrorFragmentFinite(rect.X+rect.W) && mirrorFragmentFinite(rect.Y+rect.H) &&
		rect.X+rect.W > rect.X && rect.Y+rect.H > rect.Y
}

func mirrorFragmentIntersection(rects ...MirrorRect) (MirrorRect, bool) {
	if len(rects) == 0 {
		return MirrorRect{}, false
	}
	left, top, right, bottom := math.Inf(-1), math.Inf(-1), math.Inf(1), math.Inf(1)
	for _, rect := range rects {
		if !mirrorFragmentValid(rect) {
			return MirrorRect{}, false
		}
		left, top = math.Max(left, rect.X), math.Max(top, rect.Y)
		right, bottom = math.Min(right, rect.X+rect.W), math.Min(bottom, rect.Y+rect.H)
	}
	result := MirrorRect{X: left, Y: top, W: right - left, H: bottom - top}
	return result, mirrorFragmentValid(result)
}

func mirrorFragmentFromNative(rect ui.Rect) MirrorRect {
	return MirrorRect{X: float64(rect.X), Y: float64(rect.Y), W: float64(rect.W), H: float64(rect.H)}
}

func mirrorFragmentNativeValid(rect ui.Rect) bool {
	r := mirrorFragmentFromNative(rect)
	return mirrorFragmentValid(r) && math.Abs(r.X+r.W) <= math.MaxFloat32 && math.Abs(r.Y+r.H) <= math.MaxFloat32
}

func mirrorFragmentToNative(rect MirrorRect) (ui.Rect, bool) {
	ceil := func(value float64) float32 {
		rounded := float32(value)
		if float64(rounded) < value {
			rounded = math.Nextafter32(rounded, float32(math.Inf(1)))
		}
		return rounded
	}
	floor := func(value float64) float32 {
		rounded := float32(value)
		if float64(rounded) > value {
			rounded = math.Nextafter32(rounded, float32(math.Inf(-1)))
		}
		return rounded
	}
	if !mirrorFragmentValid(rect) || math.Abs(rect.X) > math.MaxFloat32 || math.Abs(rect.Y) > math.MaxFloat32 ||
		math.Abs(rect.X+rect.W) > math.MaxFloat32 || math.Abs(rect.Y+rect.H) > math.MaxFloat32 {
		return ui.Rect{}, false
	}
	x, y := ceil(rect.X), ceil(rect.Y)
	right, bottom := floor(rect.X+rect.W), floor(rect.Y+rect.H)
	result := ui.Rect{X: x, Y: y, W: floor(float64(right) - float64(x)), H: floor(float64(bottom) - float64(y))}
	return result, mirrorFragmentNativeValid(result)
}
