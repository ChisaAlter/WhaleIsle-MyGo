package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestMCPFilterRowIsAlwaysShown covers task-50: the search box and the 全部
// dropdown keep their 32×88 geometry whether or not the catalog handshake has
// landed, and only become interactive once it has.
func TestMCPFilterRowIsAlwaysShown(t *testing.T) {
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

	// Without a catalog the row still paints, both controls disabled.
	absent := render(t, nil)
	for _, label := range []string{"MCP search", "MCP filter"} {
		if _, ok := absent.Find(label); !ok {
			t.Fatalf("filter control %q is missing before the catalog arrives: %v", label, absent.Texts())
		}
	}
	// The box is 88 DIP; mygo reports the Select label rect including its 16 DIP
	// chevron gutter, so the assertion allows that gutter and nothing more.
	if rect, _ := absent.Find("MCP filter"); rect.W < 88 || rect.W > 104 || rect.H != 32 {
		t.Fatalf("disabled dropdown is %v×%v, want 88(+16 gutter)×32", rect.W, rect.H)
	}
	textsBefore := len(absent.Texts())
	absent.ClickAt(620, 30)
	absent.Frame()
	if after := len(absent.Texts()); after != textsBefore {
		t.Fatalf("the disabled dropdown opened a menu (%d → %d texts)", textsBefore, after)
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
