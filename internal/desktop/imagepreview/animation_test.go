package imagepreview

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
	"time"
)

// Guards against first-frame flattening, incorrect disposal and endless finite GIFs.
func TestNativeGIFDisposalAndTiming(t *testing.T) {
	palette := color.Palette{color.Transparent, color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}}
	frame := func(rect image.Rectangle, index uint8) *image.Paletted {
		p := image.NewPaletted(rect, palette)
		for i := range p.Pix {
			p.Pix[i] = index
		}
		return p
	}
	g := &gif.GIF{Image: []*image.Paletted{frame(image.Rect(0, 0, 2, 1), 1), frame(image.Rect(0, 0, 1, 1), 2), frame(image.Rect(1, 0, 2, 1), 2)}, Delay: []int{10, 10, 10}, Disposal: []byte{gif.DisposalNone, gif.DisposalPrevious, gif.DisposalBackground}, LoopCount: -1, Config: image.Config{ColorModel: palette, Width: 2, Height: 1}}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	a, err := DecodeAnimation(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Frames) != 3 {
		t.Fatal("animation flattened")
	}
	if got := color.NRGBAModel.Convert(a.Frames[2].At(0, 0)).(color.NRGBA); got.R != 255 || got.G != 0 {
		t.Fatal("previous disposal lost", got)
	}
	if index, wait := a.FrameAt(100 * time.Millisecond); index != 1 || wait != 100*time.Millisecond {
		t.Fatal(index, wait)
	}
	if index, wait := a.FrameAt(time.Second); index != 2 || wait != 0 {
		t.Fatal("finite loop kept running", index, wait)
	}
	g.LoopCount = 0
	g.Disposal[1] = gif.DisposalBackground
	b.Reset()
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	a, err = DecodeAnimation(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, alpha := a.Frames[2].At(0, 0).RGBA(); alpha != 0 {
		t.Fatal("background disposal did not clear rectangle")
	}
	if index, wait := a.FrameAt(350 * time.Millisecond); index != 0 || wait != 50*time.Millisecond {
		t.Fatal("infinite loop timing", index, wait)
	}
}

func TestNativeGIFRejectsBeforeDecode(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("GIF89a"), make([]byte, MaxBytes+1)} {
		if _, err := DecodeAnimation(data); err == nil {
			t.Fatal("accepted invalid GIF")
		}
	}
}

func TestNativeGIFOpaqueBackgroundDisposal(t *testing.T) {
	background := color.NRGBA{B: 255, A: 255}
	palette := color.Palette{color.NRGBA{R: 255, A: 255}, background, color.NRGBA{G: 255, A: 255}}
	first := image.NewPaletted(image.Rect(0, 0, 1, 1), palette)
	second := image.NewPaletted(image.Rect(1, 0, 2, 1), palette)
	second.Pix[0] = 2
	g := &gif.GIF{
		Image: []*image.Paletted{first, second}, Delay: []int{5, 7},
		Disposal: []byte{gif.DisposalBackground, gif.DisposalNone}, LoopCount: -1,
		Config: image.Config{ColorModel: palette, Width: 2, Height: 1}, BackgroundIndex: 1,
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	a, err := DecodeAnimation(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ frame, x int }{{0, 1}, {1, 0}} {
		if got := color.NRGBAModel.Convert(a.Frames[tc.frame].At(tc.x, 0)).(color.NRGBA); got != background {
			t.Fatalf("frame %d pixel %d: background = %#v, want %#v", tc.frame, tc.x, got, background)
		}
	}
	if index, wait := a.FrameAt(60 * time.Millisecond); index != 1 || wait != 60*time.Millisecond {
		t.Fatal("timing changed", index, wait)
	}
	if index, wait := a.FrameAt(time.Second); index != 1 || wait != 0 {
		t.Fatal("finite loop changed", index, wait)
	}
}
