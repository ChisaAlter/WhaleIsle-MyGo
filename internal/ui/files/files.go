// Package files is the M7 workspace file tree and text preview in the right pane.
package files

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// EditViewMode is how the file editor shows draft text vs highlighted preview.
// Source remains selectable in every mode; syntax colors decorate native text.
type EditViewMode int

const (
	// EditViewSplit shows TextArea + SourceView sharing the draft (preferred).
	EditViewSplit EditViewMode = iota
	// EditViewEdit shows TextArea only.
	EditViewEdit
	// EditViewHighlight shows SourceView only (read-only preview of the draft).
	EditViewHighlight
)

// sideBySideMinWidth is the preview-pane width (DIPs) at which split uses
// TextArea | SourceView instead of stacked editor-over-preview.
const sideBySideMinWidth float32 = 520

// Model is file panel state for one frame.
type Model struct {
	OpenDocumentViewer  func(string) (DocumentSurface, error)
	ReadTextPageContext func(context.Context, string, int64) (workspacefs.TextPage, error)
	AllowPageShortcuts  func() bool
	PlanOnly            bool
	PlanSnapshot        session.ViewState
	OnLoadPlanHistory   func()
	ReadSubagentPlan    func(context.Context, string, session.PlanResourceTarget, func()) *session.PlanResourceRead
	Tabs                *Tabs
	WorkspaceRoot       string
	// BindingID identifies the current Host connection and selected session.
	SessionID       string
	OnReferenceFile func(filelink.Reference) error
	BindingID       string
	ThemeID         string
	SystemDark      bool

	Expanded     *map[string]bool // rel dir path → expanded; missing means collapsed
	SelectedFile *string          // rel file path
	Preview      *string
	PreviewErr   *string
	PreviewList  *ui.ListState
	PreviewSplit *float32 // height of tree pane (first child of vertical split)

	Editing    *bool
	EditDraft  *string
	EditStatus *string
	// EditViewMode remembers Split / Edit / Highlight across frames (and edits).
	// Nil falls back to package-local state (frame wiring optional).
	EditViewMode *EditViewMode
	// PaneWidth is the preview pane width in DIPs; used to pick side-by-side vs stack.
	PaneWidth *float32
	// EditSplit is the first-pane size for the edit/preview split (width or height).
	EditSplit         *float32
	OnShowDiff        func(rel string)
	OnSaveEdit        func(rel, text string) error
	OnOpenFloatingDoc func(string)
	OnOpenDoc         func(absPath string, kind workspacefs.DocKind, inWebView bool)
	OnOpenMedia       func(path string, floating bool)
	// OnReveal opens the selected file's folder in the OS file manager.
	OnReveal func(absPath string)
	// OnRenderPDF converts an Office doc via officeToPdf/render then previews PDF.
	OnRenderPDF         func(rel string)
	OfficePreviewStatus func(path string) string
	// OpenWithApps are session/workspacePathApplications rows for the selected file.
	OpenWithApps []OpenWithApp
	// OnOpenWith opens path via session/openWorkspacePath with application id.
	OnOpenWith func(absPath, appID string)
	// Filter narrows the tree to matching relative paths (live IndexRelPaths).
	Filter *string
	// ListRemote is optional workspaceFiles/list fallback when local ListDir fails.
	ListRemote         func(rel string) ([]workspacefs.Entry, error)
	ListRemoteContext  workspacefs.RemoteDirectoryLister
	WatchRemoteContext workspacefs.RemoteDirectoryWatcher
	// ReadRemote is optional workspaceFiles/read fallback when local preview fails.
	ReadRemote func(rel string) (string, error)
	// Context readers are captured from the same frame as BindingID.
	ReadRemoteContext func(context.Context, string) (string, error)
	ReadTableContext  func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error)
	StatRemoteContext func(context.Context, string) string
	// StatRemote is optional workspaceFiles/stat note for the selected file.
	StatRemote   func(rel string) string
	StatNote     *string
	OnAddToChat  func(string) error
	CanAddToChat bool
	OnOpenLink   func(string)
	notify       func()
}

