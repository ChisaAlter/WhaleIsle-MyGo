package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
)

// countBlue counts the pixels of img that are mostly blue.
func countBlue(img *image.RGBA) int {
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		if p := img.Pix[i:]; p[2] > 180 && p[0] < 120 && p[1] < 120 {
			n++
		}
	}
	return n
}

func TestAnimatedPathShowsInEveryFrame(t *testing.T) {
	n := 0
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreText(c, "Wave")
			coreBox(c).Grow(1).FillWidth().Draw(func(p *Painter, r Rect) {
				var path Path
				for i := 0; i <= 100; i++ {
					x := r.X + 10 + float32(i)*(r.W-20)/100
					y := r.Y + r.H/2 + float32(math.Sin(float64(i)/8+float64(n)/5))*(r.H/2-20)
					if i == 0 {
						path.MoveTo(x, y)
					} else {
						path.LineTo(x, y)
					}
				}
				p.StrokePath(&path, 3, RGB(20, 60, 230))
			})
		})
	}
	tt := coreNewTester(view, 600, 400)
	tt.SetScale(2)
	// Each frame draws another wave: the atlas must not fill up with the
	// old ones and leave a frame without its wave.
	for ; n < 80; n++ {
		tt.Frame()
		if ink := countBlue(tt.Image()); ink < 2000 {
			t.Fatalf("frame %d shows %d pixels of the wave", n, ink)
		}
		if !tt.HasText("Wave") {
			t.Fatalf("frame %d lost its text", n)
		}
	}
}

func TestFramesStayWholeWhenTheAtlasFills(t *testing.T) {
	view := func(c *context) {
		coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
			// 48 different discs, together larger than the atlas starts.
			for i := 0; i < 48; i++ {
				var path Path
				cx := r.X + 60 + float32(i%8)*110
				cy := r.Y + 60 + float32(i/8)*110
				path.Circle(cx, cy, 40+float32(i)/4)
				p.FillPath(&path, RGB(20, 60, 230))
			}
		})
	}
	tt := coreNewTester(view, 900, 700)
	tt.SetScale(2)
	want := bytes.Clone(tt.Image().Pix)
	if countBlue(tt.Image()) < 48*3000 {
		t.Fatalf("the first frame shows %d pixels of discs", countBlue(tt.Image()))
	}
	for i := 0; i < 4; i++ {
		tt.Frame()
		if !bytes.Equal(tt.Image().Pix, want) {
			t.Fatalf("frame %d differs from the first: %d pixels of discs", i+2, countBlue(tt.Image()))
		}
	}
}

func TestFramesRedrawOnlyWhatChanged(t *testing.T) {
	d := &demo{choice: "a", size: "Medium", volume: 40}
	tt := coreNewTester(d.view, 640, 600)
	tt.SetScale(2)
	full := raster.NewImage(1, 1)
	check := func(what string) {
		t.Helper()
		s := &tt.rt.scene
		full.Resize(s.Width, s.Height)
		raster.Render(full, s)
		if !bytes.Equal(tt.h.img.Image.Pix, full.Pix) {
			t.Fatalf("after %s, the frame differs from a whole drawing", what)
		}
	}
	check("the first frame")
	r, _ := tt.Find("Increment")
	tt.Move(r.X+5, r.Y+5)
	check("hovering a button")
	tt.Click("Increment")
	check("a click")
	tt.Click("I agree")
	check("a check box")
	in, _ := tt.Find("I agree")
	tt.ClickAt(in.X+20, in.Y+in.H/2+40)
	tt.Type("Ada")
	check("typing")
	tt.Key(0, KeyTab)
	check("moving the focus")
	row, _ := tt.Find("Row 3")
	tt.Scroll(row.X+10, row.Y+10, 0, 100)
	check("scrolling")
	tt.Click("Row 7")
	check("selecting a row")
	tt.SetDark(true)
	check("going dark")
}

