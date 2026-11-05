package pluginmanager

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
)

// pluginDialogFixture renders one plugin manager card at width x height after
// its opening frame, as the page shows it over the shared backdrop.
func pluginDialogFixture(t *testing.T, render func(c *ui.Context), width, height int) *ui.Tester {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { render(c) })
	}, width, height)
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
	return tt
}

// assertSharedCard checks the reference Modal metrics the plugin manager's
// cards share: a 560 DIP card on the 1440 viewport with 24 DIP of air, a 28 DIP
// close box 22 DIP below its top, and the close box inside the card's right
// edge at the reference's 14 DIP inset.
func assertSharedCard(t *testing.T, tt *ui.Tester, title string) {
	t.Helper()
	card, ok := tt.Find(title)
	if !ok {
		t.Fatalf("card %q not found: %v", title, tt.Texts())
	}
	if card.W < 559 || card.W > 561 {
		t.Fatalf("card is %.0f DIP wide, want the reference's 560", card.W)
	}
	if card.X < 439 || card.X > 441 {
		t.Fatalf("card starts at %.0f, want 440 (centered on 1440)", card.X)
	}
	if card.Y < 23 {
		t.Fatalf("card starts at %.0f, want at least the reference's 24 DIP inset", card.Y)
	}
	closeBox, ok := tt.Find(providerDialogCloseLabel())
	if !ok {
		t.Fatalf("card %q has no close box: %v", title, tt.Texts())
	}
	if closeBox.W < 27 || closeBox.W > 29 || closeBox.H < 27 || closeBox.H > 29 {
		t.Fatalf("close box is %.0fx%.0f, want the reference's 28x28", closeBox.W, closeBox.H)
	}
	if offset := closeBox.Y - card.Y; offset < 21 || offset > 23 {
		t.Fatalf("close box starts %.0f below the card, want the reference's 22", offset)
	}
	if right := card.X + card.W - (closeBox.X + closeBox.W); right < 13 || right > 15 {
		t.Fatalf("close box is %.0f from the card's right edge, want the reference's 14", right)
	}
}

func providerDialogCloseLabel() string { return copyText("关闭", "Close") }

// TestInstallDialogUsesTheSharedModalSurface pins the idle spec screen: the
// reference's 560 wide content-sized card with the shared fade, glass, radius
// and header, instead of the old 640 DIP fixed 720 DIP hand-painted panel.
func TestInstallDialogUsesTheSharedModalSurface(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "idle"}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	assertSharedCard(t, tt, copyText("添加插件", "Add plugin"))
	card, _ := tt.Find(copyText("添加插件", "Add plugin"))
	if card.H < 200 || card.H > 560 {
		t.Fatalf("idle card is %.0f DIP tall; the reference sizes it to its content", card.H)
	}
	for _, want := range []string{"包名或地址", "安装", "请确认插件来源可信", "暂不支持自动更新"} {
		if !tt.HasText(want) {
			t.Fatalf("idle card is missing %q: %v", want, tt.Texts())
		}
	}
	savePageSnapshot(t, tt, "plugins-install-idle-1440.png")
}

// TestInstallDialogSizesTheLogScreenToItsContent covers the other state: the
// card still grows with its content and stops at the window's inset, so a short
// log leaves no blank working height and a long one scrolls inside the card.
func TestInstallDialogSizesTheLogScreenToItsContent(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "failed", Spec: "fixture-plugin", DetailsOpen: true,
		Runs: []manager.InstallRun{{JobID: "pnpm-1", Command: "pnpm add fixture-plugin", CWD: "fixture-profile",
			Output: "Installing fixture\nFixture failure", Settled: true}}}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	card, ok := tt.Find(copyText("添加插件", "Add plugin"))
	if !ok {
		t.Fatalf("install card not found: %v", tt.Texts())
	}
	if card.H > 920-47 {
		t.Fatalf("running card is %.0f DIP tall, past the reference's 24 DIP inset", card.H)
	}
	if card.H >= 720 {
		t.Fatalf("running card keeps a fixed %.0f DIP height instead of following its log", card.H)
	}
	if !tt.HasText("pnpm add fixture-plugin") {
		t.Fatalf("running card lost the install log: %v", tt.Texts())
	}
	savePageSnapshot(t, tt, "plugins-install-failed-1440.png")
}

// TestSourcesDialogUsesTheSharedModalSurface covers the index sources card: the
// reference header now owns its close control, not a footer button.
func TestSourcesDialogUsesTheSharedModalSurface(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Discover.SourcesOpen = true
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		sourcesDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	assertSharedCard(t, tt, copyText("插件索引源", "Plugin index sources"))
	savePageSnapshot(t, tt, "plugins-sources-1440.png")
}

// TestMarketChoiceDialogUsesTheSharedModalSurface covers the market choice card:
// the reference header, a shared body, and the cancel/install footer row.
func TestMarketChoiceDialogUsesTheSharedModalSurface(t *testing.T) {
	m := pageModel(t)
	s := m.Controller.Snapshot()
	entry := manager.MarketEntry{Name: "fixture-discover", Title: manager.Text(`"Discover fixture"`),
		Description: manager.Text(`"A source-backed entry with exact install versions."`), Author: "Fixture author", DefaultVersion: "2.0.0"}
	entry.Installation.Status = "not-installed"
	version := manager.MarketVersion{Version: "2.0.0", Compatibility: manager.Compatibility{Compatible: true}}
	version.Install.Spec = "fixture-discover@2.0.0"
	entry.Versions = []manager.MarketVersion{version}
	entry.Title = manager.Text(`"Discover fixture"`)
	s.Discover.Choice = &manager.MarketChoice{Entry: entry, Version: version}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		marketChoiceDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	assertSharedCard(t, tt, copyText("确认安装插件", "Confirm plugin installation"))
	if !tt.HasText("Discover fixture · 2.0.0") {
		t.Fatalf("market choice card lost its heading: %v", tt.Texts())
	}
	if !tt.HasText("安装") || !tt.HasText("取消") {
		t.Fatalf("market choice card lost its actions: %v", tt.Texts())
	}
	savePageSnapshot(t, tt, "plugins-market-choice-1440.png")
}
