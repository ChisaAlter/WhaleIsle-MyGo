package imagepreview

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
)

func avifSequenceFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNativeAVIFSequenceFramesTimingAndOwnership(t *testing.T) {
	data := avifSequenceFixture(t)
	if !IsAnimatedAVIF(data) {
		t.Fatal("sequence brand missing")
	}
	a, err := DecodeAVIFAnimationContext(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 17 || a.duration != 1360*time.Millisecond {
		t.Fatalf("sequence = %d frames, %s", len(a.Frames), a.duration)
	}
	different := false
	for _, frame := range a.Frames {
		if frame.Bounds().Dx() != 500 || frame.Bounds().Dy() != 360 {
			t.Fatal("sequence dimensions", frame.Bounds())
		}
		if !bytes.Equal(frame.Pix, a.Frames[0].Pix) {
			different = true
		}
		for p := 3; p < len(frame.Pix); p += 4 {
			if frame.Pix[p] != 255 {
				t.Fatal("opaque fixture output incomplete")
			}
		}
	}
	if !different {
		t.Fatal("animated AVIF flattened to repeated first frame")
	}
	if i, w := a.FrameAt(90 * time.Millisecond); i != 1 || w != 70*time.Millisecond {
		t.Fatal("frame timing", i, w)
	}
	if i, w := a.FrameAt(1360 * time.Millisecond); i != 0 || w != 80*time.Millisecond {
		t.Fatal("absent edit list should loop infinitely", i, w)
	}
	other := append([]byte(nil), a.Frames[1].Pix...)
	a.Frames[0].Pix[0] ^= 255
	if !bytes.Equal(a.Frames[1].Pix, other) {
		t.Fatal("frames alias the same staging canvas")
	}
	if _, err := DecodeAVIF(data); err == nil {
		t.Fatal("still decoder unexpectedly accepted sequence")
	}
	if _, err := DecodeAnimation(data); err == nil {
		t.Fatal("GIF/WebP entry implicitly gained AVIF sequence support")
	}
}

func TestNativeAVIFSequenceBoundsAndCancellation(t *testing.T) {
	mutate := func(kind string, offset int, value uint32) []byte {
		data := avifSequenceFixture(t)
		p := bytes.Index(data, []byte(kind))
		if p < 0 {
			t.Fatal("fixture box missing", kind)
		}
		binary.BigEndian.PutUint32(data[p+4+offset:], value)
		return data
	}
	for name, data := range map[string][]byte{
		"frames":              mutate("stsz", 8, MaxFrames+1),
		"timing-count":        mutate("stts", 8, 18),
		"timescale":           mutate("mdhd", 20, 0),
		"sample-extents":      mutate("stco", 8, math.MaxUint32),
		"cumulative-canvases": mutate("tkhd", 88, 6000<<16),
		"external-data":       mutate("url ", 0, 0),
		"truncated":           avifSequenceFixture(t)[:32634],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateAVIFSequence(data); err == nil {
				t.Fatal("invalid sequence admitted before codec")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DecodeAVIFAnimationContext(ctx, avifSequenceFixture(t)); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled sequence decoded", err)
	}
	// The last five bytes are a real later AV1 sample in the pinned fixture.
	// The C ABI may still return success; a missing duration must reject output.
	corrupt := avifSequenceFixture(t)
	clear(corrupt[len(corrupt)-5:])
	if _, err := DecodeAVIFAnimationContext(context.Background(), corrupt); err == nil {
		t.Fatal("corrupt later frame exposed partial sequence")
	}
}

func avifEditListFixture(version byte, flags uint32, segment uint64) avifBox {
	n := 20
	if version == 1 {
		n = 28
	}
	payload := make([]byte, n)
	payload[0] = version
	payload[1] = byte(flags >> 16)
	payload[2] = byte(flags >> 8)
	payload[3] = byte(flags)
	binary.BigEndian.PutUint32(payload[4:8], 1)
	if version == 1 {
		binary.BigEndian.PutUint64(payload[8:16], segment)
		binary.BigEndian.PutUint32(payload[24:28], 0x10000)
	} else {
		binary.BigEndian.PutUint32(payload[8:12], uint32(segment))
		binary.BigEndian.PutUint32(payload[16:20], 0x10000)
	}
	data := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(data, uint32(len(data)))
	copy(data[4:8], "elst")
	copy(data[8:], payload)
	return avifBox{kind: "edts", data: data}
}

func TestNativeAVIFRepetitionMatchesChromiumEditListSemantics(t *testing.T) {
	for _, tc := range []struct {
		name     string
		edits    avifBox
		duration uint64
		want     int
	}{
		{"unknown", avifBox{}, 300, 0},
		{"once", avifEditListFixture(0, 0, 100), 300, -1},
		{"finite32", avifEditListFixture(0, 1, 100), 300, 2},
		{"finite64-round-up", avifEditListFixture(1, 1, 100), 201, 2},
		{"one-cycle", avifEditListFixture(1, 1, 100), 100, -1},
		{"infinite", avifEditListFixture(1, 1, 100), math.MaxUint64, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			budget := 256
			got, err := avifSequenceLoops(tc.edits, tc.duration, &budget)
			if err != nil || got != tc.want {
				t.Fatalf("repeat %d want%d, %v", got, tc.want, err)
			}
		})
	}
	for _, edits := range []avifBox{avifEditListFixture(0, 1, 0), {kind: "edts", data: []byte{0}}} {
		budget := 256
		if _, err := avifSequenceLoops(edits, 100, &budget); err == nil {
			t.Fatal("invalid edit list accepted")
		}
	}
}

// Use the already-bundled encoder only while this test executes; no tool,
// dependency, network, persistent fixture generation or codec rebuild is needed.
func avifTransparentSequence(t *testing.T) []byte {
	t.Helper()
	codec, err := avifRuntime()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mod, err := codec.runtime.InstantiateModule(ctx, codec.module, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(context.Background())
	alloc := func(n uint64) uint64 {
		result, err := mod.ExportedFunction("malloc").Call(ctx, n)
		if err != nil || len(result) != 1 || result[0] == 0 {
			t.Fatal("fixture allocation", err)
		}
		return result[0]
	}
	// Premultiplied RGBA: transparent first frame, half-alpha red second.
	pixels := make([]byte, 4*2*4*2)
	for p := 4 * 2 * 4; p < len(pixels); p += 4 {
		pixels[p], pixels[p+3] = 80, 128
	}
	in := alloc(uint64(len(pixels)))
	if !mod.Memory().Write(uint32(in), pixels) {
		t.Fatal("fixture write")
	}
	delays := alloc(8)
	mod.Memory().WriteUint32Le(uint32(delays), 50)
	mod.Memory().WriteUint32Le(uint32(delays)+4, 70)
	size := alloc(8)
	result, err := mod.ExportedFunction("encode_animation").Call(ctx, in, 4, 2, 2, delays, 1, 100, 100, 10, 1, 1, size)
	if err != nil || len(result) != 1 || result[0] == 0 {
		t.Fatal("fixture encode", err)
	}
	n, ok := mod.Memory().ReadUint64Le(uint32(size))
	if !ok || n == 0 || n > uint64(MaxBytes) {
		t.Fatal("fixture size")
	}
	data, ok := mod.Memory().Read(uint32(result[0]), uint32(n))
	if !ok {
		t.Fatal("fixture bytes")
	}
	return append([]byte(nil), data...)
}

func TestNativeAVIFSequenceTransparentCompletionAndFiniteLoops(t *testing.T) {
	data := avifTransparentSequence(t)
	a, err := DecodeAVIFAnimationContext(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 2 {
		t.Fatal("alpha sequence flattened")
	}
	if alpha := a.Frames[0].NRGBAAt(1, 1).A; alpha != 0 {
		t.Fatal("valid transparent frame rejected or changed", alpha)
	}
	last := a.Frames[1].NRGBAAt(1, 1)
	if last.A < 127 || last.A > 129 || last.R < 150 || last.G > 10 || last.B > 10 {
		t.Fatal("premultiplied alpha conversion", last)
	}
	if i, w := a.FrameAt(125 * time.Millisecond); i != 0 || w != 45*time.Millisecond {
		t.Fatal("finite second cycle", i, w)
	}
	if i, w := a.FrameAt(250 * time.Millisecond); i != 1 || w != 0 {
		t.Fatal("finite AVIF kept running", i, w)
	}
}

func TestAVIFSequenceCodingConstraintsAreStrictlyBounded(t *testing.T) {
	boxBytes := func(kind string, payload []byte) []byte {
		box := make([]byte, 8+len(payload))
		binary.BigEndian.PutUint32(box, uint32(len(box)))
		copy(box[4:8], kind)
		copy(box[8:], payload)
		return box
	}
	description := func(constraints []byte, duplicate bool) avifBox {
		entry := make([]byte, 78)
		binary.BigEndian.PutUint16(entry[6:8], 1)
		binary.BigEndian.PutUint16(entry[24:26], 4)
		binary.BigEndian.PutUint16(entry[26:28], 2)
		entry = append(entry, boxBytes("av1C", []byte{0x81, 0x20, 0, 0})...)
		entry = append(entry, boxBytes("ccst", constraints)...)
		if duplicate {
			entry = append(entry, boxBytes("ccst", constraints)...)
		}
		payload := make([]byte, 8)
		binary.BigEndian.PutUint32(payload[4:8], 1)
		return avifBox{kind: "stsd", data: append(payload, boxBytes("av01", entry)...)}
	}
	for _, tc := range []struct {
		name             string
		payload          []byte
		duplicate, valid bool
	}{
		{"libavif writer", []byte{0, 0, 0, 0, 0x7c, 0, 0, 0}, false, true},
		{"all six constraint bits", []byte{0, 0, 0, 0, 0xfc, 0, 0, 0}, false, true},
		{"short", make([]byte, 7), false, false},
		{"trailing", make([]byte, 9), false, false},
		{"version", []byte{1, 0, 0, 0, 0x7c, 0, 0, 0}, false, false},
		{"flags", []byte{0, 0, 0, 1, 0x7c, 0, 0, 0}, false, false},
		{"reserved", []byte{0, 0, 0, 0, 0x7c, 0, 0, 1}, false, false},
		{"duplicate", []byte{0, 0, 0, 0, 0x7c, 0, 0, 0}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, alpha := range []bool{false, true} {
				budget := 256
				_, err := avifSequenceDescription(description(tc.payload, tc.duplicate), 4, 2, alpha, &budget)
				if (err == nil) != tc.valid {
					t.Fatalf("alpha=%v accepted=%v want=%v: %v", alpha, err == nil, tc.valid, err)
				}
			}
		})
	}
}
