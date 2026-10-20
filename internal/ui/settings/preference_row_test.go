package settings

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// TestSettingsFieldRowRendersRowWithLabeledSwitch pins the row primitive's
// promise: the title and its description stay with the copy, the control sits in
// its own column, and a labeled switch there is what a click reaches.
func TestSettingsFieldRowRendersRowWithLabeledSwitch(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	value := false
	fg, muted := ui.RGB(24, 24, 24), ui.RGB(120, 120, 120)
	tt := ui.NewTester(func(c *ui.Context) {
		settingsFieldRow(c, "row", "Auto save", "Writes the draft when it changes.", fg, muted, nil, func() {
			ui.Switch(c, &value).Label("Toggle auto save")
		})
	}, 720, 200)
	if !tt.HasText("Auto save") || !tt.HasText("Writes the draft when it changes.") {
		t.Fatal("row dropped its title or description")
	}
	if err := tt.Click("Toggle auto save"); err != nil {
		t.Fatalf("row control is not reachable by its label: %v", err)
	}
	if !value {
		t.Fatal("switch did not toggle")
	}
}

// TestFieldRowBoolRendersLabeledSwitch covers the schema form's boolean branch:
// a Value::Bool field must render the switch under the field's own name, not a
// checkbox with an empty label (the row used to pass "" as the control's name).
func TestFieldRowBoolRendersLabeledSwitch(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	draft, checked := "", false
	m := Model{
		DraftPtr: func(key, seed string) *string { return &draft },
		BoolPtr:  func(key, seed string) *bool { return &checked },
	}
	snap := store.ViewState{Drafts: map[string]string{"auto": "false"}, Writable: true}
	field := store.Field{Key: "auto", Label: "Auto save", Kind: store.FieldBool}
	tt := ui.NewTester(func(c *ui.Context) {
		fieldRow(c, m, snap, field, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
	}, 720, 240)
	named := 0
	for _, text := range tt.Texts() {
		if text == "Auto save" {
			named++
		}
	}
	if named < 2 {
		t.Fatalf("switch did not carry the field name: %d labeled nodes\n%v", named, tt.Texts())
	}
}

// TestFieldRowResetStaysWithTheCopy keeps the inherit action next to the title
// instead of turning it into a second control in the row.
func TestFieldRowResetStaysWithTheCopy(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	reset, value := false, true
	tt := ui.NewTester(func(c *ui.Context) {
		settingsFieldRow(c, "row", "Display name", "Shows on the session row.", ui.RGB(24, 24, 24), ui.RGB(120, 120, 120), func() { reset = true }, func() {
			ui.Switch(c, &value).Label("Toggle display name")
		})
	}, 720, 240)
	if !tt.HasText(i18n.T(i18n.KeySurfacesSettingsReset)) {
		t.Fatal("row lost its reset affordance")
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesSettingsReset)); err != nil {
		t.Fatalf("reset affordance is not reachable: %v", err)
	}
	if !reset {
		t.Fatal("reset affordance did not call back")
	}
}

// TestImplementationNamespacesStayInsideGeneral keeps remote-workspace out of
// the primary rail; it is a configuration section, so it must stay reachable
// through the namespace browser.
func TestImplementationNamespacesStayInsideGeneral(t *testing.T) {
	m := Model{Desktop: &DesktopAccess{}, OnSelectNamespace: func(string) {}}
	snap := store.ViewState{Namespaces: []store.Namespace{{NS: "remote-workspace"}, {NS: "general"}}}
	for _, entry := range primaryNavigation(m, snap) {
		if entry.id == SectionRemoteWorkspace {
			t.Fatal("remote-workspace became a primary settings section")
		}
	}
	if dedicatedNamespace("remote-workspace") {
		t.Fatal("remote-workspace is still hidden from the namespace browser")
	}
}

// TestPreferenceRowIndexIsPerFrame pins the rule-suppression mechanism every
// settings page relies on: the first row of a frame carries no leading hairline,
// each following row does, and the next frame starts counting again. That is why
// no page ends on a stray rule above its action bar.
func TestPreferenceRowIndexIsPerFrame(t *testing.T) {
	preferenceRowFrame, preferenceRowCount = time.Time{}, 0
	base := time.Now()
	for want := 0; want < 3; want++ {
		if got := nextPreferenceRowIndex(base); got != want {
			t.Fatalf("row %d of a frame numbered %d", want, got)
		}
	}
	if got := nextPreferenceRowIndex(base.Add(time.Millisecond)); got != 0 {
		t.Fatalf("next frame kept counting: %d", got)
	}
}
