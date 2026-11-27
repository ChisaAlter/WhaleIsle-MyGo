//go:build windows

// Package d3d11 draws scenes with Direct3D 11 into a window through a DXGI
// flip model swap chain, called through syscall like the rest of the
// Windows backend (no cgo). Every op of a scene is an instanced quad drawn
// by one shader (shader.hlsl, compiled ahead of time into shaders.go);
// clips are scissor rectangles, with the innermost rounded clip computed
// in the shader. The swap chain is the window's, opaque, or with
// NewComposed shown on it through DirectComposition, with its alpha.
package d3d11

//go:generate go run gen.go

import (
	"errors"
	"fmt"
	"math"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/d3d11/device"
	"github.com/egoist/mygo/internal/scene"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	procCloseHandle         = kernel32.NewProc("CloseHandle")

	user32             = syscall.NewLazyDLL(device.SystemDir() + `\user32.dll`)
	procSetTimer       = user32.NewProc("SetTimer")
	procKillTimer      = user32.NewProc("KillTimer")
	procInvalidateRect = user32.NewProc("InvalidateRect")
)

var procDCompositionCreateDevice = syscall.NewLazyDLL(device.SystemDir() + `\dcomp.dll`).NewProc("DCompositionCreateDevice")

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidIDXGIDevice     = guid{0x54ec77fa, 0x1377, 0x44e6, [8]byte{0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c}}
	iidIDXGIDevice1    = guid{0x77db970f, 0x6276, 0x48ba, [8]byte{0xba, 0x28, 0x07, 0x01, 0x43, 0xb4, 0x39, 0x2c}}
	iidIDXGIFactory2   = guid{0x50c83a1c, 0xe072, 0x4c48, [8]byte{0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0}}
	iidIDXGISwapChain2 = guid{0xa8be2ac4, 0x199f, 0x4946, [8]byte{0xb3, 0x31, 0x79, 0x59, 0x9f, 0xb9, 0x8d, 0xe7}}
	iidID3D11Texture2D = guid{0x6f15aaf2, 0xd208, 0x4e89, [8]byte{0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c}}
)

var iidIDCompositionDevice = guid{0xc37ea93a, 0xe7aa, 0x450d, [8]byte{0xb1, 0x6f, 0x97, 0x46, 0xcb, 0x04, 0x07, 0xf3}}

// Vtable indices, from the Windows SDK headers.
const (
	release = 2

	// ID3D11Device
	devCreateBuffer             = 3
	devCreateTexture2D          = 5
	devCreateShaderResourceView = 7
	devCreateRenderTargetView   = 9
	devCreateInputLayout        = 11
	devCreateVertexShader       = 12
	devCreatePixelShader        = 15
	devCreateBlendState         = 20
	devCreateRasterizerState    = 22
	devCreateSamplerState       = 23

	// ID3D11DeviceContext
	viewGetResource                 = 7 // ID3D11View
	ctxVSSetConstantBuffers         = 7
	ctxPSSetShaderResources         = 8
	ctxPSSetShader                  = 9
	ctxPSSetSamplers                = 10
	ctxVSSetShader                  = 11
	ctxMap                          = 14
	ctxUnmap                        = 15
	ctxPSSetConstantBuffers         = 16
	ctxIASetInputLayout             = 17
	ctxIASetVertexBuffers           = 18
	ctxDrawInstanced                = 21
	ctxIASetPrimitiveTopology       = 24
	ctxOMSetRenderTargets           = 33
	ctxOMSetBlendState              = 35
	ctxRSSetState                   = 43
	ctxRSSetViewports               = 44
	ctxRSSetScissorRects            = 45
	ctxCopySubresourceRegion        = 46
	ctxUpdateSubresource            = 48
	ctxClearRenderTargetView        = 50
	ctxClearState                   = 110
	ctxFlush                        = 111
	dxgiDevGetAdapter               = 7
	dxgiDevSetMaxLatency            = 12 // IDXGIDevice1::SetMaximumFrameLatency
	dxgiGetParent                   = 6
	factoryMakeWindowAssoc          = 8
	factoryCreateSwapChainHwnd      = 15
	factoryCreateSwapChainComp      = 24 // CreateSwapChainForComposition
	scPresent                       = 8
	scGetBuffer                     = 9
	scResizeBuffers                 = 13
	scSetSourceSize                 = 29 // IDXGISwapChain2
	scSetMaximumFrameLatency        = 31 // IDXGISwapChain2
	scGetFrameLatencyWaitableObject = 33 // IDXGISwapChain2

	dxgiSwapChainFlagWaitable = 0x40 // DXGI_SWAP_CHAIN_FLAG_FRAME_LATENCY_WAITABLE_OBJECT
	waitObject0               = 0x00000000

	// IDCompositionDevice, IDCompositionTarget and IDCompositionVisual
	dcompCommit           = 3
	dcompCreateTarget     = 6 // CreateTargetForHwnd
	dcompCreateVisual     = 7
	dcompTargetSetRoot    = 3
	dcompVisualSetContent = 15 // the overload taking an IUnknown
)

const (
	formatR32G32B32A32Float = 2
	formatR8G8B8A8Unorm     = 28
	formatB8G8R8A8Unorm     = 87
	formatR8Unorm           = 61

	usageDefault = 0
	usageDynamic = 2

	bindVertexBuffer   = 0x1
	bindConstantBuffer = 0x4
	bindShaderResource = 0x8
	bindRenderTarget   = 0x20

	cpuAccessWrite  = 0x10000
	mapWriteDiscard = 4

	topologyTriangleStrip = 5
)

// ptr turns an address from native code into a pointer.
func ptr(a uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&a)) }

// comMethod resolves method i from the COM object's vtable.
func comMethod(obj uintptr, i int) uintptr {
	vtbl := *(*uintptr)(ptr(obj))
	return *(*uintptr)(unsafe.Add(ptr(vtbl), i*int(unsafe.Sizeof(uintptr(0)))))
}

// Fixed-arity COM call helpers avoid variadic slice heap escapes under
// //go:uintptrescapes on the render hot path.

//go:uintptrescapes
func call0(obj uintptr, i int) uintptr {
	r, _, _ := syscall.Syscall(comMethod(obj, i), 1, obj, 0, 0)
	return r
}

//go:uintptrescapes
func call1(obj uintptr, i int, a1 uintptr) uintptr {
	r, _, _ := syscall.Syscall(comMethod(obj, i), 2, obj, a1, 0)
	return r
}

//go:uintptrescapes
func call2(obj uintptr, i int, a1, a2 uintptr) uintptr {
	r, _, _ := syscall.Syscall(comMethod(obj, i), 3, obj, a1, a2)
	return r
}

//go:uintptrescapes
func call3(obj uintptr, i int, a1, a2, a3 uintptr) uintptr {
	r, _, _ := syscall.Syscall6(comMethod(obj, i), 4, obj, a1, a2, a3, 0, 0)
	return r
}

//go:uintptrescapes
func call4(obj uintptr, i int, a1, a2, a3, a4 uintptr) uintptr {
	r, _, _ := syscall.Syscall6(comMethod(obj, i), 5, obj, a1, a2, a3, a4, 0)
	return r
}

//go:uintptrescapes
func call5(obj uintptr, i int, a1, a2, a3, a4, a5 uintptr) uintptr {
	r, _, _ := syscall.Syscall6(comMethod(obj, i), 6, obj, a1, a2, a3, a4, a5)
	return r
}

//go:uintptrescapes
func call6(obj uintptr, i int, a1, a2, a3, a4, a5, a6 uintptr) uintptr {
	r, _, _ := syscall.Syscall9(comMethod(obj, i), 7, obj, a1, a2, a3, a4, a5, a6, 0, 0)
	return r
}

//go:uintptrescapes
func call8(obj uintptr, i int, a1, a2, a3, a4, a5, a6, a7, a8 uintptr) uintptr {
	r, _, _ := syscall.Syscall9(comMethod(obj, i), 9, obj, a1, a2, a3, a4, a5, a6, a7, a8)
	return r
}

func failed(hr uintptr) bool { return int32(uint32(hr)) < 0 }

func free(obj *uintptr) {
	if *obj != 0 {
		call0(*obj, release)
		*obj = 0
	}
}

// maxBlurTaps is the maximum number of precomputed bilinear tap pairs passed
// to blurps via PassConstants.passTaps (covering passRadius up to 128).
const maxBlurTaps = 64

