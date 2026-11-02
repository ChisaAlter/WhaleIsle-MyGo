package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSessionMenuBoundActionsKeepAddressedTargetAndSelection(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	item := session.Item{SessionID: "addressed-b", Title: "B saved title"}
	var renamed []string
	m := Model{Snapshot: session.ViewState{SelectedID: "current-a"},
		MenuShortcutLabels: map[string]string{"session.rename": "Ctrl+Alt+F8", "session.fork": "Ctrl+Shift+J", "session.archive": "Ctrl+Shift+E"},
		OnBeginRename:      func(id, title string) { renamed = append(renamed, id+"/"+title) },
		OnFork:             func(string) {}, OnArchive: func(string) {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Button(c, "Menu owner").Menu(sessionRowMenu(m, item))
	}, 620, 320)
	if err := tt.Click("Menu owner"); err != nil {
		t.Fatal(err)
	}
	// Tester keeps a native-menu simulation rather than drawing menu labels.
	// The toolkit's real-overlay tests verify ShortcutLabel presentation;
	// this application test checks the addressed action and selection boundary.
	if len(tt.Menu()) != 3 {
		t.Fatalf("menu actions: %v", tt.Menu())
	}
	if len(renamed) != 0 {
		t.Fatal("rendering a shortcut invoked an action")
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarRename)); err != nil {
		t.Fatal(err)
	}
	if len(renamed) != 1 || renamed[0] != "addressed-b/B saved title" || m.Snapshot.SelectedID != "current-a" {
		t.Fatalf("wrong target/actions: %v selected=%q", renamed, m.Snapshot.SelectedID)
	}
	m.MenuShortcutLabels["session.rename"] = ""
	if err := tt.Click("Menu owner"); err != nil {
		t.Fatal(err)
	}
	if len(tt.Menu()) != 3 {
		t.Fatal("removing a shortcut removed the action")
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarRename)); err != nil {
		t.Fatal(err)
	}
	if len(renamed) != 2 || renamed[1] != "addressed-b/B saved title" {
		t.Fatalf("unbound menu action: %v", renamed)
	}
}
