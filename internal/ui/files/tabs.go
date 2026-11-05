package files

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/planpreview"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Tabs owns drafts for all open workspace files for the lifetime of this window.
// Switching tabs/workspaces never discards a draft. Runtime save queues are
// deliberately omitted from recovery; reopening never flushes a draft.
type Tabs struct {
	explorerError    string
	resourceObserver *resourceObserver
	imageReader      *imageReadQueue
	imageReadTab     *FileTab
	textReader       *textReadQueue
	readTab          *FileTab
	readSerial       uint64
	planRead         *session.PlanResourceRead
	planReadAddress  string // runtime-only explicit user-open gate
	planSession      string
	Plans            planpreview.Resources
	Preferences      *PreviewPreferences
	floating         *floatingPreview
	Workspaces       map[string]*WorkspaceTabs
	Root             string
	binding          string
	boundRoot        string
	closeSaving      bool
	Tree             *workspacefs.DirectoryTree `json:"-"`
	Search           *workspacefs.Search        `json:"-"`
}
type WorkspaceTabs struct {
	// AutoRefreshDisabled is opt-out, so new workspaces watch by default.
	AutoRefreshDisabled bool
	Open                []*FileTab
	Active              string
	PendingClose        string
}
type FileTab struct {
	AutoRefreshDisabled                       bool
	externalChanged                           bool
	document                                  documentSurfaceState
	deliverySource                            *deliveryPreviewSource
	textPage                                  workspacefs.TextPage
	textPaged                                 bool
	tablePage                                 *workspacefs.XLSXPage
	tableRequest                              workspacefs.XLSXPageRequest
	tableCell                                 string
	tableSelectionKind                        string
	tableSelectionIndex                       int
	tableRows                                 ui.ListState
	readID                                    uint64
	readFullPath                              string
	pendingLine                               textReadLine
	readLoading, readBlocked, readResume      bool
	Path, Preview, Draft, Error, Status, Stat string
	Editing                                   bool
	sourceViews                               map[string]*sourceViewState
	sourceWidthText                           string
	sourceWidth                               float32
	sourceWidthValid                          bool
	editBuffer                                *ui.TextBuffer
	editBufferVersion                         uint64
	editBufferText, editModelText             string
	RenderedMarkdown                          bool
	selectionText                             string
	selectionStart, selectionEnd              int
	lineDraft                                 string
	revealLine                                int
	image                                     *imagePreview
	markdownImages                            map[string]*markdownImageRead
	imageView                                 imageViewState
	save                                      *workspacefs.SaveQueue
	saveRevision                              uint64
	closeWhenSaved                            bool
	Remote                                    bool
	Truncated                                 bool // incomplete local prefix; persisted with tab recovery
}

func (t *FileTab) Dirty() bool {
	t.syncNativeDraft()
	return t.Draft != t.Preview
}
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
		t.syncDraftModel(m.EditDraft)
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
	t.syncNativeDraft()
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
		t.editModelText = t.Draft
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
			m.Tabs.stopImageRead(false)
			m.Tabs.stopTextRead(true, false)
			pauseSave(findTab(w, w.Active))
		}
		w.Active = path
		restore(m, t)
		applySaveResult(m, t)
		if t.document.enabled && t.document.surface != nil && t.document.surface.Alive() {
			return true
		}
		if !t.textPaged || t.readBlocked || t.readResume {
			reloadSelectedFile(m)
		}
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
	if m.Tabs != nil && m.Tabs.imageReadTab == t {
		m.Tabs.stopImageRead(false)
	}
	if m.Tabs != nil && m.Tabs.resourceObserver != nil && m.Tabs.resourceObserver.tab == t {
		m.Tabs.stopResourceObserver()
	}
	disposeSave(t)
	t.stopMarkdownImages()
	t.closeDocumentSurface()
	if m.Tabs != nil && m.Tabs.readTab == t {
		m.Tabs.stopTextRead(false, false)
	}
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
			next.readResume = isAsyncPreviewPath(next.Path) && (!next.textPaged || next.readBlocked || next.readResume)
			w.Active = next.Path
			restore(m, next)
		} else {
			restore(m, &FileTab{})
		}
	}
	return true
}
func saveDraft(m Model) error {
	capture(m)
	if err := checkPreviewComplete(m); err != nil {
		return err
	}
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

// paintTabs paints the open-file strip: one 28px chip per tab, the active chip
// filled, the close control only while the chip is active or hovered, and the
// trailing control that returns the tree to a browsable listing. Long names
// ellipsize instead of widening the strip.
func paintTabs(c *ui.Context, m Model, fg, muted ui.Color) {
	w := tabsFor(m)
	if w == nil {
		return
	}
	ui.Column(c).Gap(2).Padding(4, 8, 4, 8).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
					for _, tab := range append([]*FileTab(nil), w.Open...) {
						paintTabChip(c, m, w, tab, fg, muted)
					}
				})
			})
			paintBrowseButton(c, m, muted)
		})
		paintPendingClose(c, m, w, muted)
	})
}

