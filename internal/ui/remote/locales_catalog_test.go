package remote

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestRemoteLocaleCatalogParity(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })
	if len(remoteLocaleCatalog) == 0 {
		t.Fatal("empty locale catalog")
	}
	for _, entry := range remoteLocaleCatalog {
		fullKey := "remote" + "." + entry.module + "." + entry.key
		i18n.SetLocale(i18n.LocaleZH)
		if got := i18n.T(fullKey); got != entry.zh {
			t.Fatalf("zh mismatch for %s: got %q want %q", fullKey, got, entry.zh)
		}
		if got := remoteLocaleText(entry.module + "." + entry.key); got != entry.zh {
			t.Fatalf("helper zh mismatch for %s: got %q want %q", entry.key, got, entry.zh)
		}
		i18n.SetLocale(i18n.LocaleEN)
		if entry.en != "" {
			if got := i18n.T(fullKey); got != entry.en {
				t.Fatalf("en mismatch for %s: got %q want %q", fullKey, got, entry.en)
			}
			if got := remoteLocaleText(entry.module + "." + entry.key); got != entry.en {
				t.Fatalf("helper en mismatch for %s: got %q want %q", entry.key, got, entry.en)
			}
		}
	}
}
