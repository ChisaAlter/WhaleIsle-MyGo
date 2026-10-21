package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestSkillStatusLabelLocalizesKnownTokens(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	cases := map[string]string{
		"installed":        "Installed",
		"conflict":         "Conflict",
		"not-installed":    "Not installed",
		"update-available": "Update available",
		"something-new":    "something-new",
	}
	for in, want := range cases {
		if got := skillStatusLabel(in); got != want {
			t.Fatalf("skillStatusLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
