package pluginmanager

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
)

// specInstallCardAt renders the install card's spec screen in a window of the
// given height with the guide unfolded, so a short window leaves the body taller
// than the card can show.
func specInstallCardAt(t *testing.T, height int) *ui.Tester {
	t.Helper()
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "idle"}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, height)
	if err := tt.Click(guideToggleLabel(false)); err != nil {
		t.Fatalf("the guide toggle is missing: %v", err)
	}
	for range 4 {
		tt.Frame()
	}
	if !tt.HasText(guideToggleLabel(true)) {
		t.Fatalf("the guide did not unfold: %v", tt.Texts())
	}
	return tt
}

// installFooterBoxes finds the three pieces of the card's footer: the card, the
// first line of the trust note and the primary action.
func installFooterBoxes(t *testing.T, tt *ui.Tester) (card, note, action ui.Rect) {
	t.Helper()
	var ok bool
	if card, ok = tt.Find(copyText("添加插件", "Add plugin")); !ok {
		t.Fatalf("install card not found: %v", tt.Texts())
	}
	safety, _ := installSafetyCopy()
	if note, ok = tt.Find(safety); !ok {
		t.Fatalf("the trust note is missing: %v", tt.Texts())
	}
	if action, ok = tt.Find(copyText("安装", "Install")); !ok {
		t.Fatalf("the primary action is missing: %v", tt.Texts())
	}
	return card, note, action
}

// TestInstallFooterKeepsTheNoteAndActionOutsideTheScroll pins the reference's
// footer (PluginManagerPage.tsx:960-974, `.installFooter`): the trust note and
// the primary action are the card's last rows, so they hold their place while
// the content above them scrolls, instead of moving with it as body items do.
func TestInstallFooterKeepsTheNoteAndActionOutsideTheScroll(t *testing.T) {
	tall, short := specInstallCardAt(t, 920), specInstallCardAt(t, 480)
	tallCard, tallNote, tallAction := installFooterBoxes(t, tall)
	shortCard, shortNote, shortAction := installFooterBoxes(t, short)

	// Both windows end the footer at the reference's 24 DIP inset.
	for _, want := range []struct {
		name string
		card ui.Rect
		act  ui.Rect
	}{
		{"920", tallCard, tallAction},
		{"480", shortCard, shortAction},
	} {
		if inset := want.card.Y + want.card.H - (want.act.Y + want.act.H); math.Abs(float64(inset-24)) > 1 {
			t.Fatalf("at %s the action ends %.1f DIP above the card's bottom, want the reference's 24", want.name, inset)
		}
	}

	// The note and the action sit in the footer's 20 DIP column, above the
	// action's 40 DIP height: the note's own 10 DIP bottom padding and the
	// footer's 20 DIP gap separate the upgrade notice from the action.
	_, upgrade := installSafetyCopy()
	for _, want := range []struct {
		name string
		tt   *ui.Tester
		note ui.Rect
		act  ui.Rect
	}{
		{"920", tall, tallNote, tallAction},
		{"480", short, shortNote, shortAction},
	} {
		second, ok := want.tt.Find(upgrade)
		if !ok {
			t.Fatalf("at %s the upgrade notice is missing: %v", want.name, want.tt.Texts())
		}
		if gap := want.act.Y - (second.Y + second.H); math.Abs(float64(gap-30)) > 1.5 {
			t.Fatalf("at %s the upgrade notice ends %.1f DIP above the action, want the note's 10 DIP padding and the footer's 20 DIP gap", want.name, gap)
		}
		if gap := second.Y - (want.note.Y + want.note.H); math.Abs(float64(gap-4)) > 1.5 {
			t.Fatalf("at %s the upgrade notice starts %.1f DIP under the first line, want the note's 4 DIP gap", want.name, gap)
		}
	}

	// The footer keeps the same distance from the card's bottom in both windows: a
	// body item would follow the content down instead and leave the card's bottom.
	var tallNoteFromBottom, tallActFromBottom float32
	for _, want := range []struct {
		name string
		card ui.Rect
		note ui.Rect
		act  ui.Rect
	}{
		{"920", tallCard, tallNote, tallAction},
		{"480", shortCard, shortNote, shortAction},
	} {
		noteFromBottom := want.card.Y + want.card.H - want.note.Y
		actFromBottom := want.card.Y + want.card.H - want.act.Y
		if want.name == "920" {
			tallNoteFromBottom, tallActFromBottom = noteFromBottom, actFromBottom
			continue
		}
		if math.Abs(float64(noteFromBottom-tallNoteFromBottom)) > 1 {
			t.Fatalf("the note sits %.1f DIP above the card's bottom at 480 and %.1f at 920: it scrolls with the body",
				noteFromBottom, tallNoteFromBottom)
		}
		if math.Abs(float64(actFromBottom-tallActFromBottom)) > 1 {
			t.Fatalf("the action sits %.1f DIP above the card's bottom at 480 and %.1f at 920: it scrolls with the body",
				actFromBottom, tallActFromBottom)
		}
	}

	// A short window still keeps the whole footer inside the card, below the body,
	// and the body really overflows it: the guide's example is laid out past the
	// note's top, so the note is painted over content that ran out of room.
	if shortCard.H > 480-47 {
		t.Fatalf("the short card is %.0f DIP tall, past the reference's 24 DIP inset", shortCard.H)
	}
	if shortNote.Y < shortCard.Y+pluginDialogHeaderHeight {
		t.Fatalf("the note starts at %.0f, above the card's body", shortNote.Y)
	}
	example, ok := short.Find("dsh-plugin-whale-pet")
	if !ok {
		t.Fatalf("the unfolded guide lost its example: %v", short.Texts())
	}
	if example.Y <= shortNote.Y {
		t.Fatalf("the guide's example ends at %.0f, inside the body of a %.0f DIP card: nothing overflows",
			example.Y+example.H, shortCard.H)
	}
	// The footer is painted, not clipped away with the content: the note's tint is
	// on screen at the card's bottom in the very window where the body overflows.
	painted := short.Image().At(int(shortCard.X+24+2), int(shortNote.Y-10+12))
	if !closeTo(painted, color.RGBA{254, 245, 231, 255}) {
		t.Fatalf("the note paints %v in the short window, want its warn tint on screen", painted)
	}
}

