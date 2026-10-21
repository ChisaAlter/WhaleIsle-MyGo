package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// The built-in plugin cards used FillWidth inside a Grid cell, so each card took
// the whole grid width and the second column was clipped. Cards must stay in
// their columns and name text must remain inside the pane.
func TestBuiltinPluginCardsStayInTheirColumns(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	rows := []PluginRow{
		{Name: "@deepseek-ai/dsh-base"},
		{Name: "@deepseek-ai/dsh-web-app"},
		{Name: "@deepseek-ai/dsh-experimental-agent-team"},
		{Name: "@deepseek-ai/dsh-experimental-inspector"},
	}
	const width = 760
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Children(func() {
			builtinPluginGrid(c, &PluginsAccess{Rows: rows}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, width, 600)
	tt.Frame()
	for _, row := range rows {
		text, ok := tt.Find(row.Name)
		if !ok {
			t.Fatalf("plugin name %q missing", row.Name)
		}
		if text.X < 16 || text.X+text.W > width-16+0.5 {
			t.Fatalf("plugin %q escapes the pane: %+v", row.Name, text)
		}
	}
}
