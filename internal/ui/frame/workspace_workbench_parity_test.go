package frame

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/files"
)

func TestAuthorizedProjectFileUsesOwnRootAndCheckedSaveClose(t *testing.T) {
	globalRoot, projectRoot := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "report.md"), []byte("saved Project report"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &WorkspacePanes{}
	w.syncRoute("files")
	ordinary := w.Root
	ordinary.Files = &WorkspaceFileState{Selected: "original.txt", Preview: "saved conversation", Draft: "conversation draft", Editing: true}
	ordinary.Files.Tabs.Workspaces = map[string]*files.WorkspaceTabs{globalRoot: {Active: "original.txt", Open: []*files.FileTab{{Path: "original.txt", Preview: "saved conversation", Draft: "conversation draft", Editing: true}}}}
	var adapter WorkspaceFileState
	sizes := DefaultLayout()
	sizes.Workspace = w
	store := session.New()
	store.Select("selected-conversation")
	m := adapter.model(Model{Sizes: &sizes, Store: store, WorkspaceRoot: globalRoot, FileBindingID: "selected-host/session"})
	ordinary.Files.publish(m)
	w.legacyFileID = ordinary.ID
	target := projectflow.FileTarget{ProjectID: "project", CoordinatorSessionID: "coordinator", Root: projectRoot, CanonicalPath: filepath.Join(projectRoot, "report.md")}
	var valid atomic.Bool
	valid.Store(true)
	var writes atomic.Int32
	access := ProjectFileAccess{Valid: valid.Load, Model: files.Model{OnSaveEdit: func(rel, text string) error {
		writes.Add(1)
		full, err := workspacefs.JoinRoot(projectRoot, rel)
		if err != nil {
			return err
		}
		return workspacefs.WriteFileText(full, text)
	}}}
	if err := w.OpenAuthorizedProjectFile(target, "project-host/coordinator", access); err != nil {
		t.Fatal(err)
	}
	pane := w.Root.find(w.Active)
	if pane == ordinary || pane.Files.ProjectTarget == nil {
		t.Fatal("Project reused the global Files occurrence")
	}
	t.Cleanup(w.DisposeContent)
	w.BeginContentFrame(m.FileBindingID, m.WorkspaceRoot)
	projectModel := pane.Files.projectFileModel()
	if projectModel.WorkspaceRoot != projectRoot || projectModel.SessionID != "coordinator" || projectModel.BindingID != "project-host/coordinator" {
		t.Fatal("Project authority was rebound to selected conversation")
	}
	projected := w.beginFileProjection(m)
	w.finishFileProjection(projected)
	if m.WorkspaceRoot != globalRoot || *m.FileEditDraft != "conversation draft" || ordinary.Files.Draft != "conversation draft" {
		t.Fatal("Project projection replaced the conversation root/draft")
	}
	tt := ui.NewTester(func(c *ui.Context) { files.View(c, pane.Files.projectFileModel()) }, 600, 400)
	waitWorkspaceClose(t, func() bool { tt.Frame(); return pane.Files.Preview == "saved Project report" })
	pane.Files.Draft, pane.Files.Editing = "edited Project report", true
	closeModel := workspaceCloseFilesModel(Model{workspacePane: pane})
	if pending, err := startWorkspacePaneCloseSave(pane, closeModel); err != nil {
		t.Fatal(pending, err)
	}
	waitWorkspaceClose(t, func() bool {
		pending, err := workspacePaneCloseSaveStatus(pane, closeModel)
		if err != nil {
			t.Fatal(err)
		}
		return !pending
	})
	data, err := os.ReadFile(target.CanonicalPath)
	if err != nil || string(data) != "edited Project report" || writes.Load() != 1 {
		t.Fatal(string(data), writes.Load(), err)
	}
	valid.Store(false)
	if err := closeModel.OnSaveEdit("report.md", "must not write"); err == nil || writes.Load() != 1 {
		t.Fatal("revoked Project authority wrote a file", err)
	}
	if !w.close(pane.ID) {
		t.Fatal("saved Project occurrence did not close")
	}
}

