//go:build windows

package d3d11

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/egoist/mygo/internal/scene"
)

// Windows SDK dxgi.h: IDXGIResource::GetSharedHandle is slot 8;
// IDXGIKeyedMutex::AcquireSync/ReleaseSync are slots 8/9. Both interfaces
// inherit IDXGIDeviceSubObject. OpenSharedResource is ID3D11Device slot 28.
const (
	fluidSharedKeyedMutex = 0x100
	fluidAcquireSync      = 8
	fluidReleaseSync      = 9
	fluidWaitTimeout      = 258
)

var (
	fluidResourceIID = guid{0x035f3ab4, 0x482e, 0x4e50, [8]byte{0xb4, 0x1f, 0x8a, 0x7f, 0x8b, 0xd8, 0x96, 0x0b}}
	fluidMutexIID    = guid{0x9d8e1289, 0xd7b3, 0x465f, [8]byte{0x81, 0x26, 0x25, 0x0e, 0x34, 0x9a, 0xf8, 0x5d}}
)

// FluidFrame is one immutable GPU handoff. A single receiver owns the lease
// and closes it after importing or abandoning it. Its texture reference keeps
// the legacy DXGI shared handle alive through solver resize/close. The handle
// is not an NT handle and must never be passed to CloseHandle.
type FluidFrame struct {
	texture, shared uintptr
	width, height   int
	once            sync.Once
}

func (f *FluidFrame) ImageSize() (int, int) { return f.width, f.height }

func (f *FluidFrame) Close() {
	if f != nil {
		f.once.Do(func() { free(&f.texture); f.shared = 0 })
	}
}

func (f *Fluid) releaseOutput() {
	free(&f.staging)
	free(&f.outputMutex)
	f.outputShared = 0
	releaseFluidTexture(&f.output)
}

func (f *Fluid) createOutput(width, height int) (err error) {
	f.output, err = f.target(width, height, formatR8G8B8A8Unorm, fluidSharedKeyedMutex)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.releaseOutput()
		}
	}()
	var resource uintptr
	if hr := call(f.output.tex, 0, uintptr(unsafe.Pointer(&fluidResourceIID)), uintptr(unsafe.Pointer(&resource))); failed(hr) {
		return fmt.Errorf("fluid: shared resource: %#x", uint32(hr))
	}
	defer free(&resource)
	if hr := call(resource, 8, uintptr(unsafe.Pointer(&f.outputShared))); failed(hr) || f.outputShared == 0 {
		return fmt.Errorf("fluid: shared output handle: %#x", uint32(hr))
	}
	if hr := call(f.output.tex, 0, uintptr(unsafe.Pointer(&fluidMutexIID)), uintptr(unsafe.Pointer(&f.outputMutex))); failed(hr) {
		return fmt.Errorf("fluid: output mutex: %#x", uint32(hr))
	}
	return nil
}

// RenderFrame submits the display pass without reading pixels. Nil with no
// error means the preceding consumer's GPU copy has not released key 0 yet;
// the worker retries only this handoff, without stepping the simulation again.
// Every AcquireSync uses a zero timeout, including on the worker.
func (f *Fluid) RenderFrame() (*FluidFrame, error) {
	if f.outputMutex == 0 || f.width <= 0 || f.height <= 0 {
		return nil, fmt.Errorf("fluid: output has not been sized")
	}
	if hr := call(f.device, 39); failed(hr) {
		return nil, fmt.Errorf("fluid: device removed: %#x", uint32(hr))
	}
	hr := call(f.outputMutex, fluidAcquireSync, 0, 0)
	if hr == fluidWaitTimeout {
		return nil, nil
	}
	// WAIT_ABANDONED is also a nonnegative result; only S_OK owns the lock.
	if hr != 0 {
		return nil, fmt.Errorf("fluid: acquire output: %#x", uint32(hr))
	}
	params := [3][4]float32{{1 / float32(f.width), 1 / float32(f.height), 0, 0}}
	f.pass("displayPS", &f.output, f.dye[0].srv, 0, params)
	call(f.ctx, ctxFlush)
	if hr := call(f.outputMutex, fluidReleaseSync, 1); hr != 0 {
		return nil, fmt.Errorf("fluid: publish output: %#x", uint32(hr))
	}
	call(f.output.tex, 1) // One reference belongs to the receiving frame lease.
	return &FluidFrame{texture: f.output.tex, shared: f.outputShared, width: f.width, height: f.height}, nil
}

