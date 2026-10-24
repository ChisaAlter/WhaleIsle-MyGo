package main

import (
	"testing"

	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/ui/settings"
)

// whaleisle://settings?section=remote must land on the embedded 远程 page; the
// section switch previously had no remote case and fell back to the desktop page.
func TestDeeplinkSettingsRemoteSection(t *testing.T) {
	for _, section := range []string{"remote", "phone", "REMOTE"} {
		s := &mainState{}
		s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: section})
		if !s.settingsOpen {
			t.Fatalf("section %q did not open settings", section)
		}
		if s.settingsEmbeddedSection != settings.SectionRemote {
			t.Fatalf("section %q -> embedded %q, want %q", section, s.settingsEmbeddedSection, settings.SectionRemote)
		}
	}
}
