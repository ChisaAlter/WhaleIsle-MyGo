package imagepreview

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image/color"
	"os"
	"testing"
	"time"

	"golang.org/x/image/webp"
)

func nativeWebPFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/animated-native.webp")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Fixture: constant-color lossless literal Huffman trees, no transforms/cache.
// Four frames exercise opaque red, half-alpha blue OVER, transparent disposal,
// offset green, and half-alpha blue REPLACE. ANIM's magenta hint stays ignored.
func TestNativeWebPCompositesPixelsAndTiming(t *testing.T) {
	a, err := DecodeAnimation(nativeWebPFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 4 {
		t.Fatal("WebP flattened", len(a.Frames))
	}
	for _, tc := range []struct {
		frame, x int
		want     color.NRGBA
	}{
		{0, 0, color.NRGBA{R: 255, A: 255}}, {0, 3, color.NRGBA{R: 255, A: 255}},
		{1, 0, color.NRGBA{R: 127, B: 128, A: 255}}, {1, 3, color.NRGBA{R: 255, A: 255}},
		{2, 0, color.NRGBA{}}, {2, 2, color.NRGBA{G: 255, A: 255}},
		{3, 0, color.NRGBA{B: 255, A: 128}}, {3, 2, color.NRGBA{G: 255, A: 255}},
	} {
		if got := a.Frames[tc.frame].NRGBAAt(tc.x, 0); got != tc.want {
			t.Fatalf("frame %d x%d = %+v want %+v", tc.frame, tc.x, got, tc.want)
		}
	}
	for _, tc := range []struct {
		elapsed time.Duration
		index   int
		wait    time.Duration
	}{
		{0, 0, 40 * time.Millisecond}, {40 * time.Millisecond, 1, 60 * time.Millisecond},
		{100 * time.Millisecond, 2, 100 * time.Millisecond}, {200 * time.Millisecond, 3, 20 * time.Millisecond},
		{220 * time.Millisecond, 0, 40 * time.Millisecond}, {440 * time.Millisecond, 3, 0},
	} {
		i, w := a.FrameAt(tc.elapsed)
		if i != tc.index || w != tc.wait {
			t.Fatalf("at %s = %d/%s want %d/%s", tc.elapsed, i, w, tc.index, tc.wait)
		}
	}
	data := nativeWebPFixture(t)
	for _, tc := range []struct {
		loops   uint16
		elapsed time.Duration
		index   int
		wait    time.Duration
	}{
		{1, 220 * time.Millisecond, 3, 0}, {0, 440 * time.Millisecond, 0, 40 * time.Millisecond},
	} {
		binary.LittleEndian.PutUint16(data[42:44], tc.loops)
		a, err = DecodeAnimation(data)
		if err != nil {
			t.Fatal(err)
		}
		i, w := a.FrameAt(tc.elapsed)
		if i != tc.index || w != tc.wait {
			t.Fatalf("loops%d = %d/%s", tc.loops, i, w)
		}
	}
}

func TestNativeWebPRejectsMalformedAndBudgetedAnimations(t *testing.T) {
	mutate := func(fn func([]byte)) []byte { b := nativeWebPFixture(t); fn(b); return b }
	tooMany := append([]byte{}, nativeWebPFixture(t)[:44]...)
	for range MaxFrames + 1 {
		tooMany = append(tooMany, nativeWebPFixture(t)[44:98]...)
	}
	binary.LittleEndian.PutUint32(tooMany[4:8], uint32(len(tooMany)-8))
	for name, data := range map[string][]byte{
		"truncated":            nativeWebPFixture(t)[:259],
		"riff-length":          mutate(func(b []byte) { b[4]-- }),
		"chunk-overflow":       mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[48:52], 0xffffffff) }),
		"frame-outside":        mutate(func(b []byte) { b[52] = 2 }),
		"bitstream-dimensions": mutate(func(b []byte) { b[58] = 1 }),
		"broken-codec":         mutate(func(b []byte) { b[76] = 0 }),
		"duplicate-animation":  mutate(func(b []byte) { copy(b[44:48], "ANIM") }),
		"canvas-budget":        mutate(func(b []byte) { copy(b[24:30], []byte{255, 255, 255, 255, 255, 255}) }),
		"cumulative-canvases":  mutate(func(b []byte) { copy(b[24:30], []byte{159, 15, 0, 160, 15, 0}) }),
		"frame-budget":         tooMany,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeAnimation(data); err == nil {
				t.Fatal("accepted invalid/budget-exceeding WebP")
			}
		})
	}
}

func TestNativeWebPStillAndLossyAlphaFrameRemainDecodable(t *testing.T) {
	// Existing one-pixel VP8 fixture plus an independent constant-red VP8L.
	for _, encoded := range []string{
		"UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAwA0JaQAA3AA/vuUAAA=",
		"UklGRiIAAABXRUJQVlA4TBUAAAAvAwAAAChA/wvQ/wIAAAAAAAAAAAAA",
	} {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if IsAnimatedWebP(data) {
			t.Fatal("still WebP misclassified")
		}
		if _, err := webp.Decode(bytes.NewReader(data)); err != nil {
			t.Fatal(err)
		}
	}
	// Lossy ALPH must be wrapped with a frame-sized VP8X, not dropped.
	lossy, _ := base64.StdEncoding.DecodeString("UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAwA0JaQAA3AA/vuUAAA=")
	frame := append([]byte{'A', 'L', 'P', 'H', 2, 0, 0, 0, 0, 128}, lossy[12:]...)
	still, err := webpFrameImage(frame, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	img, err := webp.Decode(bytes.NewReader(still))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, alpha := img.At(0, 0).RGBA(); alpha != 128*257 {
		t.Fatalf("alpha = %d", alpha)
	}
}
