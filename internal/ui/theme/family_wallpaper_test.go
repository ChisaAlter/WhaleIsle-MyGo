package theme

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestThemeFamilyWallpaperRoundTrip(t *testing.T) {
	p := Defaults()
	f := p.DuplicateFamily(Library()[0])
	f.Wallpaper = "data:image/png;base64,aW1hZ2U="
	p.Custom = append(p.Custom, f)
	raw := SerializeFamily(f)
	imported, err := Defaults().ImportFamily(raw)
	if err != nil || imported.Wallpaper != f.Wallpaper {
		t.Fatalf("family wallpaper lost %v", err)
	}
	q := p.Clone()
	q.Custom[0].Wallpaper = ""
	if preferencesEqual(p, q) {
		t.Fatal("wallpaper omitted from theme revision")
	}
	if _, err := p.ImportFamily(strings.Replace(raw, f.Wallpaper, "https://example.invalid/image.png", 1)); err == nil {
		t.Fatal("accepted external embedded wallpaper")
	}
	h, err := ReadHostSettings(json.RawMessage(`{"skinWallpaperEnabled":false}`))
	if err != nil || h.Background().SkinWallpaperEnabled {
		t.Fatal("Host theme wallpaper toggle lost")
	}
}
