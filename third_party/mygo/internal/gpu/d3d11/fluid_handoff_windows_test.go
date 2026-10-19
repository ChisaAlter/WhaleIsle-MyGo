//go:build windows

package d3d11

import (
	"bytes"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/d3d11/device"
	"github.com/egoist/mygo/internal/scene"
)

// This opt-in exercises only our own textures. It creates no HWND, swap chain,
// screen capture, user-input event or application instance. A requested run
// without hardware fails instead of using WARP as evidence for GPU sharing.
func TestFluidHardwareTwoDeviceHandoff(t *testing.T) {
	if os.Getenv("MYGO_FLUID_GPU_TEST") != "1" {
		t.Skip("set MYGO_FLUID_GPU_TEST=1 to exercise hardware D3D11 sharing")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	producer, err := NewFluid()
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	consumer := &Renderer{images: map[uint64]*texture{}}
	consumer.device, consumer.ctx, consumer.warp, err = device.New()
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Release()
	if consumer.warp || consumer.device == producer.device {
		t.Fatal("handoff requires two distinct hardware devices")
	}
	if err := producer.Resize(96, 64); err != nil {
		t.Fatal(err)
	}
	producer.Splat(.5, .5, 0, 0, [3]float32{1, 0, 0}, 100)
	producer.Step(.016666, 100)
	first := fluidHardwareFrame(t, producer)
	defer first.Close()
	image := fluidHardwareImage(96, 64)
	fluidHardwareImport(t, consumer, image, first)
	if producer.staging != 0 {
		t.Fatal("native display allocated the diagnostic CPU staging texture")
	}
	first.Close()
	first.Close()
	if first.texture != 0 || first.shared != 0 {
		t.Fatal("consumed frame retained its shared-resource lease")
	}
	private := consumer.images[image.ID()]
	if private == nil || private.tex == producer.output.tex || private.ver != image.Version() {
		t.Fatal("consumer did not own a version-matched private GPU copy")
	}
	red := fluidHardwarePixels(t, consumer, private)
	center := 4 * ((64/2)*96 + 96/2)
	if red[center] == 0 || red[center+1] != 0 || red[center+2] != 0 || red[center+3] == 0 {
		t.Fatalf("real shader/copy did not deliver red dye: %v", red[center:center+4])
	}
	for _, value := range image.Pix {
		if value != 0 {
			t.Fatal("GPU frame was copied into the transparent CPU placeholder")
		}
	}
	if consumer.imageView(image) != private.srv || !bytes.Equal(red, fluidHardwarePixels(t, consumer, private)) {
		t.Fatal("ordinary scene lookup replaced native pixels with placeholder pixels")
	}

	producer.Splat(.5, .5, 0, 0, [3]float32{0, 0, 1}, 100)
	producer.Step(.016666, 100)
	second := fluidHardwareFrame(t, producer)
	defer second.Close()
	if unexpected, err := producer.RenderFrame(); err != nil || unexpected != nil {
		if unexpected != nil {
			unexpected.Close()
		}
		t.Fatalf("producer overwrote an unconsumed published output: %v", err)
	}
	// Hold the real published key on the producer interface. The consumer's
	// zero-timeout acquire must return not-ready and preserve its old copy.
	fluidHardwareAcquire(t, producer.outputMutex, 1)
	func() {
		defer func() {
			if hr := call(producer.outputMutex, fluidReleaseSync, 1); hr != 0 {
				t.Errorf("release contended frame: %#x", uint32(hr))
			}
		}()
		ready, err := consumer.ImportFluidFrame(image, second)
		if err != nil || ready {
			t.Fatalf("contended import must be not-ready: ready=%v err=%v", ready, err)
		}
		if consumer.images[image.ID()] != private || !bytes.Equal(red, fluidHardwarePixels(t, consumer, private)) {
			t.Fatal("unready import replaced or modified the previous GPU image")
		}
	}()
	fluidHardwareImport(t, consumer, image, second)
	second.Close()
	blue := fluidHardwarePixels(t, consumer, private)
	if blue[center+2] == 0 || bytes.Equal(red, blue) || producer.staging != 0 {
		t.Fatal("released frame did not update the GPU copy without CPU readback")
	}

	// An outstanding lease keeps the old shared texture alive even after
	// resize retires its solver-owned views and Close releases its device.
	landscape := fluidHardwareFrame(t, producer)
	defer landscape.Close()
	if err := producer.Resize(64, 96); err != nil {
		t.Fatal(err)
	}
	portrait := fluidHardwareFrame(t, producer)
	defer portrait.Close()
	if w, h := landscape.ImageSize(); w != 96 || h != 64 {
		t.Fatal("resize mutated the outstanding frame dimensions")
	}
	if w, h := portrait.ImageSize(); w != 64 || h != 96 {
		t.Fatal("portrait frame did not use its own physical dimensions")
	}
	producer.Close()
	producer.Close()
	fluidHardwareImport(t, consumer, image, landscape)
	portraitImage := fluidHardwareImage(64, 96)
	fluidHardwareImport(t, consumer, portraitImage, portrait)
	if len(consumer.images) != 2 || producer.device != 0 || producer.ctx != 0 || producer.output.tex != 0 || producer.outputMutex != 0 {
		t.Fatal("solver close retained its ownership or invalidated the two frame leases")
	}
	landscape.Close()
	portrait.Close()
	consumer.ReleaseFluidImage(image)
	consumer.ReleaseFluidImage(portraitImage)
	consumer.ReleaseFluidImage(image)
	if len(consumer.images) != 0 || private.tex != 0 || private.srv != 0 {
		t.Fatal("presentation close did not immediately retire private GPU images")
	}
	if ready, err := consumer.ImportFluidFrame(image, landscape); ready || err == nil {
		t.Fatal("closed frame lease was accepted")
	}
	consumer.Release()
	consumer.ReleaseFluidImage(image)
}

func fluidHardwareImage(w, h int) *scene.Image {
	return scene.NewImageRGBA(w, h, make([]byte, w*h*4))
}

func fluidHardwareFrame(t *testing.T, producer *Fluid) *FluidFrame {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		frame, err := producer.RenderFrame()
		if err != nil {
			t.Fatal(err)
		}
		if frame != nil {
			return frame
		}
		if !time.Now().Before(deadline) {
			t.Fatal("producer key 0 did not become available")
		}
		time.Sleep(time.Millisecond)
	}
}

