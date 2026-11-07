package browserpreview

import (
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/desktop/webview"
)

// Changing the owner invalidates MyGo's pressed-element key as well as the
// resize state, so a held pointer cannot restart a drag after navigation.
type browserResizeRailKey struct {
	tabID            int
	host             *desktop.Host
	epoch            uint64
	zoom             float64
	canvasW, canvasH int
	direction        string
}

type browserViewportDrag struct {
	host                        *desktop.Host
	epoch                       uint64
	direction                   string
	start                       browserToolSettings
	canvasW, canvasH            int
	zoom, effectiveZoom, dx, dy float64
}

func (b *Controller) cancelBrowserResize(t *Tab, restore bool) {
	if t == nil || t.tools.resize == nil {
		return
	}
	drag := t.tools.resize
	t.tools.resize = nil
	if restore {
		t.tools.settings = drag.start
	}
}
func (b *Controller) browserResizeCurrent(t *Tab, drag *browserViewportDrag) bool {
	return drag != nil && b.browserToolsCurrent(t, drag.host, drag.epoch) && t.tools.settings.Device
}

// The rails belong to the native parent surface and stay outside the child
// WebView rectangle. MyGo's Dragged captures pointer deltas across the guest.
func (b *Controller) deviceResizeRails(c *ui.Context, t *Tab, canvas ui.Element, railColor ui.Color) {
	if t == nil || b.current() != t || t.Host == nil || t.Document != nil || t.HTML != nil || t.Media != nil || !t.tools.settings.Device {
		return
	}
	bounds := canvas.Bounds()
	cw, ch := int(bounds.W), int(bounds.H)
	zoom := t.Host.BrowserZoom()
	if drag := t.tools.resize; drag != nil && (!b.browserResizeCurrent(t, drag) || drag.canvasW != cw || drag.canvasH != ch || math.Abs(drag.zoom-zoom) > 0.001) {
		b.cancelBrowserResize(t, true)
	}
	s := &t.tools.settings
	x, y, w, h, scale := browserDeviceLayout(cw, ch, s.Width, s.Height, zoom)
	if w <= 0 || h <= 0 {
		return
	}
	disabled := t.Host.State().Loading || t.tools.operation != nil || b.operation != nil || t.tools.confirmation != nil
	heldAny := false
	for _, spec := range []struct {
		direction, label string
		x, y, w, h       int
		cursor           ui.Cursor
	}{
		{"west", browserPreviewText("viewportRailWest"), x - 10, y, 10, h, ui.CursorResizeEW},
		{"east", browserPreviewText("viewportRailEast"), x + w, y, 10, h, ui.CursorResizeEW},
		{"south", browserPreviewText("viewportRailSouth"), x, y + h, w, 10, ui.CursorResizeNS},
		{"southwest", browserPreviewText("viewportRailSouthwest"), x - 10, y + h, 10, 10, ui.CursorResizeNESW},
		{"southeast", browserPreviewText("viewportRailSoutheast"), x + w, y + h, 10, 10, ui.CursorResizeNWSE},
	} {
		rail := ui.Box(c).Key(browserResizeRailKey{tabID: t.ID, host: t.Host, epoch: t.Host.BrowserPageVersion(), zoom: zoom, canvasW: cw, canvasH: ch, direction: spec.direction}).Absolute().Left(float32(spec.x)).Top(float32(spec.y)).Width(float32(spec.w)).Height(float32(spec.h)).Cursor(spec.cursor).Focusable().Label(spec.label).Tooltip(spec.label + browserChromeLabel("；方向键10px，Shift+方向键50px，Esc撤销本次拖动", "; arrow keys: 10px, Shift+arrows: 50px, Esc: undo this drag")).Disabled(disabled).Background(railColor)
		dx, dy, held := rail.Dragged()
		if held && !disabled {
			heldAny = true
			if t.tools.resize == nil {
				b.CancelAction()
				t.tools.resize = &browserViewportDrag{host: t.Host, epoch: t.Host.BrowserPageVersion(), direction: spec.direction, start: *s, canvasW: cw, canvasH: ch, zoom: zoom, effectiveZoom: zoom * scale}
			}
			drag := t.tools.resize
			if drag != nil && drag.direction == spec.direction && b.browserResizeCurrent(t, drag) {
				rail.Focus()
				drag.dx += float64(dx)
				drag.dy += float64(dy)
				ratio := 0.0
				if drag.start.AspectLocked {
					ratio = float64(drag.start.Width) / float64(drag.start.Height)
				}
				nw, nh := resizeBrowserRail(drag.start.Width, drag.start.Height, drag.dx, drag.dy, float64(cw-20), float64(ch-10), drag.effectiveZoom, spec.direction, ratio)
				if nw != s.Width || nh != s.Height {
					s.setSize(nw, nh, "")
					t.Error = ""
					b.showHost(t.Host)
					b.update()
				}
			}
		}
		if rail.Shortcut(0, ui.KeyEscape) && t.tools.resize != nil {
			b.cancelBrowserResize(t, true)
			b.showHost(t.Host)
			b.update()
			return
		}
		if disabled || t.tools.resize != nil {
			continue
		}
		keyDX, keyDY := 0.0, 0.0
		for _, mods := range []ui.Modifiers{0, ui.Shift} {
			step := 10.0
			if mods == ui.Shift {
				step = 50
			}
			if strings.Contains(spec.direction, "east") || strings.Contains(spec.direction, "west") {
				if rail.Shortcut(mods, ui.KeyLeft) {
					keyDX = -step * zoom
				}
				if rail.Shortcut(mods, ui.KeyRight) {
					keyDX = step * zoom
				}
			}
			if strings.Contains(spec.direction, "south") {
				if rail.Shortcut(mods, ui.KeyUp) {
					keyDY = -step * zoom
				}
				if rail.Shortcut(mods, ui.KeyDown) {
					keyDY = step * zoom
				}
			}
		}
		if keyDX != 0 || keyDY != 0 {
			ratio := 0.0
			if s.AspectLocked {
				ratio = float64(s.Width) / float64(s.Height)
			}
			nw, nh := resizeBrowserFreeform(s.Width, s.Height, keyDX, keyDY, zoom, spec.direction, ratio)
			b.updateBrowserDevice(t, nw, nh, "")
		}
	}
	if t.tools.resize != nil && !heldAny {
		t.tools.resize = nil
	}
}
