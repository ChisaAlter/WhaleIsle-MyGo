package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestAgentPresetsI18nKeys(t *testing.T) {
	keys := []string{
		i18n.KeySettingsAgentPresetsNav,
		i18n.KeySettingsAgentPresetsTitle,
		i18n.KeySettingsAgentPresetsHint,
		i18n.KeySettingsAgentPresetsPicker,
		i18n.KeySettingsAgentPresetsEmpty,
		i18n.KeySettingsAgentPresetsSession,
		i18n.KeySettingsAgentPresetsNoSession,
	}
	i18n.SetLocale(i18n.LocaleZH)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("zh missing %q got %q", k, got)
		}
	}
	i18n.SetLocale(i18n.LocaleEN)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("en missing %q got %q", k, got)
		}
	}
	i18n.SetLocale(i18n.DefaultLocale)
}
