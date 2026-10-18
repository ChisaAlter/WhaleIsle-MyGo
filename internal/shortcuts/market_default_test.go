package shortcuts

import "testing"

func TestMarketplaceOriginalDefaultAndOverride(t *testing.T) {
	if Defaults()["market.open"] != "Mod+Shift+M" {
		t.Fatal("market default differs from e9 menu")
	}
	custom := Resolve(map[string]string{"market.open": "Mod+Alt+F12"})
	if custom["market.open"].String() != "Mod+Alt+F12" {
		t.Fatal("saved user binding replaced by default migration")
	}
}
