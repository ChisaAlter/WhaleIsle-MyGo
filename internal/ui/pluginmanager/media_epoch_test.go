package pluginmanager

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/ui/i18n"
)

// iconDataURL is a solid-colour PNG the media service decodes on its worker:
// a frame painted before the decode ran shows the fallback glyph instead.
func iconDataURL(t *testing.T, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// testMediaService installs a media service the test drives: nothing decodes
// until RunPending, so pending and ready frames are deterministic.
func testMediaService(t *testing.T) *media.Service {
	t.Helper()
	s := media.Init(media.Config{Workers: -1})
	t.Cleanup(func() { media.Init(media.Config{}) })
	return s
}

// painted counts pixels close to the fixture colour, proving the decoded
// picture reached the surface rather than the fallback glyph. The fixture
// colours are hues no design token uses, so a badge or button cannot fake it.
func painted(tt *ui.Tester, want color.RGBA) int {
	const tol = 8
	img := tt.Image()
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := img.RGBAAt(x, y)
			if d := int(px.R) - int(want.R); d < -tol || d > tol {
				continue
			}
			if d := int(px.G) - int(want.G); d < -tol || d > tol {
				continue
			}
			if d := int(px.B) - int(want.B); d < -tol || d > tol {
				continue
			}
			n++
		}
	}
	return n
}

func settledModel(t *testing.T, f *reviewFixture) Model {
	t.Helper()
	m := reviewModel(t, f)
	owner := m.Controller.Snapshot().Value.Owner
	if err := m.Controller.LoadIndexes(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	return m
}

// swapMediaLease releases the page's media lease while its pictures are still
// pending — the window hide/session swap the owner exists for — then runs the
// decodes the regions must ask the next lease for. A memo region that stays
// put keeps the dead owner's fallback forever.
func swapMediaLease(t *testing.T, tt *ui.Tester, s *media.Service, m *Model) {
	t.Helper()
	revBefore := m.Controller.Snapshot().Rev
	m.Media.Release()
	m.Media = s.NewOwner(nil, nil)
	for range 3 {
		tt.Frame()
	}
	s.RunPending()
	for range 3 {
		tt.Frame()
	}
	if revAfter := m.Controller.Snapshot().Rev; revAfter != revBefore {
		t.Fatalf("controller revision changed during media swap (%d -> %d); mediaEpoch not isolated", revBefore, revAfter)
	}
}

// Every artwork memo region builds under one media lease and must still load
// its icon after the lease is released and replaced.
func TestArtworkReloadsAfterMediaLeaseSwap(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	red := color.RGBA{R: 216, G: 0, B: 200, A: 255}
	green := color.RGBA{R: 148, G: 208, B: 0, A: 255}
	blue := color.RGBA{R: 140, G: 30, B: 200, A: 255}

	fixture := func(t *testing.T) *reviewFixture {
		f := newReviewFixture()
		f.bundles[0].Meta.Icon = iconDataURL(t, red)
		f.bundles[0].Rows[0].Meta.Icon = iconDataURL(t, green)
		f.entryIcon = iconDataURL(t, blue)
		return f
	}

	// plugins-installed and its package rows (view.go's list and row memos).
	t.Run("installed rows", func(t *testing.T) {
		s := testMediaService(t)
		m := settledModel(t, fixture(t))
		m.Media = s.NewOwner(nil, nil)
		tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
		if painted(tt, red) != 0 {
			t.Fatal("package icon painted before its decode ran")
		}
		swapMediaLease(t, tt, s, &m)
		if painted(tt, red) == 0 {
			t.Fatal("installed row kept the released lease's dead icon")
		}
	})

	// plugin-detail-head and plugin-parts (detail.go's two memos).
	t.Run("detail head and parts", func(t *testing.T) {
		s := testMediaService(t)
		m := settledModel(t, fixture(t))
		m.Media = s.NewOwner(nil, nil)
		tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
		if err := tt.Click("Fixture plugin"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if painted(tt, red) != 0 || painted(tt, green) != 0 {
			t.Fatal("detail icons painted before their decodes ran")
		}
		swapMediaLease(t, tt, s, &m)
		if painted(tt, red) == 0 {
			t.Fatal("detail head kept the released lease's dead icon")
		}
		if painted(tt, green) == 0 {
			t.Fatal("parts row kept the released lease's dead icon")
		}
	})

	// discover-listing and its entry rows (discover.go's two memos).
	t.Run("discover listing", func(t *testing.T) {
		s := testMediaService(t)
		m := settledModel(t, fixture(t))
		owner := m.Controller.Snapshot().Value.Owner
		m.Controller.Actions(owner).SelectTab("discover")
		m.Media = s.NewOwner(nil, nil)
		tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1200, 900)
		if !tt.HasText("Discover fixture") {
			t.Fatalf("listing did not load: %v", tt.Texts())
		}
		if painted(tt, blue) != 0 {
			t.Fatal("entry icon painted before its decode ran")
		}
		swapMediaLease(t, tt, s, &m)
		if painted(tt, blue) == 0 {
			t.Fatal("discover row kept the released lease's dead icon")
		}
	})
}
