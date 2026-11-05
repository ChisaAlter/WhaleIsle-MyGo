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
	const item = "Import selection fixture"
	if !tt.HasText(item) || !tt.HasText("one  0/1") || catalog.Sessions[0].Selected {
		t.Fatalf("unselected row has no distinct state: %q", tt.Texts())
	}
	if err := tt.Click(item); err != nil {
		t.Fatal(err)
	}
	if !catalog.Sessions[0].Selected || !tt.HasText("one  1/1") {
		t.Fatalf("selection not reflected in row: %q", tt.Texts())
	}
	if err := tt.Click(item); err != nil {
		t.Fatal(err)
	}
	if catalog.Sessions[0].Selected || !tt.HasText("one  0/1") {
		t.Fatalf("deselection not reflected in row: %q", tt.Texts())
	}
}
