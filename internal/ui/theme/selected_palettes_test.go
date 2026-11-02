package theme

import "testing"

func TestResolveUsesPalettesDerivedOnSetPreferences(t *testing.T) {
	original := CurrentPreferences()
	defer SetPreferences(original)
	p := Defaults()
	p.Light, p.Dark = "midnight", "celadon"
	SetPreferences(p)
	current := CurrentPreferences()
	got := Resolve("")
	if got.Light != palette(current.Family(current.Light).Light, false) || got.Dark != palette(current.Family(current.Dark).Dark, true) {
		t.Fatal("Resolve differs from the palettes the chosen families derive")
	}
	SetPreferences(Defaults())
	if got := Resolve(""); got.Light != deepseekLight || got.Dark != deepseekDark {
		t.Fatal("default families kept a previous selection")
	}
}
