package files

import (
	"errors"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/i18n"
)

func ensureSaveQueue(m Model, t *FileTab) error {
	return prepareSaveQueue(m, t, false)
}

// Explicit Save retains the desktop editor's write action for an existing
// dirty local buffer after a failed or incomplete reread. Automatic observation
// never grants that authority and never retries a failed queue.
func checkSavePermission(m Model, t *FileTab, explicit bool) error {
	if t == nil || m.OnSaveEdit == nil {
		return errors.New("no editable file")
	}
	if t.deliverySource != nil {
		return errors.New(label("交付文件预览是只读视图。", "Delivered-file previews are read-only."))
	}
	if isTablePreviewPath(t.Path) {
		return errors.New(label("表格分页是只读视图，不能保存为原文件。", "Workbook pages are read-only and cannot be saved over the original file."))
	}
	if t.readLoading || t.readResume || (t.readBlocked && !(explicit && t.Dirty())) {
		return errors.New(label("读取尚未成功完成，草稿保留；请等待或重新载入后保存。", "Read has not completed successfully. Draft kept; wait or reload before saving."))
	}
	if t.Truncated && !(explicit && t.Dirty()) {
		return errors.New(label("不完整文件预览不能保存；草稿保留，请在外部编辑器核对。", "Cannot save an incomplete file preview. Draft kept; reconcile in an external editor."))
	}
	if t.Remote {
		return errors.New(label("远程预览草稿已保留，请使用会话远程文件保存。", "Remote preview draft kept. Save through the session's remote file editor."))
	}
	if t.hasConflict {
		return errors.New(label("存在外部修改冲突，请先选择应用磁盘版本或用草稿覆盖磁盘。", "External modification conflict exists. Choose Use disk version or Overwrite disk with draft first."))
	}
	return nil
}

func prepareSaveQueue(m Model, t *FileTab, explicit bool) error {
	if err := checkSavePermission(m, t, explicit); err != nil {
		return err
	}
	if t.save != nil {
		return nil
	}
	root, path, write := m.WorkspaceRoot, t.Path, m.OnSaveEdit
	t.save = workspacefs.NewLocalEditorSaveQueue(root, path, t.Preview, write, m.notify)
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
		t.saveRevision = s.Revision
		if s.Error == "" {
			t.Preview = s.Confirmed
			t.Error = ""
			t.Status = i18n.T(i18n.KeyFilesSaved)
			t.Truncated, t.readBlocked, t.externalChanged = false, false, false
			t.hasConflict, t.conflictContent, t.conflictTruncated = false, "", false
		} else {
			// SaveConflictError increments Revision and reports disk text in s.Confirmed.
			// Keep t.Preview at its pre-conflict baseline until the user resolves it.
			t.hasConflict = true
			t.conflictContent = s.Confirmed
			t.conflictTruncated = false
			t.externalChanged = true
			t.readBlocked = true
			t.Status = s.Error
		}
	}
	if s.Error != "" {
		t.Status = s.Error
	}
	if s.Running || s.Pending {
		t.Status = label("正在保存…", "Saving…")
	}
	if m.SelectedFile != nil && selectedTab(m) == t {
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
	if m.Tabs != nil && m.Tabs.closeSaving {
		return // Do not cancel a close-save queued behind an in-flight write.
	}
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
	if err := prepareSaveQueue(m, t, true); err != nil {
		return err
	}
	t.save.Flush(t.Draft)
	applySaveResult(m, t)
	return nil
}

// applyConflictDiskVersion discards the conflicting local draft edit and adopts
// the latest verified disk version captured during reread or save conflict.
func applyConflictDiskVersion(m Model) error {
	capture(m)
	t := selectedTab(m)
	if t == nil || !t.hasConflict {
		return errors.New(label("当前没有可应用的磁盘冲突版本", "No external conflict version available to apply"))
	}
	if t.conflictTruncated {
		return errors.New(label("磁盘内容超过预览上限，无法直接应用到草稿", "Disk version exceeds preview limit and cannot replace draft directly"))
	}
	if t.save != nil && t.save.Snapshot().Running {
		return errors.New(label("保存正在进行，请稍后重试", "Save in progress; try again shortly"))
	}
	disposeSave(t)
	disk := t.conflictContent
	t.Preview = disk
	t.Draft = disk
	t.Truncated = false
	t.readBlocked = false
	t.externalChanged = false
	t.hasConflict = false
	t.conflictContent = ""
	t.conflictTruncated = false
	t.Error = ""
	t.Status = label("已应用磁盘版本到草稿", "Applied disk version to draft")
	restore(m, t)
	return nil
}

// overwriteDiskWithDraft explicitly confirms writing the current draft over the
// conflicting external disk version using conflictContent as the CAS baseline.
func overwriteDiskWithDraft(m Model) error {
	if m.SelectedFile == nil || m.EditDraft == nil || m.Preview == nil || m.OnSaveEdit == nil {
		return errors.New("no editable file")
	}
	capture(m)
	t := selectedTab(m)
	if t == nil {
		return errors.New("no editable file")
	}
	if !t.hasConflict {
		return requestSaveDraft(m)
	}
	if t.deliverySource != nil {
		return errors.New(label("交付文件预览是只读视图。", "Delivered-file previews are read-only."))
	}
	if isTablePreviewPath(t.Path) {
		return errors.New(label("表格分页是只读视图，不能保存为原文件。", "Workbook pages are read-only and cannot be saved over the original file."))
	}
	if t.readLoading || t.readResume {
		return errors.New(label("读取尚未成功完成，草稿保留；请等待或重新载入后保存。", "Read has not completed successfully. Draft kept; wait or reload before saving."))
	}
	if t.Remote {
		return errors.New(label("远程预览草稿已保留，请使用会话远程文件保存。", "Remote preview draft kept. Save through the session's remote file editor."))
	}
	if t.save != nil && t.save.Snapshot().Running {
		return errors.New(label("保存正在进行，请稍后重试", "Save in progress; try again shortly"))
	}
	baseline := t.conflictContent
	if t.Draft == baseline {
		return applyConflictDiskVersion(m)
	}
	disposeSave(t)
	t.save = workspacefs.NewLocalEditorSaveQueue(m.WorkspaceRoot, t.Path, baseline, m.OnSaveEdit, m.notify)
	t.saveRevision = 0
	t.save.Observe(t.Draft, false)
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
	t.stopResourceObserver()
	t.stopImageRead(true)
	t.stopTextRead(false, true)
	t.CancelPlanRead()
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
				tab.closeDocumentSurface()
			}
		}
	}
}

// Pause stops debounce work when the Files surface is hidden, without throwing
// away drafts or completion receipts for writes that had already started.
func (t *Tabs) Pause() {
	t.stopResourceObserver()
	t.stopImageRead(false)
	t.hideDocumentSurfaces()
	t.stopTextRead(true, false)
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
