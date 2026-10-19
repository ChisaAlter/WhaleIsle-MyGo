package frame

import (
	"errors"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/ui/files"
)

// Drain both domains even when one has failed. A remote close attempt retains
// its receipt cursor until checked; abandoning it would block the next save.
func workspacePaneCloseSaveStatus(n *WorkspacePane, m files.Model) (pending bool, err error) {
	if n.Files != nil {
		pending, err = files.CheckDirtyForClose(m)
	}
	if n.RemoteFiles != nil {
		done, remoteErr := n.RemoteFiles.CheckDirtyForClose()
		pending = pending || !done && remoteErr == nil
		err = errors.Join(err, remoteErr)
	}
	return pending, err
}

func startWorkspacePaneCloseSave(n *WorkspacePane, m files.Model) (pending bool, err error) {
	if n.Files != nil {
		err = files.PreflightDirtyForClose(m)
	}
	if err == nil && n.RemoteFiles != nil {
		err = n.RemoteFiles.PreflightDirtyForClose()
	}
	if err != nil {
		return false, err
	}
	if n.RemoteFiles != nil {
		if err = n.RemoteFiles.SaveDirtyForClose(); err != nil {
			return false, err
		}
	}
	if n.Files != nil {
		_, err = files.SaveDirtyForClose(m)
	}
	pending, receiptErr := workspacePaneCloseSaveStatus(n, m)
	return pending, errors.Join(err, receiptErr)
}

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
		pending, err = workspacePaneCloseSaveStatus(n, fileModel)
		if err != nil && w.closeError == "" {
			w.closeError = err.Error()
		}
		if !pending {
			w.closeSaving = ""
			if w.closeError == "" {
				finish()
				return
			}
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
			pending, err = startWorkspacePaneCloseSave(n, fileModel)
			w.closeError = ""
			if err != nil {
				w.closeError = err.Error()
			}
			if pending {
				w.closeSaving = n.ID
			} else if err == nil {
				finish()
			}
		}
	})
	if pending {
		c.After(50 * time.Millisecond)
	}
}
