package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func replaceActionFixture(t *testing.T) (Model, store.ViewState, *store.Namespace) {
	t.Helper()
	snap := store.ViewState{Writable: true, SelectedNS: "general", Dirty: true}
	snap.Namespaces = []store.Namespace{{NS: "general"}}
	return Model{Store: store.New()}, snap, &snap.Namespaces[0]
}

// TestReplaceAsksInTheApp keeps the destructive replace behind the application's
// own dialog: the action row opens the confirmation instead of writing at once.
func TestReplaceAsksInTheApp(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	m, snap, ns := replaceActionFixture(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			settingsActions(c, m, snap, ns, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
		})
	}, 720, 400)
	if tt.HasText(i18n.T(i18n.KeySurfacesSettingsConfirmTitle)) {
		t.Fatalf("confirmation rendered before the action: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesSettingsReplace)); err != nil {
		t.Fatalf("replace action is not reachable: %v", err)
	}
	tt.Frame()
	if !tt.HasText(i18n.T(i18n.KeySurfacesSettingsConfirmTitle)) {
		t.Fatalf("replace action did not ask for confirmation: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesSettingsCancel)); err != nil {
		t.Fatalf("cancel action is not reachable: %v", err)
	}
	tt.Frame()
	if tt.HasText(i18n.T(i18n.KeySurfacesSettingsConfirmTitle)) {
		t.Fatalf("cancel left the confirmation open: %v", tt.Texts())
	}
	if snap := m.Store.Snapshot(); snap.LastAction != "" || snap.Err != "" {
		t.Fatalf("cancelled replace mutated the store: last=%q err=%q", snap.LastAction, snap.Err)
	}
}
