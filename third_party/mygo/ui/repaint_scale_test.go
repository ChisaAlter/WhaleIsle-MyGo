package ui

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

// This uses the production windowHost (including native zoom), not only a
// headless pixel host. Its OS surface supplies actual device pixels.
type repaintPixelSurface struct {
	testSurface
	width, height, scale float64
}

func (s *repaintPixelSurface) Size() (float64, float64, float64) {
	return s.width, s.height, s.scale
}

func TestNativeRepaintKeepsClientPixelsAndGlyphsAcrossDPIAndZoom(t *testing.T) {
	t.Setenv("MYGO_GPU", "0")
	for _, dpi := range []float64{1, 1.25, 1.5, 2} {
		for _, zoom := range []float64{0.8, 1, 1.25, 2} {
			t.Run(fmt.Sprintf("dpi%g_zoom%g", dpi, zoom), func(t *testing.T) {
				s := &repaintPixelSurface{width: 240, height: 180, scale: dpi}
				h := &windowHost{conn: &surface.Conn{Surface: s}, zoom: zoom}
				builds := 0
				var logical [2]float32
				h.rt = newRuntime(func(c *context) {
					builds++
					logical[0], logical[1] = c.Size()
					coreBox(c).Fill().Background(RGB(255, 255, 255)).Children(func() {
						coreText(c, "Ag 19").FontSize(14).TextColor(RGB(20, 20, 20))
						coreBox(c).Absolute().Right(0).Bottom(0).Size(12, 12).Background(RGB(255, 0, 0))
						coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
							p.AnimationFrame()
							p.Clip(Rect{r.W - 20, r.H - 20, 10, 10}, 0, func() {
								p.Fill(Rect{r.W - 30, r.H - 30, 40, 40}, RGB(0, 0, 255), 0)
							})
						})
					})
				}, h)
				defer h.detach()
				frame := func() { h.event(platform.SurfaceEvent{Kind: platform.SurfaceFrame}) }
				frame()
				if math.Abs(float64(logical[0])-s.width/(dpi*zoom)) > .01 || math.Abs(float64(logical[1])-s.height/(dpi*zoom)) > .01 {
					t.Fatalf("logical viewport %v does not match physical client at DPI×zoom", logical)
				}
				width, height, before := h.capture()
				if width != int(s.width) || height != int(s.height) {
					t.Fatalf("initial backing=%dx%d want client %.0fx%.0f", width, height, s.width, s.height)
				}
				count := builds
				for repeat := 0; repeat < 3; repeat++ {
					frame()
					w, ht, pixels := h.capture()
					if builds != count {
						t.Fatal("paint-only animation rebuilt layout")
					}
					if w != width || ht != height || !bytes.Equal(before, pixels) {
						t.Fatalf("paint-only frame changed backing, glyphs or clipped drawing: %dx%d -> %dx%d", width, height, w, ht)
					}
				}
				// A physical resize must rebuild layout before any further repaint.
				s.width, s.height = 270, 210
				frame()
				if builds == count {
					t.Fatal("resize reused stale paint-only layout")
				}
				w, ht, after := h.capture()
				if w != 270 || ht != 210 {
					t.Fatalf("resized backing=%dx%d want270x210", w, ht)
				}
				count = builds
				frame()
				rw, rh, repaint := h.capture()
				if builds != count || rw != w || rh != ht || !bytes.Equal(after, repaint) {
					t.Fatal("resize follow-up repaint changed the new physical surface")
				}
			})
		}
	}
}
