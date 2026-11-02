package settings

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The appearance theme grid must stay inside the settings pane. A live capture
// showed a third card column clipped at the pane edge, so geometry is asserted,
// not just the presence of names.
func TestThemeGridStaysInsideThePane(t *testing.T) {
	editor, tt := themeGridFixture(t)
	families := editor.Draft.Families()
	if len(families) < 2 {
		t.Fatalf("fixture has %d families", len(families))
	}
	const width = 760
	const inset = 16.0
	for _, family := range families {
		light, ok := tt.Find(family.Name + " light")
		if !ok {
			t.Fatalf("missing light half for %q", family.Name)
		}
		dark, ok := tt.Find(family.Name + " dark")
		if !ok {
			t.Fatalf("missing dark half for %q", family.Name)
		}
		if light.X < inset || dark.X+dark.W > width-inset+0.5 {
			t.Fatalf("theme card %q escapes pane: light=%+v dark=%+v width=%d", family.Name, light, dark, width)
		}
		if light.W <= 0 || dark.W <= 0 {
			t.Fatalf("theme card %q has no width: light=%+v dark=%+v", family.Name, light, dark)
		}
	}
}

// The source uses auto-fill minmax(200px,1fr), not a fixed column count.
// These widths exercise the actual one-, two-, three- and four-column breaks.
func TestThemeGridUsesSourceTracksAndPreviewGeometry(t *testing.T) {
	old, locale := theme.CurrentPreferences(), i18n.Locale()
	t.Cleanup(func() { theme.SetPreferences(old); i18n.SetLocale(locale) })
	i18n.SetLocale("en")
	for _, tc := range []struct{ width, columns int }{{390, 1}, {560, 2}, {760, 3}, {1000, 4}} {
		t.Run(fmt.Sprint(tc.width), func(t *testing.T) {
			e := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
			tt := ui.NewTester(func(c *ui.Context) {
				colors := theme.ResolveWithSystem("system", c.Theme().Dark).Active(c.Theme().Dark)
				ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() { themeLibraryPane(c, e, colors, c.Theme().Text, c.Theme().TextMuted) })
			}, tc.width, 3000)
			find := func(label string) ui.Rect {
				t.Helper()
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %q", label)
				}
				return r
			}
			close := func(label string, got, want float32) {
				t.Helper()
				if math.Abs(float64(got-want)) > .05 {
					t.Fatalf("%s: got %.4f want %.4f", label, got, want)
				}
			}
			families := e.Draft.Families()
			first := find("Theme card " + families[0].Name)
			for index, f := range families {
				card := find("Theme card " + f.Name)
				light, dark := find(f.Name+" light"), find(f.Name+" dark")
				close("light height", light.H, 88)
				close("dark height", dark.H, 88)
				close("half widths", light.W, dark.W)
				close("adjacent halves", dark.X, light.X+light.W)
				close("source track width", card.W, (float32(tc.width)-32-float32(tc.columns-1)*14)/float32(tc.columns))
				if index < tc.columns {
					close("first row", card.Y, first.Y)
				}
				if index == tc.columns {
					close("grid row gap", card.Y, first.Y+first.H+14)
				}
				footer := find("Theme footer " + f.Name)
				close("footer height", footer.H, 54.5)
				close("footer starts below preview", footer.Y, light.Y+88)
				for _, scheme := range []string{"light", "dark"} {
					dot := find("Theme accent " + f.Name + " " + scheme)
					close("accent width", dot.W, 16)
					close("accent height", dot.H, 16)
				}
			}
			for _, label := range []string{"Create theme", "Import theme"} {
				close(label+" height", find(label).H, 36)
			}
		})
	}
}
