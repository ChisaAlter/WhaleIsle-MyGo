package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestMCPFilterRowAppearsWithTheCatalog covers task-50 (revised by pass 8):
// the filter row is not drawn before the catalog handshake, because a disabled
// search box is a dead control; it appears live with the catalog.
func TestMCPFilterRowAppearsWithTheCatalog(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	render := func(t *testing.T, a *MCPCatalogAccess) *ui.Tester {
		t.Helper()
		return ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(16).Children(func() {
				mcpCatalogControls(c, a, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 700, 240)
	}

	// Without a catalog the row is absent: the empty state explains the page,
	// and no disabled control pretends to work.
	absent := render(t, nil)
	for _, label := range []string{"MCP search", "MCP filter"} {
		if _, ok := absent.Find(label); ok {
			t.Fatalf("dead filter control %q is drawn before the catalog arrives: %v", label, absent.Texts())
		}
	}

	// With a catalog the row is live: typing reports through OnQuery.
	queried := ""
	a := &MCPCatalogAccess{OnQuery: func(q string) { queried = q }}
	a.Snapshot.Ready = true
	live := render(t, a)
	if _, ok := live.Find("MCP search"); !ok {
		t.Fatalf("filter control missing with a catalog: %v", live.Texts())
	}
	if rect, _ := live.Find("MCP filter"); rect.W < 88 || rect.W > 104 || rect.H != 32 {
		t.Fatalf("live dropdown is %v×%v, want 88(+16 gutter)×32", rect.W, rect.H)
	}
	if err := live.Click("MCP filter"); err != nil {
		t.Fatalf("live dropdown is not clickable: %v", err)
	}
	if queried != "" {
		t.Fatalf("picking a filter ran the query callback: %q", queried)
	}
}