// OpenWithApp is one harness-registered application for a workspace path.
type OpenWithApp struct {
	ID   string
	Name string
}

// editViewModeFallback persists the view mode when Model.EditViewMode is nil.
var editViewModeFallback = EditViewSplit

// editSplitFallback is the first-pane size when Model.EditSplit is nil.
var editSplitFallback float32 = 200

// View paints the file tree and plain-text preview.
func View(c *ui.Context, m Model) {
	m.notify = c.Invalidate
	previewPreferences(m)
	if m.Tabs != nil {
		m.Tabs.Bind(m.BindingID, m.WorkspaceRoot)
	}
	observePlanHistory(m)
	if paintPlanResource(c, m) {
		m.Tabs.Pause()
		return
	}
	if m.PlanOnly {
		paintPlanTabs(c, m)
		ui.Text(c, label("从计划卡打开文档，或选择已打开的计划。", "Open a document from a plan card, or select an open plan.")).Padding(8)
		return
	}
	prepareTabs(m)
	syncTextRead(m)
	syncDocumentSurfaces(m)
	syncSaveResults(m)
	observeFileResource(m)
	if t := selectedTab(m); t != nil && t.Editing {
		_ = ensureSaveQueue(m, t)
	}
	defer observeAutosave(m)
	if (m.AllowPageShortcuts == nil || m.AllowPageShortcuts()) && m.SelectedFile != nil && *m.SelectedFile != "" && c.PageShortcut(ui.Cmd, ui.KeyS) {
		if err := requestSaveDraft(m); err != nil {
			setFileStatus(m, err.Error())
		}
	}
	if m.Tabs != nil {
		if m.Tabs.Search == nil {
			m.Tabs.Search = &workspacefs.Search{}
		}
		if m.Tabs.Tree == nil {
			m.Tabs.Tree = &workspacefs.DirectoryTree{}
		}
		m.Tabs.Tree.Ensure(m.BindingID, m.WorkspaceRoot)
		automatic := true
		if w := tabsFor(m); w != nil {
			automatic = !w.AutoRefreshDisabled
		}
		m.Tabs.Tree.Observe(m.WatchRemoteContext, automatic)
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)

	root := strings.TrimSpace(m.WorkspaceRoot)
	ui.Column(c).Fill().MinWidth(0).Background(field).Children(func() {
		ui.Row(c).Padding(12, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyFilesTitle)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			if w := tabsFor(m); w != nil {
				automatic := !w.AutoRefreshDisabled
				ui.Checkbox(c, &automatic, label("自动刷新文件树", "Auto-refresh tree"))
				if automatic != !w.AutoRefreshDisabled {
					w.AutoRefreshDisabled = !automatic
					m.Tabs.Tree.Observe(m.WatchRemoteContext, automatic)
					c.Invalidate()
				}
			}
			if ui.Button(c, label("刷新", "Refresh")).FontSize(theme.InterfaceSize(10)).Clicked() && m.Tabs != nil {
				if m.Tabs.Tree != nil {
					m.Tabs.Tree.Refresh()
				}
				if m.Tabs.Search != nil {
					m.Tabs.Search.Cancel()
				}
				c.Invalidate()
			}
		})
		paintPlanTabs(c, m)
		if m.Filter != nil && root != "" {
			ui.Column(c).Padding(0, 12, 8, 12).Children(func() {
				input := ui.TextInput(c, m.Filter).FillWidth().Placeholder(i18n.T(i18n.KeyFilesFilter))
				if !input.Composing() && input.Shortcut(0, ui.KeyEscape) {
					*m.Filter = ""
				}
			})
		}
		ui.Box(c).Fill().Height(1).Background(border)
		if root == "" {
			ui.Text(c, i18n.T(i18n.KeyFilesNoWorkspace)).FontSize(theme.InterfaceSize(12)).TextColor(muted).
				Padding(12, 12, 12, 12)
			return
		}
		split := m.PreviewSplit
		if split == nil {
			// Match DefaultLayout.RightTree (file tree above preview).
			def := float32(280)
			split = &def
		}
		filter := ""
		if m.Filter != nil {
			filter = strings.TrimSpace(*m.Filter)
		}
		if m.Tabs != nil && m.Tabs.Search != nil {
			m.Tabs.Search.Ensure(m.BindingID, m.WorkspaceRoot, filter, m.ListRemoteContext, c.Invalidate)
		}
		ui.SplitVertical(c, split, func() {
			ui.Scroll(c).Fill().MinHeight(0).Children(func() {
				ui.Column(c).Padding(4, 4, 4, 4).Gap(1).Fill().Children(func() {
					name := filepath.Base(root)
					ui.Text(c, name).FontSize(theme.InterfaceSize(12)).Bold().TextColor(muted).Padding(4, 8, 4, 8)
					if filter != "" {
						paintFiltered(c, m, root, filter, fg, muted, accent)
						return
					}
					paintDir(c, m, root, "", 0, fg, muted, accent)
				})
			})
		}, func() {
			previewPane(c, m, fg, muted, border, field)
		}).Fill()
	})
}