// paintTabChip paints one 28px tab chip.
func paintTabChip(c *ui.Context, m Model, w *WorkspaceTabs, tab *FileTab, fg, muted ui.Color) {
	active := w.Active == tab.Path
	dirty := tab.Dirty()
	chip := ui.Row(c.Key("tab-"+tab.Path)).Gap(5).Height(28).MinWidth(80).MaxWidth(170).Radius(6).Padding(0, 8, 0, 10).AlignItems(ui.Center).Clip()
	// The chip's face eases between resting, hovered and active over the fast
	// token, so switching tabs and passing the pointer move the fill instead of
	// snapping it.
	chip.Transition(widgets.HoverTransition())
	hovered := chip.Hovered()
	switch {
	case active:
		chip.Background(c.Theme().SurfaceHover)
	case hovered:
		chip.Background(ui.RGBA(muted.R, muted.G, muted.B, .16))
	}
	textColor := muted
	if active {
		textColor = fg
	}
	chip.Children(func() {
		title := ui.Button(c, "").Label(tab.Path).Grow(1).MinWidth(0).Height(26).Padding(0).BorderWidth(0).Background(ui.Transparent)
		title.Children(func() {
			ui.Text(c, filepath.Base(tab.Path)).FontSize(fonttheme.InterfaceSize(13)).TextColor(textColor).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
		})
		if title.Clicked() {
			selectFile(m, tab.Path)
		}
		if dirty {
			accent := chrome.Color(fonttheme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)
			ui.Box(c).Size(6, 6).Shrink(0).Radius(3).Background(accent).Tooltip(i18n.T(i18n.KeySurfacesFilesDirty))
		}
		if active || hovered {
			closeLabel := i18n.T(i18n.KeySurfacesFilesClose)
			close := ui.Button(c, "").Label(closeLabel).Size(18, 18).Shrink(0).Padding(0).Radius(4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(closeLabel)
			close.Children(func() {
				ui.Icon(c, tabCloseIcon).Size(10, 10).TextColor(muted).PassThrough()
			})
			if close.Hovered() {
				close.Background(c.Theme().SurfacePressed)
			}
			if close.Clicked() {
				closeTab(m, tab.Path, false)
			}
		}
	})
}

// paintBrowseButton is the strip's trailing control. The owner binds OnNewTab to
// its own new-tab flow; the strip only calls it, and stays out of the way when
// the owner binds nothing.
func paintBrowseButton(c *ui.Context, m Model, muted ui.Color) {
	if m.OnNewTab == nil {
		return
	}
	caption := i18n.T(i18n.KeySurfacesFilesOpen)
	control := ui.Button(c, "").Label(caption).Size(28, 28).Shrink(0).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(caption)
	control.Children(func() {
		ui.Icon(c, tabPlusIcon).Size(12, 12).TextColor(muted).PassThrough()
	})
	if control.Hovered() {
		control.Background(c.Theme().SurfaceHover)
	}
	if control.Clicked() {
		m.OnNewTab()
	}
}

// paintPendingClose shows the unsaved-tab decision for the tab that requested a
// close, instead of a second modal over the editor.
func paintPendingClose(c *ui.Context, m Model, w *WorkspaceTabs, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	if w.PendingClose == "" {
		return
	}
	path := w.PendingClose
	busy := false
	saveErr := ""
	if tab := findTab(w, path); tab != nil {
		if tab.save != nil {
			snap := tab.save.Snapshot()
			busy = snap.Running || (tab.closeWhenSaved && snap.Pending)
			if snap.Error != "" {
				saveErr = snap.Error
			}
		}
		if saveErr == "" && tab.Error != "" {
			saveErr = tab.Error
		}
	}
	ui.Text(c, label("有未保存的修改：", "Unsaved changes: ")+path).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if saveErr != "" {
		ui.Text(c, saveErr).FontSize(fonttheme.InterfaceSize(11)).TextColor(chrome.Color(colors.Danger))
	}
	ui.Row(c).Gap(4).Children(func() {
		if widgets.Button(c, colors, label("继续编辑", "Keep editing"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(busy).Clicked() {
			w.PendingClose = ""
		}
		if widgets.Button(c, colors, label("保存并关闭", "Save and close"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(busy).Clicked() {
			if tab := findTab(w, path); tab != nil {
				capture(m)
				if path != w.Active {
					m.Tabs.stopImageRead(false)
					m.Tabs.stopTextRead(true, false)
					pauseSave(findTab(w, w.Active))
					w.Active = path
					restore(m, tab)
					applySaveResult(m, tab)
				}
				tab.Error = ""
			} else {
				selectFile(m, path)
			}
			if err := requestSaveDraft(m); err != nil {
				if m.EditStatus != nil {
					*m.EditStatus = err.Error()
				}
				if tab := findTab(w, path); tab != nil {
					tab.Error = err.Error()
				}
			} else if tab := selectedTab(m); tab != nil {
				tab.closeWhenSaved = true
			} else {
				closeTab(m, path, false)
			}
		}
		if widgets.Button(c, colors, label("丢弃", "Discard"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Disabled(busy).Clicked() {
			closeTab(m, path, true)
		}
	})
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
	t.Dispose()
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
	t.stopResourceObserver()
	t.stopImageRead(false)
	t.CancelPlanRead()
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
				disposeSave(tab)
				tab.closeDocumentSurface()
				tab.pendingLine.reset()
				tab.revealLine = 0
				if isTablePreviewPath(tab.Path) {
					tab.tablePage = nil
					tab.tableRequest = workspacefs.XLSXPageRequest{}
				}
				if isAsyncPreviewPath(tab.Path) {
					tab.readBlocked, tab.readResume = true, true
				}
			}
		}
	}
	t.closeFloatingPreview()
	t.explorerError = ""
	t.binding, t.boundRoot = binding, root
}
