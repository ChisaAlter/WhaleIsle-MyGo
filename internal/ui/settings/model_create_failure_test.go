package settings

import (
	"strings"
	"testing"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/store/storelocale"
	"whaleisle/internal/ui/i18n"
)

// The sentence a failed provider creation shows must be written for a reader,
// never lifted from the Host or the transport, and it must say what to do next.
func TestProviderCreateFailureSpeaksToTheReader(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	raw := `Provider settings were not completed; the API key was not saved. settings/rejected: Configuration for "llm-pi-ai" is overridden by a home patch or command-line overlay`
	snap := store.ViewState{Err: storelocale.T(storelocale.KeyStoreSettingsWriteNotStored), ErrDetail: "settings/mutate llm-pi-ai: settings/rejected"}
	sentence, details := providerCreateFailure(snap, raw)
	if sentence != snap.Err {
		t.Fatalf("the store's sentence was not used: %q", sentence)
	}
	for _, leaked := range []string{"llm-pi-ai", "settings/rejected", "Provider settings were not completed"} {
		if strings.Contains(sentence, leaked) {
			t.Fatalf("sentence carries %q: %q", leaked, sentence)
		}
	}
	if !strings.Contains(sentence, "Try again") {
		t.Fatalf("sentence names no next step: %q", sentence)
	}
	if len(details) != 2 || details[0] != snap.ErrDetail || details[1] != raw {
		t.Fatalf("the Host's own words did not reach the details: %v", details)
	}
}

// A failure the store did not phrase (the credential step) still gets the
// written sentence, with the raw text kept for the details block.
func TestProviderCreateFailureWithoutAStoreSentence(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	raw := "Provider settings saved; API key step: credentials/describe failed"
	sentence, details := providerCreateFailure(store.ViewState{}, raw)
	if sentence == raw || strings.Contains(sentence, "credentials/describe") {
		t.Fatalf("raw text stood in for the sentence: %q", sentence)
	}
	if !strings.Contains(sentence, "再试一次") {
		t.Fatalf("sentence names no next step: %q", sentence)
	}
	if len(details) != 1 || details[0] != raw {
		t.Fatalf("raw text was not kept for the details: %v", details)
	}
}