// TestInstallSafetyNoteMatchesTheReference pins `.installSafety`: a 12 DIP radius
// warn panel with 10/12 padding, the 14 DIP warning glyph and two 12/18 lines of
// the reference's copy, at the footer's 24 DIP inset.
func TestInstallSafetyNoteMatchesTheReference(t *testing.T) {
	tt := specInstallCardAt(t, 920)
	card, note, _ := installFooterBoxes(t, tt)

	// Padding 12 + gap 8 + the 14 DIP glyph: the copy starts 34 DIP into the note,
	// which starts 24 DIP into the card (the footer's own inset).
	if inset := note.X - card.X; math.Abs(float64(inset-58)) > 1 {
		t.Fatalf("the note's copy starts %.1f DIP into the card, want 24 + 12 + 8 + 14 = 58", inset)
	}
	// The copy stays inside the note's 12 DIP right padding.
	if right := card.X + card.W - 24 - 12; note.X+note.W > right+1 {
		t.Fatalf("the note's copy reaches %.1f, past the note's own right edge %.1f", note.X+note.W, right)
	}

	// The note's second line is the upgrade notice, directly under the first.
	_, upgrade := installSafetyCopy()
	second, ok := tt.Find(upgrade)
	if !ok {
		t.Fatalf("the upgrade notice is missing: %v", tt.Texts())
	}
	if gap := second.Y - (note.Y + note.H); math.Abs(float64(gap-4)) > 1.5 {
		t.Fatalf("the upgrade notice starts %.1f DIP under the first line, want the note's 4 DIP gap", gap)
	}

	// The panel is the amber warn tint of --dsw-alias-state-warn-tertiary, and its
	// 12 DIP radius leaves the corner in the card's own colour. The found box is
	// the first line of copy, which sits the panel's 10/12 padding inside it.
	img := tt.Image()
	panelLeft, panelTop := card.X+24, note.Y-10
	mid := img.At(int(panelLeft+2), int(panelTop+12))
	if !closeTo(mid, color.RGBA{254, 245, 231, 255}) {
		t.Fatalf("the note paints %v at its left edge, want the reference's warn tint rgb(254, 245, 231)", mid)
	}
	corner := img.At(int(panelLeft+2), int(panelTop+2))
	if closeTo(corner, color.RGBA{254, 245, 231, 255}) {
		t.Fatalf("the note's corner paints %v: the panel has no 12 DIP radius", corner)
	}
}

