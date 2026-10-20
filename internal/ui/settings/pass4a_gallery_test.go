package settings

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// saveSettingsGallery writes one tester frame into SETTINGS_GALLERY_DIR. The gate
// keeps the PNG out of normal runs; the gallery is regenerated on demand.
func saveSettingsGallery(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SETTINGS_GALLERY_DIR")
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
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func galleryPane(t *testing.T, build func(c *ui.Context)) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() { build(c) })
	}, 1000, 800)
}

// TestSettingsGalleryInterface saves the interface page (gear row included).
func TestSettingsGalleryInterface(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	tt := interfacePaneTester(t, interfaceGearFixture(), 1000, 800)
	saveSettingsGallery(t, tt, "ours-settings-interface.png")
}

// TestSettingsGalleryBuiltin saves the built-in plugin page in its loaded state.
func TestSettingsGalleryBuiltin(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	rows := []PluginRow{
		{Name: "dsh-ui-market", Status: "enabled"},
		{Name: "dsh-worktree", Status: "enabled"},
		{Name: "dsh-glass", Status: "disabled", Disabled: true},
		{Name: "dsh-usage", Status: "enabled"},
	}
	// 560 ≈ the settings content column, so the two cards land at the 273 DIP the
	// reference uses instead of stretching a 1000 px test window.
	tt := galleryPane(t, func(c *ui.Context) {
		ui.Column(c).Width(560).Children(func() {
			builtinPluginGrid(c, &PluginsAccess{Rows: rows, ClientReady: true}, c.Theme().Text, c.Theme().TextMuted)
		})
	})
	saveSettingsGallery(t, tt, "ours-settings-builtin.png")
}

// TestSettingsGalleryBuiltinLoading saves the 2×2 skeleton state.
func TestSettingsGalleryBuiltinLoading(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	tt := galleryPane(t, func(c *ui.Context) {
		ui.Column(c).Width(560).Children(func() {
			builtinPluginGrid(c, &PluginsAccess{Loading: true}, c.Theme().Text, c.Theme().TextMuted)
		})
	})
	saveSettingsGallery(t, tt, "ours-settings-builtin-loading.png")
}

// TestSettingsGalleryMCP saves the MCP page with its dashed empty card.
func TestSettingsGalleryMCP(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	a := &MCPAccess{HasListRPC: true, OnRefresh: func() {}, OnEdit: func(string) {}}
	tt := galleryPane(t, func(c *ui.Context) {
		mcpPane(c, Model{MCP: a, ThemeID: theme.IDDeepseekLight}, c.Theme().Text, c.Theme().TextMuted)
	})
	saveSettingsGallery(t, tt, "ours-settings-mcp.png")
}
