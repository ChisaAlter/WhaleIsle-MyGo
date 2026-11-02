package sidebar

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// These integration tests render the complete Sidebar View into ui.NewTester.
// All pointer/key events and clipboard values stay in its memory-only host;
// they do not create native windows or prove OS/window-manager behavior.
type noSystemInputMove struct {
	Generation              uint64
	Account, Source, Before string
	Local                   bool
}

type noSystemInputFixture struct {
	Model               Model
	Moves               []noSystemInputMove
	Selections, Renames []string
	Start               time.Time
}

func newNoSystemInputSidebar(t *testing.T, grouping string) (*ui.Tester, *noSystemInputFixture) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	f := &noSystemInputFixture{Start: time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)}
	f.Model = Model{
		ViewportW: 960, ViewportH: 720, ConnectionReady: true,
		Snapshot: session.ViewState{
			SelectedID: "a", WorkspaceReady: true, ClientReady: true,
			WorkspaceMembers: map[string]bool{"a": true, "b": true, "c": true, "host-hidden": true},
			Items: []session.Item{
				{SessionID: "a", Title: "Current session Alpha", UpdatedAt: 3000},
				{SessionID: "b", Title: "  Original title B with preserved outer spaces and a long middle that exceeds the sidebar lane  ", UpdatedAt: 2000},
				{SessionID: "c", Title: "Drag session Gamma with a long title beyond the visible lane", UpdatedAt: 1000},
			},
			Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Owned workspace", Path: "/owned/remote/workspace", SessionIDs: []string{"a", "host-hidden", "b", "c"}}},
			WorkspaceLayout: session.WorkspaceLayoutState{
				HostGeneration: 137, GroupBy: grouping, OrderBy: "manual", ArchivedFilter: "default",
				Collapsed:     map[string]bool{"ws": false},
				SessionOrders: map[string][]string{"ws": {"a", "b", "c", "host-hidden"}, "::flat": {"a", "b", "c"}},
			},
		},
	}
	f.Model.OnSelect = func(id string) { f.Selections = append(f.Selections, id); f.Model.Snapshot.SelectedID = id }
	f.Model.OnBeginRename = func(id, title string) { f.Renames = append(f.Renames, id+":"+title) }
	f.Model.OnMoveWorkspaceSessionBefore = func(account, source, before string) {
		f.Moves = append(f.Moves, noSystemInputMove{Account: account, Source: source, Before: before})
	}
	f.Model.OnMoveLocalSessionBefore = func(generation uint64, account, source, before string) {
		f.Moves = append(f.Moves, noSystemInputMove{Generation: generation, Account: account, Source: source, Before: before, Local: true})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			ui.Column(c).Width(280).FillHeight().Children(func() { View(c, f.Model) })
			ui.Box(c).Grow(1).FillHeight().Label("In-memory conversation area")
		})
	}, 960, 720)
	tt.SetScale(1.5) // 144-DPI equivalent; Find/input coordinates remain DIPs.
	tt.SetPreferences(ui.Preferences{ReduceMotion: false, TextScale: 1})
	tt.SetTime(f.Start)
	if got := tt.Image().Bounds(); got.Dx() != 1440 || got.Dy() != 1080 {
		t.Fatalf("wrong 1.5-scale memory surface: %v", got)
	}
	return tt, f
}

// Read the actual full View's retained state, not a pointer injected into a
// separate helper fixture. Reflection is read-only: no unsafe, Interface on
// private fields, mutation of runtime state, or public debug hook is used.
func noSystemInputActualMarker(t *testing.T, tt *ui.Tester) (uint64, *sessionDropMarker) {
	t.Helper()
	rt := reflect.ValueOf(tt).Elem().FieldByName("rt").Elem()
	states := rt.FieldByName("states")
	ownerCount := 0
	var generation uint64
	var marker *sessionDropMarker
	for it := states.MapRange(); it.Next(); {
		locals := it.Value().Elem().FieldByName("locals")
		for entries := locals.MapRange(); entries.Next(); {
			value := entries.Value()
			if value.Kind() == reflect.Interface {
				value = value.Elem()
			}
			if !value.IsValid() || value.Type() != reflect.TypeOf((*sidebarDragState)(nil)) {
				continue
			}
			key := entries.Key()
			if key.Kind() == reflect.Interface {
				key = key.Elem()
			}
			if key.Kind() != reflect.String || key.String() != "sidebar-drag-state" {
				t.Fatal("drag owner stored under unexpected identity")
			}
			ownerCount++
			owner := value.Elem()
			generation = owner.FieldByName("HostGeneration").Uint()
			stored := owner.FieldByName("Session")
			if stored.IsNil() {
				continue
			}
			stored = stored.Elem()
			source := stored.FieldByName("Source")
			marker = &sessionDropMarker{
				Source: sessionDrag{Generation: source.FieldByName("Generation").Uint(), Workspace: source.FieldByName("Workspace").String(), ID: source.FieldByName("ID").String(), Pinned: source.FieldByName("Pinned").Bool(), Scope: noSystemInputReadScope(source.FieldByName("Scope"))},
				Scope:  noSystemInputReadScope(stored.FieldByName("Scope")), Target: stored.FieldByName("Target").String(), After: stored.FieldByName("After").Bool(), Local: stored.FieldByName("Local").Bool(),
			}
		}
	}
	if ownerCount != 1 {
		t.Fatalf("full Sidebar has %d retained drag owners, want exactly one", ownerCount)
	}
	return generation, marker
}

