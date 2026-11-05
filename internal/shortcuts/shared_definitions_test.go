package shortcuts

import (
	"testing"

	"whaleisle/internal/shortcutprefs"
)

func TestSharedDefinitionsKeepFixedFontsAndEditableCatalogReady(t *testing.T) {
	for _, profile := range []string{"desktop:windows", "desktop:macos"} {
		t.Run(profile, func(t *testing.T) {
			store := shortcutprefs.New(profile, func() ([]byte, error) { return nil, nil }, nil)
			snapshot := store.SetDefinitions("native", SharedDefinitions(profile))
			if snapshot.Status != "ready" || snapshot.Error != nil {
				t.Fatalf("native catalog rejected: %+v", snapshot)
			}
			fixed := 0
			for _, definition := range SharedDefinitions(profile) {
				if definition.Fixed != nil {
					fixed++
					if definition.Defaults == nil || len(definition.Defaults) != 0 {
						t.Fatal("fixed font command has an invalid defaults object")
					}
				}
			}
			if fixed != 2 {
				t.Fatal("fixed font ownership lost", fixed)
			}
		})
	}
}
