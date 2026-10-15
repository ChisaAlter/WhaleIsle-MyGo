package files

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/planpreview"
)

// Tabs owns drafts for all open workspace files for the lifetime of this window.
// Switching tabs/workspaces never discards a draft. Runtime save queues are
// deliberately omitted from recovery; reopening never flushes a draft.
type Tabs struct {
	planRead        *session.PlanResourceRead
	planReadAddress string // runtime-only explicit user-open gate
	planSession     string
	Plans           planpreview.Resources
	Preferences     *PreviewPreferences
	floating        *floatingPreview
	Workspaces      map[string]*WorkspaceTabs
	Root            string
	binding         string
	boundRoot       string
	Tree            *workspacefs.DirectoryTree `json:"-"`
	Search          *workspacefs.Search        `json:"-"`
}
type WorkspaceTabs struct {
	Open         []*FileTab
	Active       string
	PendingClose string
}
type FileTab struct {
	Path, Preview, Draft, Error, Status, Stat string
	Editing                                   bool
	sourceViews                               map[string]*sourceViewState
	sourceWidthText                           string
	sourceWidth                               float32
	sourceWidthValid                          bool
	RenderedMarkdown                          bool
	selectionText                             string
	selectionStart, selectionEnd              int
	lineDraft                                 string
	revealLine                                int
	image                                     *imagePreview
	save                                      *workspacefs.SaveQueue
	saveRevision                              uint64
	closeWhenSaved                            bool
	Remote                                    bool
}