func noSystemInputReadScope(v reflect.Value) sidebarDragScope {
	return sidebarDragScope{Generation: v.FieldByName("Generation").Uint(), GroupBy: v.FieldByName("GroupBy").String(), OrderBy: v.FieldByName("OrderBy").String(), ArchiveFilter: v.FieldByName("ArchiveFilter").String(), Search: v.FieldByName("Search").String(), Batch: v.FieldByName("Batch").Bool()}
}

// Require the semantic label to identify one real visible control. Title text
// also occurs in its descendant; choose the unique labeled row via its retained
// title-crawl state rather than Find's first matching duplicate.
func noSystemInputUniqueRect(t *testing.T, tt *ui.Tester, text string, rowOnly bool) ui.Rect {
	t.Helper()
	rt := reflect.ValueOf(tt).Elem().FieldByName("rt").Elem()
	labels, states := rt.FieldByName("labels"), rt.FieldByName("states")
	matches := 0
	var box ui.Rect
	for i := 0; i < labels.Len(); i++ {
		label := labels.Index(i)
		if label.FieldByName("text").String() != text {
			continue
		}
		r := label.FieldByName("r")
		if r.FieldByName("W").Float() <= 0 || r.FieldByName("H").Float() <= 0 {
			continue
		}
		if rowOnly {
			state := states.MapIndex(label.FieldByName("id"))
			if !state.IsValid() {
				continue
			}
			isSessionRow := false
			locals := state.Elem().FieldByName("locals")
			for entry := locals.MapRange(); entry.Next(); {
				key := entry.Key()
				if key.Kind() == reflect.Interface {
					key = key.Elem()
				}
				if key.Kind() == reflect.String && strings.HasPrefix(key.String(), "title-crawl:") {
					isSessionRow = true
				}
			}
			if !isSessionRow {
				continue
			}
		}
		matches++
		box = ui.Rect{X: float32(r.FieldByName("X").Float()), Y: float32(r.FieldByName("Y").Float()), W: float32(r.FieldByName("W").Float()), H: float32(r.FieldByName("H").Float())}
	}
	if matches != 1 {
		t.Fatalf("label %q identified %d visible controls (rowOnly=%v); texts=%q", text, matches, rowOnly, tt.Texts())
	}
	return box
}