// passConstants mirrors cbuffer PassConstants : register(b1) in shader.hlsl.
type passConstants struct {
	Origin, Limit, Shift, Dir [2]int32
	Down, Radius              int32
	Sigma                     float32
	TapCount                  int32
	InvSize                   [2]float32
	CenterW, Pad              float32
	Taps                      [maxBlurTaps][4]float32
}

// computeBlurTaps precomputes normalized 1D Gaussian weights and merges pairs of
// adjacent discrete taps into hardware bilinear sample offsets + combined
// weights (Daniel Rakos' linear-sampling Gaussian blur). Returns tapCount == 0
// when radius <= 0 or exceeds maxBlurTaps*2 so the shader falls back safely.
func computeBlurTaps(radius int, sigma float32) (centerW float32, tapCount int, taps [maxBlurTaps][4]float32) {
	if radius <= 0 || (radius+1)/2 > maxBlurTaps {
		return 0, 0, taps
	}
	var raw [maxBlurTaps*2 + 1]float32
	raw[0] = scene.BlurWeight(0, sigma)
	sum := float64(raw[0])
	for o := 1; o <= radius; o++ {
		w := scene.BlurWeight(o, sigma)
		raw[o] = w
		sum += 2 * float64(w)
	}
	if sum <= 0 {
		return 1, 0, taps
	}
	invSum := 1.0 / sum
	centerW = float32(float64(raw[0]) * invSum)
	k := 0
	for o := 1; o <= radius; o += 2 {
		wa := float64(raw[o]) * invSum
		if o+1 <= radius {
			wb := float64(raw[o+1]) * invSum
			wSum := wa + wb
			off := float64(o)
			if wSum > 0 {
				off = (float64(o)*wa + float64(o+1)*wb) / wSum
			}
			taps[k] = [4]float32{float32(off), float32(wSum), 0, 0}
		} else {
			taps[k] = [4]float32{float32(o), float32(wa), 0, 0}
		}
		k++
	}
	return centerW, k, taps
}

type bufferDesc struct {
	ByteWidth, Usage, BindFlags, CPUAccessFlags, MiscFlags, StructureByteStride uint32
}

type subresourceData struct {
	SysMem                   uintptr
	SysMemPitch, SysMemSlice uint32
}

type texture2DDesc struct {
	Width, Height, MipLevels, ArraySize, Format uint32
	SampleCount, SampleQuality                  uint32
	Usage, BindFlags, CPUAccessFlags, MiscFlags uint32
}

type inputElement struct {
	SemanticName         *byte
	SemanticIndex        uint32
	Format               uint32
	InputSlot            uint32
	AlignedByteOffset    uint32
	InputSlotClass       uint32
	InstanceDataStepRate uint32
}

type rtBlend struct {
	BlendEnable                                 int32
	SrcBlend, DestBlend, BlendOp                uint32
	SrcBlendAlpha, DestBlendAlpha, BlendOpAlpha uint32
	WriteMask                                   uint8
	_                                           [3]byte
}

type blendDesc struct {
	AlphaToCoverage, IndependentBlend int32
	RenderTarget                      [8]rtBlend
}

type rasterizerDesc struct {
	FillMode, CullMode   uint32
	FrontCCW             int32
	DepthBias            int32
	DepthBiasClamp       float32
	SlopeScaledDepthBias float32
	DepthClip, Scissor   int32
	Multisample, AALines int32
}

type samplerDesc struct {
	Filter                       uint32
	AddressU, AddressV, AddressW uint32
	MipLODBias                   float32
	MaxAnisotropy                uint32
	ComparisonFunc               uint32
	BorderColor                  [4]float32
	MinLOD, MaxLOD               float32
}

type viewport struct{ X, Y, W, H, MinDepth, MaxDepth float32 }

type box struct{ Left, Top, Front, Right, Bottom, Back uint32 }

type mapped struct {
	Data                 uintptr
	RowPitch, DepthPitch uint32
}

type swapChainDesc1 struct {
	Width, Height, Format          uint32
	Stereo                         int32
	SampleCount, SampleQuality     uint32
	BufferUsage, BufferCount       uint32
	Scaling, SwapEffect, AlphaMode uint32
	Flags                          uint32
}

type texture struct {
	tex, srv  uintptr
	rtv       uintptr // of a texture drawn into
	w, h      int
	bytes     int64
	gen, ver  uint64
	lastFrame uint64
	lastUse   uint64
}

// defaultImageBudget caps GPU memory held by cached image textures (RGBA8).
const defaultImageBudget = 96 << 20 // 96 MiB

// Renderer draws scenes into a window.
type Renderer struct {
	hwnd      uintptr
	device    uintptr
	ctx       uintptr
	factory   uintptr
	swapChain uintptr
	rtv       uintptr
	w, h      int  // the size drawn, which the swap chain shows
	warp      bool // the device is WARP, Windows' software rasterizer

	// composed tells that the swap chain shows on the window through
	// DirectComposition, with its alpha, as the content of visual, the
	// root of dcomp's target for the window.
	composed              bool
	dcomp, target, visual uintptr
	rounded               *roundedFrame

	// swapChain2 is the swap chain's IDXGISwapChain2 (Windows 8.1), bw×bh
	// the size of its buffers, larger than w×h while resizing, and
	// resizedAt when the window last changed size.
	swapChain2      uintptr
	scFlags         uint32    // DXGI_SWAP_CHAIN_FLAG_* kept consistent across ResizeBuffers
	waitable        uintptr   // HANDLE from GetFrameLatencyWaitableObject (closed in Release)
	waitProc        uintptr   // cached kernel32!WaitForSingleObject address
	slotAcquired    bool      // true once a waitable frame slot has been consumed for the next frame
	unpacedPresents int       // forced/immediate presents submitted while semaphore count was 0
	lastPresentAt   time.Time // timestamp of the previous Present for distinguishing back-to-back forced frames from timeout recovery
	bw, bh          int
	resizing        bool
	resizedAt       time.Time

	vs, ps, layout                 uintptr
	blend, maskBlend, raster, samp uintptr
	cbuf                           uintptr
	instBuf                        uintptr
	instCap                        int

	mask, color texture
	images      map[uint64]*texture
	imageBytes  int64
	imageBudget int64
	imageClock  uint64
	frame       uint64

	// The passes computing the backdrops of effects: their shaders and
	// constants, grab, which holds the area of the frame read, and
	// backdrop, the textures they draw into, the second only along rows.
	passVS, downPS, blurPS, passCB uintptr
	grab                           texture
	backdrop                       [2]texture
	// bdCache are the backdrops of the last frame, which the next draws
	// again while what is under them stays the same, bdKeys the keys of
	// the frame's, and bdHits and bdMisses count the backdrops drawn again
	// and computed. imageEpoch counts the images whose contents changed.
	bdCache          []cachedBackdrop
	bdKeys           []uint64
	bdHits, bdMisses int
	imageEpoch       uint64
	// effects are the pixel shaders of the effects drawn, made as first
	// drawn.
	effects map[*scene.Effect]*effectShader

	// Precomputed Gaussian blur tap cache (shared across H/V passes and frames).
	blurCacheRadius   int
	blurCacheSigma    float32
	blurCacheCenterW  float32
	blurCacheTapCount int
	blurCacheTaps     [maxBlurTaps][4]float32

	// Scratch storage on the heap-allocated Renderer so taking addresses for
	// COM calls on the render hot path avoids per-call heap allocations.
	scratch struct {
		mapped    mapped
		globals   [4]float32
		clear     [4]float32
		boundSRV  uintptr
		scissor   gpu.Scissor
		viewport  viewport
		stride    uint32
		offset    uint32
		views     [4]uintptr
		noneViews [5]uintptr
		copyBox   box
		back      uintptr
		passSrc   uintptr
		passNone  uintptr
		passConst passConstants
	}

	b gpu.Builder

	// lastUpload, lastDraw and lastSwap are how the last frame Render drew
	// split its GPU work, for the engine's frame trace.
	lastUpload, lastDraw, lastSwap time.Duration
}

// New creates a renderer drawing into the window hwnd, on the GPU or, when
// there is none, with Windows' software rasterizer (WARP). Its frames are
// opaque: what they leave transparent shows black.
func New(hwnd uintptr) (*Renderer, error) { return newRenderer(hwnd, false) }

