package frame

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceFileRefreshAfterNavigationShellProjection(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	root := t.TempDir()
	path := filepath.Join(root, "refresh.txt")
	if err := os.WriteFile(path, []byte("saved original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &WorkspacePanes{}
	w.syncRoute("files")
	sizes := DefaultLayout()
	sizes.Workspace, sizes.Right, sizes.Bottom = w, 540, 0
	store := session.New()
	store.Select("refresh-session")
	var adapter WorkspaceFileState
	adapter.Tabs.Preferences = &files.PreviewPreferences{}
	route := "files"
	base := Model{Store: store, Sizes: &sizes, RightPaneOpen: true, RightPaneTab: &route, ExternalNavigation: true, WorkspaceRoot: root, FileBindingID: "refresh-host/session"}
	tt := ui.NewTester(func(c *ui.Context) {
		// Production main snapshots adapter values before the outer shell's
		// commands execute. A prior-frame callback must not mutate the pane
		// until the inner View has finished consuming this stale snapshot.
		snapshot := adapter
		m := snapshot.model(base)
		w.BeginContentFrame(m.FileBindingID, root)
		NavigationShell(c, m, func() { View(c, m) })
		w.EndContentFrame()
		adapter = snapshot
	}, 1180, 740)
	t.Cleanup(w.DisposeContent)
	if handled, err := w.OpenFileResource(files.Model{Tabs: &adapter.Tabs, BindingID: base.FileBindingID, SessionID: store.SelectedSessionID(), WorkspaceRoot: root}, "refresh.txt"); err != nil || !handled {
		t.Fatal("open file", handled, err)
	}
	var pane *WorkspacePane
	waitWorkspaceClose(t, func() bool {
		tt.Frame()
		pane = w.Root.find(w.Active)
		return pane != nil && pane.Files != nil && pane.Files.Preview == "saved original\n"
	})
	state := pane.Files
	workspace := state.Tabs.Workspaces[filepath.Clean(root)]
	var tab *files.FileTab
	for _, item := range workspace.Open {
		if item.Path == "refresh.txt" {
			tab = item
		}
	}
	if tab == nil {
		t.Fatal("file tab missing")
	}
	tab.AutoRefreshDisabled = true // Isolate the explicit refresh command.
	tab.Draft, tab.Editing = "unsaved local\n", true
	state.Draft, state.Editing = tab.Draft, true
	w.PublishSessionProjection(adapter.model(base))
	tt.Frame()
	if err := os.WriteFile(path, []byte("external disk\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("File preview toolbar"); err != nil {
		t.Fatal(err)
	}
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyR, "KeyR")
	waitWorkspaceClose(t, func() bool {
		tt.Frame()
		return tt.HasText("External edit conflicts with unsaved draft")
	})
	if state.Preview != "saved original\n" || state.Draft != "unsaved local\n" || !strings.Contains(state.Status, "conflicts") {
		t.Fatalf("refresh lost conflict versions: preview=%q draft=%q status=%q", state.Preview, state.Draft, state.Status)
	}
	disk, err := os.ReadFile(path)
	if err != nil || string(disk) != "external disk\n" {
		t.Fatal("external writer overwritten", string(disk), err)
	}
}

func TestWorkspaceRefreshRequestUsesCurrentOccurrenceOnly(t *testing.T) {
	for _, change := range []string{"same", "owner", "binding", "root", "kind", "hidden", "pause", "disposed"} {
		t.Run(change, func(t *testing.T) {
			w := &WorkspacePanes{}
			w.syncRoute("files")
			n := w.Root
			sizes := DefaultLayout()
			sizes.Workspace = w
			m := Model{Store: session.New(), Sizes: &sizes, workspacePane: n, FileBindingID: "host-a", WorkspaceRoot: "root-a"}
			calls := 0
			bindWorkspacePageRefresh(m, func() { t.Fatal("previous-frame callback executed") })
			n.refreshResource()
			switch change {
			case "owner":
				m.Store = session.New()
			case "binding":
				m.FileBindingID = "host-b"
			case "root":
				m.WorkspaceRoot = "root-b"
			case "kind":
				n.Kind = "diff"
			case "hidden":
				w.EndContentFrame()
			case "pause":
				w.PauseContent()
			case "disposed":
				n.disposeContent()
			}
			bindWorkspacePageRefresh(m, func() { calls++ })
			bindWorkspacePageRefresh(m, func() { calls++ })
			want := 0
			if change == "same" {
				want = 1
			}
			if calls != want || n.refreshPending != nil {
				t.Fatalf("calls=%d want=%d pending=%v", calls, want, n.refreshPending)
			}
		})
	}
}