func noSystemInputSnapshot(t *testing.T, f *noSystemInputFixture) string {
	t.Helper()
	value := struct {
		Workspaces []session.WorkspaceRow
		Members    map[string]bool
		Orders     map[string][]string
		Selected   string
	}{f.Model.Snapshot.Workspaces, f.Model.Snapshot.WorkspaceMembers, f.Model.Snapshot.WorkspaceLayout.SessionOrders, f.Model.Snapshot.SelectedID}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestNoSystemInputFullSidebarDragMarkerOwnershipAndCompletion(t *testing.T) {
	for _, grouping := range []string{"workspace", "flat"} {
		for _, completion := range []string{"outside release", "escape"} {
			t.Run(grouping+"/"+completion, func(t *testing.T) {
				tt, f := newNoSystemInputSidebar(t, grouping)
				original := noSystemInputSnapshot(t, f)
				source := noSystemInputUniqueRect(t, tt, f.Model.Snapshot.Items[2].Title, true)
				target := noSystemInputUniqueRect(t, tt, f.Model.Snapshot.Items[0].Title, true)
				tt.Press(source.X+65, source.Y+source.H/2) // Actual rename-enabled title owner.
				tt.Move(source.X+78, source.Y+source.H/2)
				tt.Move(target.X+65, target.Y+3)
				generation, marker := noSystemInputActualMarker(t, tt)
				account, local := "ws", false
				if grouping == "flat" {
					account, local = "::flat", true
				}
				wantScope := currentSidebarDragScope(f.Model)
				if generation != 137 || marker == nil || marker.Source.Generation != 137 || marker.Source.ID != "c" || marker.Source.Workspace != account || marker.Source.Pinned || marker.Target != "a" || marker.After || marker.Local != local || marker.Scope != wantScope || marker.Source.Scope != wantScope {
					t.Fatalf("wrong actual retained marker owner generation=%d marker=%+v", generation, marker)
				}
				captured := *marker
				tt.Move(880, 650) // Outside sidebar, still only an in-memory event.
				_, marker = noSystemInputActualMarker(t, tt)
				if marker == nil || *marker != captured {
					t.Fatalf("leaving sidebar lost/changed the last valid owned marker: %+v", marker)
				}
				if completion == "escape" {
					tt.Key(0, ui.KeyEscape)
				}
				tt.Release(880, 650)
				tt.Frame()
				_, marker = noSystemInputActualMarker(t, tt)
				if marker != nil {
					t.Fatalf("completed drag retained authority: %+v", marker)
				}
				if completion == "escape" {
					if len(f.Moves) != 0 {
						t.Fatalf("Escape committed: %+v", f.Moves)
					}
				} else {
					want := noSystemInputMove{Account: account, Source: "c", Before: "a", Local: local}
					if local {
						want.Generation = 137
					}
					if !reflect.DeepEqual(f.Moves, []noSystemInputMove{want}) {
						t.Fatalf("wrong completion callback: %+v want %+v", f.Moves, want)
					}
				}
				if len(f.Selections) != 0 || len(f.Renames) != 0 || noSystemInputSnapshot(t, f) != original {
					t.Fatal("drag changed selected session, rename authority, saved local order or authoritative Host members")
				}
			})
		}
	}
}

func TestNoSystemInputFullSidebarWholeTitleCopyAtNormalMotionAndScale(t *testing.T) {
	tt, f := newNoSystemInputSidebar(t, "workspace")
	original := noSystemInputSnapshot(t, f)
	title := f.Model.Snapshot.Items[1].Title
	if strings.TrimSpace(title) == title {
		t.Fatal("fixture needs exact outer whitespace")
	}
	row := noSystemInputUniqueRect(t, tt, title, true)
	tt.SetClipboard("memory clipboard sentinel")
	tt.Move(row.X+65, row.Y+row.H/2)
	tt.SetTime(f.Start.Add(799 * time.Millisecond))
	if tt.HasText("Session preview: b") || tt.Clipboard() != "memory clipboard sentinel" {
		t.Fatal("full sidebar opened/copied before800ms")
	}
	tt.SetTime(f.Start.Add(800 * time.Millisecond))
	noSystemInputUniqueRect(t, tt, "Session preview: b", false)
	card := noSystemInputUniqueRect(t, tt, "Copy session title b", false)
	if card.W != 244 || card.H <= 64 {
		t.Fatalf("full-title compact card geometry=%+v", card)
	}
	// Cross half of the8DIP gap, then enter the whole card during200ms grace.
	tt.Move(row.X+row.W+4, row.Y+8)
	tt.SetTime(f.Start.Add(900 * time.Millisecond))
	tt.Move(card.X+card.W/2, card.Y+card.H/2)
	tt.SetTime(f.Start.Add(950 * time.Millisecond))
	tt.ClickAt(card.X+card.W/2, card.Y+card.H/2)
	tt.Frame()
	if tt.Clipboard() != title || !tt.HasText("Copied") {
		t.Fatalf("whole card did not copy exact spaced title: %q texts=%q", tt.Clipboard(), tt.Texts())
	}
	feedback := noSystemInputUniqueRect(t, tt, "Copy session title b", false)
	if feedback != card {
		t.Fatalf("one-second feedback moved/resized card: before=%+v after=%+v", card, feedback)
	}
	tt.SetTime(f.Start.Add(1949 * time.Millisecond))
	if !tt.HasText("Copied") {
		t.Fatal("copy feedback ended before one second")
	}
	tt.SetTime(f.Start.Add(1950 * time.Millisecond))
	if tt.HasText("Copied") || !tt.HasText(title) {
		t.Fatal("full title did not return after one second")
	}
	if noSystemInputSnapshot(t, f) != original || len(f.Selections) != 0 || len(f.Renames) != 0 || len(f.Moves) != 0 {
		t.Fatal("copy changed selected session, navigation, saved local orders or Host membership")
	}
}
