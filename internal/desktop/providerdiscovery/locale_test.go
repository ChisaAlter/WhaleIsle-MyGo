package providerdiscovery

import (
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestDiscoveryMessagesFollowCurrentLocale(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, pair := range messages {
		i18n.SetLocale("zh")
		if got := Message(pair[0]); got != pair[1] {
			t.Fatalf("Chinese status missing: %q", got)
		}
		i18n.SetLocale("en")
		if got := Message(pair[0]); got != pair[0] {
			t.Fatalf("English status changed: %q", got)
		}
	}
	i18n.SetLocale("zh")
	if got := Message("Demo: 3 candidates. Select models, add them to the draft, then review and save."); !strings.Contains(got, "3 个候选模型") {
		t.Fatalf("candidate status not translated: %s", got)
	}
}
