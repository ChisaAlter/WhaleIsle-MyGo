package chrome

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestOfficialHeroMaskUsesExactEmbeddedAssetAlpha(t *testing.T) {
	// Exact same repository/installed whale-isle-head.png, not a replacement
	// fish, luminance extraction or hand-traced approximation.
	sum := sha256.Sum256(whaleHeadPNG)
	if got := hex.EncodeToString(sum[:]); got != "deb725bfe8cda45421160c5be8c562173fd4343893051f39127c208a093c1ba7" {
		t.Fatalf("unexpected official asset: %s", got)
	}
	source, err := png.Decode(bytes.NewReader(whaleHeadPNG))
	if err != nil {
		t.Fatal(err)
	}
	mask := officialHeroAlpha()
	if mask == nil || mask.Bounds() != source.Bounds() || mask.Bounds().Dx() != 1254 || mask.Bounds().Dy() != 1254 {
		t.Fatal("official alpha source size changed")
	}
	transparent, opaque, partial := 0, 0, 0
	for _, ink := range []ui.Color{ui.RGB(0, 0, 0), ui.RGB(255, 255, 255), ui.RGBA(63, 117, 219, .5)} {
		painted := heroBrandImage(mask, ink)
		for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
			for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
				_, _, _, rawA := source.At(x, y).RGBA()
				if got := mask.AlphaAt(x, y).A; got != uint8(rawA>>8) {
					t.Fatalf("source alpha changed at%d,%d", x, y)
				}
				// Independent semantic oracle: an opaque currentColor leaves
				// exact source alpha; partial ink multiplies it once. RGB is
				// premultiplied only by the resulting alpha, never source RGB.
				a := uint8((uint32(rawA) * uint32(ink.A) * 257 / 65535) >> 8)
				got := painted.RGBAAt(x, y)
				if got.A != a {
					t.Fatalf("ink%+v alpha%d,%d=%d want%d", ink, x, y, got.A, a)
				}
				inkR, inkG, inkB, _ := (color.NRGBA{R: ink.R, G: ink.G, B: ink.B, A: ink.A}).RGBA()
				for _, pair := range [][2]uint32{{uint32(got.R), inkR}, {uint32(got.G), inkG}, {uint32(got.B), inkB}} {
					want := (pair[1] * rawA / 65535) >> 8
					if pair[0] != want {
						t.Fatalf("colored fringe at%d,%d got%+v ink%+v alpha%d", x, y, got, ink, a)
					}
				}
				if ink.A == 255 && ink.R == 0 {
					switch {
					case a == 0:
						transparent++
						if got != (color.RGBA{}) {
							t.Fatal("transparent border contains RGB")
						}
					case a == 255:
						opaque++
					default:
						partial++
					}
				}
			}
		}
	}
	if transparent == 0 || opaque == 0 || partial == 0 {
		t.Fatalf("mask does not exercise transparent/opaque/antialias edges: %d/%d/%d", transparent, opaque, partial)
	}
}

func TestOfficialHeroMaskCacheReusesActiveInks(t *testing.T) {
	black, white := ui.RGB(0, 0, 0), ui.RGB(255, 255, 255)
	a, b := HeroBrandMark(black), HeroBrandMark(white)
	if a == nil || b == nil || a == b || HeroBrandMark(black) != a || HeroBrandMark(white) != b {
		t.Fatal("hero tint was not reused per ink")
	}
	if w, h := a.Size(); w != 1254 || h != 1254 {
		t.Fatalf("tinted asset size=%dx%d", w, h)
	}
	// No opaque background: a hand-sized synthetic transparent alpha must
	// retain its zero border even with a semi-transparent colored ink.
	m := image.NewAlpha(image.Rect(0, 0, 3, 1))
	m.SetAlpha(1, 0, color.Alpha{A: 255})
	painted := heroBrandImage(m, ui.RGBA(15, 60, 190, .5))
	if painted.RGBAAt(0, 0) != (color.RGBA{}) || painted.RGBAAt(2, 0) != (color.RGBA{}) {
		t.Fatal("tint introduced a border background")
	}
}