// NewComposed creates a renderer whose frames show on the window hwnd
// through DirectComposition, premultiplied by their alpha, over what is
// behind the window's content: in a window without a redirection bitmap
// (WS_EX_NOREDIRECTIONBITMAP), its system backdrop, as Mica.
func NewComposed(hwnd uintptr) (*Renderer, error) { return newRenderer(hwnd, true) }

func newRenderer(hwnd uintptr, composed bool) (*Renderer, error) {
	if hwnd == 0 {
		return nil, errors.New("d3d11: no window")
	}
	r := &Renderer{hwnd: hwnd, composed: composed, images: map[uint64]*texture{}}
	var err error
	if r.device, r.ctx, r.warp, err = device.New(); err != nil {
		return nil, err
	}
	if err := r.init(); err != nil {
		r.Release()
		return nil, err
	}
	return r, nil
}

func (r *Renderer) init() error {
	// The factory that made the device's adapter makes the swap chain.
	var dxgiDev, adapter uintptr
	if failed(call2(r.device, 0, uintptr(unsafe.Pointer(&iidIDXGIDevice)), uintptr(unsafe.Pointer(&dxgiDev)))) {
		return errors.New("d3d11: no DXGI device")
	}
	defer free(&dxgiDev)
	// One frame queued ahead of the screen, not three: a frame shows the
	// input of when it was drawn while frames follow each other, as when
	// scrolling or animating.
	var dxgiDev1 uintptr
	if !failed(call2(r.device, 0, uintptr(unsafe.Pointer(&iidIDXGIDevice1)), uintptr(unsafe.Pointer(&dxgiDev1)))) {
		call1(dxgiDev1, dxgiDevSetMaxLatency, 1)
		free(&dxgiDev1)
	}
	if failed(call1(dxgiDev, dxgiDevGetAdapter, uintptr(unsafe.Pointer(&adapter)))) {
		return errors.New("d3d11: no adapter")
	}
	defer free(&adapter)
	if failed(call2(adapter, dxgiGetParent, uintptr(unsafe.Pointer(&iidIDXGIFactory2)), uintptr(unsafe.Pointer(&r.factory)))) {
		return errors.New("d3d11: DXGI 1.2 is required")
	}

	code, err := shaderCode()
	if err != nil {
		return err
	}
	vsCode := code.vs
	for _, s := range []struct {
		code []byte
		out  *uintptr
		vtbl int
		name string
	}{{code.vs, &r.vs, devCreateVertexShader, "vertex"}, {code.ps, &r.ps, devCreatePixelShader, "pixel"},
		{code.passVS, &r.passVS, devCreateVertexShader, "pass vertex"}, {code.down, &r.downPS, devCreatePixelShader, "down"},
		{code.blur, &r.blurPS, devCreatePixelShader, "blur"}} {
		if len(s.code) == 0 || failed(call4(r.device, s.vtbl, uintptr(unsafe.Pointer(&s.code[0])), uintptr(len(s.code)), 0, uintptr(unsafe.Pointer(s.out)))) {
			return fmt.Errorf("d3d11: cannot create the %s shader", s.name)
		}
	}
	names := []string{"RECT", "RADII", "INNER", "COLOR", "COLOR", "COLOR", "GRAD", "UV", "CLIP", "CLIPR", "PARAMS", "CLIPF"}
	indices := []uint32{0, 0, 0, 0, 1, 2, 0, 0, 0, 0, 0, 0}
	elems := make([]inputElement, len(names))
	cstr := make([][]byte, len(names))
	for i, n := range names {
		cstr[i] = append([]byte(n), 0)
		elems[i] = inputElement{SemanticName: &cstr[i][0], SemanticIndex: indices[i], Format: formatR32G32B32A32Float,
			AlignedByteOffset: uint32(16 * i), InputSlotClass: 1, InstanceDataStepRate: 1}
	}
	if failed(call5(r.device, devCreateInputLayout, uintptr(unsafe.Pointer(&elems[0])), uintptr(len(elems)),
		uintptr(unsafe.Pointer(&vsCode[0])), uintptr(len(vsCode)), uintptr(unsafe.Pointer(&r.layout)))) {
		return errors.New("d3d11: cannot create the input layout")
	}

	// Dual-source blending: the destination times one minus the shader's
	// second color, the source's alpha of each channel (D3D11_BLEND_ONE,
	// D3D11_BLEND_INV_SRC1_COLOR and D3D11_BLEND_INV_SRC1_ALPHA).
	bd := blendDesc{}
	bd.RenderTarget[0] = rtBlend{BlendEnable: 1, SrcBlend: 2, DestBlend: 17, BlendOp: 1, SrcBlendAlpha: 2, DestBlendAlpha: 19, BlendOpAlpha: 1, WriteMask: 0xf}
	if failed(call2(r.device, devCreateBlendState, uintptr(unsafe.Pointer(&bd)), uintptr(unsafe.Pointer(&r.blend)))) {
		return errors.New("d3d11: cannot create the blend state")
	}
	mbd := blendDesc{}
	mbd.RenderTarget[0] = rtBlend{BlendEnable: 1, SrcBlend: 1, DestBlend: 6, BlendOp: 1, SrcBlendAlpha: 1, DestBlendAlpha: 6, BlendOpAlpha: 1, WriteMask: 0xf}
	if failed(call2(r.device, devCreateBlendState, uintptr(unsafe.Pointer(&mbd)), uintptr(unsafe.Pointer(&r.maskBlend)))) {
		return errors.New("d3d11: cannot create the mask blend state")
	}
	rd := rasterizerDesc{FillMode: 3, CullMode: 1, DepthClip: 1, Scissor: 1}
	if failed(call2(r.device, devCreateRasterizerState, uintptr(unsafe.Pointer(&rd)), uintptr(unsafe.Pointer(&r.raster)))) {
		return errors.New("d3d11: cannot create the rasterizer state")
	}
	sd := samplerDesc{Filter: 0x15, AddressU: 3, AddressV: 3, AddressW: 3, ComparisonFunc: 1, MaxLOD: math.MaxFloat32}
	if failed(call2(r.device, devCreateSamplerState, uintptr(unsafe.Pointer(&sd)), uintptr(unsafe.Pointer(&r.samp)))) {
		return errors.New("d3d11: cannot create the sampler")
	}
	cd := bufferDesc{ByteWidth: 16, Usage: usageDefault, BindFlags: bindConstantBuffer}
	if failed(call3(r.device, devCreateBuffer, uintptr(unsafe.Pointer(&cd)), 0, uintptr(unsafe.Pointer(&r.cbuf)))) {
		return errors.New("d3d11: cannot create the constant buffer")
	}
	cd.ByteWidth = uint32(unsafe.Sizeof(passConstants{}))
	if failed(call3(r.device, devCreateBuffer, uintptr(unsafe.Pointer(&cd)), 0, uintptr(unsafe.Pointer(&r.passCB)))) {
		return errors.New("d3d11: cannot create the constant buffer of passes")
	}
	return nil
}

// Keep three flip-model swap-chain buffers in steady state while capping
// IDXGISwapChain2::SetMaximumFrameLatency at 1: input latency stays 1 frame,
// while a free back buffer remains available when DWM holds two buffers during
// high-refresh (165Hz+) composition, eliminating double-buffer backpressure
// spikes. While resizing, buffers are oversized by a quarter so ResizeBuffers
// is only called when the window outgrows them, then settled back to w×h.
const (
	defaultBuffers = 3
	resizeBuffers  = 3
	settleTimer    = 0x6d79 // the timer's ID on the window
)

var settleDelay = time.Second

// resize makes the swap chain show w×h pixels.
func (r *Renderer) resize(w, h int) error {
	if r.swapChain != 0 && w == r.w && h == r.h {
		if r.resizing && time.Since(r.resizedAt) >= settleDelay/2 {
			r.resizing = false
			return r.buffers(w, h, w, h, defaultBuffers)
		}
		return nil
	}
	if r.swapChain == 0 || r.swapChain2 == 0 {
		return r.buffers(w, h, w, h, defaultBuffers)
	}
	r.resizedAt = time.Now()
	r.armSettle()
	if r.resizing && w <= r.bw && h <= r.bh {
		if err := r.setSourceSize(w, h); err != nil {
			return err
		}
		r.w, r.h = w, h
		return nil
	}
	r.resizing = true
	return r.buffers(w, h, w+w/4+64, h+h/4+64, resizeBuffers)
}