// ImportFluidFrame is called only by this renderer's UI-thread owner. It
// copies a completed worker texture into a renderer-private image on the GPU,
// before the scene draws it in ordinary paint order. Neither context crosses
// threads. Not-ready frames leave the previous private image untouched.
func (r *Renderer) ImportFluidFrame(img *scene.Image, frame *FluidFrame) (ready bool, err error) {
	if r == nil || r.device == 0 || r.ctx == 0 || r.warp {
		return false, fmt.Errorf("fluid: native hardware renderer is unavailable")
	}
	if img == nil || frame == nil || frame.texture == 0 || frame.shared == 0 || img.W != frame.width || img.H != frame.height {
		return false, fmt.Errorf("fluid: invalid GPU frame lease")
	}
	if hr := call(r.device, 39); failed(hr) {
		return false, fmt.Errorf("fluid: presentation device removed: %#x", uint32(hr))
	}
	var shared, mutex uintptr
	if hr := call(r.device, 28, frame.shared, uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&shared))); failed(hr) {
		return false, fmt.Errorf("fluid: import output: %#x", uint32(hr))
	}
	defer free(&shared)
	if hr := call(shared, 0, uintptr(unsafe.Pointer(&fluidMutexIID)), uintptr(unsafe.Pointer(&mutex))); failed(hr) {
		return false, fmt.Errorf("fluid: imported output mutex: %#x", uint32(hr))
	}
	defer free(&mutex)
	hr := call(mutex, fluidAcquireSync, 1, 0)
	if hr == fluidWaitTimeout {
		return false, nil
	}
	if hr != 0 {
		return false, fmt.Errorf("fluid: acquire published output: %#x", uint32(hr))
	}
	defer func() {
		call(r.ctx, ctxFlush)
		if hr := call(mutex, fluidReleaseSync, 0); hr != 0 && err == nil {
			ready, err = false, fmt.Errorf("fluid: release imported output: %#x", uint32(hr))
		}
	}()
	target := r.images[img.ID()]
	if target == nil {
		target = &texture{}
		if err := r.newTexture(target, img.W, img.H, formatR8G8B8A8Unorm, nil, 4); err != nil {
			free(&target.srv)
			free(&target.tex)
			return false, err
		}
		if r.images == nil {
			r.images = map[uint64]*texture{}
		}
		r.images[img.ID()] = target
	}
	var unbound uintptr
	call(r.ctx, ctxPSSetShaderResources, 2, 1, uintptr(unsafe.Pointer(&unbound)))
	call(r.ctx, 47, target.tex, shared) // CopyResource: GPU to GPU only.
	target.ver, target.lastFrame = img.Version(), r.frame
	return true, nil
}

// ReleaseFluidImage retires only the image identity owned by a fluid presenter.
// It is safe after Renderer.Release, which already emptied the image cache.
func (r *Renderer) ReleaseFluidImage(img *scene.Image) {
	if r == nil || img == nil {
		return
	}
	if target := r.images[img.ID()]; target != nil {
		if r.ctx != 0 {
			var unbound uintptr
			call(r.ctx, ctxPSSetShaderResources, 2, 1, uintptr(unsafe.Pointer(&unbound)))
		}
		free(&target.srv)
		free(&target.tex)
		delete(r.images, img.ID())
	}
}
