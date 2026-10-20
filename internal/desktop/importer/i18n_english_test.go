package importer

import (
	"testing"
	"unicode"

	"whaleisle/internal/ui/i18n"
)

// hasHan reports whether text still carries Chinese characters, which English
// sessions must never show.
func hasHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// TestEnglishLocaleShowsNoChineseCopy guards the import wizard: its preview and
// policy text used to be Chinese-only, and Confirmation renders both.
func TestEnglishLocaleShowsNoChineseCopy(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	if hasHan(CopyOnlyNotice()) {
		t.Fatalf("import policy text still contains Chinese: %q", CopyOnlyNotice())
	}

	cat := Catalog{
		SourceHome: "/source/home",
		DestHome:   "/dest/home",
		Sessions:   []CatalogItem{{ID: "session-1", Label: "Session one"}},
		Skills:     []CatalogItem{{ID: "skill-1", Label: "Skill one", Conflict: true}},
		Plugins:    []CatalogItem{{ID: "plugin-1", Spec: "github:owner/repo#v1.2.3"}},
		Settings:   []CatalogItem{{ID: "AGENTS.md", Label: "AGENTS.md"}},
	}
	sel := ApplySelection{
		SelectedRels:       []string{"session-1"},
		SelectedSkillIDs:   []string{"skill-1"},
		SelectedPluginIDs:  []string{"plugin-1"},
		SelectedSettingIDs: []string{"AGENTS.md"},
	}
	text := Confirmation(cat, sel)
	if hasHan(text) {
		t.Fatalf("english import confirmation still contains Chinese: %q", text)
	}
}
