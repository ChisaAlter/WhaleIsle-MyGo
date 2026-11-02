package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// The host row has padding beyond its lead/title buttons. Those pixels are
// still part of the click target; actions remain independent inside the row.
func TestWorkspaceHeaderPaddingTogglesWithoutOtherActions(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var toggled, selected, created []string
	m := Model{Snapshot: session.ViewState{SelectedID: "kept", WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 3},
		Workspaces: []session.WorkspaceRow{{WorkspaceID: "one", Title: "First workspace", Path: "/one"}, {WorkspaceID: "two", Title: "Second workspace", Path: "/two"}}},
		OnToggleWorkspaceGroup:   func(id string) { toggled = append(toggled, id) },
		OnSelectWorkspace:        func(id string) { selected = append(selected, id) },
		OnCreateWorkspaceSession: func(_ uint64, id string) { created = append(created, id) },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, nil, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), ui.Transparent) })
	}, 320, 200)
	for _, title := range []string{"First workspace", "Second workspace"} {
		row, ok := tt.Find(title)
		if !ok {
			t.Fatalf("missing %q", title)
		}
		// The root starts at x=0 and is 320 DIP wide. These positions sit in
		// the 8-DIP left/right padding, outside the title, lead and actions.
		tt.ClickAt(2, row.Y+row.H/2)
		tt.ClickAt(318, row.Y+row.H/2)
	}
	if !reflect.DeepEqual(toggled, []string{"one", "one", "two", "two"}) || len(selected) != 0 || len(created) != 0 {
		t.Fatalf("padding clicks: toggled=%v selected=%v created=%v", toggled, selected, created)
	}
}
