package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSelectedRowRetainsExplicitPluginOpenAndStopWithoutSelectingAnotherRow(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	opened, stopped := "", ""
	active := false
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Snapshot: session.ViewState{SelectedID: "a"}, SessionPluginsActive: active,
			OnOpenSessionPlugins:  func(id string) { opened, active = id, true },
			OnCloseSessionPlugins: func(id string) { stopped, active = id, false },
			OnSelect:              func(string) { t.Fatal("plugin menu changed selected session") },
			OnArchive:             func(string) {},
		}
		ui.Column(c).Fill().Children(func() {
			for _, id := range []string{"a", "b"} {
				rowSession(c.Key(id), session.Item{SessionID: id, Title: id, Blank: id == "a"}, 0, m, c.Theme().Text, c.Theme().TextMuted, ui.Transparent)
			}
		})
	}, 280, 120)
	if err := tt.Click(sessionActionsLabel("b", "b")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyCmdSessionPluginTitle)); err == nil {
		t.Fatal("foreign row offered selected session plugin")
	}
	tt.CloseMenu()
	if _, ok := tt.Find("Session time a"); ok {
		t.Fatal("blank draft gained a trailing time")
	}
	if err := tt.Click(sessionActionsLabel("a", "a")); err != nil {
		t.Fatal(err)
	}
	for _, label := range tt.Menu() {
		if label == i18n.T(i18n.KeySidebarArchive) || label == i18n.T(i18n.KeySidebarRename) {
			t.Fatal("blank plugin entry exposed general draft actions")
		}
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeyCmdSessionPluginTitle)); err != nil {
		t.Fatal(err)
	}
	if opened != "a" || !active {
		t.Fatalf("open=%q active=%v", opened, active)
	}
	tt.Frame()
	if err := tt.Click(sessionActionsLabel("a", "a")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySessionPluginsStop)); err != nil {
		t.Fatal(err)
	}
	if stopped != "a" || active {
		t.Fatalf("stop=%q active=%v", stopped, active)
	}
}
