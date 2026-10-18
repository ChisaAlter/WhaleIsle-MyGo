// Package svgrender adapts the MIT Go wrapper from pptx-glimpse-go v0.0.2.
// It runs pinned resvg 0.46.0 (MIT/Apache-2.0) with wazero and no CGo.
// Exact source, licenses and build provenance are retained under upstream/.
package svgrender

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// The checksum-pinned renderer artifact is built from the source and recipe
// retained in upstream/resvg-wasm. It is never generated at application runtime.
//
//go:embed wasm/resvg.wasm.gz
var wasmGz []byte

var (
	wasmOnce  sync.Once
	wasmBytes []byte
	wasmErr   error
)

// moduleBytes gunzips the embedded module once and caches it.
func moduleBytes() ([]byte, error) {
	wasmOnce.Do(func() {
		zr, err := gzip.NewReader(bytes.NewReader(wasmGz))
		if err != nil {
			wasmErr = err
			return
		}
		defer zr.Close()
		wasmBytes, wasmErr = io.ReadAll(zr)
	})
	return wasmBytes, wasmErr
}

// sharedModule compiles the resvg WASM exactly once on a process-wide runtime and caches the result.
// Compilation (wazero translating the module to native code) is the expensive step — doing it on every
// render was the dominant cost (~1s/render, the whole reason the Go path was ~5x slower than the native
// resvg binding). The runtime + CompiledModule are immutable and safe to reuse
// concurrently; each render still instantiates its own isolated module instance (see NewContext), so the
// per-instance "not concurrency-safe" contract is unchanged — only the compile is now amortised.
var (
	sharedOnce sync.Once
	sharedRT   wazero.Runtime
	sharedMod  wazero.CompiledModule
	sharedErr  error
)

func sharedModule() (wazero.Runtime, wazero.CompiledModule, error) {
	sharedOnce.Do(func() {
		ctx := context.Background()
		wasm, err := moduleBytes()
		if err != nil {
			sharedErr = fmt.Errorf("resvg: decompress wasm: %w", err)
			return
		}
		rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(8192).WithCloseOnContextDone(true))
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
			_ = rt.Close(ctx)
			sharedErr = err
			return
		}
		compiled, err := rt.CompileModule(ctx, wasm)
		if err != nil {
			_ = rt.Close(ctx)
			sharedErr = fmt.Errorf("resvg: compile wasm: %w", err)
			return
		}
		sharedRT, sharedMod = rt, compiled
	})
	return sharedRT, sharedMod, sharedErr
}

// Context owns a wazero runtime and the instantiated resvg module. It is NOT safe for concurrent
// use; create one per rasterisation (as png_converter.go already does).
type Context struct {
	ctx     context.Context
	runtime wazero.Runtime
	mod     api.Module
	memory  api.Memory

	fnAlloc    api.Function
	fnDealloc  api.Function
	fnNew      api.Function
	fnFree     api.Function
	fnLoadFont api.Function
	fnRender   api.Function
}

// NewContext instantiates a fresh, isolated resvg module instance from the process-wide compiled module
// (compiled once by sharedModule). Each Context owns one instance and is NOT concurrency-safe; create one
// per rasterisation (as png_converter.go does). Only the cheap instantiation happens here — the expensive
// compile is paid once across the process, not per render.
func NewContext(ctx context.Context) (*Context, error) {
	rt, compiled, err := sharedModule()
	if err != nil {
		return nil, err
	}

	// A cdylib on wasm32-wasip1 is a WASI *reactor* module: run `_initialize` (not `_start`) on
	// instantiation to set up static state before any export is called. WithName("") makes the instance
	// anonymous so the same compiled module can be instantiated many times in the shared runtime.
	cfg := wazero.NewModuleConfig().WithStartFunctions("_initialize").WithName("")
	mod, err := rt.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("resvg: instantiate wasm: %w", err)
	}

	c := &Context{
		ctx: ctx, runtime: rt, mod: mod, memory: mod.Memory(),
		fnAlloc:    mod.ExportedFunction("alloc"),
		fnDealloc:  mod.ExportedFunction("dealloc"),
		fnNew:      mod.ExportedFunction("renderer_new"),
		fnFree:     mod.ExportedFunction("renderer_free"),
		fnLoadFont: mod.ExportedFunction("renderer_load_font"),
		fnRender:   mod.ExportedFunction("renderer_render"),
	}
	if c.memory == nil || c.fnAlloc == nil || c.fnDealloc == nil || c.fnNew == nil ||
		c.fnFree == nil || c.fnLoadFont == nil || c.fnRender == nil {
		_ = mod.Close(ctx)
		return nil, errors.New("resvg: wasm module is missing expected exports")
	}
	return c, nil
}

