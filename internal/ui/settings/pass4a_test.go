package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func pass4aTester(t *testing.T, build func(c *ui.Context)) *ui.Tester {
	t.Helper()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() { build(c) })
	}, 800, 700)
}

// TestMCPEmptyStateIsADashedCard covers the MCP placeholder: the dashed outline,
// both centered lines, and the two 28 DIP header actions.
func TestMCPEmptyStateIsADashedCard(t *testing.T) {
	refreshed := 0
	edited := 0
	a := &MCPAccess{HasListRPC: true, Servers: nil, OnRefresh: func() { refreshed++ }, OnEdit: func(string) { edited++ }}
	tt := pass4aTester(t, func(c *ui.Context) { mcpPane(c, Model{MCP: a}, c.Theme().Text, c.Theme().TextMuted) })
	for _, want := range []string{i18n.T(KeyPass4AMCPEmptyPrimary), i18n.T(KeyPass4AMCPEmptySecondary)} {
		if !tt.HasText(want) {
			t.Fatalf("empty card lost %q: %v", want, tt.Texts())
		}
	}
	for _, action := range []string{"Add MCP server", "Refresh"} {
		rect, ok := tt.Find(action)
		if !ok {
			t.Fatalf("header action %q missing: %v", action, tt.Texts())
		}
		if rect.W != 28 || rect.H != 28 {
			t.Fatalf("header action %q is %v×%v, want 28×28", action, rect.W, rect.H)
		}
	}
	if err := tt.Click("Add MCP server"); err != nil {
		t.Fatalf("add action is not clickable: %v", err)
	}
	if edited != 1 {
		t.Fatalf("add action ran %d times, want 1", edited)
	}
	if err := tt.Click("Refresh"); err != nil {
		t.Fatalf("refresh action is not clickable: %v", err)
	}
	if refreshed != 1 {
		t.Fatalf("refresh action ran %d times, want 1", refreshed)
	}
}

// TestBuiltinPluginsRenderAsAGridWithASkeleton covers the built-in page: two
// columns of cards, and the 2×2 skeleton while the list is still loading.
func TestBuiltinPluginsRenderAsAGridWithASkeleton(t *testing.T) {
	rows := []PluginRow{{Name: "dsh-ui-market", Status: "enabled"}, {Name: "dsh-worktree", Status: "enabled"}, {Name: "dsh-glass", Status: "disabled", Disabled: true}}
	loaded := pass4aTester(t, func(c *ui.Context) {
		builtinPluginGrid(c, &PluginsAccess{Rows: rows}, c.Theme().Text, c.Theme().TextMuted)
	})
	for _, row := range rows {
		if !loaded.HasText(row.Name) {
			t.Fatalf("built-in grid dropped %q: %v", row.Name, loaded.Texts())
		}
	}
	loading := pass4aTester(t, func(c *ui.Context) {
		builtinPluginGrid(c, &PluginsAccess{Loading: true}, c.Theme().Text, c.Theme().TextMuted)
	})
	if loading.HasText("dsh-ui-market") {
		t.Fatal("the skeleton invented plugin rows")
	}
	if got := loading.Texts(); len(got) != 1 || got[0] != i18n.T(KeyPass4ABuiltinLoading) {
		t.Fatalf("the skeleton should paint only its loading label: %v", got)
	}
}
