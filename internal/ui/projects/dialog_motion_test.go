package projects

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

// TestProjectRenameDialogFadesInLikeTheModal covers the reference modal's enter
// (Modal.module.css:29-31, 48): the mask and the card fade in over
// --ds-transition-duration. The native lifecycle cards hand-run their
// DialogBase, so they drive the motion themselves; before this they snapped on
// at full opacity in the first frame.
func TestProjectRenameDialogFadesInLikeTheModal(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	m := projectFixture()
	draft := "Harbor renamed"
	m.RenameID = "project-fixture"
	m.RenameDraft = &draft
	m.OnRename = func(string, string) {}
	m.OnCancelRename = func() { m.RenameID = "" }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 1440, 920)

	start := time.Unix(1000, 0)
	tt.SetTime(start)
	card, ok := tt.Find("Rename Project")
	if !ok {
		t.Fatalf("rename card not found: %v", tt.Texts())
	}
	// A point inside the card's empty bottom-right corner, clear of its text,
	// input and buttons.
	inX, inY := int(card.X+card.W-10), int(card.Y+card.H-10)
	at := func(d time.Duration) uint32 {
		tt.SetTime(start.Add(d))
		r, _, _, _ := tt.Image().RGBAAt(inX, inY).RGBA()
		return r >> 8
	}
	hidden, half, settled := at(0), at(chrome.OverlayDuration/2), at(chrome.OverlayDuration)
	// The card is not painted on the first frame: the page shows through.
	if hidden != 255 {
		t.Fatalf("the card paints %d at the start of the enter, want the page's 255", hidden)
	}
	if half == hidden || half == settled {
		t.Fatalf("the card did not fade in: %d → %d → %d", hidden, half, settled)
	}
	// It settles on the light theme's Field surface, rgb(245, 246, 247).
	if settled < 242 || settled > 248 {
		t.Fatalf("the card settles at %d, want the Field surface's 245", settled)
	}
	// Evidence: the card halfway through the enter.
	tt.SetTime(start.Add(chrome.OverlayDuration / 2))
	saveProjectSnapshot(t, tt, "projects-rename-fade-1440.png")

	// The mask fades with it: the page outside the card darkens over the enter
	// instead of snapping to the dim layer.
	maskAt := func(d time.Duration) uint32 {
		tt.SetTime(start.Add(d))
		r, _, _, _ := tt.Image().RGBAAt(100, 100).RGBA()
		return r >> 8
	}
	maskStart, maskSettled := maskAt(0), maskAt(chrome.OverlayDuration)
	if maskStart != 255 {
		t.Fatalf("the mask paints %d before the enter, want the page's 255", maskStart)
	}
	// .24 of black over the page: the reference's bg-mask-1 on the light scheme.
	if want := uint32(float32(maskStart) * .76); maskSettled+3 < want || maskSettled > want+3 {
		t.Fatalf("the settled mask is %d, want %d (24%% of black over %d)", maskSettled, want, maskStart)
	}
}
