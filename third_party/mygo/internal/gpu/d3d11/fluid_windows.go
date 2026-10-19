//go:build windows

package d3d11

import (
	_ "embed"
	"fmt"
	"image"
	"math"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/d3d11/device"
)

//go:embed fluid.hlsl
var fluidSource string

// Fluid owns a persistent half-float simulation device on its worker. It needs
// no HWND, swap chain, browser or main-thread work. Close must run on its owner.
type Fluid struct {
	device, ctx, vs, constants, linear, nearest, raster uintptr
	shaders                                             map[string]uintptr
	velocity, dye, pressure                             [2]texture
	curl, divergence, output                            texture
	staging                                             uintptr
	width, height                                       int
	velocityFormat, dyeFormat, scalarFormat             uint32
}

// NewFluid uses the SDK's D3D11 loader/compiler. Shader failure is returned;
// there is no lower-resolution CPU substitution or recreation retry.
func NewFluid() (_ *Fluid, err error) {
	f := &Fluid{shaders: make(map[string]uintptr)}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	var software bool
	f.device, f.ctx, software, err = device.New()
	if err != nil {
		return nil, err
	}
	// Source mountFluidCursor requires an actual float-renderable GPU context.
	if software {
		return nil, fmt.Errorf("fluid: hardware D3D11 is unavailable")
	}
	// Match the source's explicit R16F -> RG16F -> RGBA16F format probing;
	// this changes channel storage only, never the simulation resolutions.
	if f.velocityFormat, err = f.format(34, 10); err != nil {
		return nil, err
	}
	if f.dyeFormat, err = f.format(10); err != nil {
		return nil, err
	}
	if f.scalarFormat, err = f.format(54, 34, 10); err != nil {
		return nil, err
	}
	for _, entry := range []string{"fluidVS", "copyPS", "splatPS", "curlPS", "vorticityPS", "divergencePS", "clearPS", "pressurePS", "gradientPS", "advectionPS", "displayPS"} {
		target, method := "ps_4_0", devCreatePixelShader
		if entry == "fluidVS" {
			target, method = "vs_4_0", devCreateVertexShader
		}
		code, e := compile(fluidSource, entry, target)
		if e != nil {
			return nil, e
		}
		var shader uintptr
		hr := call(f.device, method, uintptr(unsafe.Pointer(&code[0])), uintptr(len(code)), 0, uintptr(unsafe.Pointer(&shader)))
		if failed(hr) {
			return nil, fmt.Errorf("fluid: create %s: %#x", entry, uint32(hr))
		}
		if entry == "fluidVS" {
			f.vs = shader
		} else {
			f.shaders[entry] = shader
		}
	}
	desc := bufferDesc{ByteWidth: 48, BindFlags: bindConstantBuffer}
	if hr := call(f.device, devCreateBuffer, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&f.constants))); failed(hr) {
		return nil, fmt.Errorf("fluid: constants: %#x", uint32(hr))
	}
	for i, out := range []*uintptr{&f.linear, &f.nearest} {
		sampler := samplerDesc{Filter: 0x15, AddressU: 3, AddressV: 3, AddressW: 3, MaxLOD: math.MaxFloat32}
		if i == 1 {
			sampler.Filter = 0
		}
		if hr := call(f.device, devCreateSamplerState, uintptr(unsafe.Pointer(&sampler)), uintptr(unsafe.Pointer(out))); failed(hr) {
			return nil, fmt.Errorf("fluid: sampler: %#x", uint32(hr))
		}
	}
	rs := rasterizerDesc{FillMode: 3, CullMode: 1, DepthClip: 1}
	if hr := call(f.device, devCreateRasterizerState, uintptr(unsafe.Pointer(&rs)), uintptr(unsafe.Pointer(&f.raster))); failed(hr) {
		return nil, fmt.Errorf("fluid: raster: %#x", uint32(hr))
	}
	call(f.ctx, ctxIASetPrimitiveTopology, 4) // triangle list, SV_VertexID fullscreen triangle
	call(f.ctx, ctxVSSetShader, f.vs, 0, 0)
	call(f.ctx, ctxPSSetConstantBuffers, 0, 1, uintptr(unsafe.Pointer(&f.constants)))
	samplers := [2]uintptr{f.linear, f.nearest}
	call(f.ctx, ctxPSSetSamplers, 0, 2, uintptr(unsafe.Pointer(&samplers[0])))
	call(f.ctx, ctxRSSetState, f.raster)
	return f, nil
}

