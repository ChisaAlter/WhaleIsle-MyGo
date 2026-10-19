package imagepreview

import (
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"math"
	"time"

	"github.com/tetratelabs/wazero"
)

// DecodeAVIFAnimationContext uses the already-bundled libavif decode_all ABI.
// It accepts bounded AV1 tracks, not still-item collections or external data.
// The static DecodeAVIFContext policy and Files routing are unchanged.
func DecodeAVIFAnimationContext(ctx context.Context, data []byte) (*Animation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sequence, err := validateAVIFSequence(data)
	if err != nil {
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
			return 0, errors.New("AVIF sequence allocation failed")
		}
		return result[0], nil
	}
	input, err := alloc(uint64(len(data)))
	if err != nil {
		return nil, err
	}
	if !mod.Memory().Write(uint32(input), data) {
		return nil, errors.New("AVIF sequence input write failed")
	}
	info, err := alloc(16)
	if err != nil {
		return nil, err
	}
	decode := func(configOnly, all, delay, out uint64) error {
		result, err := mod.ExportedFunction("decode").Call(ctx, input, uint64(len(data)), configOnly, all, info, info+4, info+8, info+12, delay, out)
		if err != nil {
			return err
		}
		if len(result) != 1 || result[0] == 0 {
			return errors.New("AVIF sequence decoding failed")
		}
		return nil
	}
	if err := decode(1, 0, 0, 0); err != nil {
		return nil, err
	}
	meta, ok := mod.Memory().Read(uint32(info), 16)
	if !ok {
		return nil, errors.New("AVIF sequence metadata read failed")
	}
	width, height := binary.LittleEndian.Uint32(meta), binary.LittleEndian.Uint32(meta[4:])
	depth, count := binary.LittleEndian.Uint32(meta[8:]), binary.LittleEndian.Uint32(meta[12:])
	if int(width) != sequence.width || int(height) != sequence.height || int(count) != sequence.count || (depth != 8 && depth != 10 && depth != 12) {
		return nil, errors.New("AVIF codec metadata differs from validated sequence")
	}
	pixels, err := rasterPixels(uint64(width), uint64(height))
	if err != nil {
		return nil, err
	}
	if count == 0 || count > MaxFrames || int64(count) > MaxPixels/pixels {
		return nil, errors.New("AVIF sequence exceeds retained canvas budget")
	}
	bpp := uint64(4)
	if depth > 8 {
		bpp = 8
	}
	frameSize := uint64(pixels) * bpp
	outSize := frameSize * uint64(count)
	out, err := alloc(outSize)
	if err != nil {
		return nil, err
	}
	delays, err := alloc(8 * uint64(count))
	if err != nil {
		return nil, err
	}
	output, ok := mod.Memory().Read(uint32(out), uint32(outSize))
	if !ok {
		return nil, errors.New("AVIF sequence output allocation unavailable")
	}
	clear(output)
	// The bundled ABI returns success when NextImage terminates, including
	// decode errors. A duration is written only AFTER that frame's complete
	// RGBA copy. NaN sentinels prove every advertised frame was actually written,
	// including legitimate all-transparent alpha frames.
	sentinels := make([]byte, 8*int(count))
	for i := 0; i < int(count); i++ {
		binary.LittleEndian.PutUint64(sentinels[i*8:], math.Float64bits(math.NaN()))
	}
	if !mod.Memory().Write(uint32(delays), sentinels) {
		return nil, errors.New("AVIF sequence timing initialization failed")
	}
	if err := decode(0, 1, delays, out); err != nil {
		return nil, err
	}
	meta, ok = mod.Memory().Read(uint32(info), 16)
	if !ok || binary.LittleEndian.Uint32(meta) != width || binary.LittleEndian.Uint32(meta[4:]) != height || binary.LittleEndian.Uint32(meta[8:]) != depth || binary.LittleEndian.Uint32(meta[12:]) != count {
		return nil, errors.New("AVIF sequence changed during pixel decoding")
	}
	a := &Animation{loops: sequence.loops}
	if count == 1 {
		a.loops = -1
	}
	for i := 0; i < int(count); i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		wireDelay, ok := mod.Memory().ReadUint64Le(uint32(delays) + uint32(8*i))
		if !ok {
			return nil, errors.New("AVIF sequence duration read failed")
		}
		seconds := math.Float64frombits(wireDelay)
		if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 || seconds >= float64(math.MaxInt64)/float64(time.Second) {
			return nil, errors.New("AVIF sequence did not produce every complete frame")
		}
		delay := time.Duration(seconds * float64(time.Second))
		if delay <= 10*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		if difference := delay - sequence.delays[i]; difference < -time.Nanosecond || difference > time.Nanosecond {
			return nil, errors.New("AVIF decoded duration differs from sample timing")
		}
		// Use exact validated rational timing after the ABI completion check.
		delay = sequence.delays[i]
		decoded, ok := mod.Memory().Read(uint32(out+uint64(i)*frameSize), uint32(frameSize))
		if !ok {
			return nil, errors.New("AVIF sequence pixels unavailable")
		}
		if !sequence.alpha {
			for p := int(bpp) - 1; p < len(decoded); p += int(bpp) {
				if decoded[p] != 255 || (bpp == 8 && decoded[p-1] != 255) {
					return nil, errors.New("AVIF sequence failed to produce an opaque color frame")
				}
			}
		}
		var raw image.Image
		rect := image.Rect(0, 0, int(width), int(height))
		if depth > 8 {
			pix := make([]byte, len(decoded))
			for p := 0; p < len(pix); p += 2 {
				pix[p], pix[p+1] = decoded[p+1], decoded[p]
			}
			raw = &image.RGBA64{Pix: pix, Stride: int(width) * 8, Rect: rect}
		} else {
			raw = &image.RGBA{Pix: decoded, Stride: int(width) * 4, Rect: rect}
		}
		raw = applyOrientation(raw, sequence.orientation)
		frame := image.NewNRGBA(raw.Bounds())
		draw.Draw(frame, frame.Rect, raw, raw.Bounds().Min, draw.Src)
		a.Frames = append(a.Frames, frame)
		a.duration += delay
		a.ends = append(a.ends, a.duration)
	}
	return a, nil
}