// buffers makes the swap chain n buffers of bw×bh pixels, which shows w×h
// of them.
func (r *Renderer) buffers(w, h, bw, bh, n int) error {
	free(&r.rtv)
	if r.swapChain == 0 {
		desc := swapChainDesc1{Width: uint32(bw), Height: uint32(bh), Format: formatB8G8R8A8Unorm, SampleCount: 1,
			BufferUsage: 0x20, BufferCount: uint32(n), Scaling: 0, SwapEffect: 4 /* flip discard */, Flags: dxgiSwapChainFlagWaitable}
		if r.composed {
			if err := r.composeSwapChain(desc); err != nil {
				return err
			}
		} else {
			hr := call6(r.factory, factoryCreateSwapChainHwnd, r.device, r.hwnd, uintptr(unsafe.Pointer(&desc)), 0, 0, uintptr(unsafe.Pointer(&r.swapChain)))
			if failed(hr) && desc.Flags != 0 {
				// Fallback for older DXGI/drivers that reject the waitable flag.
				desc.Flags = 0
				hr = call6(r.factory, factoryCreateSwapChainHwnd, r.device, r.hwnd, uintptr(unsafe.Pointer(&desc)), 0, 0, uintptr(unsafe.Pointer(&r.swapChain)))
			}
			if failed(hr) {
				return fmt.Errorf("d3d11: cannot create the swap chain: %#x", uint32(hr))
			}
			r.scFlags = desc.Flags
			const noAltEnter = 2
			call2(r.factory, factoryMakeWindowAssoc, r.hwnd, noAltEnter)
		}
		call2(r.swapChain, 0, uintptr(unsafe.Pointer(&iidIDXGISwapChain2)), uintptr(unsafe.Pointer(&r.swapChain2)))
		r.initWaitable()
	} else if hr := call5(r.swapChain, scResizeBuffers, uintptr(n), uintptr(bw), uintptr(bh), 0, uintptr(r.scFlags)); failed(hr) {
		return fmt.Errorf("d3d11: cannot resize the swap chain: %#x", uint32(hr))
	} else {
		// ResizeBuffers flushes previously queued frames; clear any stale
		// unpaced debt so the next Present reconciles the semaphore cleanly.
		r.unpacedPresents = 0
	}
	if err := r.setSourceSize(w, h); err != nil {
		return err
	}
	r.bw, r.bh = bw, bh
	var back uintptr
	if hr := call3(r.swapChain, scGetBuffer, 0, uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&back))); failed(hr) {
		return fmt.Errorf("d3d11: no back buffer: %#x", uint32(hr))
	}
	defer free(&back)
	if hr := call3(r.device, devCreateRenderTargetView, back, 0, uintptr(unsafe.Pointer(&r.rtv))); failed(hr) {
		return fmt.Errorf("d3d11: cannot create the render target view: %#x", uint32(hr))
	}
	r.w, r.h = w, h
	return nil
}

// initWaitable configures IDXGISwapChain2 frame latency and acquires the
// waitable handle when the swap chain was created with the waitable flag.
func (r *Renderer) initWaitable() {
	if r.swapChain2 == 0 || r.scFlags&dxgiSwapChainFlagWaitable == 0 {
		return
	}
	call1(r.swapChain2, scSetMaximumFrameLatency, 1)
	h := call0(r.swapChain2, scGetFrameLatencyWaitableObject)
	if h != 0 && h != ^uintptr(0) {
		r.waitable = h
		r.slotAcquired = false
		r.unpacedPresents = 0
		r.lastPresentAt = time.Time{}
	}
}

// waitFrameNonBlocking performs a zero-timeout WaitForSingleObject on the
// waitable handle without allocating a variadic argument slice.
func (r *Renderer) waitFrameNonBlocking() bool {
	if r.waitable == 0 {
		return false
	}
	if r.waitProc == 0 {
		if err := procWaitForSingleObject.Find(); err != nil {
			return false
		}
		r.waitProc = procWaitForSingleObject.Addr()
	}
	ret, _, _ := syscall.Syscall(r.waitProc, 2, r.waitable, 0, 0)
	return ret == waitObject0
}

// drainExtraWaitable drains any excess semaphore count beyond the single frame
// slot permitted by SetMaximumFrameLatency(1). It must only be called when the
// caller already owns a slot (slotAcquired == true) or is immediately about to
// Present, so it can never starve the event loop of the next frame's signal.
func (r *Renderer) drainExtraWaitable() {
	for range 16 {
		if !r.waitFrameNonBlocking() {
			return
		}
		if r.unpacedPresents > 0 {
			r.unpacedPresents--
		}
	}
}

// WaitHandle returns the DXGI frame-latency waitable HANDLE signaled when the
// swap chain is ready to accept the next frame, or 0 if unavailable.
func (r *Renderer) WaitHandle() uintptr { return r.waitable }

// TryAcquireFrame performs a non-blocking check against the waitable handle.
// It returns true when a frame may be drawn and presented without blocking on
// VSync, or when waitable pacing is inactive.
func (r *Renderer) TryAcquireFrame() bool {
	if r.waitable == 0 {
		return true
	}
	if r.slotAcquired {
		r.drainExtraWaitable()
		return true
	}
	// Reconcile any unpaced (forced/immediate) presents before acquiring the
	// next slot so a delayed completion signal from an earlier frame cannot
	// leave the semaphore permanently shifted (+1) and force Present(1, 0) to
	// block on every subsequent frame.
	for r.unpacedPresents > 0 {
		if !r.waitFrameNonBlocking() {
			return false
		}
		r.unpacedPresents--
	}
	if !r.waitFrameNonBlocking() {
		return false
	}
	r.slotAcquired = true
	r.drainExtraWaitable()
	return true
}

// MarkFrameReady records that the platform event loop was woken by WaitHandle
// (consuming its semaphore count), allowing the next frame to proceed at once
// once any unpaced present debt has been reconciled.
func (r *Renderer) MarkFrameReady() {
	if r.waitable == 0 {
		return
	}
	if r.unpacedPresents > 0 {
		// The signal consumed by MsgWait repays an unpaced present; check
		// whether another signal is already available for the next frame.
		r.unpacedPresents--
		for r.unpacedPresents > 0 {
			if !r.waitFrameNonBlocking() {
				return
			}
			r.unpacedPresents--
		}
		if !r.waitFrameNonBlocking() {
			return
		}
	}
	r.slotAcquired = true
	r.drainExtraWaitable()
}

// setSourceSize makes the swap chain show the w×h pixels at the top left
// of its buffers, unless it is for composition, which shows them whole.
func (r *Renderer) setSourceSize(w, h int) error {
	if r.swapChain2 == 0 || r.composed {
		return nil
	}
	if hr := call2(r.swapChain2, scSetSourceSize, uintptr(w), uintptr(h)); failed(hr) {
		return fmt.Errorf("d3d11: cannot set the swap chain's source size: %#x", uint32(hr))
	}
	return nil
}

// composeSwapChain creates a swap chain for DirectComposition, whose
// pixels are premultiplied by their alpha, and shows it on the window: the
// content of the visual at the root of a target for the window, on a
// DirectComposition device of the renderer's own. The swap chain's frames
// show as it presents them, with no commit.
func (r *Renderer) composeSwapChain(desc swapChainDesc1) error {
	if err := procDCompositionCreateDevice.Find(); err != nil {
		return fmt.Errorf("d3d11: no DirectComposition: %w", err)
	}
	var dxgiDev uintptr
	if failed(call2(r.device, 0, uintptr(unsafe.Pointer(&iidIDXGIDevice)), uintptr(unsafe.Pointer(&dxgiDev)))) {
		return errors.New("d3d11: no DXGI device")
	}
	hr, _, _ := procDCompositionCreateDevice.Call(dxgiDev, uintptr(unsafe.Pointer(&iidIDCompositionDevice)), uintptr(unsafe.Pointer(&r.dcomp)))
	free(&dxgiDev)
	if failed(hr) {
		return fmt.Errorf("d3d11: cannot create the DirectComposition device: %#x", uint32(hr))
	}
	// A swap chain for composition flips in sequence and stretches, the
	// only ways it takes.
	desc.SwapEffect, desc.AlphaMode = 3 /* flip sequential */, 1 /* premultiplied */
	hr = call4(r.factory, factoryCreateSwapChainComp, r.device, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&r.swapChain)))
	if failed(hr) && desc.Flags != 0 {
		// Fallback for older DXGI/drivers that reject the waitable flag.
		desc.Flags = 0
		hr = call4(r.factory, factoryCreateSwapChainComp, r.device, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&r.swapChain)))
	}
	if failed(hr) {
		return fmt.Errorf("d3d11: cannot create the swap chain for composition: %#x", uint32(hr))
	}
	r.scFlags = desc.Flags
	if hr := call3(r.dcomp, dcompCreateTarget, r.hwnd, 1, uintptr(unsafe.Pointer(&r.target))); failed(hr) {
		return fmt.Errorf("d3d11: cannot compose the window: %#x", uint32(hr))
	}
	if hr := call1(r.dcomp, dcompCreateVisual, uintptr(unsafe.Pointer(&r.visual))); failed(hr) {
		return fmt.Errorf("d3d11: cannot create the visual: %#x", uint32(hr))
	}
	if hr := call1(r.visual, dcompVisualSetContent, r.swapChain); failed(hr) {
		return fmt.Errorf("d3d11: cannot show the swap chain in the visual: %#x", uint32(hr))
	}
	if hr := call1(r.target, dcompTargetSetRoot, r.visual); failed(hr) {
		return fmt.Errorf("d3d11: cannot show the visual on the window: %#x", uint32(hr))
	}
	if hr := call0(r.dcomp, dcompCommit); failed(hr) {
		return fmt.Errorf("d3d11: cannot commit the composition: %#x", uint32(hr))
	}
	return nil
}