func (f *Fluid) format(formats ...uint32) (uint32, error) {
	const required = 0x20 | 0x200 | 0x4000 // TEXTURE2D, SHADER_SAMPLE, RENDER_TARGET
	for _, format := range formats {
		var support uint32
		if hr := call(f.device, 29, uintptr(format), uintptr(unsafe.Pointer(&support))); !failed(hr) && support&required == required {
			return format, nil
		}
	}
	return 0, fmt.Errorf("fluid: half-float render textures are unavailable")
}

func fluidResolution(width, height, resolution int) (int, int) {
	if width > height {
		return int(math.Round(float64(resolution) * float64(width) / float64(height))), resolution
	}
	return resolution, int(math.Round(float64(resolution) * float64(height) / float64(width)))
}

// FluidFits bounds actual texture extents and retained GPU/readback memory.
// The short edges remain 128/1440; an excessive aspect fails instead of degrading.
func FluidFits(width, height int) bool {
	if width <= 0 || height <= 0 || width > 16384 || height > 16384 {
		return false
	}
	sw, sh := fluidResolution(width, height, 128)
	dw, dh := fluidResolution(width, height, 1440)
	return sw <= 16384 && sh <= 16384 && dw <= 16384 && dh <= 16384 && int64(sw)*int64(sh)*48+int64(dw)*int64(dh)*16+int64(width)*int64(height)*16 <= 256<<20
}

func (f *Fluid) target(width, height int, format uint32) (t texture, err error) {
	t.w, t.h = width, height
	desc := texture2DDesc{Width: uint32(width), Height: uint32(height), MipLevels: 1, ArraySize: 1, Format: format, SampleCount: 1, BindFlags: bindShaderResource | bindRenderTarget}
	defer func() {
		if err != nil {
			free(&t.srv)
			free(&t.rtv)
			free(&t.tex)
		}
	}()
	if hr := call(f.device, devCreateTexture2D, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&t.tex))); failed(hr) {
		return t, fmt.Errorf("fluid: texture: %#x", uint32(hr))
	}
	if hr := call(f.device, devCreateShaderResourceView, t.tex, 0, uintptr(unsafe.Pointer(&t.srv))); failed(hr) {
		return t, fmt.Errorf("fluid: texture view: %#x", uint32(hr))
	}
	if hr := call(f.device, devCreateRenderTargetView, t.tex, 0, uintptr(unsafe.Pointer(&t.rtv))); failed(hr) {
		return t, fmt.Errorf("fluid: target view: %#x", uint32(hr))
	}
	zero := [4]float32{}
	call(f.ctx, ctxClearRenderTargetView, t.rtv, uintptr(unsafe.Pointer(&zero[0])))
	return t, nil
}

func releaseFluidTexture(t *texture) { free(&t.srv); free(&t.rtv); free(&t.tex); *t = texture{} }

func (f *Fluid) resizePair(pair *[2]texture, w, h int, format uint32, preserve bool) error {
	if pair[0].w == w && pair[0].h == h {
		return nil
	}
	var next [2]texture
	for i := range next {
		t, err := f.target(w, h, format)
		if err != nil {
			for j := range next {
				releaseFluidTexture(&next[j])
			}
			return err
		}
		next[i] = t
	}
	if preserve && pair[0].tex != 0 {
		f.pass("copyPS", &next[0], pair[0].srv, 0, [3][4]float32{})
	}
	for i := range pair {
		releaseFluidTexture(&pair[i])
	}
	*pair = next
	return nil
}

