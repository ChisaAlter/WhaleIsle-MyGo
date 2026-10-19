package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEmptyPreferencesRestoreOriginalTextFallbacks(t *testing.T) {
	p := (Preferences{}).Normalized()
	for name, f := range map[string]FontPreference{"interface": p.Interface, "chat": p.Chat, "composer": p.Composer} {
		if f.Family != DefaultTextFamily || f.Size != 14 {
			t.Errorf("%s default=%+v", name, f)
		}
	}
	if p.Code.Family != DefaultCodeFamily || p.Code.Size != 13 {
		t.Errorf("code default = %+v", p.Code)
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
		if after.Font != DefaultTextFamily || after.FontSize != 14 {
			t.Errorf("native interface font=%q size=%g", after.Font, after.FontSize)
		}
		after.Font, after.FontSize = before.Font, before.FontSize
		if after != before {
			t.Error("applying typography changed native control palette")
		}
		ui.Text(c, "WhaleIsle 鲸屿").FontSize(14)
	}, 320, 80)
}
