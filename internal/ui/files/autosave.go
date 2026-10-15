package files

import (
	"context"
	"errors"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func ensureSaveQueue(m Model, t *FileTab) error {
	if t == nil || m.OnSaveEdit == nil {
		return errors.New("no editable file")
	}
	if t.Remote {
		return errors.New(label("远程预览草稿已保留，请使用会话远程文件保存。", "Remote preview draft kept. Save through the session's remote file editor."))
	}
	if t.save != nil {
		return nil
	}
	root, path, write := m.WorkspaceRoot, t.Path, m.OnSaveEdit
	t.save = workspacefs.NewSaveQueue(t.Preview, func(ctx context.Context, base, text string) error {
		return workspacefs.SaveTextChecked(ctx, root, path, base, text, write)
	}, m.notify)
	t.saveRevision = 0
	// Loading/restoring/revisiting a draft is not a new typing event.
	t.save.Observe(t.Draft, false)
	return nil
}

func applySaveResult(m Model, t *FileTab) {
	if t == nil || t.save == nil {
		return
	}
	s := t.save.Snapshot()
	if s.Revision != t.saveRevision {
		t.Preview = s.Confirmed
		t.saveRevision = s.Revision
		t.Error = ""
		t.Status = i18n.T(i18n.KeyFilesSaved)
	}
	if s.Error != "" {
		t.Status = s.Error
	}
	if s.Running || s.Pending {
		t.Status = label("正在保存…", "Saving…")
	}
	if m.SelectedFile != nil && *m.SelectedFile == t.Path {
		if m.Preview != nil {
			*m.Preview = t.Preview
		}
		if m.PreviewErr != nil {
			*m.PreviewErr = t.Error
		}
		if m.EditStatus != nil {
			*m.EditStatus = t.Status
		}
	}
}

func syncSaveResults(m Model) {
	w := tabsFor(m)
	if w == nil {
		return
	}
	for _, t := range append([]*FileTab(nil), w.Open...) {
		applySaveResult(m, t)
		if t.closeWhenSaved && t.save != nil {
			s := t.save.Snapshot()
			if s.Error != "" {
				t.closeWhenSaved = false
			}
			if !s.Running && !s.Pending && !t.Dirty() {
				t.closeWhenSaved = false
				closeTab(m, t.Path, false)
			}
		}
	}
}

func observeAutosave(m Model) {
	capture(m)
	t := selectedTab(m)
	if t == nil {
		return
	}
	if t.save == nil && !t.Editing {
		return
	}
	if err := ensureSaveQueue(m, t); err != nil {
		return
	}
	automatic := t.Editing
	if w := tabsFor(m); w != nil && w.PendingClose == t.Path && !t.closeWhenSaved {
		automatic = false
	}
	t.save.Observe(t.Draft, automatic)
	applySaveResult(m, t)
}

func requestSaveDraft(m Model) error {
	if m.SelectedFile == nil || m.EditDraft == nil || m.Preview == nil {
		return errors.New("no editable file")
	}
	capture(m)
	t := selectedTab(m)
	if t == nil {
		return saveDraft(m)
	}
	if err := ensureSaveQueue(m, t); err != nil {
		return err
	}
	t.save.Flush(t.Draft)
	applySaveResult(m, t)
	return nil
}

func pauseSave(t *FileTab) {
	if t != nil && t.save != nil {
		t.save.Pause()
	}
}
func disposeSave(t *FileTab) {
	if t != nil && t.save != nil {
		t.save.Dispose()
		t.save = nil
		t.saveRevision = 0
		t.closeWhenSaved = false
	}
}

// Dispose cancels queued file work on window shutdown without flushing drafts.
// Existing recovery serialization still owns draft persistence.
func (t *Tabs) Dispose() {
	t.closeFloatingPreview()
	if t.Search != nil {
		t.Search.Cancel()
	}
	if t.Tree != nil {
		t.Tree.Cancel()
	}
	for _, w := range t.Workspaces {
		if w != nil {
			for _, tab := range w.Open {
				disposeSave(tab)
			}
		}
	}
}

// Pause stops debounce work when the Files surface is hidden, without throwing
// away drafts or completion receipts for writes that had already started.
func (t *Tabs) Pause() {
	if t.Search != nil {
		t.Search.Cancel()
	}
	if t.Tree != nil {
		t.Tree.Cancel()
	}
	for _, w := range t.Workspaces {
		if w != nil {
			for _, tab := range w.Open {
				pauseSave(tab)
			}
		}
	}
}
