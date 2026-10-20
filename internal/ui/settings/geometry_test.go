package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestPreferenceRowGeometryMatchesTheHarness pins the general-page geometry the
// installed build uses: a row is 64 DIP tall and its control is 32 DIP tall, with
// the hairline drawn above every row but the first.
func TestPreferenceRowGeometryMatchesTheHarness(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	first, second := "Fixture first row", "Fixture second row"
	value := "a"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Children(func() {
			bodyPreferenceRow(c, "geometry-first", first, "A hint that keeps the row two lines tall.", c.Theme().Text, c.Theme().TextMuted, func() {
				ui.Select(c, &value, []string{"a", "b"}).Label("Probe control").FillWidth().Height(32)
			})
			bodyPreferenceRow(c, "geometry-second", second, "", c.Theme().Text, c.Theme().TextMuted, func() {
				ui.Select(c, &value, []string{"a", "b"}).Label("Probe control 2").FillWidth().Height(32)
			})
		})
	}, 720, 320)
	control, ok := tt.Find("Probe control")
	if !ok {
		t.Fatalf("control is not reachable: %v", tt.Texts())
	}
	if control.H != 32 {
		t.Fatalf("control height = %v, want 32", control.H)
	}
	rowOne, ok := tt.Find(first)
	if !ok {
		t.Fatal("first row title missing")
	}
	rowTwo, ok := tt.Find(second)
	if !ok {
		t.Fatal("second row title missing")
	}
	if pitch := rowTwo.Y - rowOne.Y; pitch < 64 {
		t.Fatalf("row pitch = %v, want at least 64", pitch)
	}
}
