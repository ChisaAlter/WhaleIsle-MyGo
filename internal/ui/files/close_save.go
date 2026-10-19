package files

import (
	"errors"
	"path/filepath"
)

// SaveDirtyForClose is an explicit Save-and-close gesture. Preflight every
// retained dirty buffer before queuing a write; the supplied writer belongs to
// this root only. Call CheckDirtyForClose on later frames, never Flush again.
func SaveDirtyForClose(m Model) (pending bool, err error) {
	if m.Tabs == nil {
		return false, errors.New("file pane is unavailable")
	}
	capture(m)
	root := filepath.Clean(m.WorkspaceRoot)
	var dirty []*FileTab
	for workspaceRoot, workspace := range m.Tabs.Workspaces {
		if workspace == nil {
			continue
		}
		for _, tab := range workspace.Open {
			applySaveResult(m, tab)
			if tab == nil || !tab.Dirty() {
				continue
			}
			if m.WorkspaceRoot == "" || filepath.Clean(workspaceRoot) != root {
				return false, errors.New(label("其他工作区仍有草稿，请切回其工作区保存后关闭。", "A different workspace has a draft. Return to that workspace and save it before closing."))
			}
			dirty = append(dirty, tab)
		}
	}
	for _, tab := range dirty {
		if err := prepareSaveQueue(m, tab, true); err != nil {
			return false, errors.New(tab.Path + ": " + err.Error())
		}
	}
	m.Tabs.closeSaving = true
	for _, tab := range dirty {
		tab.save.Flush(tab.Draft)
	}
	return CheckDirtyForClose(m)
}

// CheckDirtyForClose applies only same-tab queue receipts. New input after a
// save's captured draft remains dirty, and failure never removes the pane.
func CheckDirtyForClose(m Model) (pending bool, err error) {
	if m.Tabs == nil {
		return false, errors.New("file pane is unavailable")
	}
	capture(m)
	var dirty bool
	for _, workspace := range m.Tabs.Workspaces {
		if workspace == nil {
			continue
		}
		for _, tab := range workspace.Open {
			if tab == nil {
				continue
			}
			applySaveResult(m, tab)
			if tab.save != nil {
				save := tab.save.Snapshot()
				pending = pending || save.Running || save.Pending
				if save.Error != "" && tab.Dirty() && err == nil {
					err = errors.New(tab.Path + ": " + save.Error)
				}
			}
			dirty = dirty || tab.Dirty()
		}
	}
	if pending {
		return true, nil
	}
	m.Tabs.closeSaving = false
	if err == nil && dirty {
		err = errors.New(label("仍有较新的未保存草稿，请再次保存或保留面板。", "Newer unsaved edits remain. Save again or keep the pane."))
	}
	return false, err
}