func TestAuthorizedExternalProjectReadNeverUsesDesktopCollision(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "same.txt")
	if err := os.WriteFile(path, []byte("unrelated desktop bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &WorkspacePanes{}
	target := projectflow.FileTarget{ProjectID: "project", CoordinatorSessionID: "coordinator", Root: root, CanonicalPath: path}
	reads := 0
	access := ProjectFileAccess{Valid: func() bool { return true }, Model: files.Model{ReadTextPageContext: func(context.Context, string, int64) (workspacefs.TextPage, error) {
		reads++
		return workspacefs.TextPage{Text: "authorized Host bytes", EOF: true}, nil
	}}}
	if err := w.OpenAuthorizedProjectFile(target, "external-host/coordinator", access); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.DisposeContent)
	pane := w.Root.find(w.Active)
	tt := ui.NewTester(func(c *ui.Context) { files.View(c, pane.Files.projectFileModel()) }, 600, 400)
	waitWorkspaceClose(t, func() bool { tt.Frame(); return pane.Files.Preview == "authorized Host bytes" })
	if reads != 1 {
		t.Fatal(reads)
	}
	for _, workspace := range pane.Files.Tabs.Workspaces {
		for _, tab := range workspace.Open {
			if !tab.Remote {
				t.Fatal("Host resource gained local save permission")
			}
		}
	}
	raw := w.Export()
	restored, err := RestoreWorkspacePanes(raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(restored.DisposeContent)
	state := restored.Root.find(restored.Active).Files
	if state.ProjectTarget == nil || state.projectAccess != nil || state.projectFileModel().OnSaveEdit != nil {
		t.Fatal("recovery serialized runtime Project authority")
	}
}

func TestInShellMiniAndRepeatedHostTerminalOccurrences(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	fileID := w.Active
	if err := w.PresentOpenedResource("mini"); err != nil || w.Root.find(fileID).Float == nil {
		t.Fatal("Files mini is not in-shell", err)
	}
	var browserID string
	w.pending = func() { target := w.selectKind(w.Root.find(fileID), "browser"); browserID = target.ID }
	if err := w.PresentOpenedResource("mini"); err != nil {
		t.Fatal(err)
	}
	action := w.pending
	w.pending = nil
	action()
	if browserID == "" || w.Root.find(browserID).Float == nil {
		t.Fatal("pending Browser preview did not float its actual occurrence")
	}
	ids := map[string]bool{}
	for range 3 {
		id := w.NewHostTerminal()
		if ids[id] || !w.Root.find(id).terminalFresh {
			t.Fatal("new terminal reused a route/occurrence")
		}
		ids[id] = true
	}
	recovered, err := RestoreWorkspacePanes(w.Export())
	if err != nil {
		t.Fatal(err)
	}
	recovered.Root.eachResource(func(n *WorkspacePane) {
		if n.Kind == "host-terminal" && n.terminalFresh {
			t.Fatal("recovery would create another Host process")
		}
	})
}

func TestPluginResourceRetainsExactOccurrenceAndConfirmsClose(t *testing.T) {
	w := &WorkspacePanes{}
	first := w.OpenPluginRightPaneTab("owner:session:tab1", "First plugin")
	if again := w.OpenPluginRightPaneTab("owner:session:tab1", "First plugin"); again != first {
		t.Fatal("same plugin occurrence cloned")
	}
	second := w.OpenPluginRightPaneTab("owner:session:tab2", "Second plugin")
	if second == first || w.Root.find(first).PluginRightPaneHandle != "owner:session:tab1" {
		t.Fatal("source handle transferred to another resource")
	}
	closeCalls := 0
	m := Model{PluginRightPaneTabs: []pluginui.RightPaneTab{{Handle: "owner:session:tab1"}, {Handle: "owner:session:tab2"}}, OnClosePluginRightPaneTab: func(handle string) {
		if handle != "owner:session:tab1" {
			t.Fatal(handle)
		}
		closeCalls++
	}}
	w.syncPluginRightPanes(m)
	if w.close(first) || closeCalls != 1 || w.Root.find(first) == nil {
		t.Fatal("unconfirmed source close removed native resource")
	}
	w.syncPluginRightPanes(m)
	if w.Root.find(first) == nil {
		t.Fatal("source refusal was ignored")
	}
	m.PluginRightPaneTabs = m.PluginRightPaneTabs[1:]
	w.syncPluginRightPanes(m)
	if w.Root.find(first) != nil || w.Root.find(second) == nil {
		t.Fatal("source close confirmation affected wrong occurrence")
	}
}

func TestFullscreenPadReserveKeepsCaptionFullWidth(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("browser")
	w.Fullscreen = w.Active
	sizes := DefaultLayout()
	sizes.Workspace = w
	route := "browser"
	m := Model{Sizes: &sizes, Store: session.New(), RightPaneOpen: true, RightPaneTab: &route, ViewportW: 1200, RightOverlayReserve: 320,
		BrowserContent: func(c *ui.Context) { ui.Box(c).Label("Reserved Browser body").Fill() }}
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceColumns(c, m, func() {}, func() { ui.Box(c).Label("Full-width caption").FillWidth().Height(48) })
	}, 1200, 700)
	caption, ok := tt.Find("Full-width caption")
	if !ok || caption.W != 1200 {
		t.Fatal(caption)
	}
	dock, ok := tt.Find("Workspace resource dock")
	if !ok || dock.W != 880 {
		t.Fatal("Pad did not reserve body room exactly once", dock)
	}
}

func TestFileOccurrenceDoesNotBorrowOtherOpenWithRequest(t *testing.T) {
	selected := "different.txt"
	state := WorkspaceFileState{Selected: "this.txt"}
	m := state.model(Model{SelectedFile: &selected, FileOpenWithApps: []files.OpenWithApp{{ID: "other-app"}}, FileOpenWithState: &files.OpenWithState{Loading: true}, OnRefreshFileOpenWith: func() { t.Fatal("refreshed the wrong file") }})
	if m.FileOpenWithApps != nil || m.FileOpenWithState != nil || m.OnRefreshFileOpenWith != nil {
		t.Fatal("unfocused occurrence borrowed another file's request")
	}
}