func fluidHardwareImport(t *testing.T, consumer *Renderer, image *scene.Image, frame *FluidFrame) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ready, err := consumer.ImportFluidFrame(image, frame)
		if err != nil {
			t.Fatal(err)
		}
		if ready {
			return
		}
		if !time.Now().Before(deadline) {
			t.Fatal("consumer key 1 did not become available")
		}
		time.Sleep(time.Millisecond)
	}
}

func fluidHardwareAcquire(t *testing.T, mutex uintptr, key uintptr) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		hr := call(mutex, fluidAcquireSync, key, 0)
		if hr == 0 {
			return
		}
		if hr != fluidWaitTimeout || !time.Now().Before(deadline) {
			t.Fatalf("acquire test-owned key %d: %#x", key, uint32(hr))
		}
		time.Sleep(time.Millisecond)
	}
}

// This reads only a test-owned offscreen texture to verify GPU-to-GPU copy
// contents. It is not a window/screen capture or part of production handoff.
func fluidHardwarePixels(t *testing.T, consumer *Renderer, source *texture) []byte {
	t.Helper()
	var staging uintptr
	desc := texture2DDesc{Width: uint32(source.w), Height: uint32(source.h), MipLevels: 1, ArraySize: 1, Format: formatR8G8B8A8Unorm, SampleCount: 1, Usage: 3, CPUAccessFlags: 0x20000}
	if hr := call(consumer.device, devCreateTexture2D, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&staging))); failed(hr) {
		t.Fatalf("test-owned staging: %#x", uint32(hr))
	}
	defer free(&staging)
	call(consumer.ctx, 47, staging, source.tex)
	var mapped mapped
	if hr := call(consumer.ctx, ctxMap, staging, 0, 1, 0, uintptr(unsafe.Pointer(&mapped))); failed(hr) {
		t.Fatalf("read test-owned GPU copy: %#x", uint32(hr))
	}
	defer call(consumer.ctx, ctxUnmap, staging, 0)
	pixels := make([]byte, source.w*source.h*4)
	for y := 0; y < source.h; y++ {
		copy(pixels[y*source.w*4:(y+1)*source.w*4], unsafe.Slice((*byte)(ptr(mapped.Data+uintptr(y)*uintptr(mapped.RowPitch))), source.w*4))
	}
	return pixels
}
