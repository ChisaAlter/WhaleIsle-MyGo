//go:build windows && (amd64 || arm64)

package windows

import (
	"log"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/d3d11/roundclip"
)

const (
	dwmwaNCRenderingPolicy  = 2
	dwmwaCornerPreference   = 33
	dwmNCRenderingDisabled  = 1
	dwmDoNotRound           = 1
	wmDwmNCRenderingChanged = 0x031F
	wmRestoreRoundedFrame   = 0x85D3
)

// frameRadius is an opt-in for native UI; no existing window changes its
// redirection mode, frame policy, or rendering path without all three options.
func (w *window) frameRadius() int {
	if !w.opts.Transparent || !w.opts.Surface {
		return 0
	}
	return max(w.opts.WindowsFrameCornerRadius, 0)
}

// configureRoundedFrame keeps WS_CAPTION and WS_THICKFRAME (native motion,
// resize and Snap), suppresses DWM's rectangular non-client backing, and tells
// DWM not to round the frame.
//
// The silhouette is the content's own, one contour: the 20 DIP anti-aliased
// clip of applyFrameClip with the app's hairline ring inside it, which is what
// the original draws in its page on a transparent window with
// roundedCorners:false. Asking DWM to round as well (DWMWCP_ROUND) put a second
// outline of the system's 8 DIP radius under the clip: the corners showed two
// curves with a light wedge between them, a 1 px light line along the top edge
// and a shadow shaped by the smaller radius. With DoNotRound DWM paints no
// frame, border or shadow for this window, so only the content shows, like the
// original.
func (w *window) configureRoundedFrame() error {
	if w.frameRadius() <= 0 {
		return nil
	}
	policy := uint32(dwmNCRenderingDisabled)
	if hr, _, _ := procDwmSetWindowAttribute.Call(w.hwnd, dwmwaNCRenderingPolicy, uintptr(unsafe.Pointer(&policy)), 4); failed(hr) {
		return hresultError("disabling rounded window non-client rendering", hr)
	}
	if windows11() {
		corner := uint32(dwmDoNotRound)
		if hr, _, _ := procDwmSetWindowAttribute.Call(w.hwnd, dwmwaCornerPreference, uintptr(unsafe.Pointer(&corner)), 4); failed(hr) {
			return hresultError("leaving the window frame unrounded", hr)
		}
	}
	return nil
}

func (w *window) roundedFrameMessage(m uint32, wp, lp uintptr) (uintptr, bool) {
	if w.frameRadius() <= 0 {
		return 0, false
	}
	switch m {
	case wmNCHitTest:
		if hit := w.roundedFrameHit(lp); hit != 0 {
			return hit, true
		}
	case wmDwmNCRenderingChanged:
		if wp != 0 && !w.roundedFramePending {
			// Like the original's setImmediate: the operation enabling DWM
			// frame paint must finish before we restore the alpha contract.
			w.roundedFramePending = true
			postMessage(w.hwnd, wmRestoreRoundedFrame, 0, 0)
		}
	case wmRestoreRoundedFrame:
		w.roundedFramePending = false
		if err := w.configureRoundedFrame(); err != nil {
			// Existing compositor logging reports presentation failures; this
			// policy failure is separate and must not silently claim alpha.
			log.Printf("mygo: cannot restore transparent window frame: %v", err)
		}
		return 0, true
	}
	return 0, false
}

// Removing non-client insets must not remove native sizing. The invisible
// frame still gets Windows' sizing hit codes, on both the parent and surface.
func (w *window) roundedFrameHit(lp uintptr) uintptr {
	if w.frameRadius() <= 0 || w.fullScreen || w.IsMaximized() || !w.hasStyle(wsThickFrame) {
		return 0
	}
	var bounds rect
	procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&bounds)))
	x, y := getXParam(lp), getYParam(lp)
	border := resizeBorder(dpiOf(w.hwnd))
	left, right := x < bounds.Left+border, x >= bounds.Right-border
	top, bottom := y < bounds.Top+border, y >= bounds.Bottom-border
	switch {
	case top && left:
		return htTopLeft
	case top && right:
		return htTopRight
	case bottom && left:
		return htBottomLeft
	case bottom && right:
		return htBottomRight
	case left:
		return htLeft
	case right:
		return htRight
	case top:
		return htTop
	case bottom:
		return htBottom
	}
	return 0
}

func (s *surface) roundedSurfaceMessage(m uint32, wp, lp uintptr) (uintptr, bool) {
	if s.w.frameRadius() <= 0 {
		return 0, false
	}
	switch m {
	case wmNCHitTest:
		if hit := s.w.roundedFrameHit(lp); hit != 0 {
			return hit, true
		}
	case wmNCLButtonDown, wmNCLButtonDblClk:
		if wp >= htLeft && wp <= htBottomRight {
			r, _, _ := procSendMessageW.Call(s.w.hwnd, uintptr(m), wp, lp)
			return r, true
		}
	}
	return 0, false
}

// frameClip is independent of captions: only the native content's visual is
// clipped. Caption buttons already sit inside the silhouette's safe insets.
type frameClip struct{ clip roundclip.Clip }

func (c *compositor) applyFrameClip(dev *composition, width, height int32) error {
	s := c.b.surfaces[c.hwnd]
	if s == nil || s.w.frameRadius() <= 0 {
		return nil
	}
	if c.frame == nil {
		c.frame = &frameClip{}
	}
	radius := float32(s.w.frameRadius()) * float32(dpiOf(s.w.hwnd)) / 96
	if s.w.IsMaximized() || s.w.fullScreen {
		radius = 0
	}
	_, err := c.frame.clip.Apply(dev.dcomp, c.visual, float32(width), float32(height), radius)
	return err
}

func (c *compositor) freeFrameClip() {
	if c.frame != nil {
		c.frame.clip.Close()
		c.frame = nil
	}
}
