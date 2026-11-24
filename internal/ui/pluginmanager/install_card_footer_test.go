package pluginmanager

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// specInstallCardAt renders the install card's spec screen in a window of the
// given height with the guide unfolded, so a short window leaves the body taller
// than the card can show.
func specInstallCardAt(t *testing.T, height int) *ui.Tester {
	t.Helper()
	m := pageModel(t)
	s := m.Controller.Snapshot().Value
	s.Install = manager.InstallState{Open: true, Phase: "idle"}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, testPage(m, s))
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

	// The upgrade notice is the note's second line, and the note and the action
	// sit in one footer cluster: the notice ends just above the action, the way
	// the reference's `.installFooter` column reads.
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
		if gap := want.act.Y - (second.Y + second.H); gap < 8 || gap > 48 {
			t.Fatalf("at %s the upgrade notice ends %.1f DIP above the action, want one footer cluster", want.name, gap)
		}
		if gap := second.Y - (want.note.Y + want.note.H); gap < 0 || gap > 12 {
			t.Fatalf("at %s the upgrade notice starts %.1f DIP under the first line, want the note's own line gap", want.name, gap)
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
	desc, ok := short.Find(copyText("输入完整 npm 包名、Git 仓库地址或本地目录。", "Enter a full npm package name, Git repository address, or local directory."))
	if !ok {
		t.Fatalf("the spec screen lost its header line: %v", short.Texts())
	}
	if shortNote.Y <= desc.Y+desc.H {
		t.Fatalf("the note starts at %.0f, inside the card's header", shortNote.Y)
	}
	// The body really ran out of room: the unfolded guide is clipped away by
	// the scroll viewport, so the note is painted over content that ran out
	// of room.
	overflows := false
	for _, r := range short.FindAll("dsh-plugin-whale-pet") {
		if r.W == 0 || r.H == 0 || r.Y+r.H > shortNote.Y-16 {
			overflows = true
		}
	}
	if !overflows {
		t.Fatalf("the unfolded guide is fully visible in a %.0f DIP card: nothing overflows", shortCard.H)
	}
	// The footer is painted, not clipped away with the content: the note's
	// warning border is on screen at the card's bottom in the very window
	// where the body overflows.
	if !warnishNear(short.Image(), int(shortNote.X+8), int(shortNote.Y)-16, int(shortNote.Y)) {
		t.Fatalf("the note has no warning-tinted panel edge in the short window")
	}
}

// TestInstallSafetyNoteMatchesTheReference pins the spec screen's `.installSafety`
// panel on the shared alert: a warning-tinted note holding the reference's two
// lines of trust copy, past its icon and inside the footer's 24 DIP inset.
func TestInstallSafetyNoteMatchesTheReference(t *testing.T) {
	tt := specInstallCardAt(t, 920)
	card, note, _ := installFooterBoxes(t, tt)

	// The copy starts past the panel's icon inside the footer's inset: the
	// shared alert's padding, icon and gap indent it deep into the card.
	if inset := note.X - card.X; inset < 50 || inset > 74 {
		t.Fatalf("the note's copy starts %.1f DIP into the card, want the footer inset plus the panel's icon column", inset)
	}
	// The copy stays inside the panel's own right padding.
	if right := card.X + card.W - (note.X + note.W); right < 30 || right > 50 {
		t.Fatalf("the note's copy ends %.1f DIP before the card's right edge, want it inside the panel's padding", right)
	}

	// The note's second line is the upgrade notice, directly under the first.
	_, upgrade := installSafetyCopy()
	second, ok := tt.Find(upgrade)
	if !ok {
		t.Fatalf("the upgrade notice is missing: %v", tt.Texts())
	}
	if gap := second.Y - (note.Y + note.H); gap < 0 || gap > 12 {
		t.Fatalf("the upgrade notice starts %.1f DIP under the first line, want the note's own line gap", gap)
	}

	// The note reads as a warning panel: the shared alert's tinted border shows
	// just above the first line of copy.
	if !warnishNear(tt.Image(), int(note.X+8), int(note.Y)-16, int(note.Y)) {
		t.Fatalf("the note has no warning-tinted panel edge")
	}
}

