package imagepreview

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed avifcodec/avif.wasm.gz
var avifCodec []byte

type avifCompiled struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
}

var avifRuntime = sync.OnceValues(func() (avifCompiled, error) {
	ctx := context.Background()
	r, err := gzip.NewReader(bytes.NewReader(avifCodec))
	if err != nil {
		return avifCompiled{}, err
	}
	defer r.Close()
	wasm, err := io.ReadAll(r)
	if err != nil {
		return avifCompiled{}, err
	}
	config := wazero.NewRuntimeConfig().WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesThreads).WithMemoryLimitPages(8192).WithCloseOnContextDone(true)
	rt := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return avifCompiled{}, err
	}
	compiled, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		rt.Close(ctx)
		return avifCompiled{}, err
	}
	return avifCompiled{rt, compiled}, nil
})

// DecodeAVIF retains the existing single-item still-image acceptance policy.
// Each decode has isolated WASM memory, no filesystem/network/environment, and
// returns Go-owned pixels for the ordinary native zoom and drag viewer.
func DecodeAVIF(data []byte) (image.Image, error) {
	return DecodeAVIFContext(context.Background(), data)
}

// DecodeAVIFContext observes cancellation while running the isolated WASM decoder.
func DecodeAVIFContext(ctx context.Context, data []byte) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateAVIF(data); err != nil {
		return nil, err
	}
	codec, err := avifRuntime()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	mod, err := codec.runtime.InstantiateModule(ctx, codec.module, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, err
	}
	defer mod.Close(context.Background())
	alloc := func(n uint64) (uint64, error) {
		result, err := mod.ExportedFunction("malloc").Call(ctx, n)
		if err != nil {
			return 0, err
		}
		if len(result) != 1 || result[0] == 0 {
			return 0, errors.New("AVIF decoder allocation failed")
		}
		return result[0], nil
	}
	input, err := alloc(uint64(len(data)))
	if err != nil {
		return nil, err
	}
	if !mod.Memory().Write(uint32(input), data) {
		return nil, errors.New("AVIF decoder input write failed")
	}
	info, err := alloc(24)
	if err != nil {
		return nil, err
	}
	decode := func(configOnly, out uint64) error {
		result, err := mod.ExportedFunction("decode").Call(ctx, input, uint64(len(data)), configOnly, 0, info, info+4, info+8, info+12, info+16, out)
		if err != nil {
			return err
		}
		if len(result) != 1 || result[0] == 0 {
			return errors.New("AVIF pixel decoding failed")
		}
		return nil
	}
	if err := decode(1, 0); err != nil {
		return nil, err
	}
	meta, ok := mod.Memory().Read(uint32(info), 16)
	if !ok {
		return nil, errors.New("AVIF decoder metadata read failed")
	}
	width, height := binary.LittleEndian.Uint32(meta), binary.LittleEndian.Uint32(meta[4:])
	depth, count := binary.LittleEndian.Uint32(meta[8:]), binary.LittleEndian.Uint32(meta[12:])
	pixels, err := rasterPixels(uint64(width), uint64(height))
	if err != nil {
		return nil, err
	}
	if count != 1 || (depth != 8 && depth != 10 && depth != 12) {
		return nil, errors.New("unsupported AVIF frame count or bit depth")
	}
	bpp := uint64(4)
	if depth > 8 {
		bpp = 8
	}
	size := uint64(pixels) * bpp
	out, err := alloc(size)
	if err != nil {
		return nil, err
	}
	// The upstream ABI reports success even if no frame is produced.
	// This subset excludes alpha items, so opaque alpha proves that output
	// pixels were actually written, rather than exposing uninitialized memory.
	output, ok := mod.Memory().Read(uint32(out), uint32(size))
	if !ok {
		return nil, errors.New("AVIF decoder output allocation is unavailable")
	}
	clear(output)
	if err := decode(0, out); err != nil {
		return nil, err
	}
	decoded, ok := mod.Memory().Read(uint32(out), uint32(size))
	if !ok {
		return nil, errors.New("AVIF decoder pixel read failed")
	}
	for i := int(bpp) - 1; i < len(decoded); i += int(bpp) {
		if decoded[i] != 255 || (bpp == 8 && decoded[i-1] != 255) {
			return nil, errors.New("AVIF decoder did not produce an opaque still frame")
		}
	}
	pix := append([]byte(nil), decoded...)
	var img image.Image
	rect := image.Rect(0, 0, int(width), int(height))
	if depth > 8 {
		for i := 0; i < len(pix); i += 2 {
			pix[i], pix[i+1] = pix[i+1], pix[i]
		}
		img = &image.RGBA64{Pix: pix, Stride: int(width) * 8, Rect: rect}
	} else {
		img = &image.RGBA{Pix: pix, Stride: int(width) * 4, Rect: rect}
	}
	orientation, err := avifOrientation(data)
	if err != nil {
		return nil, fmt.Errorf("AVIF orientation: %w", err)
	}
	return applyOrientation(img, orientation), nil
}

// The validator requires every property to belong to the one primary image.
func avifOrientation(data []byte) (int, error) {
	budget := 256
	boxes, err := avifBoxes(data, 0, &budget)
	if err != nil {
		return 0, err
	}
	var props []avifBox
	for _, box := range boxes {
		if box.kind == "meta" {
			children, err := avifBoxes(box.data[4:], box.start+4, &budget)
			if err != nil {
				return 0, err
			}
			for _, child := range children {
				if child.kind == "iprp" {
					groups, err := avifBoxes(child.data, child.start, &budget)
					if err != nil {
						return 0, err
					}
					for _, group := range groups {
						if group.kind == "ipco" {
							props, err = avifBoxes(group.data, group.start, &budget)
							if err != nil {
								return 0, err
							}
						}
					}
				}
			}
		}
	}
	haveRot, haveMir := false, false
	angle, axis := 0, 0
	for _, p := range props {
		switch p.kind {
		case "irot":
			haveRot = true
			angle = int(p.data[0])
		case "imir":
			haveMir = true
			axis = int(p.data[0])
		}
	}
	return exifOrientationFromIrotImir(haveRot, angle, haveMir, axis), nil
}
