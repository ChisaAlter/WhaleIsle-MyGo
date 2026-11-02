package frame

import (
	"bytes"
	"image"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestRenderSnapshotReusesPassAndSeesSameFrameRevisions(t *testing.T) {
	store := session.New()
	store.Select("first")
	pass := store.BeginFrame()
	defer pass.EndFrame()
	m := Model{Store: store, FrameView: pass}
	first := renderSnapshot(m)
	builds, hits := store.SnapshotStats()
	second := renderSnapshot(m)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("unchanged pass changed rendered state")
	}
	if b, h := store.SnapshotStats(); b != builds || h != hits+1 {
		t.Fatal("unchanged render did not reuse existing pass")
	}
	store.SetLastAction("same-frame")
	if got := renderSnapshot(m).LastAction; got != "same-frame" {
		t.Fatalf("same-frame mutation stale: %q", got)
	}
	store.Select("second")
	if got := renderSnapshot(m).SelectedID; got != "second" {
		t.Fatalf("same-frame selection stale: %q", got)
	}
	if first.SelectedID != "first" {
		t.Fatal("later revision changed prior rendered state")
	}
	ordinary := store.Snapshot()
	ordinary.LastAction = "caller-owned"
	if got := renderSnapshot(m).LastAction; got == "caller-owned" {
		t.Fatal("mutable ordinary Snapshot changed pass state")
	}
}

func TestRenderSnapshotFallsBackForMissingOrDifferentOwner(t *testing.T) {
	first, second := session.New(), session.New()
	first.Select("primary")
	second.Select("child")
	pass := first.BeginFrame()
	defer pass.EndFrame()
	for _, m := range []Model{{Store: second}, {Store: second, FrameView: pass}} {
		if got := renderSnapshot(m).SelectedID; got != "child" {
			t.Fatalf("wrong owner snapshot selected=%q", got)
		}
		second.SetLastAction("fresh")
		if got := renderSnapshot(m).LastAction; got != "fresh" {
			t.Fatal("fallback failed to read current owner")
		}
	}
	if got := pass.Snapshot().SelectedID; got != "primary" {
		t.Fatal("foreign Store lookup changed primary pass")
	}
}

func TestFrameAndNavigationReadOnlyPassMatchFallbackPixels(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	store := session.New()
	store.Select("fixture")
	store.SetLastAction("stable")
	for _, external := range []bool{false, true} {
		paint := func(reuse bool) ([]string, *image.RGBA) {
			layout := DefaultLayout()
			layout.Bottom, layout.Right = 0, 0
			draft := ""
			capture := ui.NewTester(func(c *ui.Context) {
				pass := store.BeginFrame()
				defer pass.EndFrame()
				m := Model{Store: store, Sizes: &layout, Draft: &draft, ThemeID: "deepseek.light", ExternalNavigation: external}
				if reuse {
					m.FrameView = pass
				}
				c.SetTheme(ui.LightTheme())
				ui.Column(c).Fill().Children(func() {
					if external {
						NavigationShell(c, m, func() { View(c, m) })
					} else {
						View(c, m)
					}
				})
			}, 1440, 864)
			capture.Frame()
			return capture.Texts(), capture.Image()
		}
		oldLabels, oldPixels := paint(false)
		newLabels, newPixels := paint(true)
		if !reflect.DeepEqual(oldLabels, newLabels) || oldPixels.Rect != newPixels.Rect || !bytes.Equal(oldPixels.Pix, newPixels.Pix) {
			t.Fatalf("external=%v frame pass changed rendered output", external)
		}
	}
}
