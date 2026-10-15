package browserpreview

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// This catalog and zoom ladder mirror ui-preview at WhaleIsle 90465b67.
// Device mode changes the native guest viewport; it does not spoof a user agent.
type browserDevicePreset struct {
	Name          string
	Width, Height int
}

var browserDevicePresets = []browserDevicePreset{
	{"iPhone SE", 375, 667}, {"iPhone XR", 414, 896}, {"iPhone 12 Pro", 390, 844},
	{"iPhone 14 Pro Max", 430, 932}, {"Pixel 7", 412, 915}, {"Samsung Galaxy S8+", 360, 740},
	{"Samsung Galaxy S20 Ultra", 412, 915}, {"iPad Mini", 768, 1024}, {"iPad Air", 820, 1180},
	{"iPad Pro", 1024, 1366}, {"Surface Pro 7", 912, 1368}, {"Surface Duo", 540, 720},
	{"Galaxy Z Fold 5", 344, 882}, {"Asus Zenbook Fold", 853, 1280}, {"Samsung Galaxy A51/71", 412, 914},
	{"Nest Hub", 1024, 600}, {"Nest Hub Max", 1280, 800},
}
var browserZoomLevels = [...]float64{0.25, 0.33, 0.5, 0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4, 5}

type browserToolSettings struct {
	Device                  bool
	Width, Height           int
	WidthDraft, HeightDraft string
	Preset                  string
	AspectLocked            bool
	Scheme                  string
}

func normalizedBrowserZoom(z float64) float64 {
	if z <= 0 || math.IsNaN(z) || math.IsInf(z, 0) {
		return 1
	}
	return z
}
func nextBrowserZoom(current float64, direction int) float64 {
	current = normalizedBrowserZoom(current)
	step := len(browserZoomLevels) - 1
	for i, z := range browserZoomLevels {
		if math.Abs(z-current) < 0.001 || z > current {
			step = i
			if math.Abs(z-current) >= 0.001 {
				step--
			}
			break
		}
	}
	if direction > 0 {
		step++
	} else {
		step--
	}
	return browserZoomLevels[max(0, min(step, len(browserZoomLevels)-1))]
}
func validBrowserDimensions(w, h int) bool {
	return w >= 240 && h >= 240 && w <= 3840 && h <= 3840 && int64(w)*int64(h) <= 3840*2160
}
func (s *browserToolSettings) setSize(w, h int, preset string) {
	s.Width, s.Height, s.Preset = w, h, preset
	s.WidthDraft, s.HeightDraft = strconv.Itoa(w), strconv.Itoa(h)
}
func (s *browserToolSettings) rotate() { s.setSize(s.Height, s.Width, s.Preset) }
func (s browserToolSettings) dimensions(width, height string) (int, int, error) {
	w, e1 := strconv.Atoi(strings.TrimSpace(width))
	h, e2 := strconv.Atoi(strings.TrimSpace(height))
	if e1 != nil || e2 != nil {
		return 0, 0, errors.New("宽和高必须是整数")
	}
	if s.AspectLocked && s.Width > 0 && s.Height > 0 {
		if w != s.Width {
			h = int(math.Round(float64(w) * float64(s.Height) / float64(s.Width)))
		} else if h != s.Height {
			w = int(math.Round(float64(h) * float64(s.Width) / float64(s.Height)))
		}
	}
	if !validBrowserDimensions(w, h) {
		return 0, 0, errors.New("宽高须为 240–3840，面积不能超过 3840 × 2160")
	}
	return w, h, nil
}

// Fit the requested CSS viewport times page zoom into the native panel. Like
// the original, this is bounds fitting, not device-metrics or UA emulation.
func fitBrowserDevice(availableW, availableH, width, height int, zoom float64) (x, y, w, h int, scale float64) {
	if availableW <= 0 || availableH <= 0 {
		return 0, 0, 0, 0, 0
	}
	zoom = normalizedBrowserZoom(zoom)
	targetW, targetH := float64(width)*zoom, float64(height)*zoom
	if targetW <= 0 || targetH <= 0 {
		return 0, 0, availableW, availableH, 1
	}
	scale = math.Min(1, math.Min(float64(availableW)/targetW, float64(availableH)/targetH))
	w = max(1, min(availableW, int(math.Floor(targetW*scale))))
	h = max(1, min(availableH, int(math.Floor(targetH*scale))))
	return (availableW - w) / 2, 0, w, h, scale
}
