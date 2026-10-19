package theme_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/theme"
)

func TestRenderPreferencesFallbackRolesAndIsolation(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	theme.SetPreferences(theme.Defaults())
	if !reflect.DeepEqual(theme.RenderPreferences(), theme.Defaults()) {
		t.Fatal("default render stack changed or gained duplicate families")
	}
	raw := theme.Defaults()
	raw.Interface.Family = "  Arial, 'Microsoft YaHei'  "
	raw.Chat.Family = "Chat Face"
	raw.Composer.Family = "Composer Face"
	raw.Code.Family = "Code Face"
	raw.Terminal.Family = "Terminal Face"
	raw.Custom = []theme.Family{{ID: "owned", Name: "Owned"}}
	theme.SetPreferences(raw)
	for range 8 {
		got := theme.RenderPreferences()
		for _, pair := range []struct{ got, raw, suffix string }{
			{got.Interface.Family, raw.Interface.Family, theme.DefaultTextFamily},
			{got.Chat.Family, raw.Chat.Family, theme.DefaultTextFamily},
			{got.Composer.Family, raw.Composer.Family, theme.DefaultTextFamily},
			{got.Code.Family, raw.Code.Family, theme.DefaultCodeFamily},
		} {
			if pair.got != strings.TrimSpace(pair.raw)+", "+pair.suffix {
				t.Errorf("render family=%q", pair.got)
			}
		}
		if got.Terminal != raw.Terminal || !reflect.DeepEqual(theme.CurrentPreferences(), raw) {
			t.Fatal("drawing changed raw preferences or the independent terminal stack")
		}
		got.Custom[0].Name = "mutated snapshot"
		if theme.RenderPreferences().Custom[0].Name != "Owned" {
			t.Fatal("render snapshot shares mutable theme data")
		}
	}
	for _, family := range []string{"", theme.DefaultTextFamily, theme.TextFamily("Arial")} {
		if theme.TextFamily(theme.TextFamily(family)) != theme.TextFamily(family) {
			t.Fatal("text rendering grows the fallback stack")
		}
	}
	for _, family := range []string{"", theme.DefaultCodeFamily, theme.CodeFamily("Consolas")} {
		if theme.CodeFamily(theme.CodeFamily(family)) != theme.CodeFamily(family) {
			t.Fatal("code rendering grows the fallback stack")
		}
	}
	ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		if c.Theme().Font != theme.TextFamily(raw.Interface.Family) {
			t.Error("native control root bypassed rendered text fallbacks")
		}
	}, 100, 60)
}

func TestRenderPreferencesRawPersistenceAndPreviewCancel(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	dir := t.TempDir()
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw := theme.Defaults()
	raw.Interface.Family = "Arial, 'Microsoft YaHei'"
	raw.Code.Family = "Consolas"
	e := theme.NewEditor(raw, s.SaveAppearance)
	e.Save()
	if e.Preview || !reflect.DeepEqual(e.Saved, raw) {
		t.Fatal("save changed raw font families")
	}
	baseline := theme.RenderPreferences()
	for range 4 {
		e.Draft.Interface.Family = "Preview Face"
		e.ApplyPreview()
		if !e.Preview || theme.RenderPreferences().Interface.Family != theme.TextFamily("Preview Face") {
			t.Fatal("preview did not apply render stack")
		}
		e.Cancel()
		if !reflect.DeepEqual(e.Draft, raw) || !reflect.DeepEqual(e.Saved, raw) || !reflect.DeepEqual(theme.CurrentPreferences(), raw) || !reflect.DeepEqual(theme.RenderPreferences(), baseline) {
			t.Fatal("cancel grew or persisted a render stack")
		}
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Appearance()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, raw) {
		t.Fatalf("saved raw preferences changed: %+v", got)
	}
}
