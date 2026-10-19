package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEmptyPreferencesPreserveRawFamiliesAndRestoreRoleSizes(t *testing.T) {
	p := (Preferences{}).Normalized()
	for name, pair := range map[string]struct {
		got  FontPreference
		size float32
	}{"interface": {p.Interface, 16}, "chat": {p.Chat, 14}, "composer": {p.Composer, 14}, "code": {p.Code, 13}, "terminal": {p.Terminal, 13}} {
		if pair.got.Family != "" || pair.got.Size != pair.size {
			t.Errorf("%s raw default=%+v", name, pair.got)
		}
	}
	if TextFamily(p.Interface.Family) != DefaultTextFamily || CodeFamily(p.Code.Family) != DefaultCodeFamily {
		t.Fatal("render fallback missing for empty raw Host family")
	}
}

func TestNormalizePreservesExplicitTextPreferences(t *testing.T) {
	want := FontPreference{Family: "'Custom Sans', 'Microsoft YaHei'", Size: 18}
	p := Preferences{Interface: want, Chat: want, Composer: want}.Normalized()
	if p.Interface != want || p.Chat != want || p.Composer != want {
		t.Fatalf("normalization changed explicit fonts: %+v", p)
	}
	explicitCode := FontPreference{Family: "monospace", Size: 16}
	if got := (Preferences{Code: explicitCode}).Normalized().Code; got != explicitCode {
		t.Fatal("explicit saved code family was migrated", got)
	}
}

func TestApplyInterfaceUsesFallbackStackAndPreservesPalette(t *testing.T) {
	previous := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(previous) })
	SetPreferences(Preferences{})
	ui.NewTester(func(c *ui.Context) {
		before := *c.Theme()
		ApplyInterface(c)
		after := *c.Theme()
		if after.Font != DefaultTextFamily || after.FontSize != 16 {
			t.Errorf("native interface font=%q size=%g", after.Font, after.FontSize)
		}
		after.Font, after.FontSize = before.Font, before.FontSize
		if after != before {
			t.Error("applying typography changed native control palette")
		}
		ui.Text(c, "WhaleIsle 鲸屿").FontSize(14)
	}, 320, 80)
}