// Composed reports whether the frames show through DirectComposition, with
// their alpha (NewComposed).
func (r *Renderer) Composed() bool { return r.composed }

// Software reports whether the renderer draws with WARP, on the CPU, for
// want of a GPU.
func (r *Renderer) Software() bool { return r.warp }

// settling holds the windows whose settle timer is armed; settleProc is
// the timers' callback, made once.
var (
	settling   = map[uintptr]bool{}
	settleProc uintptr
)

// armSettle has the window ask for a frame settleDelay from now, unless
// it changes size again before.
func (r *Renderer) armSettle() {
	if settleProc == 0 {
		settleProc = syscall.NewCallback(func(hwnd, msg, id, ms uintptr) uintptr {
			procKillTimer.Call(hwnd, id)
			if settling[hwnd] {
				delete(settling, hwnd)
				procInvalidateRect.Call(hwnd, 0, 0)
			}
			return 0
		})
	}
	settling[r.hwnd] = true
	procSetTimer.Call(r.hwnd, settleTimer, uintptr(settleDelay/time.Millisecond), settleProc)
}

// Release frees the renderer's GPU objects.
func (r *Renderer) Release() {
	if r.rounded != nil {
		r.rounded.clip.Close()
	}
	if settling[r.hwnd] {
		delete(settling, r.hwnd)
		procKillTimer.Call(r.hwnd, settleTimer)
	}
	if r.ctx != 0 {
		call0(r.ctx, ctxClearState)
		call0(r.ctx, ctxFlush)
	}
	for _, t := range r.images {
		free(&t.srv)
		free(&t.tex)
	}
	r.images = nil
	r.imageBytes = 0
	for _, t := range []*texture{&r.mask, &r.color} {
		free(&t.srv)
		free(&t.tex)
	}
	for _, p := range r.effects {
		free(&p.ps)
	}
	for _, t := range []*texture{&r.grab, &r.backdrop[0], &r.backdrop[1]} {
		free(&t.rtv)
		free(&t.srv)
		free(&t.tex)
	}
	r.trimBackdrops(0)
	if r.waitable != 0 {
		procCloseHandle.Call(r.waitable)
		r.waitable = 0
		r.slotAcquired = false
		r.unpacedPresents = 0
	}
	// The target goes first: a window has one at most, which another
	// renderer of it then makes.
	for _, p := range []*uintptr{&r.visual, &r.target, &r.dcomp, &r.rtv, &r.swapChain2, &r.swapChain, &r.instBuf, &r.cbuf, &r.passCB, &r.samp, &r.raster, &r.maskBlend, &r.blend, &r.layout,
		&r.blurPS, &r.downPS, &r.passVS, &r.ps, &r.vs, &r.factory, &r.ctx, &r.device} {
		free(p)
	}
}

// newTexture creates a texture of w×h pixels from pix.
func (r *Renderer) newTexture(t *texture, w, h int, format uint32, pix []byte, bpp int) error {
	free(&t.srv)
	free(&t.tex)
	desc := texture2DDesc{Width: uint32(w), Height: uint32(h), MipLevels: 1, ArraySize: 1, Format: format, SampleCount: 1, Usage: usageDefault, BindFlags: bindShaderResource}
	var init *subresourceData
	if len(pix) >= w*h*bpp {
		init = &subresourceData{SysMem: uintptr(unsafe.Pointer(&pix[0])), SysMemPitch: uint32(w * bpp)}
	}
	if hr := call3(r.device, devCreateTexture2D, uintptr(unsafe.Pointer(&desc)), uintptr(unsafe.Pointer(init)), uintptr(unsafe.Pointer(&t.tex))); failed(hr) {
		return fmt.Errorf("d3d11: cannot create a %dx%d texture: %#x", w, h, uint32(hr))
	}
	if hr := call3(r.device, devCreateShaderResourceView, t.tex, 0, uintptr(unsafe.Pointer(&t.srv))); failed(hr) {
		return fmt.Errorf("d3d11: cannot create a texture view: %#x", uint32(hr))
	}
	t.w, t.h = w, h
	return nil
}

// syncAtlas uploads what changed in an atlas.
func (r *Renderer) syncAtlas(t *texture, a *scene.Atlas, format uint32) error {
	if a == nil {
		return nil
	}
	if t.tex == 0 || t.gen != a.Generation() || t.w != a.W || t.h != a.H {
		if err := r.newTexture(t, a.W, a.H, format, a.Pix, a.BPP); err != nil {
			return err
		}
		t.gen, t.ver = a.Generation(), a.Version()
		return nil
	}
	rects, full := a.Changes(t.gen, t.ver)
	if full {
		call6(r.ctx, ctxUpdateSubresource, t.tex, 0, 0, uintptr(unsafe.Pointer(&a.Pix[0])), uintptr(a.W*a.BPP), 0)
	}
	for _, rc := range rects {
		r.scratch.copyBox = box{Left: uint32(rc.Min.X), Top: uint32(rc.Min.Y), Front: 0, Right: uint32(rc.Max.X), Bottom: uint32(rc.Max.Y), Back: 1}
		src := a.Pix[(rc.Min.Y*a.W+rc.Min.X)*a.BPP:]
		call6(r.ctx, ctxUpdateSubresource, t.tex, 0, uintptr(unsafe.Pointer(&r.scratch.copyBox)), uintptr(unsafe.Pointer(&src[0])), uintptr(a.W*a.BPP), 0)
	}
	t.ver = a.Version()
	return nil
}

// effectiveImageBudget returns the byte budget for cached image textures.
func (r *Renderer) effectiveImageBudget() int64 {
	if r.imageBudget > 0 {
		return r.imageBudget
	}
	return defaultImageBudget
}

// deleteImage releases one cached image texture and updates accounting.
func (r *Renderer) deleteImage(id uint64, t *texture) {
	free(&t.srv)
	free(&t.tex)
	r.imageBytes -= t.bytes
	if r.imageBytes < 0 {
		r.imageBytes = 0
	}
	delete(r.images, id)
}

// evictImages releases the least-recently-used image textures until cached
// image memory is at or below budget. When allowCurrentFrame is false,
// textures used in the current frame (t.lastFrame == r.frame) are preserved
// so in-flight batch SRVs remain valid.
func (r *Renderer) evictImages(budget int64, allowCurrentFrame bool) {
	for r.imageBytes > budget && len(r.images) > 0 {
		var victimID uint64
		var victim *texture
		for id, t := range r.images {
			if !allowCurrentFrame && t.lastFrame == r.frame {
				continue
			}
			if victim == nil || t.lastFrame < victim.lastFrame || (t.lastFrame == victim.lastFrame && t.lastUse < victim.lastUse) {
				victimID, victim = id, t
			}
		}
		if victim == nil {
			break
		}
		r.deleteImage(victimID, victim)
	}
}