// Resize preserves the readable dye and velocity through linear resampling;
// pressure/curl/divergence restart whenever the drawing buffer changes.
func (f *Fluid) Resize(width, height int) error {
	if width == f.width && height == f.height {
		return nil
	}
	if !FluidFits(width, height) {
		return fmt.Errorf("fluid: drawing-buffer dimensions exceed the GPU budget")
	}
	sw, sh := fluidResolution(width, height, 128)
	dw, dh := fluidResolution(width, height, 1440)
	if err := f.resizePair(&f.velocity, sw, sh, f.velocityFormat, true); err != nil {
		return err
	} // R16G16_FLOAT
	if err := f.resizePair(&f.dye, dw, dh, f.dyeFormat, true); err != nil {
		return err
	} // R16G16B16A16_FLOAT
	for i := range f.pressure {
		releaseFluidTexture(&f.pressure[i])
	}
	if err := f.resizePair(&f.pressure, sw, sh, f.scalarFormat, false); err != nil {
		return err
	} // R16_FLOAT
	for _, t := range []*texture{&f.curl, &f.divergence, &f.output} {
		releaseFluidTexture(t)
	}
	var err error
	if f.curl, err = f.target(sw, sh, f.scalarFormat); err != nil {
		return err
	}
	if f.divergence, err = f.target(sw, sh, f.scalarFormat); err != nil {
		return err
	}
	if f.output, err = f.target(width, height, formatR8G8B8A8Unorm); err != nil {
		return err
	}
	free(&f.staging)
	desc := texture2DDesc{Width: uint32(width), Height: uint32(height), MipLevels: 1, ArraySize: 1, Format: formatR8G8B8A8Unorm, SampleCount: 1, Usage: 3, CPUAccessFlags: 0x20000}
	if hr := call(f.device, devCreateTexture2D, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&f.staging))); failed(hr) {
		return fmt.Errorf("fluid: staging: %#x", uint32(hr))
	}
	f.width, f.height = width, height
	return nil
}

func (f *Fluid) pass(name string, dst *texture, src, aux uintptr, params [3][4]float32) {
	none := [2]uintptr{}
	call(f.ctx, ctxPSSetShaderResources, 0, 2, uintptr(unsafe.Pointer(&none[0])))
	call(f.ctx, ctxOMSetRenderTargets, 1, uintptr(unsafe.Pointer(&dst.rtv)), 0)
	vp := viewport{W: float32(dst.w), H: float32(dst.h), MaxDepth: 1}
	call(f.ctx, ctxRSSetViewports, 1, uintptr(unsafe.Pointer(&vp)))
	call(f.ctx, ctxUpdateSubresource, f.constants, 0, 0, uintptr(unsafe.Pointer(&params[0][0])), 0, 0)
	views := [2]uintptr{src, aux}
	call(f.ctx, ctxPSSetShaderResources, 0, 2, uintptr(unsafe.Pointer(&views[0])))
	call(f.ctx, ctxPSSetShader, f.shaders[name], 0, 0)
	call(f.ctx, 13, 3, 0) // ID3D11DeviceContext::Draw
	// Unbind both sides before any later resize/release/copy of these resources.
	call(f.ctx, ctxPSSetShaderResources, 0, 2, uintptr(unsafe.Pointer(&none[0])))
	call(f.ctx, ctxOMSetRenderTargets, 0, 0, 0)
}

func swapFluid(pair *[2]texture) { pair[0], pair[1] = pair[1], pair[0] }

