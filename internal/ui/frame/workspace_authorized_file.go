package frame

import (
	"errors"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/files"
)

// ProjectFileAccess contains the exact Host/client/coordinator authorization
// captured by the main consumer. Model readers/openers never resolve the
// currently selected conversation. Managed local files use the checked local
// save path; an external Host reader stays read-only under Files' old contract.
// Runtime callbacks are deliberately not part of recovery serialization.
type ProjectFileAccess struct {
	Valid func() bool
	Model files.Model
}

func (a *ProjectFileAccess) current() bool { return a != nil && a.Valid != nil && a.Valid() }

func absoluteResourcePath(path string) bool {
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\\`) || len(path) > 2 && path[1] == ':' && (path[2] == '/' || path[2] == '\\')
}

func (s *WorkspaceFileState) projectFileModel() files.Model {
	m := files.Model{}
	if s.projectAccess != nil {
		m = s.projectAccess.Model
	}
	target := s.ProjectTarget
	if target != nil {
		m.WorkspaceRoot, m.SessionID = target.Root, target.CoordinatorSessionID
	}
	m.BindingID = s.projectBinding
	m.ResourceOnly = true
	m.HostOnly = m.ReadRemoteContext != nil || m.ReadTextPageContext != nil
	m.Tabs, m.Expanded = &s.Tabs, &s.Expanded
	m.SelectedFile, m.Filter = &s.Selected, &s.Filter
	m.Preview, m.PreviewErr, m.PreviewList = &s.Preview, &s.Error, &s.List
	m.EditDraft, m.EditStatus, m.StatNote = &s.Draft, &s.Status, &s.Stat
	m.Editing, m.EditViewMode, m.EditSplit = &s.Editing, &s.Mode, &s.Split
	m.PreviewSplit = &s.TreeSplit
	m.OnNewTab = nil
	if save := m.OnSaveEdit; save != nil {
		access := s.projectAccess
		m.OnSaveEdit = func(rel, text string) error {
			if !access.current() {
				return errors.New("Project file owner changed; reopen it from the delivery report")
			}
			expected, ok := files.WorkspaceRelativePath(target.Root, target.CanonicalPath)
			if !ok || !samePreviewPath(expected, rel) {
				return errors.New("Project file save target changed")
			}
			return save(rel, text)
		}
	}
	return m
}

// OpenAuthorizedProjectFile creates/reuses a distinct Files occurrence. It does
// not select a Session or change the shell's global workspace root. Existing
// drafts are retained, and changing Host authority cannot adopt a dirty draft.
func (w *WorkspacePanes) OpenAuthorizedProjectFile(target projectflow.FileTarget, binding string, access ProjectFileAccess) error {
	if target.ProjectID == "" || target.CoordinatorSessionID == "" || binding == "" || !access.current() || !absoluteResourcePath(target.Root) || !absoluteResourcePath(target.CanonicalPath) {
		return errors.New("Project file authorization is no longer available")
	}
	if rel, ok := files.WorkspaceRelativePath(target.Root, target.CanonicalPath); !ok || rel == "." {
		return errors.New("Project file is outside its authorized root")
	}
	var pane *WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Files != nil && n.Files.ProjectTarget != nil && *n.Files.ProjectTarget == target {
			pane = n
		}
	})
	added := pane == nil
	if added {
		pane = w.leaf("files")
		pane.Files = &WorkspaceFileState{Expanded: map[string]bool{}, Mode: files.EditViewEdit, Split: 200}
	}
	state := pane.Files
	if state.unsaved() && state.projectBinding != binding {
		return errors.New("This Project file has a draft from another Host owner. Save or close that pane first")
	}
	if !added && state.unsaved() {
		state.projectAccess = &access
		w.activateDockTab(pane.ID)
		w.focusPending = pane.ID
		if pane.Float != nil {
			w.MinimizeWorkspaceFloat(pane.ID, false)
		}
		return nil
	}
	copyTarget := target
	state.ProjectTarget, state.projectBinding, state.projectAccess = &copyTarget, binding, &access
	state.Tabs.Bind(binding, target.Root)
	state.previewBinding, state.previewRoot = binding, target.Root
	handled, err := files.OpenPresentedPath(state.projectFileModel(), target.CanonicalPath)
	if err != nil {
		return err
	}
	if !handled {
		return errors.New("Project file is unavailable in its authorized workspace")
	}
	if !access.current() {
		state.Tabs.Pause()
		return errors.New("Project file owner changed while opening")
	}
	if added {
		if w.Root == nil {
			w.Root = pane
		} else {
			w.placeDockResource(pane, w.Root.find(w.Active), false)
		}
	}
	w.activateDockTab(pane.ID)
	w.focusPending = pane.ID
	if pane.Float != nil {
		w.MinimizeWorkspaceFloat(pane.ID, false)
	}
	return nil
}

func paintAuthorizedProjectFile(c *ui.Context, m Model, n *WorkspacePane) bool {
	if n.Kind != "files" || n.Files == nil || n.Files.ProjectTarget == nil {
		return false
	}
	state := n.Files
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Children(func() {
		workspacePaneControls(c, m)
		if !state.projectAccess.current() {
			state.Tabs.Pause()
			ui.Text(c, paneLabel("请从项目交付报告重新打开此文件；未保存草稿仍保留。", "Reopen this file from its Project delivery report. Unsaved drafts are retained.")).Padding(12)
			return
		}
		state.rendered = true
		fm := state.projectFileModel()
		fm.SVGImages, fm.ThemeID, fm.SystemDark = m.SVGImages, m.ThemeID, m.SystemDark
		fm.AllowPageShortcuts = func() bool { return m.Sizes.Workspace.Active == n.ID }
		local := m
		local.workspacePane = n
		bindWorkspacePageRefresh(local, func() {
			if state.projectAccess.current() {
				files.ReloadSelectedFile(fm)
			}
		})
		files.View(c, fm)
	})
	return true
}
