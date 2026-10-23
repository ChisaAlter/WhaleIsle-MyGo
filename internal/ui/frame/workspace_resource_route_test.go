package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceResourceMenuOpensRetainedInspectorThroughFrame(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(oldLocale)
	sizes := DefaultLayout()
	sizes.Bottom = 0
	selected := "files"
	right := float32(540)
	open := true
	store := session.New()
	store.Select("retained-session")
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: store, Sizes: &sizes, RightPaneOpen: open, RightPaneTab: &selected, FilePaneWidth: &right,
			OnToggleTools: func() { open = !open }})
	}, 1600, 900)
	if err := tt.Click("Pane actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Session Log · Inspect this session's raw events"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		tt.Frame()
		if !open || selected != "session-inspector-log" || !tt.HasText("Raw Log") || !tt.HasText("No records") {
			t.Fatalf("inspector route lost after menu choice: open=%v route=%q texts=%q", open, selected, tt.Texts())
		}
	}
}
