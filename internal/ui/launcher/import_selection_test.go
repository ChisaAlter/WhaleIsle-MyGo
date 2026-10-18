package launcher

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/importer"
)

func TestImportSelectionFeedbackTracksToggle(t *testing.T) {
	previous := importCatActive
	importCatActive = "sessions"
	defer func() { importCatActive = previous }()
	catalog := &importer.Catalog{Sessions: []importer.CatalogItem{{ID: "one", Label: "Import selection fixture"}}}
	tt := ui.NewTester(func(c *ui.Context) { importPanel(c, Model{ImportCatalog: catalog}) }, 1200, 1000)
	const unchecked = "□  Import selection fixture"
	const checked = "✓  Import selection fixture"
	if !tt.HasText(unchecked) {
		t.Fatalf("unselected row has no distinct state: %q", tt.Texts())
	}
	if err := tt.Click(unchecked); err != nil {
		t.Fatal(err)
	}
	if !catalog.Sessions[0].Selected || !tt.HasText(checked) {
		t.Fatalf("selection not reflected in row: %q", tt.Texts())
	}
	if err := tt.Click(checked); err != nil {
		t.Fatal(err)
	}
	if catalog.Sessions[0].Selected || !tt.HasText(unchecked) {
		t.Fatalf("deselection not reflected in row: %q", tt.Texts())
	}
}
