package frame

import (
	"testing"
	"whaleisle/internal/ui/browserpreview"
)

// Model/interaction fixture only: root runs after the shared source freeze.
// Removing the float transition, geometry clamp, or retained pointer breaks it.
func TestWorkspaceFloatLiveResource(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	files := w.Root
	browser := w.selectKind(files, "browser")
	controller := &browserpreview.Controller{}
	browser.browser = controller
	terminal := w.selectKind(browser, "host-terminal")
	terminal.TerminalSource = "existing-pty"
	if !w.FloatWorkspaceTab(browser.ID) || browser.browser != controller || w.Root.find(browser.ID) != browser {
		t.Fatal("float replaced live resource")
	}
	if w.Root.dockedLeaf() != terminal || !w.visible("browser") {
		t.Fatal("float hid remaining dock or live float")
	}
	w.MoveWorkspaceFloat(browser.ID, -20, -50)
	w.ResizeWorkspaceFloat(browser.ID, 10, 20)
	if browser.Float.X != 0 || browser.Float.Y != workspaceFloatTop || browser.Float.Width != 220 || browser.Float.Height != 140 {
		t.Fatal("float bounds not constrained")
	}
	fitted := boundedFloat(WorkspaceFloat{X: 900, Y: 700, Width: 380, Height: 300}, 640, 480)
	if fitted.X != 260 || fitted.Y != 180 {
		t.Fatal("viewport did not retain reachable controls")
	}
	w.MinimizeWorkspaceFloat(browser.ID, true)
	if w.visible("browser") || browser.browser != controller {
		t.Fatal("minimize destroyed content or retained visibility")
	}
	w.MinimizeWorkspaceFloat(browser.ID, false)
	if !w.visible("browser") {
		t.Fatal("restore failed")
	}
	if !w.FloatWorkspaceTab(terminal.ID) || terminal.TerminalSource != "existing-pty" {
		t.Fatal("float changed PTY identity")
	}
	browser.browser = nil // Do not invoke a zero-value native controller during export.
	saved, err := RestoreWorkspacePanes(w.Export())
	if err != nil {
		t.Fatal(err)
	}
	restored := saved.Root.find(browser.ID)
	if restored.Float == nil || restored.Float.Width != 220 || saved.Root.find(terminal.ID).TerminalSource != "existing-pty" {
		t.Fatal("float persistence lost resource/geometry")
	}
	browser.browser = controller
	if !w.RedockWorkspaceTab(browser.ID, files.ID) || browser.Float != nil || browser.browser != controller {
		t.Fatal("redock replaced live resource")
	}
	if !w.RedockWorkspaceTab(terminal.ID, browser.ID) || terminal.TerminalSource != "existing-pty" {
		t.Fatal("redock changed PTY")
	}
	if len(w.Root.dockTabs()) != 3 {
		t.Fatal("redock duplicated resource")
	}
}

// Original observer tab flags apply to command/API paths, not just buttons.
func TestWorkspaceObserverRejectsUserCloseAndDock(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("files")
	files := w.Root
	observer := w.selectKind(files, "project-observer")
	w.FloatWorkspaceTab(observer.ID)
	if w.close(observer.ID) {
		t.Fatal("observer accepted user close")
	}
	w.closeNow(observer.ID)
	w.CloseKind("project-observer")
	if w.Root.find(observer.ID) != observer {
		t.Fatal("close command removed observer")
	}
	if w.RedockWorkspaceTab(observer.ID, files.ID) || w.MoveWorkspaceTab(observer.ID, files.ID, 0) || w.MoveWorkspaceTab(files.ID, observer.ID, 0) {
		t.Fatal("observer accepted dock/drop")
	}
	if observer.Float == nil || w.Root.find(observer.ID) != observer {
		t.Fatal("rejected intent mutated observer")
	}
	// Internal lifecycle removal must remain possible for resets/Host teardown.
	w.disposeWorkspaceResource(observer.ID)
	if w.Root.find(observer.ID) != nil || w.Root.find(files.ID) != files {
		t.Fatal("internal cleanup blocked or removed unrelated resource")
	}
}