// Close releases this module instance. The shared runtime + compiled module persist for the process
// lifetime (reclaimed by the OS on exit), so only the per-render instance is torn down here.
func (c *Context) Close() error { return c.mod.Close(c.ctx) }

// Renderer is a resvg renderer handle with its own font database.
type Renderer struct {
	c      *Context
	handle uint64 // pointer to the Rust Renderer in linear memory
}

// NewRenderer allocates a renderer (empty font database) inside the WASM instance.
func (c *Context) NewRenderer() (*Renderer, error) {
	res, err := c.fnNew.Call(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("resvg: renderer_new: %w", err)
	}
	if res[0] == 0 {
		return nil, errors.New("resvg: renderer_new returned null")
	}
	return &Renderer{c: c, handle: res[0]}, nil
}

// Close frees the renderer handle.
func (r *Renderer) Close() error {
	_, err := r.c.fnFree.Call(r.c.ctx, r.handle)
	return err
}

// LoadFontData registers raw TTF/OTF/TTC bytes as font faces for <text> rendering.
func (r *Renderer) LoadFontData(font []byte) error {
	if len(font) == 0 {
		return nil
	}
	ptr, err := r.c.writeBytes(font)
	if err != nil {
		return err
	}
	defer r.c.free(ptr, uint32(len(font)))
	_, err = r.c.fnLoadFont.Call(r.c.ctx, r.handle, uint64(ptr), uint64(len(font)))
	return err
}

// RenderWithSize rasterises the SVG into a width x height PNG bitmap.
func (r *Renderer) RenderWithSize(svg []byte, width, height uint32) ([]byte, error) {
	c := r.c
	svgPtr, err := c.writeBytes(svg)
	if err != nil {
		return nil, err
	}
	defer c.free(svgPtr, uint32(len(svg)))

	outLenPtr, err := c.alloc(4) // *usize is 4 bytes on wasm32
	if err != nil {
		return nil, err
	}
	defer c.free(outLenPtr, 4)

	res, err := c.fnRender.Call(c.ctx, r.handle, uint64(svgPtr), uint64(len(svg)),
		uint64(width), uint64(height), uint64(outLenPtr))
	if err != nil {
		return nil, fmt.Errorf("resvg: render: %w", err)
	}
	pngPtr := uint32(res[0])
	if pngPtr == 0 {
		return nil, errors.New("resvg: render returned null (parse or rasterise failure)")
	}
	pngLen, ok := c.memory.ReadUint32Le(outLenPtr)
	if !ok {
		return nil, errors.New("resvg: failed to read output length")
	}
	src, ok := c.memory.Read(pngPtr, pngLen)
	if !ok {
		c.free(pngPtr, pngLen)
		return nil, errors.New("resvg: failed to read PNG bytes from linear memory")
	}
	if pngLen > 128<<20 {
		c.free(pngPtr, pngLen)
		return nil, errors.New("resvg: output exceeds image budget")
	}
	out := make([]byte, pngLen)
	copy(out, src)
	c.free(pngPtr, pngLen)
	return out, nil
}

// --- low-level linear-memory helpers ---

func (c *Context) alloc(n uint32) (uint32, error) {
	res, err := c.fnAlloc.Call(c.ctx, uint64(n))
	if err != nil {
		return 0, fmt.Errorf("resvg: alloc: %w", err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, errors.New("resvg: alloc returned null")
	}
	return ptr, nil
}

func (c *Context) free(ptr, n uint32) { _, _ = c.fnDealloc.Call(c.ctx, uint64(ptr), uint64(n)) }

func (c *Context) writeBytes(b []byte) (uint32, error) {
	ptr, err := c.alloc(uint32(len(b)))
	if err != nil {
		return 0, err
	}
	if !c.memory.Write(ptr, b) {
		c.free(ptr, uint32(len(b)))
		return 0, errors.New("resvg: memory write out of range")
	}
	return ptr, nil
}
