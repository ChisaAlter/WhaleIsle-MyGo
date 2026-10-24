package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// These presets reproduce the independent installed DOM inputs at its420/210px
// row widths. They verify line boxes and gallery frames, not font raster equality.
func TestToolImageGeometryInstalledPresets(t *testing.T) {
	toolImageViewLoader(t)
	toolImageViewSetLoad(t, "geometry-single", toolImageViewBitmap(t, 640, 360), nil)
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	name := "scene.png"
	extra := "  leading spaces  \n\tTabbed line\n\ntrailing  "
	base := toolImageAttachment{AttachmentID: "geometry-single", MediaType: "image/png", Width: 640, Height: 360, Bytes: 12345, Name: &name}
	all := map[string]map[string]ui.Rect{}
	for _, tc := range []struct {
		name, locale string
		width        int
		data         toolImageModel
		wantHeight   float32
	}{
		{"single-en", i18n.LocaleEN, 420, toolImageModel{Path: `C:\fixture\scene.png`, HasPath: true, Images: []toolImageAttachment{base}, ExtraText: &extra}, 263},
		{"no-extra", i18n.LocaleEN, 420, toolImageModel{Path: `C:\fixture\scene.png`, HasPath: true, Images: []toolImageAttachment{base}}, 179},
		{"multi5", i18n.LocaleEN, 210, toolImageModel{Images: []toolImageAttachment{base, base, base, base, base}}, 304},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			var body, following ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				theme.ApplyInterface(c)
				palette := theme.Resolve(theme.DefaultID).Active(false)
				st := toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border))
				ui.Column(c).Width(float32(tc.width)).Children(func() {
					ui.Column(c).FillWidth().ObserveLayout(func(r ui.Rect) { body = r }).Children(func() { toolImageCard(c, &tc.data, st, "geometry", nil) })
					ui.Box(c).Height(1).ObserveLayout(func(r ui.Rect) { following = r })
				})
			}, tc.width, 800)
			tt.SetTime(time.Unix(12200, 0))
			tt.Frame()
			if body.H != tc.wantHeight || body.W != float32(tc.width) || following.Y != body.H {
				t.Fatalf("preset %s body%v following%v expectedH%v", tc.name, body, following, tc.wantHeight)
			}
			boxes := map[string]ui.Rect{"body": body, "following": following}
			for _, label := range []string{"scene.png, click to view original", `C:\fixture\scene.png`, "640x360px, 12345 bytes", extra} {
				if r, ok := tt.Find(label); ok {
					boxes[label] = r
				}
			}
			all[tc.name] = boxes
		})
	}
	if dir := os.Getenv("TOOL_IMAGE_EVIDENCE_DIR"); dir != "" {
		raw, _ := json.MarshalIndent(all, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "tool-image-pass266-native-preset-geometry.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
