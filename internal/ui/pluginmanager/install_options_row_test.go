package pluginmanager

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
)

// idleInstallCard renders the install card's spec screen with the npm official
// registry chosen, so the chip's copy is the reference's.
func idleInstallCard(t *testing.T) *ui.Tester {
	t.Helper()
	official := manager.OfficialNPMRegistry
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "idle",
		RegistryInfo:      &manager.RegistryInfo{Registry: &official, Resolved: &official},
		RegistrySelection: manager.RegistryChoice{Kind: "offered", Registry: &official}}
	return pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)
}

// TestInstallOptionsRowMatchesTheReference pins the reference's `.optionsRow`
// (PluginManagerPage.module.css:714): the guide's text toggle at the card's left
// inset and the 28 DIP registry chip at its right, on one line.
func TestInstallOptionsRowMatchesTheReference(t *testing.T) {
	tt := idleInstallCard(t)
	card, ok := tt.Find(copyText("添加插件", "Add plugin"))
	if !ok {
		t.Fatalf("install card not found: %v", tt.Texts())
	}

	guide, ok := tt.Find(guideToggleLabel(false))
	if !ok {
		t.Fatalf("guide toggle is missing: %v", tt.Texts())
	}
	if guide.H > 20 {
		t.Fatalf("guide toggle is %.0f DIP tall, want the reference's 12.5 DIP text line", guide.H)
	}
	if inset := guide.X - card.X; math.Abs(float64(inset-24)) > 1 {
		t.Fatalf("guide toggle starts %.0f DIP into the card, want the reference's 24", inset)
	}
	if guide.W > card.W/2 {
		t.Fatalf("guide toggle is %.0f DIP wide, want an inline control, not a full-width button", guide.W)
	}

	chip, ok := tt.Find(copyText("安装源", "Registry") + " " + copyText("npm npm 官方源", "npm official registry"))
	if !ok {
		t.Fatalf("registry chip is missing its chosen name: %v", tt.Texts())
	}
	if math.Abs(float64(chip.H-28)) > 1 {
		t.Fatalf("registry chip is %.0f DIP tall, want the reference's 28", chip.H)
	}
	if right := card.X + card.W - (chip.X + chip.W); math.Abs(float64(right-24)) > 1 {
		t.Fatalf("registry chip ends %.0f DIP before the card's right edge, want the reference's 24", right)
	}
	if math.Abs(float64(chip.Y+chip.H/2-(guide.Y+guide.H/2))) > 1.5 {
		t.Fatalf("the options are on two lines: guide %.1f, chip %.1f", guide.Y+guide.H/2, chip.Y+chip.H/2)
	}
}

// TestInstallGuideToggleUnfoldsWithTheReferenceCopy opens the guide: the toggle
// takes the reference's second name and the example unfolds below the row.
func TestInstallGuideToggleUnfoldsWithTheReferenceCopy(t *testing.T) {
	tt := idleInstallCard(t)
	if _, ok := tt.Find(guideToggleLabel(false)); !ok {
		t.Fatalf("guide toggle is missing: %v", tt.Texts())
	}
	if err := tt.Click(guideToggleLabel(false)); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		tt.Frame()
	}
	if !tt.HasText(guideToggleLabel(true)) {
		t.Fatalf("the open guide kept the closed label: %v", tt.Texts())
	}
	if !tt.HasText("dsh-plugin-whale-pet") {
		t.Fatalf("opening the guide did not unfold its example: %v", tt.Texts())
	}
}

// TestInstallChevronAngleEasesOverTheReferenceDuration covers the reference's
// `transition: transform 160ms ease` on the disclosure chevron: a state change
// turns the arrow over instead of swapping glyphs.
func TestInstallChevronAngleEasesOverTheReferenceDuration(t *testing.T) {
	start := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	open := false
	var angle float32
	tt := ui.NewTester(func(c *ui.Context) {
		box := ui.Box(c).Size(16, 16)
		angle = installChevronAngle(c, box, open)
	}, 200, 200)

	tt.SetTime(start)
	tt.Frame()
	if angle != 0 {
		t.Fatalf("the closed chevron is at %.0f degrees, want 0", angle)
	}
	// The frame that notices the change starts the turn at the current time.
	open = true
	tt.SetTime(start.Add(10 * time.Millisecond))
	tt.Frame()
	tt.SetTime(start.Add(90 * time.Millisecond))
	tt.Frame()
	if angle <= 0 || angle >= 180 {
		t.Fatalf("80ms into the 160ms turn the chevron is at %.0f degrees, want it between 0 and 180", angle)
	}
	tt.SetTime(start.Add(installChevronSettle + 10*time.Millisecond))
	tt.Frame()
	if angle != 180 {
		t.Fatalf("the chevron settled at %.0f degrees, want 180 after the reference's 160ms", angle)
	}
	open = false
	tt.SetTime(start.Add(installChevronSettle + 20*time.Millisecond))
	tt.Frame()
	tt.SetTime(start.Add(installChevronSettle + 100*time.Millisecond))
	tt.Frame()
	if angle >= 180 || angle <= 0 {
		t.Fatalf("80ms into the return the chevron is at %.0f degrees, want it between 0 and 180", angle)
	}
	tt.SetTime(start.Add(3 * installChevronSettle))
	tt.Frame()
	if angle != 0 {
		t.Fatalf("the chevron returned to %.0f degrees, want 0", angle)
	}
}

// TestInstallRegistryChipUnfoldsItsChoices covers the chip's expanded state: the
// choices float from it as the reference's `.registry` panel does.
func TestInstallRegistryChipUnfoldsItsChoices(t *testing.T) {
	official := manager.OfficialNPMRegistry
	m := pageModel(t)
	s := m.Controller.Snapshot()
	s.Install = manager.InstallState{Open: true, Phase: "idle", RegistryOpen: true,
		RegistryInfo:      &manager.RegistryInfo{Registry: &official, Resolved: &official},
		RegistrySelection: manager.RegistryChoice{Kind: "offered", Registry: &official}}
	tt := pluginDialogFixture(t, func(c *ui.Context) {
		installDialog(c, m, s, m.Controller.Actions(s.Owner))
	}, 1440, 920)

	if _, ok := tt.Find(copyText("安装源", "Registry") + " " + copyText("npm npm 官方源", "npm official registry")); !ok {
		t.Fatalf("registry chip is missing: %v", tt.Texts())
	}
	for _, want := range []string{
		copyText("自定义地址", "Custom address"),
		copyText("自定义安装源 URL", "Custom registry URL"),
	} {
		if !tt.HasText(want) {
			t.Fatalf("unfolded registry choices are missing %q: %v", want, tt.Texts())
		}
	}
}