func paintFiltered(c *ui.Context, m Model, root, filter string, fg, muted, accent ui.Color) {
	if m.Tabs == nil || m.Tabs.Search == nil {
		return
	}
	state := m.Tabs.Search.Snapshot()
	if state.Busy {
		ui.Text(c, label("正在搜索全部文件…", "Searching all files…")).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 8, 8, 8)
		return
	}
	if state.Error != "" {
		ui.Text(c, state.Error).FontSize(theme.InterfaceSize(11)).TextColor(accent).Padding(8, 8, 8, 8)
		return
	}
	for _, path := range state.Paths {
		row := ui.Row(c).Key(treeFileRowKey(m, path)).Label("File result "+path).Focusable().Height(28).Padding(0, 8).Gap(6).FillWidth().MinWidth(0).ContextMenu(treeFileMenu(m, path)).Drag(workspaceReference(m, path))
		if m.SelectedFile != nil && *m.SelectedFile == path {
			row.Background(accent.Alpha(0.12))
		}
		row.Children(func() {
			ui.Text(c, path).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
			paintTreeReferenceButton(c, m, row, path, muted)
		})
		if row.Clicked() {
			selectFile(m, path)
		}
	}
	if len(state.Paths) < state.Total {
		if ui.Button(c, fmt.Sprintf(label("加载更多（%d / %d）", "Show more (%d / %d)"), len(state.Paths), state.Total)).Clicked() {
			m.Tabs.Search.More()
		}
	}
	if state.Total == 0 {
		ui.Text(c, i18n.T(i18n.KeyFilesFilterEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(8, 8, 8, 8)
	}
}

func paintDir(c *ui.Context, m Model, root, rel string, depth int, fg, muted, accent ui.Color) {
	if m.Tabs == nil || m.Tabs.Tree == nil {
		return
	}
	state := m.Tabs.Tree.Get(rel, m.ListRemoteContext, c.Invalidate)
	if state.Busy {
		ui.Text(c, label("加载中…", "Loading…")).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 8, 8, 8)
		return
	}
	if state.Error != "" {
		ui.Text(c, state.Error).FontSize(theme.InterfaceSize(11)).TextColor(accent).Padding(8, 8, 8, 8)
		return
	}
	if state.WatchError != "" {
		ui.Text(c, state.WatchError).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(4, 8, 4, 8)
	}
	ents := state.Entries
	if len(ents) == 0 && rel == "" {
		ui.Text(c, label("此文件夹为空", "This folder is empty")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(8)
	}
	for _, e := range ents {
		childRel := e.Name
		if rel != "" {
			childRel = rel + "/" + e.Name
		}
		if e.IsDir {
			rowDir(c, m, childRel, e.Name, depth, fg, muted, accent)
			if isExpanded(m, childRel) {
				paintDir(c, m, root, childRel, depth+1, fg, muted, accent)
			}
		} else {
			rowFile(c, m, childRel, e.Name, depth, fg, muted, accent)
		}
	}
}

func rowDir(c *ui.Context, m Model, rel, name string, depth int, fg, muted, accent ui.Color) {
	pad := float32(8 + depth*12)
	expanded := isExpanded(m, rel)
	row := ui.Row(c).Key(workspaceReference(m, rel)).Label("Folder "+rel).Focusable().Padding(0, 8, 0, pad).Gap(6).FillWidth().MinWidth(0).Height(28).AlignItems(ui.Center).ContextMenu(treeDirectoryMenu(m, rel))
	if row.Hovered() || row.FocusWithin() {
		row.Background(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Hover))
	}
	row.Children(func() {
		paintTreeTwist(c, expanded, muted)
		paintTreeEntryIcon(c, true, muted)
		ui.Text(c, name).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
	})
	if row.Clicked() {
		toggleExpanded(m, rel)
	}
	_ = accent
}