// TestInstallActionMatchesTheWidePrimary pins `.wide`
// (PluginManagerPage.module.css:864-869): the footer's action fills the card's
// content width at 40 DIP.
func TestInstallActionMatchesTheWidePrimary(t *testing.T) {
	tt := specInstallCardAt(t, 920)
	card, _, action := installFooterBoxes(t, tt)
	// The panel's 1 DIP border and the footer's 24 DIP inset leave 510 of
	// content width in the 560 DIP card.
	if math.Abs(float64(action.W-(card.W-50))) > 1 {
		t.Fatalf("the action is %.1f DIP wide in a %.0f DIP card, want the footer's content width", action.W, card.W)
	}
	if math.Abs(float64(action.H-40)) > 1 {
		t.Fatalf("the action is %.1f DIP tall, want the reference's 40", action.H)
	}
	if math.Abs(float64(action.X-card.X-25)) > 1 {
		t.Fatalf("the action starts %.1f DIP into the card, want the border plus the footer's 24", action.X-card.X)
	}
}

// closeTo compares a painted pixel with a token colour.
func closeTo(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	return math.Abs(float64(r>>8)-float64(want.R)) < 3 &&
		math.Abs(float64(g>>8)-float64(want.G)) < 3 &&
		math.Abs(float64(b>>8)-float64(want.B)) < 3
}

// warnishNear reports whether a warning-tinted pixel (a warm hue, red over
// green over blue) shows anywhere in column x over the y range [top, bottom):
// the safety note's warn border on the shared alert, on either scheme.
func warnishNear(img *image.RGBA, x, top, bottom int) bool {
	for y := top; y < bottom; y++ {
		c := img.At(x, y)
		r, g, b, _ := c.RGBA()
		if r>>8 > 80 && r>>8 > g>>8+10 && g>>8 > b>>8+10 {
			return true
		}
	}
	return false
}

// TestInstallFooterKeepsTheLongCopyInsideTheCard covers the wider locale: the
// English safety note and upgrade notice are longer than the Chinese ones, so
// the footer must grow instead of pushing the action past the card's edge.
func TestInstallFooterKeepsTheLongCopyInsideTheCard(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	m := pageModel(t)
	s := m.Controller.Snapshot().Value
	s.Install = manager.InstallState{Open: true, Phase: "idle"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			installDialog(c, testPage(m, s))
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
	desc, ok := tt.Find("Enter a full npm package name, Git repository address, or local directory.")
	if !ok {
		t.Fatalf("the English card lost its header line: %v", tt.Texts())
	}
	if note.H < 18 || note.Y <= desc.Y+desc.H {
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
	s := m.Controller.Snapshot().Value
	s.Install = manager.InstallState{Open: true, Phase: "checking", Spec: "dsh-plugin-whale-pet"}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, testPage(m, s))
	}, 1440, 920)

	if !tt.HasText(copyText("正在检查…", "Checking…")) {
		t.Fatalf("the checking card lost the reference's checking label: %v", tt.Texts())
	}
	if _, ok := tt.Find(copyText("安装", "Install")); ok {
		t.Fatalf("the checking card still offers the install action: %v", tt.Texts())
	}
	// The way back to editing stays in the body; the reference has no such
	// control, and it is the native recovery path for a check that hangs.
	if !tt.HasText(copyText("取消安装并返回编辑", "Cancel check and return to editing")) {
		t.Fatalf("the checking card lost its way back to editing: %v", tt.Texts())
	}
	safety, _ := installSafetyCopy()
	if !tt.HasText(safety) {
		t.Fatalf("the checking card lost the trust note: %v", tt.Texts())
	}
}

// marketChoiceFixture renders the native market confirmation card over the
// shared backdrop, as the market's install button opens it.
func marketChoiceFixture(t *testing.T) *ui.Tester {
	t.Helper()
	m := pageModel(t)
	s := m.Controller.Snapshot().Value
	entry := manager.MarketEntry{Name: "fixture-discover", Title: manager.Text(`"Discover fixture"`),
		Description: manager.Text(`"A source-backed entry with exact install versions."`), Author: "Fixture author", DefaultVersion: "2.0.0"}
	entry.Installation.Status = "not-installed"
	version := manager.MarketVersion{Version: "2.0.0", Compatibility: manager.Compatibility{Compatible: true}}
	version.Install.Spec = "fixture-discover@2.0.0"
	entry.Versions = []manager.MarketVersion{version}
	s.Discover.Choice = &manager.MarketChoice{Entry: entry, Version: version}
	return pluginDialogFixture(t, func(c *ui.Context) {
		marketChoiceDialog(c, testPage(m, s))
	}, 1440, 920)
}