// Glyphs land on an opaque background, which subpixel antialiasing needs,
// where the root's covers the window, or, under a transparent root over a
// window's material, where an element's opaque background holds all of
// what shows of them: a pane beside a sidebar, as the pane's list scrolls.
func TestOpaqueUnderElements(t *testing.T) {
	transparent := false
	opaque := map[string]bool{}
	probe := func(c *context, name string) *node {
		return coreBox(c).Height(20).FillWidth().Draw(func(p *Painter, r Rect) { opaque[name] = p.opaque })
	}
	var list ListState
	tt := coreNewTester(func(c *context) {
		if transparent {
			c.Root().Background(Transparent)
		}
		coreRow(c).Fill().AlignItems(Stretch).Children(func() {
			coreColumn(c).Width(100).Children(func() { probe(c, "sidebar") })
			coreColumn(c).Grow(1).Background(RGB(255, 255, 255)).Children(func() {
				coreList(c, &list, 40, func(i int) {
					if i == 3 {
						probe(c, "row")
					} else {
						coreBox(c).Height(20)
					}
				}).Height(100)
				probe(c, "pane")
				// Over the sidebar, out of the pane.
				probe(c, "overflow").Absolute().Left(-50).Top(150).Width(100)
				coreColumn(c).Background(RGB(255, 255, 255).Alpha(0.5)).Children(func() { probe(c, "translucent") })
			})
		})
	}, 400, 300)
	frame := func(when string, want map[string]bool) {
		t.Helper()
		clear(opaque)
		tt.Frame()
		for name, w := range want {
			if got, ok := opaque[name]; !ok || got != w {
				t.Errorf("%s: %s opaque %v (painted %v), want %v", when, name, got, ok, w)
			}
		}
	}
	frame("an opaque root", map[string]bool{"sidebar": true, "pane": true, "row": true, "overflow": true, "translucent": true})
	transparent = true
	frame("a transparent root", map[string]bool{"sidebar": false, "pane": true, "row": true, "overflow": false, "translucent": true})
	// The row half out of the list's top, and so of the pane's: what shows
	// of it is in the pane.
	list.ScrollTo(3, Start)
	tt.Frame()
	tt.Scroll(250, 50, 0, 10)
	frame("a row the list clips", map[string]bool{"row": true})
}

