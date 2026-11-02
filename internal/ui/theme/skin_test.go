package theme

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSkinFollowsSelectedHalfAndPinnedScheme(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	p := Defaults()
	p.Light, p.Dark = "codex", "claude"
	SetPreferences(p)
	for _, tc := range []struct {
		id   string
		dark bool
		want string
	}{{IDSystem, false, SkinCodex}, {IDSystem, true, SkinClaude}, {IDDeepseekLight, true, SkinCodex}, {IDDeepseekDark, false, SkinClaude}} {
		if got := ResolvedSkin(tc.id, tc.dark); got != tc.want {
			t.Errorf("skin %s systemDark=%v: %s want %s", tc.id, tc.dark, got, tc.want)
		}
	}
	if !reflect.DeepEqual(CurrentPreferences(), p) {
		t.Fatal("resolving a skin changed raw preferences")
	}
}

func TestSkinRoleFontsPreserveRawDefaultsAndCode(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	raw := Defaults()
	raw.Interface.Family, raw.Chat.Family, raw.Composer.Family = "User Sans", "User Chat", "User Draft"
	SetPreferences(raw)
	p := RenderPreferences()
	for _, skin := range []string{SkinDefault, SkinCodex} {
		if got := FontForSkin(p, skin, SkinInterfaceFont); got != p.Interface {
			t.Fatal(skin, got)
		}
		if got := FontForSkin(p, skin, SkinProseFont); got != p.Chat {
			t.Fatal(skin, got)
		}
		if got := FontForSkin(p, skin, SkinComposerFont); got != p.Composer {
			t.Fatal(skin, got)
		}
	}
	if err := RegisterSkinFonts(); err != nil {
		t.Fatal(err)
	}
	if got := FontForSkin(p, SkinClaude, SkinInterfaceFont); !strings.HasPrefix(got.Family, `"Inter", User Sans, `) || got.Size != p.Interface.Size {
		t.Fatal("Claude interface does not prepend Inter", got)
	}
	for _, role := range []SkinFontRole{SkinProseFont, SkinComposerFont, SkinHeroFont} {
		if got := FontForSkin(p, SkinClaude, role); got.Family != ClaudeProseFamily {
			t.Fatal(role, got)
		}
	}
	if !reflect.DeepEqual(CurrentPreferences(), raw) || !reflect.DeepEqual(RenderPreferences(), p) {
		t.Fatal("drawing a skin contaminated editable font preferences")
	}
}

func TestSkinContextFontUsesCurrentFrameMode(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	p := Defaults()
	p.Light, p.Dark = "deepseek", "claude"
	SetPreferences(p)
	for _, dark := range []bool{false, true, false} {
		ui.NewTester(func(c *ui.Context) {
			next := *c.Theme()
			next.Dark = dark
			c.SetTheme(&next)
			got := ContextFont(c, SkinComposerFont)
			want := RenderPreferences().Composer.Family
			if dark {
				want = ClaudeProseFamily
			}
			if got.Family != want {
				t.Errorf("dark %v font %q want %q", dark, got.Family, want)
			}
			ui.Text(c, "Mode text").Font(got.Family)
		}, 200, 80)
	}
}