// Idle reclaims transient GPU resources (backdrop scratch targets, cached
// backdrops, oversized instance buffers and stale images) once the window
// stops drawing frames.
func (r *Renderer) Idle() {
	if r.ctx != 0 {
		call0(r.ctx, ctxClearState)
		call0(r.ctx, ctxFlush)
	}
	for _, t := range []*texture{&r.grab, &r.backdrop[0], &r.backdrop[1]} {
		free(&t.rtv)
		free(&t.srv)
		free(&t.tex)
		*t = texture{}
	}
	r.trimBackdrops(0)
	if r.instCap > 4096 {
		free(&r.instBuf)
		r.instCap = 0
	}
	for id, t := range r.images {
		if t.lastFrame < r.frame {
			r.deleteImage(id, t)
		}
	}
	r.evictImages(r.effectiveImageBudget(), true)
}

// Trim performs a deeper GPU reclaim for an occluded (hidden or minimized)
// window, releasing reconstructible image caches, glyph atlases and scratch
// buffers until the next frame draws.
func (r *Renderer) Trim() {
	r.Idle()
	r.evictImages(0, true)
	for _, t := range []*texture{&r.mask, &r.color} {
		free(&t.srv)
		free(&t.tex)
		*t = texture{}
	}
	free(&r.instBuf)
	r.instCap = 0
	r.b = gpu.Builder{}
	if r.ctx != 0 {
		call0(r.ctx, ctxClearState)
		call0(r.ctx, ctxFlush)
	}
}

// imageView returns the texture view of an image, uploading it when new or
// changed.
func (r *Renderer) imageView(img *scene.Image) uintptr {
	if r.images == nil {
		r.images = map[uint64]*texture{}
	}
	t := r.images[img.ID()]
	size := int64(img.W) * int64(img.H) * 4
	if t == nil {
		t = &texture{}
		if err := r.newTexture(t, img.W, img.H, formatR8G8B8A8Unorm, img.Pix, 4); err != nil {
			return 0
		}
		t.ver = img.Version()
		t.bytes = size
		r.images[img.ID()] = t
		r.imageBytes += size
		r.imageClock++
		t.lastUse = r.imageClock
		t.lastFrame = r.frame
		r.evictImages(r.effectiveImageBudget(), false)
		return t.srv
	}
	if t.ver != img.Version() {
		r.imageEpoch++
		if t.w != img.W || t.h != img.H {
			oldBytes := t.bytes
			if err := r.newTexture(t, img.W, img.H, formatR8G8B8A8Unorm, img.Pix, 4); err != nil {
				r.imageBytes -= oldBytes
				if r.imageBytes < 0 {
					r.imageBytes = 0
				}
				delete(r.images, img.ID())
				return 0
			}
			r.imageBytes += size - oldBytes
			t.bytes = size
		} else {
			call6(r.ctx, ctxUpdateSubresource, t.tex, 0, 0, uintptr(unsafe.Pointer(&img.Pix[0])), uintptr(img.W*4), 0)
		}
		t.ver = img.Version()
	}
	r.imageClock++
	t.lastUse = r.imageClock
	t.lastFrame = r.frame
	if r.imageBytes > r.effectiveImageBudget() {
		r.evictImages(r.effectiveImageBudget(), false)
	}
	return t.srv
}

// Render draws s into the window and presents it.
func (r *Renderer) Render(s *scene.Scene) error {
	r.lastUpload, r.lastDraw, r.lastSwap = 0, 0, 0
	if s.Width <= 0 || s.Height <= 0 {
		return nil
	}
	resized := r.swapChain != 0 && (s.Width != r.w || s.Height != r.h)
	if err := r.draw(s); err != nil {
		return err
	}
	if r.rounded != nil {
		applied := time.Now()
		if err := r.rounded.apply(r, s); err != nil {
			return err
		}
		r.lastDraw += time.Since(applied)
	}
	// A frame of a new size is presented at once (sync interval 0), not
	// after the frame queued before it, which would miss the next
	// composition. It is not flushed with DwmFlush: that waited for the
	// compositor after every resized frame and blocked the UI thread for
	// hundreds of milliseconds while DWM caught up, which stalled a drag.
	// The next composition shows this frame, as a live resize holds the
	// last one stretched until the size stops changing.
	sync := uintptr(1)
	if resized {
		sync = 0
	}
	presented := time.Now()
	if err := r.present(sync); err != nil {
		return err
	}
	r.lastSwap = time.Since(presented)
	return nil
}

// FrameLaps reports how the last frame Render drew split its GPU work: the
// upload lap syncs the atlases, image textures and instance buffer, the draw
// lap builds the batches and submits them, backdrop passes and the rounded
// mask included, and the swap lap is the present call itself.
func (r *Renderer) FrameLaps() (upload, draw, swap time.Duration) {
	return r.lastUpload, r.lastDraw, r.lastSwap
}

// draw draws s into the swap chain's back buffer.
func (r *Renderer) draw(s *scene.Scene) error {
	drawStart := time.Now()
	r.frame++
	if err := r.resize(s.Width, s.Height); err != nil {
		return err
	}
	uploadStart := time.Now()
	if err := r.syncAtlas(&r.mask, s.MaskAtlas, formatR8Unorm); err != nil {
		return err
	}
	if err := r.syncAtlas(&r.color, s.ColorAtlas, formatR8G8B8A8Unorm); err != nil {
		return err
	}
	upload := time.Since(uploadStart)
	r.b.Build(s, r.imageView)
	if err := r.fitBackdrop(); err != nil {
		return err
	}
	var salt uint64
	if len(r.b.Backdrops) > 0 {
		r.bdKeys = r.b.BackdropKeys(r.bdKeys)
		salt = r.backdropSalt(s)
	}
	r.trimBackdrops(len(r.b.Backdrops))
	maskIdx := -1
	if r.rounded != nil {
		if radius := r.rounded.maskRadius(r, s); radius > 0 {
			w, h := float32(s.Width), float32(s.Height)
			maskIdx = len(r.b.Instances)
			r.b.Instances = append(r.b.Instances, gpu.Instance{
				Rect:   [4]float32{0, 0, w, h},
				Inner:  [4]float32{radius, radius, radius, radius},
				Color:  [4]float32{1, 1, 1, 1},
				UV:     [4]float32{0, 0, w, h},
				Clip:   [4]float32{-1e6, -1e6, 2e6, 2e6},
				Params: [4]float32{1, 0, 0, 1},
			})
		}
	}

	// Upload the instances, growing the buffer as needed.
	uploadRest := time.Now()
	if n := len(r.b.Instances); n > 0 {
		if n > r.instCap {
			free(&r.instBuf)
			capacity := max(n*3/2, 1024)
			bd := bufferDesc{ByteWidth: uint32(capacity * gpu.InstanceSize), Usage: usageDynamic, BindFlags: bindVertexBuffer, CPUAccessFlags: cpuAccessWrite}
			if hr := call3(r.device, devCreateBuffer, uintptr(unsafe.Pointer(&bd)), 0, uintptr(unsafe.Pointer(&r.instBuf))); failed(hr) {
				r.instCap = 0
				return fmt.Errorf("d3d11: cannot create the instance buffer: %#x", uint32(hr))
			}
			r.instCap = capacity
		}
		r.scratch.mapped = mapped{}
		if hr := call5(r.ctx, ctxMap, r.instBuf, 0, mapWriteDiscard, 0, uintptr(unsafe.Pointer(&r.scratch.mapped))); failed(hr) {
			return fmt.Errorf("d3d11: cannot map the instance buffer: %#x", uint32(hr))
		}
		dst := unsafe.Slice((*gpu.Instance)(ptr(r.scratch.mapped.Data)), n)
		copy(dst, r.b.Instances)
		call2(r.ctx, ctxUnmap, r.instBuf, 0)
	}
	r.scratch.globals = [4]float32{float32(s.Width), float32(s.Height), 0, 0}
	call6(r.ctx, ctxUpdateSubresource, r.cbuf, 0, 0, uintptr(unsafe.Pointer(&r.scratch.globals[0])), 0, 0)
	upload += time.Since(uploadRest)

	ctx := r.ctx
	call3(ctx, ctxOMSetRenderTargets, 1, uintptr(unsafe.Pointer(&r.rtv)), 0)
	r.scratch.clear = s.Clear.Premul(1)
	call2(ctx, ctxClearRenderTargetView, r.rtv, uintptr(unsafe.Pointer(&r.scratch.clear[0])))
	call1(ctx, ctxRSSetState, r.raster)
	call1(ctx, ctxIASetPrimitiveTopology, topologyTriangleStrip)
	r.bindState(s, 0)
	r.scratch.boundSRV = 0
	ps := r.ps
	for i := range r.b.Batches {
		b := &r.b.Batches[i]
		if b.Count == 0 {
			continue
		}
		want := r.ps
		if b.Effect != nil {
			if want = r.effectShader(b.Effect); want == 0 {
				continue
			}
		}
		if b.Backdrop != 0 {
			// The effect shows what is drawn so far.
			r.bindState(s, r.backdropView(b.Backdrop-1, salt))
			r.scratch.boundSRV, ps = 0, r.ps
		}
		if want != ps {
			ps = want
			call3(ctx, ctxPSSetShader, ps, 0, 0)
		}
		if b.Image != 0 && b.Image != r.scratch.boundSRV {
			r.scratch.boundSRV = b.Image
			call3(ctx, ctxPSSetShaderResources, 2, 1, uintptr(unsafe.Pointer(&r.scratch.boundSRV)))
		}
		call2(ctx, ctxRSSetScissorRects, 1, uintptr(unsafe.Pointer(&b.Scissor)))
		call4(ctx, ctxDrawInstanced, 4, uintptr(b.Count), 0, uintptr(b.Start))
	}
	if maskIdx >= 0 && r.maskBlend != 0 {
		if ps != r.ps {
			call3(ctx, ctxPSSetShader, r.ps, 0, 0)
		}
		r.scratch.scissor = gpu.Scissor{Left: 0, Top: 0, Right: int32(s.Width), Bottom: int32(s.Height)}
		call2(ctx, ctxRSSetScissorRects, 1, uintptr(unsafe.Pointer(&r.scratch.scissor)))
		call3(ctx, ctxOMSetBlendState, r.maskBlend, 0, 0xffffffff)
		call4(ctx, ctxDrawInstanced, 4, 1, 0, uintptr(maskIdx))
	}
	r.lastUpload = upload
	r.lastDraw = time.Since(drawStart) - upload
	return nil
}