// TestOneDIPBordersAndDividersPhysicalPixels verifies Blink-aligned floor
// rounding for hairline widths (<= 1 DIP): 1 DIP borders and dividers render
// at 1 physical pixel across 1.0x/1.25x/1.5x/1.75x and step to 2px at 2.0x,
// while widths > 1 DIP continue rounding to the nearest pixel.
func TestOneDIPBordersAndDividersPhysicalPixels(t *testing.T) {
	red := RGB(220, 20, 20)
	blue := RGB(20, 40, 220)
	green := RGB(20, 180, 40)
	cases := []struct {
		scale   float32
		want1px float32
		want2px float32
	}{
		{1.0, 1, 2},
		{1.25, 1, 3},
		{1.5, 1, 3},
		{1.75, 1, 4},
		{2.0, 2, 4},
	}
	for _, tc := range cases {
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Background(RGB(255, 255, 255)).Gap(8).Padding(8).Children(func() {
				coreBox(c).Size(60, 24).Border(1, red).Label("b1")
				coreBox(c).Size(60, 24).Border(2, blue).Label("b2")
				coreColumn(c).Width(60).Gap(10).Dividers(1, green).Children(func() {
					coreBox(c).Height(10)
					coreBox(c).Height(10)
				})
				coreRow(c).Height(20).Gap(10).Dividers(1, green).Children(func() {
					coreBox(c).Width(20)
					coreBox(c).Width(20)
				})
			})
		}, 120, 160)
		tt.SetScale(tc.scale)
		tt.Frame()

		var gotB1, gotB2 [4]float32
		var hDivH, vDivW float32
		for _, op := range tt.rt.scene.Ops {
			if op.Kind != scene.OpFill {
				continue
			}
			switch {
			case op.BorderColor == red.scene() && scene.HasBorder(op.Border):
				gotB1 = op.Border
			case op.BorderColor == blue.scene() && scene.HasBorder(op.Border):
				gotB2 = op.Border
			case op.Color == green.scene():
				if op.Rect.W > op.Rect.H {
					hDivH = op.Rect.H
				} else {
					vDivW = op.Rect.W
				}
			}
		}
		want1 := [4]float32{tc.want1px, tc.want1px, tc.want1px, tc.want1px}
		want2 := [4]float32{tc.want2px, tc.want2px, tc.want2px, tc.want2px}
		if gotB1 != want1 {
			t.Errorf("scale %.2f: 1 DIP border = %v, want %v", tc.scale, gotB1, want1)
		}
		if gotB2 != want2 {
			t.Errorf("scale %.2f: 2 DIP border = %v, want %v", tc.scale, gotB2, want2)
		}
		if hDivH != tc.want1px || vDivW != tc.want1px {
			t.Errorf("scale %.2f: 1 DIP dividers h=%v w=%v, want %v", tc.scale, hDivH, vDivW, tc.want1px)
		}

		// Verify rendered physical pixel thickness of the 1 DIP top border.
		img := tt.Image()
		b1, _ := tt.Find("b1")
		cx := int(round((b1.X + b1.W/2) * tc.scale))
		y0 := int(round(b1.Y * tc.scale))
		redRows := 0
		for y := y0; y < y0+6; y++ {
			p := img.RGBAAt(cx, y)
			if p.R > 180 && p.G < 80 {
				redRows++
			}
		}
		if redRows != int(tc.want1px) {
			t.Errorf("scale %.2f: rendered 1 DIP top border is %d px thick, want %d px", tc.scale, redRows, int(tc.want1px))
		}
	}
}

// TestBorderOneClipNoGapAtFractionalDPI pins that a Border(1, blue).Clip()
// container with a flush red child over a green backdrop renders [G B... R...]
// with zero 1px green inner-gap bleed across 100%/125%/150%/175%/200%/250% DPI
// and across all subpixel x/y parities on all four sides.
func TestBorderOneClipNoGapAtFractionalDPI(t *testing.T) {
	green := RGB(20, 200, 40)
	blue := RGB(20, 60, 230)
	red := RGB(230, 30, 30)
	classify := func(c color.RGBA) byte {
		switch {
		case c.G > 140 && c.R < 80 && c.B < 100:
			return 'G'
		case c.B > 150 && c.R < 80 && c.G < 120:
			return 'B'
		case c.R > 160 && c.G < 80 && c.B < 80:
			return 'R'
		default:
			return '?'
		}
	}

	scales := []struct {
		scale  float32
		wantPx int
	}{
		{1.0, 1},
		{1.25, 1},
		{1.5, 1},
		{1.75, 1},
		{2.0, 2},
		{2.5, 2},
	}

	for _, sc := range scales {
		for _, off := range []float32{10, 11, 12, 13} {
			for _, nested := range []bool{false, true} {
				var innerRect Rect
				tt := coreNewTester(func(c *context) {
					coreBox(c).Fill().Background(green).Children(func() {
						coreBox(c).Absolute().Left(off).Top(off).Size(40, 40).Border(1, blue).Clip().Children(func() {
							if nested {
								coreRow(c).Fill().Children(func() {
									innerRect = coreBox(c).Fill().Background(red).Bounds()
								})
							} else {
								innerRect = coreBox(c).Fill().Background(red).Bounds()
							}
						})
					})
				}, 80, 80)
				tt.SetScale(sc.scale)
				tt.Frame()
				// Layout remains DIP-invariant across fractional DPI scales.
				if innerRect.W != 38 || innerRect.H != 38 {
					t.Fatalf("scale=%.2f off=%g nested=%v layout drifted from DIP-invariant 38x38: %+v", sc.scale, off, nested, innerRect)
				}

				img := tt.Image()
				x0 := int(round(off * sc.scale))
				x1 := int(round((off + 40) * sc.scale))
				y0 := int(round(off * sc.scale))
				y1 := int(round((off + 40) * sc.scale))
				midX := (x0 + x1) / 2
				midY := (y0 + y1) / 2

				checkRun := func(side string, sample func(i int) color.RGBA, n int) {
					t.Helper()
					if got := classify(sample(-1)); got != 'G' {
						t.Fatalf("scale=%.2f off=%g nested=%v %s outside=%c (%v), want G", sc.scale, off, nested, side, got, sample(-1))
					}
					for i := 0; i < sc.wantPx; i++ {
						if got := classify(sample(i)); got != 'B' {
							t.Fatalf("scale=%.2f off=%g nested=%v %s border[%d]=%c (%v), want B", sc.scale, off, nested, side, i, got, sample(i))
						}
					}
					for i := sc.wantPx; i < sc.wantPx+4 && i < n; i++ {
						if got := classify(sample(i)); got != 'R' {
							t.Fatalf("scale=%.2f off=%g nested=%v %s inner[%d]=%c (%v), want R (1px seam bleed!)", sc.scale, off, nested, side, i, got, sample(i))
						}
					}
				}

				checkRun("left", func(i int) color.RGBA { return img.RGBAAt(x0+i, midY) }, x1-x0)
				checkRun("right", func(i int) color.RGBA { return img.RGBAAt(x1-1-i, midY) }, x1-x0)
				checkRun("top", func(i int) color.RGBA { return img.RGBAAt(midX, y0+i) }, y1-y0)
				checkRun("bottom", func(i int) color.RGBA { return img.RGBAAt(midX, y1-1-i) }, y1-y0)
			}
		}
	}
}

