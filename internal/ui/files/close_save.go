package files

import (
	"errors"
	"path/filepath"
)

// dirtyTabsForClose validates the same captured draft and root ownership for
// the read-only preflight and the subsequent explicit save gesture.
func dirtyTabsForClose(m Model) ([]*FileTab, error) {
	if m.Tabs == nil {
		return nil, errors.New("file pane is unavailable")
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
				return nil, errors.New(label("其他工作区仍有草稿，请切回其工作区保存后关闭。", "A different workspace has a draft. Return to that workspace and save it before closing."))
			}
			dirty = append(dirty, tab)
		}
	}
	for _, tab := range dirty {
		if err := checkSavePermission(m, tab, true); err != nil {
			return nil, errors.New(tab.Path + ": " + err.Error())
		}
	}
	return dirty, nil
}

// PreflightDirtyForClose captures committed input and checks every local draft
// without creating queues or scheduling writes. Mixed panes call all preflights
// before starting either local or remote saves.
func PreflightDirtyForClose(m Model) error {
	_, err := dirtyTabsForClose(m)
	return err
}

// SaveDirtyForClose queues one explicit save after all draft permissions pass.
// Poll CheckDirtyForClose on later frames; never repeat Flush while waiting.
func SaveDirtyForClose(m Model) (pending bool, err error) {
	dirty, err := dirtyTabsForClose(m)
	if err != nil {
		return false, err
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
