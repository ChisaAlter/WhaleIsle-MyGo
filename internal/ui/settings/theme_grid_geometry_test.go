package settings

import (
	"testing"
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
