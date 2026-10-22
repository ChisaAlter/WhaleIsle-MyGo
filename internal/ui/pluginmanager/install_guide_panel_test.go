package pluginmanager

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
)

// guideSpecCard renders the install card's spec screen with the guide unfolded.
func guideSpecCard(t *testing.T, height int) *ui.Tester {
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

// TestInstallGuidePanelMatchesTheReference pins the reference's `.guide`
// (PluginManagerPage.module.css:619-711, PluginManagerPage.tsx:1032-1059): a
// hairline rounded panel holding one ordered example row — the title, the hint,
// an "example" line with the package name — beside the button that fills it in.
func TestInstallGuidePanelMatchesTheReference(t *testing.T) {
	tt := guideSpecCard(t, 920)
	card, ok := tt.Find(copyText("添加插件", "Add plugin"))
	if !ok {
		t.Fatalf("install card not found: %v", tt.Texts())
	}
	title, hint, example := installGuideExample()

	titleBox, ok := tt.Find(title)
	if !ok {
		t.Fatalf("the guide lost its title: %v", tt.Texts())
	}
	// The panel starts at the body's 24 inset and pads 14: the copy starts at 38.
	if inset := titleBox.X - card.X; math.Abs(float64(inset-38)) > 1 {
		t.Fatalf("the guide's title starts %.1f DIP into the card, want the body's 24 plus the panel's 14", inset)
	}
	if math.Abs(float64(titleBox.H-20)) > 1 {
		t.Fatalf("the guide's title is %.1f DIP tall, want the reference's 13/20 line", titleBox.H)
	}

	hintBox, ok := tt.Find(hint)
	if !ok {
		t.Fatalf("the guide lost its hint: %v", tt.Texts())
	}
	if gap := hintBox.Y - (titleBox.Y + titleBox.H); math.Abs(float64(gap-6)) > 1.5 {
		t.Fatalf("the hint starts %.1f DIP under the title, want `.guideHint`'s 6", gap)
	}

	label := copyText("示例：", "Example: ")
	labelBox, ok := tt.Find(label)
	if !ok {
		t.Fatalf("the guide lost its example label: %v", tt.Texts())
	}
	exampleBox, ok := tt.Find(example)
	if !ok {
		t.Fatalf("the guide lost the example: %v", tt.Texts())
	}
	if gap := labelBox.Y - (hintBox.Y + hintBox.H); math.Abs(float64(gap)) > 1.5 {
		t.Fatalf("the example line starts %.1f DIP under the hint, want `.guideMain`'s 0 gap", gap)
	}
	if gap := exampleBox.X - (labelBox.X + labelBox.W); math.Abs(float64(gap)) > 2 {
		t.Fatalf("the package name starts %.1f DIP after the label, want it inline", gap)
	}

	// `.guideItem > button { align-self: center }`: the 28 DIP outline button ends
	// at the panel's 14 DIP right padding, on the row's centreline.
	fill := copyText("填入示例", "Fill the example") + " " + example
	button, ok := tt.Find(fill)
	if !ok {
		t.Fatalf("the guide lost its fill button: %v", tt.Texts())
	}
	if math.Abs(float64(button.H-28)) > 1 {
		t.Fatalf("the fill button is %.1f DIP tall, want the reference's small 28", button.H)
	}
	if right := card.X + card.W - 24 - 14 - (button.X + button.W); math.Abs(float64(right)) > 1.5 {
		t.Fatalf("the fill button ends %.1f DIP before the panel's right padding, want the panel's 14 inset", right)
	}
	centres := (titleBox.Y + exampleBox.Y + exampleBox.H) / 2
	if math.Abs(float64(button.Y+button.H/2-centres)) > 1.5 {
		t.Fatalf("the fill button is centred at %.1f, the row's copy at %.1f", button.Y+button.H/2, centres)
	}

	// The panel paints the raised layer (--dsw-alias-bg-layer-1, the shell's
	// #f5f6f7 light well) behind the copy, not the card's own white.
	img := tt.Image()
	panel := img.At(int(card.X+24+2), int(titleBox.Y-4))
	if !closeTo(panel, color.RGBA{245, 246, 247, 255}) {
		t.Fatalf("the guide panel paints %v, want the raised layer rgb(245, 246, 247)", panel)
	}
	outside := img.At(int(card.X+8), int(titleBox.Y-4))
	if closeTo(outside, color.RGBA{245, 246, 247, 255}) {
		t.Fatalf("the card paints %v behind the panel: the panel has no fill of its own", outside)
	}
	savePageSnapshot(t, tt, "plugins-install-guide-1440.png")
}
