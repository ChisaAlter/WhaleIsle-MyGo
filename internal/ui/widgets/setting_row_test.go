package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// settingRows builds two rows in a window of the given width, the second
// ending the group.
func settingRows(t *testing.T, width, height int, on *bool) (*ui.Tester, *ui.Theme) {
	t.Helper()
	return newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			SettingRow(c, SettingRowOptions{
				Title:       "Theme",
				Description: "Follows the system",
				Control:     func() { Switch(c, on).Label("Theme switch") },
			})
			SettingRow(c, SettingRowOptions{
				Title:       "Language",
				Description: "The interface language",
				Control:     func() { Switch(c, on).Label("Language switch") },
				Last:        true,
			})
		})
	}, width, height)
}

func TestSettingRowLaysOutTitleAndDescription(t *testing.T) {
	on := false
	tt, _ := settingRows(t, 800, 300, &on)
	title, ok := tt.Find("Theme")
	if !ok {
		t.Fatal("the title is missing")
	}
	if title.Y != 16 || title.H != 22 {
		t.Fatalf("the title is at y=%v, %v high; want y=16, 22 high", title.Y, title.H)
	}
	desc, ok := tt.Find("Follows the system")
	if !ok {
		t.Fatal("the description is missing")
	}
	if desc.Y != 42 || desc.H != 18 {
		t.Fatalf("the description is at y=%v, %v high; want y=42, 18 high", desc.Y, desc.H)
	}
	if desc.X != 0 {
		t.Fatalf("the description starts at x=%v", desc.X)
	}
}

func TestSettingRowPutsTheControlOnTheRight(t *testing.T) {
	on := false
	tt, _ := settingRows(t, 800, 300, &on)
	control, ok := tt.Find("Theme switch")
	if !ok {
		t.Fatal("the control is missing")
	}
	if control.X+control.W > 800 || control.X < 500 {
		t.Fatalf("the control sits at x=%v, want the right column", control.X)
	}
	title, _ := tt.Find("Theme")
	if control.Y+control.H/2 < title.Y-4 || control.Y > title.Y+44 {
		t.Fatalf("the control sits at y=%v, beside the row's text at %v", control.Y, title.Y)
	}
}

func TestSettingRowHairlineEndsAtTheLastRow(t *testing.T) {
	on := false
	tt, theme := settingRows(t, 800, 300, &on)
	line := 16 + 22 + 4 + 18 + 16
	if got := at(tt.Image(), 400, line); near(got, rgb(theme.Background), 2) {
		t.Fatalf("no hairline under the row: %v", got)
	}
	// The second row is the last of the group, so nothing follows it.
	title, _ := tt.Find("Language")
	bottom := int(title.Y) + 16 + 22 + 4 + 18 + 16
	if got := at(tt.Image(), 400, bottom); !near(got, rgb(theme.Background), 2) {
		t.Fatalf("the last row still draws a hairline: %v", got)
	}
}

func TestSettingRowStacksWhenNarrow(t *testing.T) {
	on := false
	tt, _ := settingRows(t, 360, 300, &on)
	desc, ok := tt.Find("Follows the system")
	if !ok {
		t.Fatal("the description is missing")
	}
	control, ok := tt.Find("Theme switch")
	if !ok {
		t.Fatal("the control is missing")
	}
	if control.Y <= desc.Y+desc.H {
		t.Fatalf("the control at y=%v did not move under the text ending at %v", control.Y, desc.Y+desc.H)
	}
	if control.X != 0 {
		t.Fatalf("the stacked control starts at x=%v", control.X)
	}
}

func TestSettingRowControlIsOneHundredEightyWide(t *testing.T) {
	on := false
	var controlWidth float32
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			SettingRow(c, SettingRowOptions{
				Title:       "Theme",
				Description: "Follows the system",
				Control: func() {
					control := ui.Box(c).FillWidth()
					control.Children(func() { Switch(c, &on).Label("Theme switch") })
					controlWidth = control.Bounds().W
				},
			})
		})
	}, 800, 200)
	tt.Frame()
	if controlWidth != settingRowControl {
		t.Fatalf("the control column is %v wide, want %v", controlWidth, settingRowControl)
	}
}
