package interaction

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestInteractionLocaleCatalogParity(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })
	if len(interactionLocaleCatalog) == 0 {
		t.Fatal("empty locale catalog")
	}
	for _, entry := range interactionLocaleCatalog {
		fullKey := "interaction" + "." + entry.module + "." + entry.key
		i18n.SetLocale(i18n.LocaleZH)
		if got := i18n.T(fullKey); got != entry.zh {
			t.Fatalf("zh mismatch for %s: got %q want %q", fullKey, got, entry.zh)
		}
		if got := interactionLocaleText(entry.module + "." + entry.key); got != entry.zh {
			t.Fatalf("helper zh mismatch for %s: got %q want %q", entry.key, got, entry.zh)
		}
		i18n.SetLocale(i18n.LocaleEN)
		if entry.en != "" {
			if got := i18n.T(fullKey); got != entry.en {
				t.Fatalf("en mismatch for %s: got %q want %q", fullKey, got, entry.en)
			}
			if got := interactionLocaleText(entry.module + "." + entry.key); got != entry.en {
				t.Fatalf("helper en mismatch for %s: got %q want %q", entry.key, got, entry.en)
			}
		}
	}
}