// TestBorderOneClipZeroSizedChildPaintsNoGhostPixels pins that zero-width or
// zero-height children (such as 0% progress bars or collapsed panels) inside a
// Border(1).Clip() container paint strictly 0 pixels across 100%/150%/250% DPI
// instead of expanding into 1px ghost strips via borderSeam snapping.
func TestBorderOneClipZeroSizedChildPaintsNoGhostPixels(t *testing.T) {
	green := RGB(20, 200, 40)
	blue := RGB(20, 60, 230)
	red := RGB(230, 30, 30)

	cases := []struct {
		name  string
		child func(c *context)
	}{
		{"width-0-left", func(c *context) { coreBox(c).Width(0).FillHeight().Background(red) }},
		{"width-0-right", func(c *context) { coreBox(c).Absolute().Right(0).Top(0).Bottom(0).Width(0).Background(red) }},
		{"height-0-top", func(c *context) { coreBox(c).FillWidth().Height(0).Background(red) }},
		{"height-0-bottom", func(c *context) { coreBox(c).Absolute().Left(0).Right(0).Bottom(0).Height(0).Background(red) }},
	}

	for _, sc := range []float32{1.0, 1.25, 1.5, 1.75, 2.0, 2.5} {
		for _, off := range []float32{10, 11, 12, 13} {
			for _, tc := range cases {
				tt := coreNewTester(func(c *context) {
					coreBox(c).Fill().Background(green).Children(func() {
						coreBox(c).Absolute().Left(off).Top(off).Size(40, 40).Border(1, blue).Clip().Children(func() {
							tc.child(c)
						})
					})
				}, 80, 80)
				tt.SetScale(sc)
				tt.Frame()
				img := tt.Image()
				b := img.Bounds()
				redCount := 0
				for y := b.Min.Y; y < b.Max.Y; y++ {
					for x := b.Min.X; x < b.Max.X; x++ {
						px := img.RGBAAt(x, y)
						if px.R > 160 && px.G < 80 && px.B < 80 {
							redCount++
						}
					}
				}
				if redCount != 0 {
					t.Fatalf("scale=%.2f off=%g case=%s: painted %d ghost red pixels, want 0", sc, off, tc.name, redCount)
				}
			}
		}
	}
}
