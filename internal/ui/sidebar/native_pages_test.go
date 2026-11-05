package sidebar

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/session"
)

func TestNativeSidebarPagesUseLiveHandlesAndIgnoreOtherSlots(t *testing.T) {
	pages := []pluginui.NativeView{
		{Handle: "a-live-handle", Slot: "sidebar.page", ID: "shared-id", Label: "Plugin page A", Registrant: "owner-a"},
		{Handle: "b-live-handle", Slot: "main", ID: "shared-id", Label: "Plugin page B", Registrant: "owner-b"},
		{Handle: "settings-handle", Slot: "settings.section", ID: "settings-id", Label: "Not a global page"},
		{Handle: "label-missing", Slot: "main", ID: "Never invent this label"},
		{Handle: "label-whitespace", Slot: "sidebar.page", ID: "Whitespace is not a label", Label: "  \t"},
		{Slot: "main", ID: "no-live-handle", Label: "Unroutable page"},
	}
	for _, collapsed := range []bool{false, true} {
		t.Run(fmt.Sprintf("collapsed-%v", collapsed), func(t *testing.T) {
			selected := ""
			m := Model{Collapsed: collapsed, ThemeID: "deepseek.light", NativePages: pages, OnToggle: func() {}, Snapshot: session.ViewState{WorkspaceReady: true}}
			m.OnOpenNativePage = func(handle string) { selected, m.ActivePanel = handle, handle }
			width := 280
			if collapsed {
				width = 56
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, width, 800)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			for _, hidden := range []string{"Not a global page", "Never invent this label", "Whitespace is not a label", "Unroutable page"} {
				if _, ok := tt.Find(hidden); ok {
					t.Fatalf("invented or misrouted navigation %q", hidden)
				}
			}
			for _, page := range pages[:2] {
				if err := tt.Click(page.Label); err != nil {
					t.Fatal(err)
				}
				if selected != page.Handle || m.ActivePanel != page.Handle {
					t.Fatalf("page used label/ID instead of live handle: %q", selected)
				}
				row := navigationRect(t, tt, page.Label)
				if row.X < 0 || row.Y < 0 || row.X+row.W > float32(width) || row.Y+row.H > 800 {
					t.Fatalf("global page outside sidebar: %+v", row)
				}
			}
			navigationEvidence(t, tt, fmt.Sprintf("native-plugin-pages-collapsed-%v", collapsed))
			m.NativePages = nil
			tt.Frame()
			if _, ok := tt.Find(pages[0].Label); ok {
				t.Fatal("withdrawn native page retained a navigation entry")
			}
			m.NativePages, m.OnOpenNativePage = pages, nil
			tt.Frame()
			if _, ok := tt.Find(pages[0].Label); ok {
				t.Fatal("unowned native page rendered a dead control")
			}
		})
	}
}

func TestNativeSidebarPageKeyboardActivatesSameLiveHandle(t *testing.T) {
	opened := ""
	m := Model{NativePages: []pluginui.NativeView{{Handle: "keyboard-handle", Slot: "main", ID: "keyboard-id", Label: "Keyboard plugin page"}}, OnOpenNativePage: func(handle string) { opened = handle }, Snapshot: session.ViewState{WorkspaceReady: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 700)
	for count := 0; count < 12 && !tt.Focused("Keyboard plugin page"); count++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Keyboard plugin page") {
		t.Fatal("native global entry is not keyboard reachable")
	}
	tt.Key(0, ui.KeyEnter)
	if opened != "keyboard-handle" {
		t.Fatalf("keyboard route=%q", opened)
	}
}