// effectShader is the pixel shader of an effect, or why it has none.
type effectShader struct {
	ps  uintptr
	err error
}

// effectShader returns the pixel shader of e: that e compiled ahead of
// time, unless it was compiled from another source, as when the
// renderer's head or tail changed since, which it then compiles. It is 0
// for an effect that does not compile, which draws nothing.
func (r *Renderer) effectShader(e *scene.Effect) uintptr {
	if p := r.effects[e]; p != nil {
		return p.ps
	}
	p := &effectShader{}
	if r.effects == nil {
		r.effects = map[*scene.Effect]*effectShader{}
	}
	r.effects[e] = p
	code := e.HLSL.Compiled
	if len(code) == 0 || e.HLSL.Sum != gpu.SourceSum(EffectSource(e.HLSL.Source)) {
		code, p.err = CompileEffect(e.HLSL.Source)
	}
	if p.err == nil {
		if hr := call4(r.device, devCreatePixelShader, uintptr(unsafe.Pointer(&code[0])), uintptr(len(code)), 0, uintptr(unsafe.Pointer(&p.ps))); failed(hr) {
			p.err = fmt.Errorf("d3d11: cannot create the pixel shader of the effect %s: %#x", e.Name, uint32(hr))
		}
	} else {
		p.err = fmt.Errorf("d3d11: the effect %s: %w", e.Name, p.err)
	}
	if p.err != nil {
		// Shown once, as frames go on without the effect.
		fmt.Fprintln(os.Stderr, p.err)
	}
	return p.ps
}

// bindState sets the state the instances of s draw with, into the back
// buffer, with the backdrop of an effect.
func (r *Renderer) bindState(s *scene.Scene, backdrop uintptr) {
	ctx := r.ctx
	call3(ctx, ctxOMSetRenderTargets, 1, uintptr(unsafe.Pointer(&r.rtv)), 0)
	r.scratch.viewport = viewport{W: float32(s.Width), H: float32(s.Height), MaxDepth: 1}
	call2(ctx, ctxRSSetViewports, 1, uintptr(unsafe.Pointer(&r.scratch.viewport)))
	call3(ctx, ctxOMSetBlendState, r.blend, 0, 0xffffffff)
	call1(ctx, ctxIASetInputLayout, r.layout)
	r.scratch.stride, r.scratch.offset = uint32(gpu.InstanceSize), 0
	call5(ctx, ctxIASetVertexBuffers, 0, 1, uintptr(unsafe.Pointer(&r.instBuf)), uintptr(unsafe.Pointer(&r.scratch.stride)), uintptr(unsafe.Pointer(&r.scratch.offset)))
	call3(ctx, ctxVSSetShader, r.vs, 0, 0)
	call3(ctx, ctxVSSetConstantBuffers, 0, 1, uintptr(unsafe.Pointer(&r.cbuf)))
	call3(ctx, ctxPSSetShader, r.ps, 0, 0)
	call3(ctx, ctxPSSetSamplers, 0, 1, uintptr(unsafe.Pointer(&r.samp)))
	r.scratch.views = [4]uintptr{r.mask.srv, r.color.srv, 0, backdrop}
	call3(ctx, ctxPSSetShaderResources, 0, 4, uintptr(unsafe.Pointer(&r.scratch.views[0])))
}

// readBackdrop computes the backdrop bk of what the back buffer holds,
// into r.backdrop[0]: it copies the area into grab, averages its squares
// and blurs them.
func (r *Renderer) readBackdrop(bk scene.Backdrop) {
	ctx := r.ctx
	// No texture drawn into may be bound for reading.
	call3(ctx, ctxPSSetShaderResources, 0, 5, uintptr(unsafe.Pointer(&r.scratch.noneViews[0])))
	r.scratch.back = 0
	call1(r.rtv, viewGetResource, uintptr(unsafe.Pointer(&r.scratch.back)))
	r.scratch.copyBox = box{Left: uint32(bk.Area.Min.X), Top: uint32(bk.Area.Min.Y), Right: uint32(bk.Area.Max.X), Bottom: uint32(bk.Area.Max.Y), Back: 1}
	call8(ctx, ctxCopySubresourceRegion, r.grab.tex, 0, 0, 0, 0, r.scratch.back, 0, uintptr(unsafe.Pointer(&r.scratch.copyBox)))
	free(&r.scratch.back)
	call3(ctx, ctxOMSetBlendState, 0, 0, 0xffffffff)
	call1(ctx, ctxIASetInputLayout, 0)
	call3(ctx, ctxVSSetShader, r.passVS, 0, 0)
	call3(ctx, ctxPSSetConstantBuffers, 1, 1, uintptr(unsafe.Pointer(&r.passCB)))
	call3(ctx, ctxPSSetSamplers, 0, 1, uintptr(unsafe.Pointer(&r.samp)))
	w, h := bk.Size()
	r.prepareDownPass(bk, [2]int32{int32(bk.Area.Min.X), int32(bk.Area.Min.Y)}, r.grab.w, r.grab.h)
	r.pass(r.downPS, r.grab.srv, &r.backdrop[0], w, h)
	if bk.Radius > 0 {
		r.prepareBlurPass(bk, [2]int32{1, 0}, r.backdrop[0].w, r.backdrop[0].h)
		r.pass(r.blurPS, r.backdrop[0].srv, &r.backdrop[1], w, h)
		r.prepareBlurPass(bk, [2]int32{0, 1}, r.backdrop[1].w, r.backdrop[1].h)
		r.pass(r.blurPS, r.backdrop[1].srv, &r.backdrop[0], w, h)
	}
}

func (r *Renderer) prepareDownPass(bk scene.Backdrop, shift [2]int32, srcW, srcH int) {
	p := gpu.DownPass(bk, shift)
	c := &r.scratch.passConst
	c.Origin, c.Limit, c.Shift, c.Dir = p.Origin, p.Limit, p.Shift, p.Dir
	c.Down, c.Radius, c.Sigma, c.TapCount = p.Down, 0, 0, 0
	c.InvSize = [2]float32{1 / float32(max(srcW, 1)), 1 / float32(max(srcH, 1))}
	c.CenterW, c.Pad = 0, 0
}

