package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

func testButtonTester(t *testing.T, colors theme.Colors, spec ButtonSpec) *ui.Tester {
	t.Helper()
	label := "Save"
	return ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Padding(8).AlignItems(ui.Start).Children(func() {
			Button(c, colors, label, spec)
		})
	}, 240, 80)
}

// The harness button is 36 DIP tall with 14px text at md and 28 DIP with 12px
// at sm (ui-primitives Button.module.css), not MyGo's compact theme face.
func TestButtonUsesHarnessMetrics(t *testing.T) {
	colors := theme.Resolve(theme.IDDeepseekLight).Light
	md := testButtonTester(t, colors, ButtonSpec{Variant: ButtonPrimary})
	rect, ok := md.Find("Save")
	if !ok {
		t.Fatal("md button missing")
	}
	if rect.H != 36 {
		t.Fatalf("md height = %v, want 36", rect.H)
	}
	sm := testButtonTester(t, colors, ButtonSpec{Variant: ButtonOutline, Size: ButtonSmall})
	rect, ok = sm.Find("Save")
	if !ok {
		t.Fatal("sm button missing")
	}
	if rect.H != 28 {
		t.Fatalf("sm height = %v, want 28", rect.H)
	}
}

// The primary face is the monochrome brand fill, not the blue info fill; the
// rounded corner leaves the page background visible.
func TestPrimaryButtonIsTheBrandMonochromeFill(t *testing.T) {
	colors := theme.Resolve(theme.IDDeepseekLight).Light
	tt := testButtonTester(t, colors, ButtonSpec{Variant: ButtonPrimary})
	rect, ok := tt.Find("Save")
	if !ok {
		t.Fatal("button missing")
	}
	img := tt.Image()
	fill := at(img, int(rect.X+rect.W/2), int(rect.Y+6))
	if want := rgb(theme.Color(colors.BrandPrimary)); !near(fill, want, 3) {
		t.Fatalf("primary fill = %v, want %v", fill, want)
	}
	if info := rgb(theme.Color(colors.ButtonInfoFill)); near(fill, info, 6) {
		t.Fatalf("primary button still uses the info blue: %v", fill)
	}
	corner := at(img, int(rect.X), int(rect.Y))
	if near(corner, fill, 6) {
		t.Fatalf("corner = %v, want the page behind the 12 DIP radius", corner)
	}
}

func TestDisabledButtonDimsToFortyPercent(t *testing.T) {
	colors := theme.Resolve(theme.IDDeepseekLight).Light
	tt := testButtonTester(t, colors, ButtonSpec{Variant: ButtonPrimary, Disabled: true})
	rect, ok := tt.Find("Save")
	if !ok {
		t.Fatal("button missing")
	}
	got := at(tt.Image(), int(rect.X+rect.W/2), int(rect.Y+6))
	full := rgb(theme.Color(colors.BrandPrimary))
	if near(got, full, 2) {
		t.Fatalf("disabled face is not dimmed: %v", got)
	}
	// 0.4 of near-black over white lands well above the full fill.
	if got.R < 100 {
		t.Fatalf("disabled face too dark: %v", got)
	}
}

// A ghost button paints nothing of its own: its face is the page.
func TestGhostButtonHasNoFace(t *testing.T) {
	colors := theme.Resolve(theme.IDDeepseekLight).Light
	tt := testButtonTester(t, colors, ButtonSpec{Variant: ButtonGhost})
	rect, ok := tt.Find("Save")
	if !ok {
		t.Fatal("button missing")
	}
	got := at(tt.Image(), int(rect.X+2), int(rect.Y+rect.H/2))
	want := rgb(theme.Color(colors.BG))
	if !near(got, want, 2) {
		t.Fatalf("ghost face = %v, want the page %v", got, want)
	}
}
// Nothing outside the label paints for a ghost button: no frame, no fill.
func TestGhostButtonDrawsOnlyItsLabel(t *testing.T) {
	colors := theme.Resolve(theme.IDDeepseekLight).Light
	tt := testButtonTester(t, colors, ButtonSpec{Variant: ButtonGhost})
	rect, ok := tt.Find("Save")
	if !ok {
		t.Fatal("button missing")
	}
	img := tt.Image()
	page := rgb(theme.Color(colors.BG))
	for x := int(rect.X); x <= int(rect.X+rect.W); x++ {
		for _, y := range []int{int(rect.Y), int(rect.Y + rect.H - 1), int(rect.Y + rect.H/2)} {
			if got := at(img, x, y); !near(got, page, 2) {
				// The label glyphs sit inside the rect; edges must stay page.
				if x == int(rect.X) || x == int(rect.X+rect.W) || y == int(rect.Y) || y == int(rect.Y+rect.H-1) {
					t.Fatalf("ghost edge %d,%d = %v, want page %v", x, y, got, page)
				}
			}
		}
	}
}