func (f *Fluid) Splat(x, y, dx, dy float32, color [3]float32, size float32) {
	aspect := float32(f.width) / float32(f.height)
	radius := .2 * (size / 100) / 100
	if aspect > 1 {
		radius *= aspect
	}
	params := [3][4]float32{{}, {x, y, radius, aspect}, {dx, dy, 0, 0}}
	f.pass("splatPS", &f.velocity[1], f.velocity[0].srv, 0, params)
	swapFluid(&f.velocity)
	params[2] = [4]float32{color[0], color[1], color[2], 0}
	f.pass("splatPS", &f.dye[1], f.dye[0].srv, 0, params)
	swapFluid(&f.dye)
}

func (f *Fluid) Step(dt, speed float32) {
	params := [3][4]float32{{1 / float32(f.velocity[0].w), 1 / float32(f.velocity[0].h), min(max(dt, 0), .016666), 0}}
	f.pass("curlPS", &f.curl, f.velocity[0].srv, 0, params)
	f.pass("vorticityPS", &f.velocity[1], f.velocity[0].srv, f.curl.srv, params)
	swapFluid(&f.velocity)
	f.pass("divergencePS", &f.divergence, f.velocity[0].srv, 0, params)
	f.pass("clearPS", &f.pressure[1], f.pressure[0].srv, 0, params)
	swapFluid(&f.pressure)
	for range 20 {
		f.pass("pressurePS", &f.pressure[1], f.pressure[0].srv, f.divergence.srv, params)
		swapFluid(&f.pressure)
	}
	f.pass("gradientPS", &f.velocity[1], f.velocity[0].srv, f.pressure[0].srv, params)
	swapFluid(&f.velocity)
	params[0][3] = 2
	f.pass("advectionPS", &f.velocity[1], f.velocity[0].srv, f.velocity[0].srv, params)
	swapFluid(&f.velocity)
	params[0][3] = 3.5 * speed / 100
	f.pass("advectionPS", &f.dye[1], f.dye[0].srv, f.velocity[0].srv, params)
	swapFluid(&f.dye)
}

func (f *Fluid) ImageSize() (int, int) { return f.width, f.height }

// Render is the existing native bitmap bridge. The simulation stays on GPU at
// source resolution; only the final drawing-buffer-sized RGBA frame is read.
func (f *Fluid) Render(dst *image.RGBA) error {
	params := [3][4]float32{{1 / float32(f.width), 1 / float32(f.height), 0, 0}}
	f.pass("displayPS", &f.output, f.dye[0].srv, 0, params)
	call(f.ctx, 47, f.staging, f.output.tex) // CopyResource
	var m mapped
	if hr := call(f.ctx, ctxMap, f.staging, 0, 1, 0, uintptr(unsafe.Pointer(&m))); failed(hr) {
		return fmt.Errorf("fluid: GPU readback: %#x", uint32(hr))
	}
	defer call(f.ctx, ctxUnmap, f.staging, 0)
	for y := 0; y < f.height; y++ {
		copy(dst.Pix[y*dst.Stride:y*dst.Stride+f.width*4], unsafe.Slice((*byte)(ptr(m.Data+uintptr(y)*uintptr(m.RowPitch))), f.width*4))
	}
	if hr := call(f.device, 39); failed(hr) {
		return fmt.Errorf("fluid: device removed: %#x", uint32(hr))
	}
	return nil
}

func (f *Fluid) Close() {
	if f.ctx != 0 {
		call(f.ctx, ctxClearState)
	}
	for _, pair := range []*[2]texture{&f.velocity, &f.dye, &f.pressure} {
		for i := range pair {
			releaseFluidTexture(&pair[i])
		}
	}
	for _, t := range []*texture{&f.curl, &f.divergence, &f.output} {
		releaseFluidTexture(t)
	}
	free(&f.staging)
	for name, shader := range f.shaders {
		free(&shader)
		delete(f.shaders, name)
	}
	for _, p := range []*uintptr{&f.vs, &f.constants, &f.linear, &f.nearest, &f.raster, &f.ctx, &f.device} {
		free(p)
	}
}