// TestMarketChoiceCarriesTheReferenceSafetyNote covers the two install cards
// speaking in one voice: the native market confirmation used to warn with a
// one-line sentence of its own ("请确认来源可信。插件以当前 profile 的权限运行。")
// while the spec screen warned with the reference's `.installSafety` panel. It
// now shows the same panel and the same reference copy.
func TestMarketChoiceCarriesTheReferenceSafetyNote(t *testing.T) {
	tt := marketChoiceFixture(t)
	card, ok := tt.Find(copyText("确认安装插件", "Confirm plugin installation"))
	if !ok {
		t.Fatalf("the market choice card is missing: %v", tt.Texts())
	}
	safety, upgrade := installSafetyCopy()
	note, ok := tt.Find(safety)
	if !ok {
		t.Fatalf("the market card lost the reference's trust note: %v", tt.Texts())
	}
	if !tt.HasText(upgrade) {
		t.Fatalf("the market card lost the upgrade notice: %v", tt.Texts())
	}
	if tt.HasText("请确认来源可信。插件以当前 profile 的权限运行。") {
		t.Fatalf("the market card still warns with its own one-line sentence: %v", tt.Texts())
	}

	// The copy starts past the shared alert's icon inside the body's inset and
	// stays inside the card, as in the spec screen's footer.
	if inset := note.X - card.X; inset < 50 || inset > 74 {
		t.Fatalf("the market note's copy starts %.1f DIP into the card, want the body inset plus the panel's icon column", inset)
	}
	if right := card.X + card.W - (note.X + note.W); right < 30 || right > 50 {
		t.Fatalf("the market note's copy ends %.1f DIP before the card's right edge, want it inside the panel's padding", right)
	}
	// The panel reads as a warning note: its tinted border shows above the copy.
	if !warnishNear(tt.Image(), int(note.X+8), int(note.Y)-16, int(note.Y)) {
		t.Fatalf("the market note has no warning-tinted panel edge")
	}
}

// TestInstallSafetyNoteReadsAsAWarnPanelOnDark keeps the note readable on the
// dark scheme: the shared alert still frames it with a warning-tinted edge
// inside the dark card.
func TestInstallSafetyNoteReadsAsAWarnPanelOnDark(t *testing.T) {
	m := pageModel(t)
	m.SystemDark = true
	m.ThemeID = theme.IDDeepseekDark
	s := m.Controller.Snapshot().Value
	s.Install = manager.InstallState{Open: true, Phase: "idle"}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ui.Column(c).Fill().Children(func() { installDialog(c, testPage(m, s)) })
	}, 1440, 920)
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
	card, note, _ := installFooterBoxes(t, tt)
	if note.Y <= card.Y || note.Y+note.H >= card.Y+card.H {
		t.Fatalf("the dark note is out of place: %v in a %.0f DIP card", note, card.H)
	}
	if !warnishNear(tt.Image(), int(note.X+8), int(note.Y)-16, int(note.Y)) {
		t.Fatalf("the dark note has no warning-tinted panel edge")
	}
	savePageSnapshot(t, tt, "plugins-install-dark-1440.png")
}

// TestInstallFooterIsAbsentFromTheLogScreen keeps the footer to the spec screen:
// the installing, installed and failed screens carry their own actions, as the
// reference's wizard does (PluginManagerPage.tsx:930-974).
func TestInstallFooterIsAbsentFromTheLogScreen(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot().Value
	s.Install = manager.InstallState{Open: true, Phase: "failed", Spec: "fixture-plugin",
		Result: &manager.Change{Application: "failed"}}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, testPage(m, s))
	}, 1440, 920)

	safety, _ := installSafetyCopy()
	if tt.HasText(safety) {
		t.Fatalf("the failed screen kept the spec screen's footer note: %v", tt.Texts())
	}
	if !tt.HasText(copyText("重试安装", "Retry installation")) {
		t.Fatalf("the failed screen lost its own action: %v", tt.Texts())
	}
}
