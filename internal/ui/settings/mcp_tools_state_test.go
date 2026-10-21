package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

// TestMCPRowStatesTheToolCount covers the Pass 9 audit fix: a connected server
// with tools offers the count button, and one with none says so instead of
// leaving the same silent row a not-yet-read server shows.
func TestMCPRowStatesTheToolCount(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	render := func(t *testing.T, tools []string) *ui.Tester {
		t.Helper()
		rows := []api.McpServerSnapshotServersItem{}
		item := api.McpServerSnapshotServersItem{Id: "srv"}
		connection := api.McpServerSnapshotServersItemConnection{Tools: tools}
		item.Connection = &connection
		_ = connection
		rows = append(rows, item)
		catalog := &MCPCatalogAccess{Snapshot: mcpconfig.Snapshot{Rows: rows}}
		row := MCPServerRow{ID: "srv", ServerName: "tools", Enabled: true, Writable: true}
		return ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(16).Children(func() {
				mcpCompactRow(c, &MCPAccess{Catalog: catalog, HasListRPC: true}, row, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 760, 320)
	}
	with := render(t, []string{"read_file", "write_file"})
	if !with.HasText("2 tools") {
		t.Fatalf("tool count is not painted: %v", with.Texts())
	}
	empty := render(t, nil)
	if !empty.HasText("no tools") {
		t.Fatalf("the zero-tool state is silent: %v", empty.Texts())
	}
	if empty.HasText("0 tools") {
		t.Fatal("the zero-tool state still paints a count button")
	}
}
