package pluginui

import (
	"strings"
	"testing"
)

// The native Settings panel selects navigation:settings:<section> on the page so
// the page can follow it; the page echoes that subject back in every snapshot.
func TestValidSnapshotAcceptsSettingsNavigationSubject(t *testing.T) {
	snapshot := func(name string) Snapshot {
		return Snapshot{Revision: 1, Subject: &Subject{Kind: "navigation", Name: name}}
	}
	for _, name := range []string{"", "models", "settings:general", "settings:market"} {
		if !validSnapshot(snapshot(name)) {
			t.Fatalf("navigation subject %q must be accepted", name)
		}
	}
	for _, name := range []string{"other", "settings", "x:settings:general", "settings:" + strings.Repeat("a", 300)} {
		if validSnapshot(snapshot(name)) {
			t.Fatalf("navigation subject %q must be rejected", name)
		}
	}
}
