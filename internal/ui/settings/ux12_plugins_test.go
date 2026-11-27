package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"
)

func TestConfirmDialogLocaleAndBackdropDismissDisabled(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	cancelled, confirmed := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		ConfirmDialog(c, ConfirmOptions{
			Title:        "Delete item",
			Body:         "This action cannot be undone.",
			ConfirmLabel: "Delete",
			Danger:       true,
			OnConfirm:    func() { confirmed++ },
			OnCancel:     func() { cancelled++ },
		})
	}, 800, 600)

	if !tt.HasText("Cancel") {
		t.Fatalf("expected localized English Cancel button; texts=%q", tt.Texts())
	}
	if tt.HasText("取消") {
		t.Fatalf("hardcoded Chinese cancel label leaked in EN locale; texts=%q", tt.Texts())
	}

	// Clicking backdrop corner must NOT dismiss destructive confirmation dialog.
	tt.ClickAt(4, 4)
	tt.Frame()
	if cancelled != 0 || confirmed != 0 {
		t.Fatalf("backdrop click dismissed ConfirmDialog: cancelled=%d confirmed=%d", cancelled, confirmed)
	}

	// Explicit Cancel button click dismisses.
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("click Cancel: %v", err)
	}
	if cancelled != 1 {
		t.Fatalf("cancelled = %d, want 1", cancelled)
	}
}

func TestAppearanceColorSchemeTileRendersInLightAndDark(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	for _, dark := range []bool{false, true} {
		selectedTheme := "deepseek.dark"
		if !dark {
			selectedTheme = "deepseek.light"
		}
		tt := ui.NewTester(func(c *ui.Context) {
			appearancePane(c, Model{
				ThemeID:    selectedTheme,
				SystemDark: dark,
				Appearance: &AppearanceAccess{
					Theme:      &selectedTheme,
					OnSetTheme: func(string) error { return nil },
				},
			}, c.Theme().Text, c.Theme().TextMuted)
		}, 900, 640)
		tt.SetDark(dark)
		tt.Frame()
		for _, label := range []string{"Light", "Dark", "System"} {
			if !tt.HasText(label) {
				t.Fatalf("dark=%v missing color scheme tile %q; texts=%q", dark, label, tt.Texts())
			}
		}
	}
}

func TestPluginsOperationMessageFailureUsesAlert(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	rawErr := "dial tcp 127.0.0.1:443: connectex: connection refused"
	access := &PluginsAccess{
		ClientReady:      true,
		HasListRPC:       true,
		OperationMessage: rawErr,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		pluginPackages(c, access, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 640)

	wantSentence := errcopy.SentenceText(rawErr)
	if !tt.HasText(wantSentence) {
		t.Fatalf("missing sanitized alert sentence %q; texts=%q", wantSentence, tt.Texts())
	}
	if tt.HasText(rawErr) {
		t.Fatalf("raw error string leaked outside details disclosure; texts=%q", tt.Texts())
	}
	if !tt.HasText("Details") {
		t.Fatalf("expected alert Details disclosure button; texts=%q", tt.Texts())
	}

	// Non-failure status keeps notice style without Details disclosure.
	okAccess := &PluginsAccess{
		ClientReady:      true,
		HasListRPC:       true,
		OperationMessage: "Applied; checking inventory.",
	}
	ttOK := ui.NewTester(func(c *ui.Context) {
		pluginPackages(c, okAccess, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 640)
	if !ttOK.HasText("Applied; checking inventory.") {
		t.Fatalf("missing success operation notice; texts=%q", ttOK.Texts())
	}
	if ttOK.HasText("Details") {
		t.Fatalf("success notice unexpectedly rendered error alert; texts=%q", ttOK.Texts())
	}
}
