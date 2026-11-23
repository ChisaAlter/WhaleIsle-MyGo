package market

import (
	"testing"

	marketstore "whaleisle/internal/store/market"
)

func TestDesktopInstalledProjectionRetainsPartialDependenciesOnly(t *testing.T) {
	disabled := false
	rows := []marketstore.Entry{
		{Label: "saved-awaiting-restart", IdentityKnown: true, Installed: true, Enabled: &disabled, Error: "restart required"},
		{Label: "missing-files", IdentityKnown: true, Installed: true, ReadOnlyReason: "files missing"},
		{Label: "runtime-optional", IdentityKnown: true, Installed: false},
		{Label: "identity-not-read", Installed: true},
	}
	got := desktopInstalledRows(rows)
	if len(got) != 2 || got[0].Label != "saved-awaiting-restart" || got[1].Label != "missing-files" || got[0].Enabled == nil || *got[0].Enabled {
		t.Fatalf("partial dependency state was confused with runtime availability: %+v", got)
	}
}
