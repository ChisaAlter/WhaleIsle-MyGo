package files

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func TestDirtyExternalRereadNeverTriggersAutosaveOverwrite(t *testing.T) {
	m, done := autosaveModel(t)
	var writes atomic.Int32
	baseWrite := m.OnSaveEdit
	m.OnSaveEdit = func(rel, text string) error {
		writes.Add(1)
		return baseWrite(rel, text)
	}
	if err := os.WriteFile(filepath.Join(m.WorkspaceRoot, "b.txt"), []byte("other-b"), 0600); err != nil {
		t.Fatal(err)
	}
	*m.Editing = true
	*m.EditDraft = "unsaved-local-draft"
	capture(m)

	target := filepath.Join(m.WorkspaceRoot, "a.txt")
	if err := os.WriteFile(target, []byte("external-v2"), 0600); err != nil {
		t.Fatal(err)
	}

	// Both explicit reread and tab switch back trigger reloadSelectedFile -> reconcileTextRead.
	reloadSelectedFile(m)
	awaitFileTextRead(t, m, done)

	tab := selectedTab(m)
	if tab == nil || *m.Preview != "original-a" || *m.EditDraft != "unsaved-local-draft" ||
		!tab.hasConflict || tab.conflictContent != "external-v2" || !tab.externalChanged || !tab.readBlocked {
		t.Fatalf("reread did not preserve baseline+conflict guard: preview=%q draft=%q tab=%+v", *m.Preview, *m.EditDraft, tab)
	}

	selectFile(m, "b.txt")
	awaitFileTextRead(t, m, done)
	selectFile(m, "a.txt")
	awaitFileTextRead(t, m, done)

	tab = selectedTab(m)
	if tab == nil || *m.Preview != "original-a" || *m.EditDraft != "unsaved-local-draft" ||
		!tab.hasConflict || tab.conflictContent != "external-v2" || !tab.externalChanged || !tab.readBlocked {
		t.Fatalf("tab switch corrupted conflict guard: preview=%q draft=%q tab=%+v", *m.Preview, *m.EditDraft, tab)
	}

	// Simulate UI frames running observeAutosave while Editing stays true.
	for range 3 {
		observeAutosave(m)
	}
	time.Sleep(workspacefs.AutoSaveDelay + 150*time.Millisecond)
	syncSaveResults(m)
	if writes.Load() != 0 {
		t.Fatalf("autosave executed %d write(s) while conflict existed", writes.Load())
	}
	if disk, err := os.ReadFile(target); err != nil || string(disk) != "external-v2" {
		t.Fatalf("disk external edit overwritten: disk=%q err=%v", disk, err)
	}
	if err := requestSaveDraft(m); err == nil {
		t.Fatal("plain explicit save bypassed unresolved conflict")
	}
}

func TestConflictUIResolutionAdoptAndCASOverwrite(t *testing.T) {
	prevLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(prevLocale)

	t.Run("adopt disk version", func(t *testing.T) {
		m, done := autosaveModel(t)
		*m.Editing = true
		*m.EditDraft = "local-draft"
		capture(m)

		target := filepath.Join(m.WorkspaceRoot, "a.txt")
		if err := os.WriteFile(target, []byte("disk-v2"), 0600); err != nil {
			t.Fatal(err)
		}
		reloadSelectedFile(m)
		awaitFileTextRead(t, m, done)

		tt := ui.NewTester(func(c *ui.Context) {
			paintResourceRefreshControls(c, m)
		}, 720, 120)
		if err := tt.Click("Use disk version"); err != nil {
			t.Fatal(err)
		}
		tab := selectedTab(m)
		if *m.Preview != "disk-v2" || *m.EditDraft != "disk-v2" || tab.Dirty() ||
			tab.hasConflict || tab.conflictContent != "" || tab.externalChanged || tab.readBlocked {
			t.Fatalf("adopt disk version failed: preview=%q draft=%q tab=%+v", *m.Preview, *m.EditDraft, tab)
		}
	})

	t.Run("overwrite disk uses V2 CAS baseline and catches V3 race", func(t *testing.T) {
		m, done := autosaveModel(t)
		*m.Editing = true
		*m.EditDraft = "local-draft"
		capture(m)

		target := filepath.Join(m.WorkspaceRoot, "a.txt")
		if err := os.WriteFile(target, []byte("disk-v2"), 0600); err != nil {
			t.Fatal(err)
		}
		reloadSelectedFile(m)
		awaitFileTextRead(t, m, done)

		// Disk advances to V3 before user confirms overwrite against known V2!
		if err := os.WriteFile(target, []byte("disk-v3"), 0600); err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) {
			paintResourceRefreshControls(c, m)
		}, 720, 120)
		if err := tt.Click("Overwrite disk with draft"); err != nil {
			t.Fatal(err)
		}
		awaitAutosave(t, done)
		syncSaveResults(m)

		tab := selectedTab(m)
		if disk, err := os.ReadFile(target); err != nil || string(disk) != "disk-v3" {
			t.Fatalf("V2 CAS baseline failed to catch V3 race: disk=%q err=%v", disk, err)
		}
		if *m.Preview != "original-a" || *m.EditDraft != "local-draft" || !tab.hasConflict || tab.conflictContent != "disk-v3" {
			t.Fatalf("V3 race did not refresh conflict baseline safely: preview=%q draft=%q tab=%+v", *m.Preview, *m.EditDraft, tab)
		}

		// User confirms overwrite a second time against the refreshed V3 baseline -> succeeds.
		if err := tt.Click("Overwrite disk with draft"); err != nil {
			t.Fatal(err)
		}
		awaitAutosave(t, done)
		syncSaveResults(m)

		if disk, err := os.ReadFile(target); err != nil || string(disk) != "local-draft" {
			t.Fatalf("explicit overwrite against refreshed V3 did not write draft: disk=%q err=%v", disk, err)
		}
		if *m.Preview != "local-draft" || tab.Dirty() || tab.hasConflict || tab.conflictContent != "" || tab.externalChanged || tab.readBlocked {
			t.Fatalf("overwrite did not settle clean tab state: preview=%q tab=%+v", *m.Preview, tab)
		}
	})
}
