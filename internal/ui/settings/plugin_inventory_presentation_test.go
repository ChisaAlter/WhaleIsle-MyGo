package settings

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/plugininventory"
)

func TestPluginInventoryCardsReflowAndExpandTheirOwnEntry(t *testing.T) {
	rows := []plugininventory.DisplayRow{
		{Row: plugininventory.Row{Key: "preset-a/one", EntryID: "loader-one", Module: "module-one", Enabled: "enabled", Phase: "active"}, Title: "First entry", Description: "First description"},
		{Row: plugininventory.Row{Key: "preset-a/two", EntryID: "loader-two", Module: "module-two", Enabled: "conditional", Phase: "failed", Condition: "fixture-condition"}, Title: "Second entry", Description: "Second description"},
	}
	a := &PluginInventoryAccess{}
	a.OnToggleRow = func(key string) {
		if a.Snapshot.Expanded == key {
			a.Snapshot.Expanded = ""
		} else {
			a.Snapshot.Expanded = key
		}
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { pluginInventoryCards(c, a, rows, c.Theme().Text, c.Theme().TextMuted) })
	}, 760, 1000)
	first, ok1 := tt.Find("First entry")
	second, ok2 := tt.Find("Second entry")
	if !ok1 || !ok2 || first.Y != second.Y || second.X <= first.X {
		t.Fatalf("inventory cards were not in two columns: %v %v", first, second)
	}
	saveInventorySnapshot(t, tt, "plugin-inventory-wide.png")
	if err := tt.Click("Second entry"); err != nil {
		t.Fatal(err)
	}
	if a.Snapshot.Expanded != rows[1].Key || !tt.HasText("fixture-condition") || tt.HasText("Module: module-one") {
		t.Fatal("expansion did not preserve the selected Loader identity")
	}
	saveInventorySnapshot(t, tt, "plugin-inventory-expanded.png")
	tt.SetSize(420, 1000)
	first, ok1 = tt.Find("First entry")
	second, ok2 = tt.Find("Second entry")
	if !ok1 || !ok2 || second.Y <= first.Y || first.X != second.X || second.X+second.W > 420 {
		t.Fatalf("narrow inventory did not stack within viewport: %v %v", first, second)
	}
	saveInventorySnapshot(t, tt, "plugin-inventory-narrow.png")
}

func saveInventorySnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
