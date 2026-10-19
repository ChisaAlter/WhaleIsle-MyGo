//go:build windows

package ui

import (
	"fmt"
	"image"
	"time"

	"github.com/egoist/mygo/internal/gpu/d3d11"
)

type windowsSplashBackend struct{ *d3d11.Fluid }

func newSplashBackend() (splashBackend, error) {
	fluid, err := d3d11.NewFluid()
	if err != nil {
		return nil, err
	}
	return &windowsSplashBackend{fluid}, nil
}

func (b *windowsSplashBackend) RenderNative() (splashNativeFrame, error) {
	frame, err := b.RenderFrame()
	if frame == nil {
		return nil, err
	}
	return frame, err
}

type windowsSplashPresenter struct {
	renderer     *d3d11.Renderer
	bitmap, next *Bitmap
}

func newNativeSplashPresenter(p *Painter) (splashNativePresenter, error) {
	host, ok := p.rt.host.(*windowHost)
	if !ok {
		return nil, fmt.Errorf("fluid: native GPU presentation requires a window surface")
	}
	// The first surface frame chooses its renderer in present(), after
	// decoration paint. Keep the lease pending until that real choice exists.
	if !host.gpuTried {
		return nil, nil
	}
	renderer, ok := host.gpu.(*d3d11.Renderer)
	if !ok || renderer.Software() {
		return nil, fmt.Errorf("fluid: native hardware presentation is unavailable")
	}
	return &windowsSplashPresenter{renderer: renderer}, nil
}

func (s *windowsSplashPresenter) paint(p *Painter, frame splashNativeFrame, w, h float32) (bool, error) {
	host, ok := p.rt.host.(*windowHost)
	if !ok || host.gpu != s.renderer || s.renderer.Software() {
		return false, fmt.Errorf("fluid: native presentation owner changed")
	}
	ready := false
	if frame != nil {
		native, ok := frame.(*d3d11.FluidFrame)
		if !ok {
			return false, fmt.Errorf("fluid: unsupported native frame lease")
		}
		width, height := native.ImageSize()
		target := s.bitmap
		if target == nil || target.w != width || target.h != height {
			if s.next == nil || s.next.w != width || s.next.h != height {
				// Transparent CPU backing makes a later renderer failure inert.
				// Native pixels never enter it, and ImportFluidFrame preloads
				// the image cache so imageView never uploads this placeholder.
				s.next = NewBitmap(image.NewRGBA(image.Rect(0, 0, width, height)))
			}
			target = s.next
		}
		var err error
		ready, err = s.renderer.ImportFluidFrame(target.img, native)
		if err != nil {
			return false, err
		}
		if ready && target != s.bitmap {
			if s.bitmap != nil {
				s.renderer.ReleaseFluidImage(s.bitmap.img)
			}
			s.bitmap, s.next = target, nil
		}
	}
	if s.bitmap != nil {
		p.Image(s.bitmap, Rect{W: w, H: h}, FillBox)
	}
	return ready, nil
}

func (s *windowsSplashPresenter) close() {
	for _, bitmap := range []*Bitmap{s.bitmap, s.next} {
		if bitmap != nil {
			s.renderer.ReleaseFluidImage(bitmap.img)
		}
	}
	s.bitmap, s.next = nil, nil
}
func nativeSplashFits(width, height int) bool  { return d3d11.FluidFits(width, height) }
func splashPacing(time.Duration) time.Duration { return splashFrameInterval }