func rowFile(c *ui.Context, m Model, rel, name string, depth int, fg, muted, accent ui.Color) {
	pad := float32(8 + depth*12)
	selected := m.SelectedFile != nil && *m.SelectedFile == rel
	row := ui.Row(c).Key(treeFileRowKey(m, rel)).Label("File "+rel).Focusable().Padding(0, 8, 0, pad).Gap(6).FillWidth().MinWidth(0).Height(28).AlignItems(ui.Center).ContextMenu(treeFileMenu(m, rel)).Drag(workspaceReference(m, rel))
	if selected {
		row.Background(accent.Alpha(0.12))
	} else if row.Hovered() || row.FocusWithin() {
		row.Background(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Hover))
	}
	row.Children(func() {
		ui.Box(c).Width(12)
		paintTreeEntryIcon(c, false, muted)
		ui.Text(c, name).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
		paintTreeReferenceButton(c, m, row, rel, muted)
	})
	if row.Clicked() {
		selectFile(m, rel)
	}
	_ = muted
}

func previewPane(c *ui.Context, m Model, fg, muted, border, field ui.Color) {
	ui.Column(c).Fill().MinHeight(0).Background(field).Children(func() {
		ui.Box(c).Fill().Height(1).Background(border)
		paintTabs(c, m, fg, muted)
		sel := ""
		if m.SelectedFile != nil {
			sel = *m.SelectedFile
		}
		if sel == "" {
			ui.Text(c, i18n.T(i18n.KeyFilesPickFile)).FontSize(theme.InterfaceSize(12)).TextColor(muted).
				Padding(12, 12, 12, 12)
			return
		}
		editing := m.Editing != nil && *m.Editing
		canHL := false
		if m.EditDraft != nil {
			canHL = markdown.CanSourceHighlight(sel, *m.EditDraft)
		} else if m.Preview != nil {
			canHL = markdown.CanSourceHighlight(sel, *m.Preview)
		}
		mode := resolveEditViewMode(m, canHL && editing)

		paintPreviewToolbar(c, m, sel, muted, border)
		if paintDocumentSurface(c, m) {
			return
		}
		if tab := selectedTab(m); tab != nil && tab.readLoading {
			ui.Row(c).Gap(8).Padding(0, 12, 4, 12).Children(func() {
				ui.Text(c, label("正在读取文件…", "Reading file…")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if ui.Button(c, label("取消读取", "Cancel read")).Clicked() {
					m.Tabs.stopTextRead(false, false)
					setFileStatus(m, label("读取已取消，草稿保留。点击重新载入可重试。", "Read cancelled; draft kept. Choose Reload to retry."))
				}
			})
		}
		if previewTruncated(m) {
			ui.Text(c, previewTruncationNotice(m)).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(0, 12, 4, 12)
		}
		paintTextPageControls(c, m, muted)
		paintTableControls(c, m, muted)
		if m.OfficePreviewStatus != nil {
			if status := m.OfficePreviewStatus(sel); status != "" {
				ui.Text(c, status).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(0, 12, 4, 12)
			}
		}
		paintResourceRefreshControls(c, m)
		if m.EditStatus != nil && *m.EditStatus != "" {
			ui.Text(c, *m.EditStatus).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(0, 12, 4, 12)
		}
		if m.PreviewErr != nil && *m.PreviewErr != "" {
			ui.Text(c, *m.PreviewErr).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Accent)).
				Padding(4, 12, 8, 12)
			return
		}
		if paintTableGrid(c, m) {
			return
		}
		if mediaMode(m, sel) {
			ui.Text(c, label("点击音视频预览或浮动预览；仅当前 Host/会话单文件，最大 32 MiB，30 分钟有效。播放取决于 WebView2 编解码支持。", "Choose Media preview or Floating preview. One current Host/session file, up to 32 MiB, valid for 30 minutes. Playback depends on WebView2 codecs.")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(12)
			return
		}
		if imageMode(m, sel) {
			paintImagePreview(c, m, sel, muted)
			return
		}
		if tab := selectedTab(m); tab != nil && tab.RenderedMarkdown && isMarkdown(sel) {
			ui.Scroll(c).Fill().MinHeight(0).Children(func() {
				ui.Column(c).Padding(8, 12, 8, 12).FillWidth().Children(func() {
					paintMarkdownResource(c, m, fg, muted)
				})
			})
			return
		}
		editing = m.Editing != nil && *m.Editing
		mode = resolveEditViewMode(m, canHL && editing)
		if editing && m.EditDraft != nil {
			paintEditBody(c, m, sel, *m.EditDraft, mode, fg, muted)
			return
		}
		text := ""
		if m.Preview != nil {
			text = *m.Preview
		}
		if m.EditDraft != nil && m.Preview != nil && *m.EditDraft != *m.Preview {
			text = *m.EditDraft
			ui.Text(c, label("未保存草稿（仅保留在此窗口）", "Unsaved draft (kept in this window)")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		paintSourceArea(c, m, sel, &text, true, "source", fg, muted)
	})
}

func paintEditBody(c *ui.Context, m Model, sel, draft string, mode EditViewMode, fg, muted ui.Color) {
	switch mode {
	case EditViewHighlight:
		ui.Column(c).Fill().MinHeight(0).Padding(8, 12, 8, 12).Children(func() {
			paintSourceArea(c, m, sel, &draft, true, "highlight", fg, muted)
		})
	case EditViewSplit:
		split := editSplitHandle(m)
		side := preferSideBySide(paneWidthOf(m))
		if side {
			ui.Split(c, split, func() {
				paintSourceArea(c, m, sel, m.EditDraft, false, "editor", fg, muted)
			}, func() {
				ui.Column(c).Fill().MinHeight(0).MinWidth(0).Padding(8, 8, 8, 8).Children(func() {
					paintSourceArea(c, m, sel, &draft, true, "highlight", fg, muted)
				})
			}).Fill()
			return
		}
		ui.SplitVertical(c, split, func() {
			paintSourceArea(c, m, sel, m.EditDraft, false, "editor", fg, muted)
		}, func() {
			ui.Column(c).Fill().MinHeight(0).Padding(8, 8, 8, 8).Children(func() {
				paintSourceArea(c, m, sel, &draft, true, "highlight", fg, muted)
			})
		}).Fill()
	default:
		paintSourceArea(c, m, sel, m.EditDraft, false, "editor", fg, muted)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}

func isExpanded(m Model, rel string) bool {
	if m.Expanded == nil {
		return false
	}
	return (*m.Expanded)[rel]
}

func toggleExpanded(m Model, rel string) {
	if m.Expanded == nil {
		return
	}
	exp := *m.Expanded
	if exp == nil {
		exp = map[string]bool{}
	}
	exp[rel] = !exp[rel]
	if !exp[rel] && m.Tabs != nil && m.Tabs.Tree != nil {
		m.Tabs.Tree.Collapse(rel)
	}
	*m.Expanded = exp
}

func selectFile(m Model, rel string) {
	if m.Tabs != nil {
		m.Tabs.Plans.ShowFiles(m.Tabs.planSession)
	}
	if selectTab(m, rel) {
		return
	}
	// Legacy callers without a tab owner must never silently overwrite edits.
	if m.Tabs == nil && m.EditDraft != nil && m.Preview != nil && *m.EditDraft != *m.Preview {
		if m.EditStatus != nil {
			*m.EditStatus = label("请先保存当前草稿再切换文件", "Save the current draft before switching files")
		}
		return
	}
	pauseSave(selectedTab(m))
	if m.Tabs != nil {
		m.Tabs.stopImageRead(false)
		m.Tabs.stopTextRead(true, false)
		if isAsyncPreviewPath(rel) {
			restore(m, &FileTab{Path: rel})
			rememberTab(m, rel)
			beginTextRead(m)
			return
		}
	}
	remotePreview, truncated := false, false
	defer func() {
		rememberTab(m, rel)
		if t := selectedTab(m); t != nil {
			t.Remote = remotePreview
			t.Truncated = truncated
		}
	}()
	if m.SelectedFile != nil {
		*m.SelectedFile = rel
	}
	if m.Editing != nil {
		*m.Editing = false
	}
	if m.EditStatus != nil {
		*m.EditStatus = ""
	}
	text, errMsg, prefix := loadPreview(m.WorkspaceRoot, rel)
	truncated = prefix
	if truncated && m.Tabs == nil {
		// Legacy callers cannot retain read metadata; do not expose a prefix as complete.
		errMsg = truncatedPreviewNotice()
	}
	if errMsg != "" && m.ReadRemote != nil && !isTablePreviewPath(rel) {
		if remote, rerr := LoadPreviewRemote(m.ReadRemote, rel); rerr == "" {
			text, errMsg = remote, ""
			remotePreview = true
			truncated = false
		}
	}
	if m.Preview != nil {
		*m.Preview = text
	}
	if m.PreviewErr != nil {
		*m.PreviewErr = errMsg
	}
	if m.EditDraft != nil {
		*m.EditDraft = text
	}
	if m.StatNote != nil {
		*m.StatNote = ""
		if m.StatRemote != nil {
			*m.StatNote = m.StatRemote(rel)
		}
	}
}

func sourceStyle(m Model, fg, muted ui.Color) markdown.Style {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	return markdown.Style{
		FontSize: 12,
		FG:       fg,
		Muted:    muted,
		CodeBG:   chrome.Color(colors.Field),
		Accent:   chrome.Color(colors.Accent),
	}
}

func editViewModeOf(m Model) EditViewMode {
	if m.EditViewMode != nil {
		return *m.EditViewMode
	}
	return editViewModeFallback
}

func setEditViewMode(m Model, mode EditViewMode) {
	if m.EditViewMode != nil {
		*m.EditViewMode = mode
		return
	}
	editViewModeFallback = mode
}

// resolveEditViewMode returns the effective mode while editing.
// Non-highlightable drafts always use EditViewEdit.
// Unknown / unset modes prefer Split.
func resolveEditViewMode(m Model, canHighlight bool) EditViewMode {
	if !canHighlight {
		return EditViewEdit
	}
	switch mode := editViewModeOf(m); mode {
	case EditViewSplit, EditViewEdit, EditViewHighlight:
		return mode
	default:
		return EditViewSplit
	}
}

// cycleEditViewMode advances Split → Edit → Highlight → Split.
func cycleEditViewMode(mode EditViewMode) EditViewMode {
	switch mode {
	case EditViewSplit:
		return EditViewEdit
	case EditViewEdit:
		return EditViewHighlight
	default:
		return EditViewSplit
	}
}

// preferSideBySide reports whether the preview pane is wide enough for
// TextArea | SourceView instead of stacked preview under the editor.
func preferSideBySide(paneWidth float32) bool {
	return paneWidth >= sideBySideMinWidth
}

func paneWidthOf(m Model) float32 {
	if m.PaneWidth != nil {
		return *m.PaneWidth
	}
	return 0
}

func editSplitHandle(m Model) *float32 {
	if m.EditSplit != nil {
		return m.EditSplit
	}
	return &editSplitFallback
}

// languageChipLabel is the auto-detected chroma language for the path.
func languageChipLabel(path string) string {
	lang := markdown.LanguageFromPath(path)
	if lang == "" {
		return "text"
	}
	return lang
}

func editViewModeLabel(mode EditViewMode) string {
	switch mode {
	case EditViewSplit:
		return i18n.T(i18n.KeyFilesModeSplit)
	case EditViewHighlight:
		return i18n.T(i18n.KeyFilesModeHighlight)
	default:
		return i18n.T(i18n.KeyFilesModeEdit)
	}
}

// editViewModeCycleLabel is the toolbar action that switches to the next mode.
func editViewModeCycleLabel(current EditViewMode) string {
	return i18n.T(i18n.KeyFilesModeNext) + ": " + editViewModeLabel(cycleEditViewMode(current))
}

// LoadPreviewRemote reads via workspaceFiles/read when local preview fails.
func mediaMode(m Model, path string) bool {
	return mediapreview.Supports(path) && !(m.EditDraft != nil && m.Preview != nil && *m.EditDraft != *m.Preview)
}

func LoadPreviewRemote(readRemote func(rel string) (string, error), rel string) (text, errMsg string) {
	if readRemote == nil {
		return "", "no remote reader"
	}
	body, err := readRemote(rel)
	if err != nil {
		return "", err.Error()
	}
	return body, ""
}

// LoadPreview reads the selected file for display.
func LoadPreview(root, rel string) (text, errMsg string) {
	text, errMsg, truncated := loadPreview(root, rel)
	if truncated {
		return "", truncatedPreviewNotice()
	}
	return text, errMsg
}

func loadPreview(root, rel string) (text, errMsg string, truncated bool) {
	if mediapreview.Supports(rel) {
		return "", "", false
	} // No client-side path resolution or binary text read.
	full, err := workspacefs.JoinRoot(root, rel)
	if err != nil {
		return "", err.Error(), false
	}
	if isPreviewImage(rel) {
		return "", "", false // Decoded separately; never offer binary image bytes for editing.
	}
	switch workspacefs.ClassifyDoc(rel) {
	case workspacefs.DocPDF:
		return "", i18n.T(i18n.KeyFilesPDFHint), false
	case workspacefs.DocXLSX:
		page, xerr := readTablePreview(context.Background(), full, workspacefs.XLSXPageRequest{})
		table := ""
		if page != nil {
			table = page.Text
		}
		if xerr == nil && table != "" {
			return table, "", false
		}
		hint := i18n.T(i18n.KeyFilesXLSXHint)
		if xerr != nil {
			hint = hint + " (" + xerr.Error() + ")"
		}
		return "", hint, false
	case workspacefs.DocOther:
		return "", i18n.T(i18n.KeyFilesOfficeHint), false
	}
	res, err := workspacefs.ReadFileTextPreview(full)
	if err != nil {
		switch {
		case errors.Is(err, workspacefs.ErrTooLarge):
			return "", i18n.T(i18n.KeyFilesTooLarge), false
		case errors.Is(err, workspacefs.ErrBinary):
			return "", i18n.T(i18n.KeyFilesBinary), false
		case errors.Is(err, workspacefs.ErrNotFile):
			return "", err.Error(), false
		default:
			return "", err.Error(), false
		}
	}
	return res.Text, "", res.Truncated
}
