package im

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestDisabledPluginOffersARealEnableAction(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	rendered := false
	enabled := 0
	tt := ui.NewTester(func(c *ui.Context) {
		rendered = true
		View(c, Model{ThemeID: "deepseek-light", PluginDisabled: true, OnEnablePlugin: func() { enabled++ }})
	}, 900, 700)
	if !rendered {
		t.Fatal("IM view did not render")
	}
	if err := tt.Click("Enable the dsh-im plugin"); err != nil {
		t.Fatalf("enable action missing: %v", err)
	}
	if enabled != 1 {
		t.Fatalf("enable action fired %d times, want 1", enabled)
	}

	// With the plugin enabled the action disappears.
	on := ui.NewTester(func(c *ui.Context) {
		View(c, Model{ThemeID: "deepseek-light", OnEnablePlugin: func() { enabled++ }})
	}, 900, 700)
	if _, ok := on.Find("Enable the dsh-im plugin"); ok {
		t.Fatal("enable action still drawn for an enabled plugin")
	}
}
