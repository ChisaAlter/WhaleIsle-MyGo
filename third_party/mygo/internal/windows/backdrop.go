//go:build windows && (amd64 || arm64)

package windows

import (
	"log"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/d3d11/roundclip"
	"github.com/egoist/mygo/internal/platform"
)

// backdrop fills a window without a redirection bitmap with the color its
// content starts on, through DirectComposition, from the moment the window
// appears. Such
// a window paints nothing of its own before its content's first frame: GDI
// reaches no screen, the frames show through a composition that does not
// exist yet, and what the system paints behind the window shows instead.
// A window that appears a few hundred milliseconds before its first frame
// therefore shows the wrong color, and then cuts to the right one. The
// backdrop is that right color, behind the content; the first frame covers
// it, and ui drops it then.
type backdrop struct {
	dev                     *composition
	target, visual, surface uintptr
	clip                    roundclip.Clip
	color                   platform.Color
	w, h                    int32
	radius                  float32
	px                      []uint32
}

// showBackdrop puts the window's startup color behind its content. It does
// nothing for a window that shows a material, has no redirection bitmap to
// leave blank, or names no color that its content starts on.
func (w *window) showBackdrop() {
	if w.backdrop != nil || !w.noRedirect || w.vibrancy != "" {
		return
	}
	color := w.opts.BackdropColor
	if color == nil && w.bg != nil && w.bg.A != 0 {
		color = w.bg
	}
	if color == nil {
		return
	}
	dev := w.b.composition()
	if dev == nil {
		return
	}
	d := &backdrop{color: *color}
	if err := d.draw(dev, w); err != nil {
		log.Printf("mygo: cannot fill the window with its background color: %v", err)
		d.free()
		return
	}
	w.backdrop = d
}

// DropBackdrop drops the backdrop, which the first frame covers. ui calls it
// once the window has drawn, and a window may be created without one.
func (w *window) DropBackdrop() {
	if w.backdrop != nil {
		w.backdrop.free()
		w.backdrop = nil
	}
}

// sizeBackdrop draws the backdrop again at the window's new size.
func (w *window) sizeBackdrop() {
	if w.backdrop == nil {
		return
	}
	dev := w.b.composition()
	if dev == nil {
		w.DropBackdrop()
		return
	}
	if err := w.backdrop.draw(dev, w); err != nil {
		log.Printf("mygo: cannot resize the window's background: %v", err)
		w.DropBackdrop()
	}
}

func (d *backdrop) draw(dev *composition, w *window) error {
	var client rect
	procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&client)))
	width, height := client.Right-client.Left, client.Bottom-client.Top
	if width <= 0 || height <= 0 {
		return nil
	}
	if d.dev != dev {
		d.free()
		if hr := comCall(dev.dcomp, dcompDevCreateTargetForHwnd, w.hwnd, 1, uintptr(unsafe.Pointer(&d.target))); failed(hr) {
			return hresultError("IDCompositionDevice::CreateTargetForHwnd", hr)
		}
		if hr := comCall(dev.dcomp, dcompDevCreateVisual, uintptr(unsafe.Pointer(&d.visual))); failed(hr) {
			return hresultError("IDCompositionDevice::CreateVisual", hr)
		}
		if hr := comCall(d.target, dcompTargetSetRoot, d.visual); failed(hr) {
			return hresultError("IDCompositionTarget::SetRoot", hr)
		}
		d.dev = dev
	}
	d.radius = float32(w.frameRadius()) * float32(dpiOf(w.hwnd)) / 96
	if w.IsMaximized() || w.fullScreen {
		d.radius = 0
	}
	if d.surface == 0 || width != d.w || height != d.h {
		release(d.surface)
		d.surface = 0
		if hr := comCall(dev.dcomp, dcompDevCreateSurface, uintptr(width), uintptr(height), dxgiFormatB8G8R8A8, dxgiAlphaPremultiplied,
			uintptr(unsafe.Pointer(&d.surface))); failed(hr) {
			return hresultError("IDCompositionDevice::CreateSurface", hr)
		}
		d.w, d.h = width, height
		if hr := comCall(d.visual, dcompVisualSetContent, d.surface); failed(hr) {
			return hresultError("IDCompositionVisual::SetContent", hr)
		}
	}
	var tex uintptr
	var offset point
	if hr := comCall(d.surface, dcompSurfaceBeginDraw, 0, uintptr(unsafe.Pointer(&iidID3D11Texture2D)),
		uintptr(unsafe.Pointer(&tex)), uintptr(unsafe.Pointer(&offset))); failed(hr) {
		return hresultError("IDCompositionSurface::BeginDraw", hr)
	}
	px := d.pixels(int(width) * int(height))
	// The surface holds premultiplied BGRA, one uint32 a pixel.
	c := d.color
	r := uint32(c.R) * uint32(c.A) / 255
	g := uint32(c.G) * uint32(c.A) / 255
	b := uint32(c.B) * uint32(c.A) / 255
	color := b | g<<8 | r<<16 | uint32(c.A)<<24
	for i := range px {
		px[i] = color
	}
	box := [6]uint32{uint32(offset.X), uint32(offset.Y), 0, uint32(offset.X + width), uint32(offset.Y + height), 1}
	comCall(dev.ctx, d3dCtxUpdateSubresource, tex, 0, uintptr(unsafe.Pointer(&box)), uintptr(unsafe.Pointer(&px[0])), uintptr(width*4), 0)
	release(tex)
	if hr := comCall(d.surface, dcompSurfaceEndDraw); failed(hr) {
		return hresultError("IDCompositionSurface::EndDraw", hr)
	}
	if _, err := d.clip.Apply(dev.dcomp, d.visual, float32(width), float32(height), d.radius); err != nil {
		return err
	}
	if hr := comCall(dev.dcomp, dcompDevCommit); failed(hr) {
		return hresultError("IDCompositionDevice::Commit", hr)
	}
	return nil
}

func (d *backdrop) free() {
	d.clip.Close()
	for _, p := range []*uintptr{&d.surface, &d.visual, &d.target} {
		if *p != 0 {
			release(*p)
			*p = 0
		}
	}
	d.dev, d.w, d.h, d.radius, d.px = nil, 0, 0, 0, nil
}

// pixels returns the room for n pixels to fill, kept from one draw to the
// next, as a resize draws again.
func (d *backdrop) pixels(n int) []uint32 {
	if cap(d.px) < n {
		d.px = make([]uint32, n)
	}
	return d.px[:n]
}
