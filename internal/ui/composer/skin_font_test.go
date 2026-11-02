package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

func TestSkinDraftFourModesRetainTextOwnerAndUseCurrentRoleFont(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old) })
	p := theme.Defaults()
	p.Light, p.Dark = "codex", "claude"
	p.Composer.Family = "User Draft"
	theme.SetPreferences(p)
	d := NewDraftState("Serif Āé 鲸😀")
	client := d.Client
	for _, tc := range []struct {
		id   string
		dark bool
		skin string
	}{{theme.IDSystem, false, theme.SkinCodex}, {theme.IDSystem, true, theme.SkinClaude}, {theme.IDDeepseekLight, true, theme.SkinCodex}, {theme.IDDeepseekDark, false, theme.SkinClaude}} {
		tt := ui.NewTester(func(c *ui.Context) {
			resolved := theme.ResolveWithSystem(tc.id, tc.dark)
			next := *c.Theme()
			next.Dark = resolved.Scheme == theme.SchemeDark
			c.SetTheme(&next)
			View(c, Model{State: d, Draft: &d.Text, TextInput: client, UseTextInputClient: true, Hero: true, ThemeID: tc.id, SystemDark: tc.dark, WorkspaceRoot: "fixture"})
		}, 800, 260)
		if _, ok := tt.Find("Composer draft"); !ok {
			t.Fatal("draft missing", tc)
		}
		want := theme.FontForSkin(theme.RenderPreferences(), tc.skin, theme.SkinComposerFont)
		if client.font.Family != want.Family {
			t.Fatal("draft shape font mismatched current half", tc, client.font.Family, want.Family)
		}
		client.Sync()
		if client.font.Family != want.Family {
			t.Fatal("programmatic sync discarded frame font", tc)
		}
		if d.Client != client || d.Text != "Serif Āé 鲸😀" {
			t.Fatal("font transition replaced text owner or text")
		}
		client.SetSelectionUTF16(0, 1)
		if r, _, ok := client.BoundsForRange(ui.TextInputRange{Start: 0, End: 1}); !ok || r.W <= 0 {
			t.Fatal("caret range has no shaped bounds", tc, r, ok)
		}
	}
}
