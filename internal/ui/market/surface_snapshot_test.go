package market

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
)

// saveSurfaceSnapshot writes the rendered market for the evidence gallery when
// SURFACES_GALLERY_DIR is set; the path is named after the surface.
func saveSurfaceSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SURFACES_GALLERY_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestMarketSurfaceSnapshot renders the discovery page's structure — header,
// category tabs and the plugin cards — for evidence/after/surface-market.png.
func TestMarketSurfaceSnapshot(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	catalog := desktopmarket.Catalog{Categories: []desktopmarket.Category{
		{ID: "all", Label: "All", Count: 4},
		{ID: "ui", Label: "Interface", Count: 2},
		{ID: "tool", Label: "Tools", Count: 2},
	}}
	rows := []desktopmarket.CatalogItem{
		{ID: "dsh-worktree-1", Repo: "dsh-worktree", Owner: "deepseek-ai", PackageName: "@deepseek-ai/dsh-worktree", InstallSpec: "@deepseek-ai/dsh-worktree", Description: "Browse and install plugins from the registry without leaving the window.", Stars: 1200, Category: "ui"},
		{ID: "dsh-worktree", Repo: "dsh-worktree", Owner: "deepseek-ai", Description: "Manage git worktrees for parallel agent sessions.", Stars: 860, Category: "tool"},
		{ID: "dsh-glass", Repo: "dsh-glass", Owner: "community", Description: "Frosted glass effect for the window chrome and composer.", Stars: 240, Category: "ui"},
		{ID: "dsh-usage", Repo: "dsh-usage", Owner: "community", Description: "Token usage dashboard with a contribution calendar.", Stars: 180, Category: "tool"},
	}
	d := marketstore.Discovery{Category: "all", Period: "all", Tab: "discover", RuntimeReady: true, Catalog: catalog, Rows: rows}
	inventory := marketstore.ViewState{ClientReady: true, Rows: []marketstore.Entry{{Label: "@deepseek-ai/dsh-glass", Detail: "1.2.0"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Gap(16).Children(func() {
			catalogCards(c, Model{Store: store, ThemeID: "deepseek.light"}, d, inventory, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Border, c.Theme().Accent)
		})
	}, 800, 640)
	saveSurfaceSnapshot(t, tt, "surface-market.png")
}
