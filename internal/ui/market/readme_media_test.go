package market

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestMarketReadmeNativeAnimatedMediaPaintAndHide(t *testing.T) {
	webp, err := os.ReadFile("../../desktop/imagepreview/testdata/animated-native.webp")
	if err != nil {
		t.Fatal(err)
	}
	palette := color.Palette{color.NRGBA{R: 255, A: 255}, color.NRGBA{G: 255, A: 255}}
	first, last := image.NewPaletted(image.Rect(0, 0, 4, 1), palette), image.NewPaletted(image.Rect(0, 0, 4, 1), palette)
	for i := range last.Pix {
		last.Pix[i] = 1
	}
	var gifBytes bytes.Buffer
	if err := gif.EncodeAll(&gifBytes, &gif.GIF{Image: []*image.Paletted{first, last}, Delay: []int{5, 5}, LoopCount: -1}); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"moving.webp": webp, "moving.gif": gifBytes.Bytes()} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cache := readmeImageCache(t)
				url := "https://example.invalid/" + name
				if err := os.WriteFile(cache.FilePath(url), data, 0600); err != nil {
					t.Fatal(err)
				}
				visible := true
				tt := ui.NewTester(func(c *ui.Context) {
					if visible {
						marketReadme(c, Model{}, "![Motion]("+url+")", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
					}
				}, 160, 100)
				synctest.Wait()
				tt.Frame()
				sample := func(x float32) color.RGBA {
					t.Helper()
					r, ok := tt.Find("README image " + url)
					if !ok || r.W != 4 || r.H != 1 {
						t.Fatalf("native natural size lost: %+v", r)
					}
					return tt.Image().RGBAAt(int(r.X+x), int(r.Y))
				}
				if p := sample(2); p.R < 240 || p.G > 20 {
					t.Fatal("initial image not red", p)
				}
				time.Sleep(110 * time.Millisecond)
				tt.Frame()
				if p := sample(2); p.G < 240 || p.R > 20 || p.B > 20 {
					t.Fatal("README image stayed first frame", p)
				}
				visible = false
				tt.Frame()
				time.Sleep(time.Second)
				synctest.Wait()
				tt.Frame()
				if _, ok := tt.Find("README image " + url); ok {
					t.Fatal("hidden README retained image seat")
				}
			})
		})
	}
}

func TestMarketReadmeAsyncDecodeNotifiesOnceAndFinitePlaybackStops(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/animated-native.webp")
	if err != nil {
		t.Fatal(err)
	}
	synctest.Test(t, func(t *testing.T) {
		path := t.TempDir() + "/motion.webp"
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		var notified atomic.Int32
		d := readmeDecodedImage{}
		d.load(path, func() { notified.Add(1) })
		synctest.Wait()
		d.poll(path)
		if d.animation == nil || len(d.frames) != 4 || len(d.animation.Frames) != 0 || d.pending != nil {
			t.Fatal("animation/readiness or staging cleanup missing")
		}
		start := time.Now()
		d.advance(start)
		if wait := d.advance(start.Add(time.Second)); wait != 0 || d.bitmap != d.frames[3] {
			t.Fatal("finite README animation did not stop")
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if notified.Load() != 1 {
			t.Fatal("decoder spawned recurring invalidation", notified.Load())
		}
	})
}

func TestMarketReadmeBroaderStaticSVGActuallyRenders(t *testing.T) {
	var raster bytes.Buffer
	pixel := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	pixel.SetNRGBA(0, 0, color.NRGBA{B: 255, A: 255})
	if err := png.Encode(&raster, pixel); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"text":           `<text x="4" y="23" font-size="20" fill="black">HI</text>`,
		"filter":         `<defs><filter id="shift" x="-100%" y="-100%" width="400%" height="400%"><feOffset dx="8" dy="0"/></filter></defs><rect x="8" y="8" width="8" height="8" fill="blue" filter="url(#shift)"/>`,
		"embedded-image": fmt.Sprintf(`<image x="8" y="8" width="8" height="8" href="data:image/png;base64,%s"/>`, base64.StdEncoding.EncodeToString(raster.Bytes())),
		"pattern":        `<defs><pattern id="tile" patternUnits="userSpaceOnUse" width="8" height="8"><rect width="4" height="8" fill="blue"/></pattern></defs><rect x="8" y="8" width="24" height="8" fill="url(#tile)"/>`,
		"marker":         `<defs><marker id="arrow" viewBox="0 0 8 8" refX="4" refY="4" markerWidth="8" markerHeight="8" markerUnits="userSpaceOnUse"><path d="M0 0L8 4L0 8Z" fill="blue"/></marker></defs><path d="M4 12L24 12" stroke="black" marker-end="url(#arrow)"/>`,
		"complex-css":    `<style>svg > g > rect {fill:blue}</style><g><rect x="8" y="8" width="8" height="8"/></g>`,
	} {
		t.Run(name, func(t *testing.T) {
			cache := readmeImageCache(t)
			source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="32"><rect width="64" height="32" fill="white"/>` + content + `</svg>`)
			if !supportedReadmeSVG(source) {
				t.Fatal("shared static capability rejected")
			}
			url := "https://example.invalid/actual-" + name + ".svg"
			if err := os.WriteFile(cache.FilePath(url), source, 0600); err != nil {
				t.Fatal(err)
			}
			tt := ui.NewTester(func(c *ui.Context) {
				marketReadme(c, Model{}, "![Authored alt]("+url+")", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
			}, 200, 100)
			deadline := time.Now().Add(12 * time.Second)
			var r ui.Rect
			for {
				tt.Frame()
				r, _ = tt.Find("README image " + url)
				if r.W > 0 || time.Now().After(deadline) {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if r.W != 64 || r.H != 32 || tt.HasText("Authored alt") {
				t.Fatalf("decoded static SVG missing or wrong size: %+v", r)
			}
			colored, dark := 0, 0
			for y := int(r.Y) + 2; y < int(r.Y+r.H)-2; y++ {
				for x := int(r.X) + 2; x < int(r.X+r.W)-2; x++ {
					p := tt.Image().RGBAAt(x, y)
					if p.B > 200 && p.R < 40 && p.G < 40 {
						colored++
					}
					if p.R < 80 && p.G < 80 && p.B < 80 {
						dark++
					}
				}
			}
			if name == "text" {
				if dark < 10 {
					t.Fatal("SVG diagram text omitted")
				}
			} else if colored < 4 {
				t.Fatal("static SVG content omitted", colored)
			}
			if name == "filter" {
				if p := tt.Image().RGBAAt(int(r.X+18), int(r.Y+10)); p.B < 200 || p.R > 40 {
					t.Fatal("filter offset not applied", p)
				}
			}
		})
	}
}
