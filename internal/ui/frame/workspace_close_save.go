package frame

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/files"
)

func workspaceCloseFilesModel(m Model) files.Model {
	if m.workspacePane.Files != nil {
		m = m.workspacePane.Files.model(m)
	}
	return files.Model{
		Tabs: m.FileTabs, WorkspaceRoot: m.WorkspaceRoot, BindingID: m.FileBindingID,
		SelectedFile: m.SelectedFile, Preview: m.FilePreview, PreviewErr: m.FilePreviewErr,
		Editing: m.FileEditing, EditDraft: m.FileEditDraft, EditStatus: m.FileEditStatus,
		OnSaveEdit: func(rel, text string) error {
			full, err := workspacefs.JoinRoot(m.WorkspaceRoot, rel)
			if err != nil {
				return err
			}
			return workspacefs.WriteFileText(full, text)
		},
	}
}

func workspacePaneClosePrompt(c *ui.Context, m Model) {
	n, w := m.workspacePane, m.Sizes.Workspace
	fileModel := workspaceCloseFilesModel(m)
	pending := w.closeSaving == n.ID
	finish := func() {
		w.closeSaving, w.closeError = "", ""
		w.pending = func() {
			if w.Root.find(n.ID) == n {
				w.closeNow(n.ID)
			}
		}
	}
	if pending {
		var err error
		pending = false
		if n.Files != nil {
			pending, err = files.CheckDirtyForClose(fileModel)
		}
		if err == nil && n.RemoteFiles != nil {
			done, remoteErr := n.RemoteFiles.CheckDirtyForClose()
			pending, err = pending || !done, remoteErr
		}
		if err != nil {
			pending = false
		}
		if !pending {
			w.closeSaving = ""
			if err == nil {
				finish()
				return
			}
			w.closeError = err.Error()
		}
	}
	ui.Text(c, paneLabel("此面板有未保存草稿。", "This pane has unsaved drafts."))
	if w.closeError != "" {
		ui.Text(c, w.closeError)
	}
	ui.Row(c).Gap(4).Wrap().Children(func() {
		if ui.Button(c, paneLabel("保留面板", "Keep pane")).Disabled(pending).Clicked() {
			w.closing, w.closeSaving, w.closeError = "", "", ""
		}
		if ui.Button(c, paneLabel("放弃并关闭", "Discard and close")).Disabled(pending).Clicked() {
			finish()
		}
		if ui.PrimaryButton(c, paneLabel("保存并关闭", "Save and close")).Disabled(pending).Clicked() {
			var err error
			if n.Files != nil {
				err = files.PreflightDirtyForClose(fileModel)
			}
			if err == nil && n.RemoteFiles != nil {
				err = n.RemoteFiles.PreflightDirtyForClose()
			}
			if err == nil && n.RemoteFiles != nil {
				err = n.RemoteFiles.SaveDirtyForClose()
			}
			if err == nil {
				if n.Files != nil {
					pending, err = files.SaveDirtyForClose(fileModel)
				}
				if err == nil && n.RemoteFiles != nil {
					done, remoteErr := n.RemoteFiles.CheckDirtyForClose()
					pending, err = pending || !done, remoteErr
				}
			}
			w.closeError = ""
			if err != nil {
				pending = false
				w.closeError = err.Error()
			} else if pending {
				w.closeSaving = n.ID
			} else {
				finish()
			}
		}
	})
	if pending {
		c.After(50 * time.Millisecond)
	}
}