func (r *Renderer) prepareBlurPass(bk scene.Backdrop, dir [2]int32, srcW, srcH int) {
	p := gpu.BlurPass(bk, dir)
	if r.blurCacheRadius != bk.Radius || r.blurCacheSigma != bk.Sigma {
		r.blurCacheCenterW, r.blurCacheTapCount, r.blurCacheTaps = computeBlurTaps(bk.Radius, bk.Sigma)
		r.blurCacheRadius = bk.Radius
		r.blurCacheSigma = bk.Sigma
	}
	c := &r.scratch.passConst
	c.Origin, c.Limit, c.Shift, c.Dir = p.Origin, p.Limit, p.Shift, p.Dir
	c.Down, c.Radius, c.Sigma = 0, p.Radius, p.Sigma
	c.TapCount = int32(r.blurCacheTapCount)
	c.InvSize = [2]float32{1 / float32(max(srcW, 1)), 1 / float32(max(srcH, 1))}
	c.CenterW, c.Pad = r.blurCacheCenterW, 0
	if r.blurCacheTapCount > 0 {
		copy(c.Taps[:r.blurCacheTapCount], r.blurCacheTaps[:r.blurCacheTapCount])
	}
}

// pass draws ps into the w×h texels at the start of dst, reading src and
// r.scratch.passConst.
func (r *Renderer) pass(ps, src uintptr, dst *texture, w, h int) {
	ctx := r.ctx
	call6(ctx, ctxUpdateSubresource, r.passCB, 0, 0, uintptr(unsafe.Pointer(&r.scratch.passConst)), 0, 0)
	call3(ctx, ctxOMSetRenderTargets, 1, uintptr(unsafe.Pointer(&dst.rtv)), 0)
	r.scratch.viewport = viewport{W: float32(dst.w), H: float32(dst.h), MaxDepth: 1}
	call2(ctx, ctxRSSetViewports, 1, uintptr(unsafe.Pointer(&r.scratch.viewport)))
	r.scratch.scissor = gpu.Scissor{Right: int32(w), Bottom: int32(h)}
	call2(ctx, ctxRSSetScissorRects, 1, uintptr(unsafe.Pointer(&r.scratch.scissor)))
	call3(ctx, ctxPSSetShader, ps, 0, 0)
	r.scratch.passSrc = src
	call3(ctx, ctxPSSetShaderResources, 4, 1, uintptr(unsafe.Pointer(&r.scratch.passSrc)))
	call4(ctx, ctxDrawInstanced, 4, 1, 0, 0)
	r.scratch.passNone = 0
	call3(ctx, ctxPSSetShaderResources, 4, 1, uintptr(unsafe.Pointer(&r.scratch.passNone)))
}

// shouldShrinkTarget reports whether an existing scratch target of curW×curH
// is significantly oversized for a needW×needH request (with hysteresis so
// small popovers and minor resizes do not churn).
func shouldShrinkTarget(curW, curH, needW, needH int) bool {
	if curW == 0 || curH == 0 || needW <= 0 || needH <= 0 {
		return false
	}
	wantW, wantH := needW+needW/4, needH+needH/4
	if curW < wantW || curH < wantH {
		return false
	}
	const minShrinkPixels = 65536 // 256 KiB at 4 Bpp
	if curW*curH <= minShrinkPixels {
		return false
	}
	return curW*curH >= 4*needW*needH || curW >= 2*wantW || curH >= 2*wantH
}

// fitBackdrop makes the textures of backdrops as large as the scene built
// needs, and the texture the area read is copied to, shrinking them back down
// when the window or backdrop area shrinks significantly.
func (r *Renderer) fitBackdrop() error {
	w, h := r.b.BackdropSize()
	aw, ah := 0, 0
	for _, bk := range r.b.Backdrops {
		aw, ah = max(aw, bk.Area.Dx()), max(ah, bk.Area.Dy())
	}
	// Somewhat larger, so that a pane growing does not make them again
	// every frame.
	if aw > r.grab.w || ah > r.grab.h {
		if err := r.newTarget(&r.grab, max(aw+aw/4, r.grab.w), max(ah+ah/4, r.grab.h), false); err != nil {
			return err
		}
	} else if shouldShrinkTarget(r.grab.w, r.grab.h, aw, ah) {
		if err := r.newTarget(&r.grab, aw+aw/4, ah+ah/4, false); err != nil {
			return err
		}
	}
	if w > r.backdrop[0].w || h > r.backdrop[0].h {
		w, h = max(w+w/4, r.backdrop[0].w), max(h+h/4, r.backdrop[0].h)
		for i := range r.backdrop {
			if err := r.newTarget(&r.backdrop[i], w, h, true); err != nil {
				return err
			}
		}
	} else if shouldShrinkTarget(r.backdrop[0].w, r.backdrop[0].h, w, h) {
		w, h = w+w/4, h+h/4
		for i := range r.backdrop {
			if err := r.newTarget(&r.backdrop[i], w, h, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// newTarget makes t a w×h texture of the back buffer's format, which
// shaders read and, when drawn, draw into.
func (r *Renderer) newTarget(t *texture, w, h int, drawn bool) error {
	free(&t.rtv)
	free(&t.srv)
	free(&t.tex)
	*t = texture{}
	desc := texture2DDesc{Width: uint32(w), Height: uint32(h), MipLevels: 1, ArraySize: 1, Format: formatB8G8R8A8Unorm, SampleCount: 1,
		Usage: usageDefault, BindFlags: bindShaderResource}
	if drawn {
		desc.BindFlags |= bindRenderTarget
	}
	if hr := call3(r.device, devCreateTexture2D, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&t.tex))); failed(hr) {
		return fmt.Errorf("d3d11: cannot create a %dx%d backdrop texture: %#x", w, h, uint32(hr))
	}
	if hr := call3(r.device, devCreateShaderResourceView, t.tex, 0, uintptr(unsafe.Pointer(&t.srv))); failed(hr) {
		return fmt.Errorf("d3d11: cannot create a backdrop texture view: %#x", uint32(hr))
	}
	if drawn {
		if hr := call3(r.device, devCreateRenderTargetView, t.tex, 0, uintptr(unsafe.Pointer(&t.rtv))); failed(hr) {
			return fmt.Errorf("d3d11: cannot create a backdrop render target: %#x", uint32(hr))
		}
	}
	t.w, t.h = w, h
	return nil
}

// present shows the back buffer, after sync vertical blanks.
func (r *Renderer) present(sync uintptr) error {
	acquired := r.slotAcquired
	r.slotAcquired = false
	if r.waitable != 0 {
		if !acquired {
			// Consume an available slot before presenting (covers first frame,
			// immediatePaint, inSizeMove, inMenu, deferredForce and direct
			// Render calls) so the initial/idle semaphore count never drifts.
			for r.unpacedPresents > 0 && r.waitFrameNonBlocking() {
				r.unpacedPresents--
			}
			if r.unpacedPresents == 0 && r.waitFrameNonBlocking() {
				acquired = true
			}
		}
		if acquired {
			// We hold this frame's slot and are about to Present; drain any
			// excess drifted count safely back to zero.
			r.drainExtraWaitable()
		}
	}
	hr := call2(r.swapChain, scPresent, sync, 0)
	if failed(hr) {
		if acquired && r.waitable != 0 {
			// Present did not enqueue a frame, so preserve the acquired slot.
			r.slotAcquired = true
		}
		return fmt.Errorf("d3d11: present failed: %#x", uint32(hr))
	}
	now := time.Now()
	backToBack := !r.lastPresentAt.IsZero() && now.Sub(r.lastPresentAt) < 30*time.Millisecond
	r.lastPresentAt = now
	if r.waitable != 0 && !acquired {
		// Present ran without holding a semaphore token first (an earlier frame
		// was still in flight). Reconcile the earlier frame's completion signal
		// immediately if Present waited for it; otherwise, when this was a
		// back-to-back forced/resize present (not a 32ms timeout fallback after
		// an empty/coalesced queue), record 1 unit of pending debt (capped at 1
		// to match MaxFrameLatency=1) so the next TryAcquireFrame/MarkFrameReady
		// absorbs the delayed signal instead of letting S+I drift to >= 2.
		if !r.waitFrameNonBlocking() && backToBack {
			r.unpacedPresents = 1
		}
	}
	// Forget the textures of images no frame drew for a while, and keep total
	// cached image VRAM within budget.
	if r.frame%120 == 0 {
		for id, t := range r.images {
			if r.frame-t.lastFrame > 240 {
				r.deleteImage(id, t)
			}
		}
	}
	if r.imageBytes > r.effectiveImageBudget() {
		r.evictImages(r.effectiveImageBudget(), false)
	}
	return nil
}
