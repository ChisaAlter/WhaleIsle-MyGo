package browserpreview

import (
	"math"
	"strings"
)

const browserResizeRailSize = 10
const browserViewportMaxArea = 3840 * 2160

// Device-toolbar chrome is already above our native canvas. Reserve the same
// ten-DIP left/right/bottom rails as the original, outside the child WebView.
func browserDeviceLayout(canvasW, canvasH, width, height int, zoom float64) (x, y, w, h int, scale float64) {
	aw, ah := canvasW-2*browserResizeRailSize, canvasH-browserResizeRailSize
	if aw <= 0 || ah <= 0 {
		return 0, 0, 0, 0, 0
	}
	x, _, w, h, scale = fitBrowserDevice(aw, ah, width, height, zoom)
	return x + browserResizeRailSize, (ah - h) / 2, w, h, scale
}
func resizeEndRail(start, delta, available float64) float64 {
	edge := start
	if start < available {
		edge = (available + start) / 2
	}
	target := edge + delta
	if target <= available {
		return target*2 - available
	}
	return target
}
func resizeStartRail(start, delta, available float64) float64 {
	if start > available {
		distance := start - available
		if delta <= distance {
			return start - delta
		}
		return available - (delta-distance)*2
	}
	target := (available-start)/2 + delta
	if target >= 0 {
		return available - target*2
	}
	return available - target
}
func finiteResizeDelta(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}
func resizeBrowserRail(width, height int, dx, dy, availableW, availableH, zoom float64, direction string, ratio float64) (int, int) {
	zoom = normalizedBrowserZoom(zoom)
	dx = finiteResizeDelta(dx)
	dy = finiteResizeDelta(dy)
	sw, sh := float64(width)*zoom, float64(height)*zoom
	dw, dh := sw, sh
	if strings.Contains(direction, "east") {
		dw = resizeEndRail(sw, dx, availableW)
	} else if strings.Contains(direction, "west") {
		dw = resizeStartRail(sw, dx, availableW)
	}
	if strings.Contains(direction, "south") {
		dh = resizeEndRail(sh, dy, availableH)
	}
	dx, dy = dw-sw, dh-sh
	if strings.Contains(direction, "west") {
		dx = -dx
	}
	return resizeBrowserFreeform(width, height, dx, dy, zoom, direction, ratio)
}
func resizeBrowserFreeform(width, height int, dx, dy, zoom float64, direction string, ratio float64) (int, int) {
	zoom = normalizedBrowserZoom(zoom)
	dx = finiteResizeDelta(dx)
	dy = finiteResizeDelta(dy)
	horizontal, vertical := 0.0, 0.0
	if strings.Contains(direction, "east") {
		horizontal = dx
	} else if strings.Contains(direction, "west") {
		horizontal = -dx
	}
	if strings.Contains(direction, "south") {
		vertical = dy
	}
	wantW, wantH := float64(width)+horizontal/zoom, float64(height)+vertical/zoom
	if ratio > 0 && !math.IsNaN(ratio) && !math.IsInf(ratio, 0) {
		useW := horizontal != 0 || direction == "east" || direction == "west"
		useH := vertical != 0 || direction == "south"
		primaryW := useW && !useH || (useW == useH && math.Abs(wantW-float64(width))/float64(width) >= math.Abs(wantH-float64(height))/float64(height))
		return resizeBrowserAspect(wantW, wantH, ratio, primaryW)
	}
	w, h := int(math.Round(math.Max(240, math.Min(3840, wantW)))), int(math.Round(math.Max(240, math.Min(3840, wantH))))
	if w*h > browserViewportMaxArea {
		if math.Abs(horizontal) >= math.Abs(vertical) {
			w = browserViewportMaxArea / h
		} else {
			h = browserViewportMaxArea / w
		}
	}
	return w, h
}
func resizeBrowserAspect(wantW, wantH, ratio float64, primaryW bool) (int, int) {
	if primaryW {
		low := math.Ceil(math.Max(240, 240*ratio))
		high := math.Floor(math.Min(3840, math.Min(3840*ratio, math.Sqrt(browserViewportMaxArea*ratio))))
		w := int(math.Min(high, math.Max(low, math.Round(wantW))))
		h := int(math.Round(float64(w) / ratio))
		for w*h > browserViewportMaxArea && float64(w) > low {
			w--
			h = int(math.Round(float64(w) / ratio))
		}
		return w, h
	}
	low := math.Ceil(math.Max(240, 240/ratio))
	high := math.Floor(math.Min(3840, math.Min(3840/ratio, math.Sqrt(browserViewportMaxArea/ratio))))
	h := int(math.Min(high, math.Max(low, math.Round(wantH))))
	w := int(math.Round(float64(h) * ratio))
	for w*h > browserViewportMaxArea && float64(h) > low {
		h--
		w = int(math.Round(float64(h) * ratio))
	}
	return w, h
}
