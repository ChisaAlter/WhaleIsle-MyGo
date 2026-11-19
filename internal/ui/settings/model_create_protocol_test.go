package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// hostProtocols is the order the Host serializes its protocol union in, which
// is what the card reads: chat completions first, responses second.
var hostProtocols = []string{"openai-completions", "openai-responses", "anthropic-messages", "google-generative-ai"}

func providerCreateTester(s *store.Store, fields, models []store.Field) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			configurationCreateProvider(c, Model{Store: s, Models: &ModelsAccess{}},
				s.Snapshot(), map[string]any{}, fields, models, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120))
		})
	}, 760, 900)
}

// The protocol field must not answer for the reader. Its first option is chat
// completions, so a default silently created a provider that saved, was listed,
// and then failed every turn against a gateway that only speaks Responses.
func TestProviderCreateAsksForTheProtocolInsteadOfGuessing(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	labels, selected := configurationProtocolChoices("", hostProtocols)
	if len(labels) != len(hostProtocols)+1 {
		t.Fatalf("the field offers %v", labels)
	}
	if selected != labels[0] {
		t.Fatalf("an untouched field selects %q, want the placeholder %q", selected, labels[0])
	}
	if selected == configurationProtocolLabel(hostProtocols[0]) {
		t.Fatalf("an untouched field defaults to %q", hostProtocols[0])
	}
	if id := configurationProtocolChoice(selected, hostProtocols); id != "" {
		t.Fatalf("the placeholder writes protocol %q", id)
	}

	// A chosen protocol is what the card shows and what it writes.
	_, selected = configurationProtocolChoices("openai-responses", hostProtocols)
	if selected != configurationProtocolLabel("openai-responses") {
		t.Fatalf("a Responses provider shows %q", selected)
	}
	if id := configurationProtocolChoice(selected, hostProtocols); id != "openai-responses" {
		t.Fatalf("the shown choice writes %q", id)
	}
	for _, id := range hostProtocols {
		if got := configurationProtocolChoice(configurationProtocolLabel(id), hostProtocols); got != id {
			t.Fatalf("choosing %q writes %q", id, got)
		}
	}
}

// The blocked create says which requirement is unmet, and it does so in the
// reader's language: no protocol id, no internal name.
func TestProviderCreateExplainsTheMissingProtocol(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	rows := []any{map[string]any{"id": "auto/gpt-6-luna"}}
	hint := configurationCreateHint("magpie", "http://127.0.0.1:3425/v1", "", rows, false, false)
	if hint == "" {
		t.Fatal("a card with no protocol says nothing about why it is blocked")
	}
	if !strings.Contains(hint, "协议") {
		t.Fatalf("the hint does not name the protocol: %q", hint)
	}
	for _, leaked := range []string{"openai", "anthropic", "llm-pi-ai", "api"} {
		if strings.Contains(hint, leaked) {
			t.Fatalf("the hint carries %q: %q", leaked, hint)
		}
	}
	if hint := configurationCreateHint("magpie", "http://127.0.0.1:3425/v1", "openai-responses", rows, false, false); hint != "" {
		t.Fatalf("a complete card still reports %q", hint)
	}
}

// Rendering the card with nothing chosen shows the placeholder, not the first
// protocol the Host offers.
func TestProviderCreateFieldShowsThePlaceholder(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	s := twoProviderStore(t)
	fields := []store.Field{{Key: "api", EnumOptions: hostProtocols}}
	models := []store.Field{{Key: "id"}}
	tt := providerCreateTester(s, fields, models)
	_, selected := configurationProtocolChoices("", hostProtocols)
	if !tt.HasText(selected) {
		t.Fatalf("the field does not show the placeholder %q: %v", selected, tt.Texts())
	}
	if tt.HasText("OpenAI Chat Completions") {
		t.Fatalf("the field shows a protocol before a choice: %v", tt.Texts())
	}
}