// TestInstallActionMatchesTheWidePrimary pins `.wide`
// (PluginManagerPage.module.css:864-869): the footer's action fills the card's
// content width at 40 DIP.
func TestInstallActionMatchesTheWidePrimary(t *testing.T) {
	tt := specInstallCardAt(t, 920)
	card, _, action := installFooterBoxes(t, tt)
	if math.Abs(float64(action.W-(card.W-48))) > 1 {
		t.Fatalf("the action is %.1f DIP wide in a %.0f DIP card, want the content width of 512", action.W, card.W)
	}
	if math.Abs(float64(action.H-40)) > 1 {
		t.Fatalf("the action is %.1f DIP tall, want the reference's 40", action.H)
	}
	if math.Abs(float64(action.X-card.X-24)) > 1 {
		t.Fatalf("the action starts %.1f DIP into the card, want the footer's 24", action.X-card.X)
	}
}

// closeTo compares a painted pixel with a token colour.
func closeTo(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	return math.Abs(float64(r>>8)-float64(want.R)) < 3 &&
		math.Abs(float64(g>>8)-float64(want.G)) < 3 &&
		math.Abs(float64(b>>8)-float64(want.B)) < 3
}

// TestInstallFooterKeepsTheLongCopyInsideTheCard covers the wider locale: the
// English safety note and upgrade notice are longer than the Chinese ones, so
// the footer must grow instead of pushing the action past the card's edge.
func TestInstallFooterKeepsTheLongCopyInsideTheCard(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "idle"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			installDialog(c, m, s, m.Controller.Actions(s.Owner))
		})
	}, 1440, 920)
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}

	safety, upgrade := installSafetyCopy()
	if !tt.HasText(safety) || !tt.HasText(upgrade) {
		t.Fatalf("the English card lost the reference's note: %v", tt.Texts())
	}
	card, note, action := installFooterBoxes(t, tt)
	if note.H < 18 || note.Y < card.Y+pluginDialogHeaderHeight {
		t.Fatalf("the English note is out of place: %v in a %.0f DIP card", note, card.H)
	}
	if inset := card.Y + card.H - (action.Y + action.H); math.Abs(float64(inset-24)) > 1 {
		t.Fatalf("the English action ends %.1f DIP above the card's bottom, want the reference's 24", inset)
	}
	if action.Y+action.H > card.Y+card.H {
		t.Fatalf("the English action ends at %.0f, past the card's bottom %.0f", action.Y+action.H, card.Y+card.H)
	}
}

// TestInstallFooterReadsCheckingWhileTheSpecChecks covers the reference's busy
// action (PluginManagerPage.tsx:969-972): the footer's button names the check
// instead of offering the install again.
func TestInstallFooterReadsCheckingWhileTheSpecChecks(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "checking", Spec: "dsh-plugin-whale-pet"}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	if !tt.HasText(copyText("正在检查…", "Checking…")) {
		t.Fatalf("the checking card lost the reference's checking label: %v", tt.Texts())
	}
	if _, ok := tt.Find(copyText("安装", "Install")); ok {
		t.Fatalf("the checking card still offers the install action: %v", tt.Texts())
	}
	// The way back to editing stays in the body; the reference has no such
	// control, and it is the native recovery path for a check that hangs.
	if !tt.HasText(copyText("取消检查并返回编辑", "Cancel check and return to editing")) {
		t.Fatalf("the checking card lost its way back to editing: %v", tt.Texts())
	}
	safety, _ := installSafetyCopy()
	if !tt.HasText(safety) {
		t.Fatalf("the checking card lost the trust note: %v", tt.Texts())
	}
}

// TestInstallFooterIsAbsentFromTheLogScreen keeps the footer to the spec screen:
// the installing, installed and failed screens carry their own actions, as the
// reference's wizard does (PluginManagerPage.tsx:930-974).
func TestInstallFooterIsAbsentFromTheLogScreen(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "failed", Spec: "fixture-plugin",
		Result: &manager.Change{Application: "failed"}}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	safety, _ := installSafetyCopy()
	if tt.HasText(safety) {
		t.Fatalf("the failed screen kept the spec screen's footer note: %v", tt.Texts())
	}
	if !tt.HasText(copyText("重试安装", "Retry installation")) {
		t.Fatalf("the failed screen lost its own action: %v", tt.Texts())
	}
}
