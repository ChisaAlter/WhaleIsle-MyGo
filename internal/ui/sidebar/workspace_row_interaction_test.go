package sidebar

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// The entire non-action area toggles only that workspace. Hover actions are
// independent and pass both the drawn Host generation and addressed identity.
func TestWorkspaceHeaderOnlyTogglesAndNewButtonTargetsHoveredGroup(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var toggled, selected, created []string
	var generations []uint64
	m := Model{Snapshot: session.ViewState{SelectedID: "keep", WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 42},
		Workspaces: []session.WorkspaceRow{{WorkspaceID: "one", Title: "First workspace", Path: "/one"}, {WorkspaceID: "two", Title: "Second workspace", Path: "/two"}}},
		OnToggleWorkspaceGroup: func(id string) { toggled = append(toggled, id) },
		OnSelectWorkspace:      func(id string) { selected = append(selected, id) },
		OnSelect:               func(id string) { selected = append(selected, id) },
		OnCreateWorkspaceSession: func(generation uint64, id string) {
			generations = append(generations, generation)
			created = append(created, id)
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, nil, ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), ui.Transparent) })
	}, 320, 200)
	for _, test := range []struct{ title, id string }{{"First workspace", "one"}, {"Second workspace", "two"}} {
		row, ok := tt.Find(test.title)
		if !ok {
			t.Fatalf("missing row %q: %q", test.title, tt.Texts())
		}
		// Hover reveals unreserved actions and narrows the title seat. Settle
		// that layout first; each subsequent click uses the live title bounds.
		tt.Move(row.X+row.W/2, row.Y+row.H/2)
		tt.Frame()
		for _, position := range []string{"left", "middle", "right"} {
			row, ok = tt.Find(test.title)
			if !ok {
				t.Fatalf("missing live title %q", test.title)
			}
			x := row.X + 2
			if position == "middle" {
				x = row.X + row.W/2
			} else if position == "right" {
				x = row.X + row.W - 2
			}
			tt.ClickAt(x, row.Y+row.H/2)
		}
		if err := tt.Click(workspaceText("折叠工作区 ", "Toggle workspace ") + test.id); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(toggled, []string{"one", "one", "one", "one", "two", "two", "two", "two"}) || len(selected) != 0 || len(created) != 0 {
		t.Fatalf("row clicks toggled=%v selected=%v created=%v", toggled, selected, created)
	}
	for _, test := range []struct{ title, id string }{{"First workspace", "one"}, {"Second workspace", "two"}} {
		row, _ := tt.Find(test.title)
		tt.Move(row.X+row.W/2, row.Y+row.H/2)
		tt.Frame()
		label := workspaceText("新建会话 ", "New session ") + test.id
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("hover action missing %q: %q", label, tt.Texts())
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(created, []string{"one", "two"}) || !reflect.DeepEqual(generations, []uint64{42, 42}) || len(selected) != 0 || len(toggled) != 8 {
		t.Fatalf("action created=%v generations=%v selected=%v toggled=%v", created, generations, selected, toggled)
	}
}

func TestScratchHeaderUsesHostOwnedCreationAndKeepsFallback(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "host-owned", true: "legacy-fallback"}[fallback], func(t *testing.T) {
			var toggled, created []string
			var generations []uint64
			var scratch int
			items := []session.Item{{SessionID: "unfiled", Cwd: "/scratch"}}
			m := Model{Snapshot: session.ViewState{Items: items, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 9, Collapsed: map[string]bool{"::scratch": true}}, ScratchCwd: "/scratch"},
				OnToggleWorkspaceGroup: func(id string) { toggled = append(toggled, id) },
				OnNoDirectory:          func() { scratch++ },
			}
			if !fallback {
				m.OnCreateWorkspaceSession = func(generation uint64, id string) {
					generations = append(generations, generation)
					created = append(created, id)
				}
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(1, 2, 3), ui.RGB(4, 5, 6), ui.Transparent) })
			}, 320, 120)
			row, ok := tt.Find("No working directory")
			if !ok {
				t.Fatalf("scratch row missing: %q", tt.Texts())
			}
			tt.Move(row.X+30, row.Y+row.H/2)
			tt.Frame()
			if err := tt.Click(workspaceText("无目录会话", "No directory session")); err != nil {
				t.Fatal(err)
			}
			if fallback {
				if scratch != 1 || !reflect.DeepEqual(toggled, []string{"::scratch"}) || len(created) != 0 {
					t.Fatalf("scratch=%d toggled=%v created=%v", scratch, toggled, created)
				}
			} else if scratch != 0 || len(toggled) != 0 || !reflect.DeepEqual(created, []string{"::scratch"}) || !reflect.DeepEqual(generations, []uint64{9}) {
				t.Fatalf("scratch=%d toggled=%v created=%v generations=%v", scratch, toggled, created, generations)
			}
		})
	}
}
