package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/desktop/workspacefs"
)

func autosaveModel(t *testing.T) (Model, chan struct{}) {
	t.Helper()
	root := t.TempDir()
	for p, text := range map[string]string{"a.txt": "original-a", "b.txt": "original-b"} {
		if err := os.WriteFile(filepath.Join(root, p), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selected, preview, draft, problem, status := "", "", "", "", ""
	editing := false
	tabs := &Tabs{}
	done := make(chan struct{}, 20)
	m := Model{WorkspaceRoot: root, BindingID: "host/session", Tabs: tabs, SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, EditStatus: &status, Editing: &editing, notify: func() { done <- struct{}{} }, OnSaveEdit: func(rel, text string) error {
		full, err := workspacefs.JoinRoot(root, rel)
		if err != nil {
			return err
		}
		return workspacefs.WriteFileText(full, text)
	}}
	tabs.Bind(m.BindingID, root)
	prepareTabs(m)
	selectFile(m, "a.txt")
	// Text files use the production async reader. Complete that initial read
	// before exercising save-queue behavior so the fixture is not testing the
	// read-blocked guard itself.
	awaitFileTextRead(t, m, done)
	*m.Editing = true
	observeAutosave(m)
	t.Cleanup(tabs.Dispose)
	return m, done
}
func awaitAutosave(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("save timeout")
	}
}

func awaitFileTextRead(t *testing.T, m Model, done <-chan struct{}) {
	t.Helper()
	for selectedTab(m) != nil && selectedTab(m).readBlocked {
		select {
		case <-done:
			syncTextRead(m)
		case <-time.After(3 * time.Second):
			t.Fatal("text read timeout")
		}
	}
}

func TestAutosaveKeepsNewerDraftAndFileIdentity(t *testing.T) {
	m, done := autosaveModel(t)
	// Replace the queue before changing text, so its captured writer is controlled.
	disposeSave(selectedTab(m))
	started, release := make(chan struct{}), make(chan struct{})
	write := m.OnSaveEdit
	m.OnSaveEdit = func(rel, text string) error { close(started); <-release; return write(rel, text) }
	observeAutosave(m)
	*m.EditDraft = "first-write"
	if err := requestSaveDraft(m); err != nil {
		t.Fatal(err)
	}
	awaitAutosave(t, started)
	*m.EditDraft = "newer-keystrokes"
	observeAutosave(m)
	selectFile(m, "b.txt")
	awaitFileTextRead(t, m, done)
	close(release)
	awaitAutosave(t, done)
	syncSaveResults(m)
	if *m.SelectedFile != "b.txt" || *m.Preview != "original-b" || *m.EditDraft != "original-b" {
		t.Fatal("late save changed the new file")
	}
	selectFile(m, "a.txt")
	awaitFileTextRead(t, m, done)
	if *m.Preview != "first-write" || *m.EditDraft != "newer-keystrokes" {
		t.Fatalf("newer draft lost: preview=%q draft=%q", *m.Preview, *m.EditDraft)
	}
	disk, _ := os.ReadFile(filepath.Join(m.WorkspaceRoot, "a.txt"))
	if string(disk) != "first-write" {
		t.Fatalf("switch flushed pending draft: %q", disk)
	}
}

func TestAutosaveConflictRecoveryAndBindingChange(t *testing.T) {
	m, done := autosaveModel(t)
	*m.EditDraft = "draft"
	observeAutosave(m)
	pending := selectedTab(m).save
	m.Tabs.Bind("different-host/session", m.WorkspaceRoot)
	if pending.Snapshot().Pending || selectedTab(m).save != nil {
		t.Fatal("old binding save not cancelled")
	}
	if *m.EditDraft != "draft" {
		t.Fatal("binding change lost draft")
	}
	m.BindingID = "different-host/session"
	if err := os.WriteFile(filepath.Join(m.WorkspaceRoot, "a.txt"), []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	// The conflict case exercises SaveTextChecked against the external edit;
	// mark the already captured tab as readable without starting a second
	// asynchronous reload that would replace the test's explicit baseline.
	if tab := selectedTab(m); tab != nil {
		tab.readBlocked, tab.readLoading, tab.readResume = false, false, false
	}
	if err := requestSaveDraft(m); err != nil {
		t.Fatal(err)
	}
	awaitAutosave(t, done)
	syncSaveResults(m)
	if *m.Preview != "external" || *m.EditDraft != "draft" || !strings.Contains(*m.EditStatus, "changed") {
		t.Fatalf("conflict corrupted baseline/draft: %q %q %q", *m.Preview, *m.EditDraft, *m.EditStatus)
	}
	if closeTab(m, "a.txt", false) {
		t.Fatal("dirty close bypassed")
	}
	reloadSelectedFile(m)
	awaitFileTextRead(t, m, done)
	if *m.Preview != "external" || *m.EditDraft != "draft" {
		t.Fatal("reload overwrote conflict draft")
	}
	data, err := m.Tabs.Export()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "SaveQueue") {
		t.Fatal("runtime queue serialized")
	}
	var restored Tabs
	if err := restored.Restore(data); err != nil {
		t.Fatal(err)
	}
	if restored.Workspaces[filepath.Clean(m.WorkspaceRoot)].Open[0].Draft != "draft" {
		t.Fatal("recovery lost draft")
	}
}

func TestAutosaveInitialRestoredAndRemoteDraftsDoNotFlush(t *testing.T) {
	m, _ := autosaveModel(t)
	disposeSave(selectedTab(m))
	*m.EditDraft = "recovered-draft"
	capture(m)
	observeAutosave(m)
	if selectedTab(m).save.Snapshot().Pending {
		t.Fatal("restoration started an automatic write")
	}
	disposeSave(selectedTab(m))
	selectedTab(m).Remote = true
	if err := requestSaveDraft(m); err == nil {
		t.Fatal("remote preview authorized a local mirror save")
	}
}

func TestAutosaveDoesNotRestartWhileDirtyCloseIsPending(t *testing.T) {
	m, _ := autosaveModel(t)
	// The close click can happen in the same build pass as the edit, before
	// deferred observation has seen the new draft.
	*m.EditDraft = "new-in-this-frame"
	if closeTab(m, "a.txt", false) {
		t.Fatal("closed a dirty tab")
	}
	observeAutosave(m)
	if selectedTab(m).save.Snapshot().Pending {
		t.Fatal("deferred observation restarted save behind close confirmation")
	}
	if *m.EditDraft != "new-in-this-frame" {
		t.Fatal("close lost draft")
	}
}
