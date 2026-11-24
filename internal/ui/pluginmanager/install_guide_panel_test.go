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
	// The panel starts at the body's 24 inset and pads 16: the copy starts at 40.
	if inset := titleBox.X - card.X; math.Abs(float64(inset-40)) > 2 {
		t.Fatalf("the guide's title starts %.1f DIP into the card, want the body's 24 plus the panel's 16", inset)
	}
	if math.Abs(float64(titleBox.H-20)) > 1 {
		t.Fatalf("the guide's title is %.1f DIP tall, want the reference's 13/20 line", titleBox.H)
	}

	hintBox, ok := tt.Find(hint)
	if !ok {
		t.Fatalf("the guide lost its hint: %v", tt.Texts())
	}
	if gap := hintBox.Y - (titleBox.Y + titleBox.H); math.Abs(float64(gap-4)) > 1.5 {
		t.Fatalf("the hint starts %.1f DIP under the title, want the panel column's 4 DIP gap", gap)
	}

	label := copyText("示例：", "Example: ")
	labelBox, ok := tt.Find(label)
	if !ok {
		t.Fatalf("the guide lost its example label: %v", tt.Texts())
	}
	// The spec input's placeholder is the same package name: the guide's copy
	// is the match on the example row, right of its label.
	var exampleBox ui.Rect
	for _, r := range tt.FindAll(example) {
		if r.Y >= labelBox.Y-2 && r.X > labelBox.X {
			exampleBox = r
		}
	}
	if exampleBox.W == 0 {
		t.Fatalf("the guide lost the example: %v", tt.Texts())
	}
	if gap := labelBox.Y - (hintBox.Y + hintBox.H); math.Abs(float64(gap-4)) > 1.5 {
		t.Fatalf("the example line starts %.1f DIP under the hint, want the panel column's 4 DIP gap", gap)
	}
	if gap := exampleBox.X - (labelBox.X + labelBox.W); math.Abs(float64(gap-4)) > 2 {
		t.Fatalf("the package name starts %.1f DIP after the label, want it inline", gap)
	}

	// The small outline button ends at the panel's 16 DIP right padding, on the
	// row's centreline.
	fill := copyText("填入示例", "Fill the example") + " " + example
	button, ok := tt.Find(fill)
	if !ok {
		t.Fatalf("the guide lost its fill button: %v", tt.Texts())
	}
	if math.Abs(float64(button.H-28)) > 1 {
		t.Fatalf("the fill button is %.1f DIP tall, want the reference's small 28", button.H)
	}
	if right := card.X + card.W - 24 - 16 - (button.X + button.W); math.Abs(float64(right)) > 1.5 {
		t.Fatalf("the fill button ends %.1f DIP before the panel's right padding, want the panel's 16 inset", right)
	}
	centres := (titleBox.Y + exampleBox.Y + exampleBox.H) / 2
	if math.Abs(float64(button.Y+button.H/2-centres)) > 1.5 {
		t.Fatalf("the fill button is centred at %.1f, the row's copy at %.1f", button.Y+button.H/2, centres)
	}

	// The panel paints the shared muted wash (--muted, black at 4% over the
	// card's white) behind the copy, not the card's own white. Sample at the
	// copy block's centreline, clear of the XL corner arcs.
	img := tt.Image()
	panel := img.At(int(card.X+24+8), int(centres))
	if !closeTo(panel, color.RGBA{245, 245, 245, 255}) {
		t.Fatalf("the guide panel paints %v, want the muted wash rgb(245, 245, 245)", panel)
	}
	outside := img.At(int(card.X+8), int(centres))
	if closeTo(outside, color.RGBA{245, 245, 245, 255}) {
		t.Fatalf("the card paints %v behind the panel: the panel has no fill of its own", outside)
	}
	savePageSnapshot(t, tt, "plugins-install-guide-1440.png")
}