func (t *FileTab) Dirty() bool { return t.Draft != t.Preview }
func label(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}
func tabsFor(m Model) *WorkspaceTabs {
	if m.Tabs == nil {
		return nil
	}
	if m.Tabs.Workspaces == nil {
		m.Tabs.Workspaces = map[string]*WorkspaceTabs{}
	}
	root := filepath.Clean(m.WorkspaceRoot)
	w := m.Tabs.Workspaces[root]
	if w == nil {
		w = &WorkspaceTabs{}
		m.Tabs.Workspaces[root] = w
	}
	return w
}
func findTab(w *WorkspaceTabs, path string) *FileTab {
	if w != nil {
		for _, t := range w.Open {
			if t.Path == path {
				return t
			}
		}
	}
	return nil
}
func capture(m Model) {
	w := tabsFor(m)
	if w == nil || m.SelectedFile == nil {
		return
	}
	t := findTab(w, *m.SelectedFile)
	if t == nil {
		return
	}
	if m.Preview != nil {
		t.Preview = *m.Preview
	}
	if m.EditDraft != nil {
		t.Draft = *m.EditDraft
	}
	if m.PreviewErr != nil {
		t.Error = *m.PreviewErr
	}
	if m.EditStatus != nil {
		t.Status = *m.EditStatus
	}
	if m.Editing != nil {
		t.Editing = *m.Editing
	}
	if m.StatNote != nil {
		t.Stat = *m.StatNote
	}
}
func restore(m Model, t *FileTab) {
	t.selectionText = ""
	t.selectionStart, t.selectionEnd = 0, 0
	if m.SelectedFile != nil {
		*m.SelectedFile = t.Path
	}
	if m.Preview != nil {
		*m.Preview = t.Preview
	}
	if m.EditDraft != nil {
		*m.EditDraft = t.Draft
	}
	if m.PreviewErr != nil {
		*m.PreviewErr = t.Error
	}
	if m.EditStatus != nil {
		*m.EditStatus = t.Status
	}
	if m.Editing != nil {
		*m.Editing = t.Editing
	}
	if m.StatNote != nil {
		*m.StatNote = t.Stat
	}
}
func prepareTabs(m Model) {
	w := tabsFor(m)
	if w == nil {
		return
	}
	root := filepath.Clean(m.WorkspaceRoot)
	if m.Tabs.Root != root {
		m.Tabs.Root = root
		if t := findTab(w, w.Active); t != nil {
			restore(m, t)
			reloadSelectedFile(m)
		} else {
			restore(m, &FileTab{})
		}
		return
	}
	if m.SelectedFile != nil && *m.SelectedFile != "" && *m.SelectedFile != w.Active {
		requested := *m.SelectedFile
		if active := findTab(w, w.Active); active != nil {
			restore(m, active)
		}
		selectFile(m, requested)
	}
	capture(m)
}
func selectTab(m Model, path string) bool {
	w := tabsFor(m)
	if w == nil {
		return false
	}
	capture(m)
	if t := findTab(w, path); t != nil {
		if path != w.Active {
			pauseSave(findTab(w, w.Active))
		}
		w.Active = path
		restore(m, t)
		applySaveResult(m, t)
		reloadSelectedFile(m)
		return true
	}
	return false
}
func rememberTab(m Model, path string) {
	w := tabsFor(m)
	if w == nil {
		return
	}
	w.Active = path
	if findTab(w, path) == nil {
		w.Open = append(w.Open, &FileTab{Path: path})
	}
	capture(m)
}
func closeTab(m Model, path string, discard bool) bool {
	w := tabsFor(m)
	if w == nil {
		return false
	}
	capture(m)
	t := findTab(w, path)
	if t == nil {
		return true
	}
	applySaveResult(m, t)
	if t.save != nil && t.save.Snapshot().Running {
		w.PendingClose = path
		setFileStatus(m, label("保存正在进行，草稿保留。请稍候再关闭。", "A save is in progress; draft kept. Wait before closing."))
		return false
	}
	if t.Dirty() && !discard {
		pauseSave(t)
		w.PendingClose = path
		return false
	}
	disposeSave(t)
	for i, v := range w.Open {
		if v.Path == path {
			w.Open = append(w.Open[:i], w.Open[i+1:]...)
			break
		}
	}
	w.PendingClose = ""
	if w.Active == path {
		w.Active = ""
		if len(w.Open) > 0 {
			next := w.Open[len(w.Open)-1]
			w.Active = next.Path
			restore(m, next)
		} else {
			restore(m, &FileTab{})
		}
	}
	return true
}
func saveDraft(m Model) error {
	if m.SelectedFile == nil || m.EditDraft == nil || m.Preview == nil || m.OnSaveEdit == nil {
		return errors.New("no editable file")
	}
	if t := selectedTab(m); t != nil && t.save != nil {
		if t.save.Snapshot().Running {
			return errors.New("save already in progress")
		}
		disposeSave(t)
	}
	if err := workspacefs.SaveTextChecked(context.Background(), m.WorkspaceRoot, *m.SelectedFile, *m.Preview, *m.EditDraft, m.OnSaveEdit); err != nil {
		return err
	}
	*m.Preview = *m.EditDraft
	if m.PreviewErr != nil {
		*m.PreviewErr = ""
	}
	if m.EditStatus != nil {
		*m.EditStatus = i18n.T(i18n.KeyFilesSaved)
	}
	capture(m)
	return nil
}
func paintTabs(c *ui.Context, m Model, fg, muted ui.Color) {
	w := tabsFor(m)
	if w == nil {
		return
	}
	ui.Column(c).Gap(2).Padding(4, 8, 4, 8).Children(func() {
		for _, t := range append([]*FileTab(nil), w.Open...) {
			path := t.Path
			name := path
			if t.Dirty() {
				name += " •"
			}
			if w.Active == path {
				name = "▸ " + name
			}
			ui.Row(c).Gap(4).FillWidth().Children(func() {
				if ui.Button(c, name).FontSize(11).Clicked() {
					selectFile(m, path)
				}
				if ui.Button(c, "×").FontSize(11).Clicked() {
					closeTab(m, path, false)
				}
			})
		}
		if w.PendingClose != "" {
			path := w.PendingClose
			ui.Text(c, label("未保存：", "Unsaved: ")+path).FontSize(11).TextColor(muted)
			ui.Row(c).Gap(4).Children(func() {
				if ui.Button(c, label("保留草稿", "Keep draft")).Clicked() {
					w.PendingClose = ""
				}
				if ui.Button(c, label("保存并关闭", "Save and close")).Clicked() {
					selectFile(m, path)
					if err := requestSaveDraft(m); err != nil {
						if m.EditStatus != nil {
							*m.EditStatus = err.Error()
						}
					} else {
						if t := selectedTab(m); t != nil && t.save != nil {
							t.closeWhenSaved = true
						} else {
							closeTab(m, path, false)
						}
					}
				}
				if ui.Button(c, label("丢弃并关闭", "Discard and close")).Clicked() {
					closeTab(m, path, true)
				}
			})
		}
	})
	_ = fg
}

// Export captures tab/draft state for the application recovery store.
func (t *Tabs) Export() ([]byte, error) { return json.Marshal(t) }

// Restore rehydrates drafts without saving them to workspace files.
func (t *Tabs) Restore(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var restored Tabs
	if err := json.Unmarshal(data, &restored); err != nil {
		return err
	}
	for _, w := range restored.Workspaces {
		if w == nil {
			return errors.New("invalid saved workspace")
		}
		for _, tab := range w.Open {
			if tab == nil || tab.Path == "" {
				return errors.New("invalid saved tab")
			}
			if err := workspacefs.ValidateRelativePath(tab.Path); err != nil {
				return err
			}
		}
		w.PendingClose = ""
	}
	restored.Root = ""
	*t = restored
	return nil
}

// Bind cancels obsolete background reads even while the Files pane is hidden.
// Drafts stay attached to their workspace and are never flushed on a switch.
func (t *Tabs) Bind(binding, root string) {
	t.Plans.BindOwner(planpreview.HostOwner(binding))
	if t.binding == binding && t.boundRoot == root {
		return
	}
	t.CancelPlanRead()
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
	t.closeFloatingPreview()
	t.binding, t.boundRoot = binding, root
}
