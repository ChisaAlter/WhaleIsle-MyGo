// Package files is the M7 workspace file tree and text preview in the right pane.
package files

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// EditViewMode is how the file editor shows draft text vs highlighted preview.
// Source remains selectable in every mode; syntax colors decorate native text.
type EditViewMode int

const (
	// EditViewSplit shows TextArea + SourceView sharing the draft when selected.
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
	// TreeOnly is the Files dock resource: the explorer owns this entire seat.
	// File activation is handed to OnOpenResource instead of nesting an editor.
	TreeOnly       bool
	OnOpenResource func(path string) error
	// ResourceOnly is an explicitly addressed resource, not an explorer grant.
	ResourceOnly bool
	// HostOnly forbids reading a coincidentally matching desktop path.
	HostOnly           bool
	Media              *media.Owner
	OpenDocumentViewer func(string) (DocumentSurface, error)
	// OpenDeliveryDocument accepts a validated local saved file in a read-only
	// document route. Nil means that route was accepted, not that bytes rendered.
	// It must report asynchronous loading/errors and must never use Host readers
	// or a system/default-application fallback.
	OpenDeliveryDocument func(absPath string, kind DeliveryPreviewKind) error
	ReadTextPageContext  func(context.Context, string, int64) (workspacefs.TextPage, error)
	AllowPageShortcuts   func() bool
	PlanOnly             bool
	PlanSnapshot         session.ViewState
	OnLoadPlanHistory    func()
	ReadSubagentPlan     func(context.Context, string, session.PlanResourceTarget, func()) *session.PlanResourceRead
	Tabs                 *Tabs
	WorkspaceRoot        string
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
	EditSplit *float32
	// OnNewTab opens the owner's new-tab flow from the strip's trailing control.
	OnNewTab          func()
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
	// OpenWithState belongs to the captured Host/session/root/path request.
	OpenWithState     *OpenWithState
	OnRefreshOpenWith func()
	// EditorApps are probed external editors (Electron editors.js) offered in
	// the same open-with menu.
	EditorApps []OpenWithApp
	// OnOpenWith opens path via session/openWorkspacePath with application id.
	OnOpenWith func(absPath, appID string)
	// OnOpenWithEditor launches a probed external editor at line (0 = no line).
	// The caller falls back to the system default handler when not installed.
	OnOpenWithEditor func(absPath, editorID string, line int)
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
	ReadImageContext  func(context.Context, string, int64) ([]byte, error)
	ImageGeneration   uint64
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
	ID      string
	Name    string
	Default bool
}

type OpenWithState struct {
	Loading bool
	Error   string
}

// editViewModeFallback persists the view mode when Model.EditViewMode is nil.
var editViewModeFallback = EditViewEdit

// editSplitFallback is the first-pane size when Model.EditSplit is nil.
var editSplitFallback float32 = 200

// View paints the file tree and plain-text preview.
func View(c *ui.Context, m Model) {
	m.notify = c.Invalidate
	previewPreferences(m)
	if m.Tabs != nil {
		m.Tabs.Bind(m.BindingID, m.WorkspaceRoot)
	}
	if m.TreeOnly {
		viewExplorer(c, m)
		return
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
	syncDeliveryPreview(m)
	syncTextRead(m)
	syncDocumentSurfaces(m)
	syncSaveResults(m)
	observeFileResource(m)
	if t := selectedTab(m); t != nil && t.Editing {
		_ = ensureSaveQueue(m, t)
	}
	defer observeAutosave(m)
	if (m.AllowPageShortcuts == nil || m.AllowPageShortcuts()) && m.SelectedFile != nil && *m.SelectedFile != "" {
		if c.PageShortcut(ui.Cmd, ui.KeyS) {
			if err := requestSaveDraft(m); err != nil {
				setFileError(m, err)
			}
		}
		if c.PageShortcut(ui.Cmd, ui.KeyF) {
			triggerPreviewFind(m)
		}
	}
	if m.Tabs != nil {
		if m.Tabs.Search == nil {
			m.Tabs.Search = &workspacefs.Search{}
		}
		if m.Tabs.Tree == nil {
			m.Tabs.Tree = &workspacefs.DirectoryTree{}
		}
		m.Tabs.Tree.EnsureHostOnly(m.BindingID, m.WorkspaceRoot, m.HostOnly)
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
	if m.ResourceOnly {
		previewPane(c, m, fg, muted, border, field)
		return
	}

	root := strings.TrimSpace(m.WorkspaceRoot)
	paneBg := field
	if chrome.BackdropActive() {
		paneBg = ui.Transparent
	}
	ui.Column(c).Fill().MinWidth(0).Background(paneBg).Children(func() {
		ui.Row(c).Padding(12, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyFilesTitle)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			if w := tabsFor(m); w != nil {
				automatic := !w.AutoRefreshDisabled
				if coss.Checkbox(c, &automatic, label("自动刷新文件树", "Auto-refresh tree"), coss.CheckOptions{}).Changed() {
					w.AutoRefreshDisabled = !automatic
					m.Tabs.Tree.Observe(m.WatchRemoteContext, automatic)
					c.Invalidate()
				}
			}
			if widgets.Button(c, colors, label("刷新", "Refresh"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() && m.Tabs != nil {
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
				input := coss.Input(c, m.Filter, coss.InputOptions{
					Size:        coss.InputSM,
					Label:       i18n.T(i18n.KeyFilesFilter),
					Placeholder: i18n.T(i18n.KeyFilesFilter),
				})
				if !input.Composing() && input.Shortcut(0, ui.KeyEscape) {
					*m.Filter = ""
				}
			})
		}
		ui.Box(c).FillWidth().Height(0.5).Background(border)
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
			m.Tabs.Search.EnsureHostOnly(m.BindingID, m.WorkspaceRoot, filter, m.ListRemoteContext, c.Invalidate, m.HostOnly)
		}
		ui.SplitVertical(c, split, func() {
			paintVirtualizedFileTree(c, m, root, filter, true, fg, muted, accent)
		}, func() {
			previewPane(c, m, fg, muted, border, field)
		}).Fill()
	})
}

type fileTreeRowKind uint8

const (
	vtreeRootHeader fileTreeRowKind = iota
	vtreeStatusText
	vtreeErrorText
	vtreeEmptyDir
	vtreeOtherEntry
	vtreeDirEntry
	vtreeFileEntry
	vtreeSearchResult
	vtreeSearchMore
)

type fileTreeVirtualRow struct {
	kind       fileTreeRowKind
	key        string
	rel        string
	name       string
	depth      int
	shownCount int
	totalCount int
}

var filesTreeVirtualRowHook func(mode string, index int, key string)

func collectFilteredTreeRows(m Model, out *[]fileTreeVirtualRow) {
	if m.Tabs == nil || m.Tabs.Search == nil {
		return
	}
	state := m.Tabs.Search.Snapshot()
	if state.Busy {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeStatusText, key: "search:busy", name: label("正在搜索文件…", "Searching all files…")})
		return
	}
	if state.Error != "" {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeErrorText, key: "search:err", name: state.Error})
		return
	}
	for _, path := range state.Paths {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeSearchResult, key: "hit:" + path, rel: path})
	}
	if len(state.Paths) < state.Total {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeSearchMore, key: "search:more", shownCount: len(state.Paths), totalCount: state.Total})
	}
	if state.Total == 0 {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeStatusText, key: "search:empty", name: i18n.T(i18n.KeyFilesFilterEmpty)})
	}
}

func collectDirTreeRows(m Model, rel string, depth int, invalidate func(), out *[]fileTreeVirtualRow) {
	if m.Tabs == nil || m.Tabs.Tree == nil {
		return
	}
	state := m.Tabs.Tree.Get(rel, m.ListRemoteContext, invalidate)
	if state.Busy {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeStatusText, key: "busy:" + rel, name: label("加载中…", "Loading…")})
		return
	}
	if state.Error != "" {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeErrorText, key: "err:" + rel, name: state.Error})
		if len(state.Entries) == 0 {
			return
		}
	}
	if state.WatchError != "" {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeStatusText, key: "watch:" + rel, name: state.WatchError})
	}
	ents := state.Entries
	if len(ents) == 0 {
		*out = append(*out, fileTreeVirtualRow{kind: vtreeEmptyDir, key: "empty:" + rel, name: label("空目录", "This folder is empty"), depth: depth})
		return
	}
	for _, e := range ents {
		childRel := e.Name
		if rel != "" {
			childRel = rel + "/" + e.Name
		}
		if e.Other {
			*out = append(*out, fileTreeVirtualRow{kind: vtreeOtherEntry, key: "other:" + childRel, rel: childRel, name: e.Name, depth: depth})
		} else if e.IsDir {
			*out = append(*out, fileTreeVirtualRow{kind: vtreeDirEntry, key: "dir:" + childRel, rel: childRel, name: e.Name, depth: depth})
			if isExpanded(m, childRel) {
				collectDirTreeRows(m, childRel, depth+1, invalidate, out)
			}
		} else {
			*out = append(*out, fileTreeVirtualRow{kind: vtreeFileEntry, key: "file:" + childRel, rel: childRel, name: e.Name, depth: depth})
		}
	}
}

func paintFileTreeRow(rc *ui.Context, m Model, vr fileTreeVirtualRow, colors theme.Colors, fg, muted, accent ui.Color) {
	switch vr.kind {
	case vtreeRootHeader:
		ui.Text(rc, vr.name).FontSize(theme.InterfaceSize(12)).Bold().TextColor(muted).Padding(4, 8, 4, 8)
	case vtreeStatusText:
		ui.Text(rc, vr.name).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(8, 8, 8, 8)
	case vtreeErrorText:
		ui.Text(rc, vr.name).FontSize(theme.InterfaceSize(11)).TextColor(accent).Padding(8, 8, 8, 8)
	case vtreeEmptyDir:
		ui.Text(rc, vr.name).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(4, 8, 4, float32(8+vr.depth*12))
	case vtreeOtherEntry:
		ui.Text(rc, vr.name).FillWidth().FontSize(theme.InterfaceSize(12.5)).TextColor(muted).SingleLine().Ellipsis("…").
			Padding(4, 8, 4, float32(26+vr.depth*12)).
			Tooltip(label("此项目不是普通文件或文件夹，无法打开", "This entry is not a regular file or directory and cannot be opened"))
	case vtreeDirEntry:
		rowDir(rc, m, vr.rel, vr.name, vr.depth, fg, muted, accent)
	case vtreeFileEntry:
		rowFile(rc, m, vr.rel, vr.name, vr.depth, fg, muted, accent)
	case vtreeSearchResult:
		path := vr.rel
		row := ui.Row(rc).Key(treeFileRowKey(m, path)).Label("File result "+path).Focusable().Height(28).Radius(8).Padding(0, 8).Gap(6).FillWidth().MinWidth(0).AlignItems(ui.Center).ContextMenu(treeFileMenu(m, path)).Drag(workspaceReference(m, path))
		if m.SelectedFile != nil && *m.SelectedFile == path {
			row.Background(accent.Alpha(0.12))
		}
		row.Children(func() {
			ui.Text(rc, path).FontSize(theme.InterfaceSize(12.5)).FixedLineHeight(18).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
			paintTreeReferenceButton(rc, m, row, path, muted)
		})
		if row.Clicked() {
			openTreeFile(m, path)
		}
	case vtreeSearchMore:
		if widgets.Button(rc, colors, fmt.Sprintf(label("加载更多（%d/%d）", "Show more (%d/%d)"), vr.shownCount, vr.totalCount), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			if m.Tabs != nil && m.Tabs.Search != nil {
				m.Tabs.Search.More()
			}
		}
	}
}

// paintDir renders a single directory subtree directly for standalone callers
// and parity tests; interactive file views use paintVirtualizedFileTree.
func paintDir(c *ui.Context, m Model, _ string, rel string, depth int, fg, muted, accent ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	var rows []fileTreeVirtualRow
	collectDirTreeRows(m, rel, depth, c.Invalidate, &rows)
	for _, vr := range rows {
		paintFileTreeRow(c, m, vr, colors, fg, muted, accent)
	}
}

func paintVirtualizedFileTree(c *ui.Context, m Model, root, filter string, showRootName bool, fg, muted, accent ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	var rows []fileTreeVirtualRow
	if showRootName {
		rows = append(rows, fileTreeVirtualRow{kind: vtreeRootHeader, key: "root:" + root, name: filepath.Base(root)})
	}
	mode := "tree"
	if filter != "" {
		mode = "search"
		collectFilteredTreeRows(m, &rows)
	} else {
		collectDirTreeRows(m, "", 0, c.Invalidate, &rows)
	}
	// .body in the original's FilesPanel.module.css: rows sit flush, inset 8/12/16.
	host := ui.Column(c.Key("file-tree-host")).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Padding(8, 12, 16, 12).Label(label("文件树", "File tree"))
	listState := ui.Local(host, "file-tree-list-state:"+mode, func() ui.ListState { return ui.ListState{} })
	host.Children(func() {
		if len(rows) == 0 {
			return
		}
		ui.List(c.Key("file-tree-list:"+mode), listState, len(rows)).
			ItemKey(func(i int) any { return rows[i].key }).
			FillWidth().Grow(1).MinWidth(0).MinHeight(0).
			Rows(func(lr ui.ListRow) {
				vr := rows[lr.Index]
				if filesTreeVirtualRowHook != nil {
					filesTreeVirtualRowHook(mode, lr.Index, vr.key)
				}
				paintFileTreeRow(lr.Context, m, vr, colors, fg, muted, accent)
			})
	})
}

func rowDir(c *ui.Context, m Model, rel, name string, depth int, fg, muted, accent ui.Color) {
	expanded := isExpanded(m, rel)
	// The original indents by the nested ul's padding-left, so the row itself
	// narrows with depth; its inner padding stays 8.
	row := ui.Row(c).Key(workspaceReference(m, rel)).Label("Folder "+rel).Focusable().Margin(0, 0, 0, float32(depth*12)).Padding(0, 8).Gap(6).MinWidth(0).Height(28).Radius(8).AlignItems(ui.Center).ContextMenu(treeDirectoryMenu(m, rel))
	if row.Hovered() || row.FocusWithin() {
		row.Background(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Hover))
	}
	row.Children(func() {
		paintTreeTwist(c, expanded, muted)
		paintTreeEntryIcon(c, true, muted)
		ui.Text(c, name).FontSize(theme.InterfaceSize(12.5)).FixedLineHeight(18).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
	})
	if row.Clicked() {
		toggleExpanded(m, rel)
	}
	_ = accent
}

func rowFile(c *ui.Context, m Model, rel, name string, depth int, fg, muted, accent ui.Color) {
	selected := m.SelectedFile != nil && *m.SelectedFile == rel
	// The original's .item li lays the row button (flex: 1) and the 20px
	// mention button side by side; the row narrows by that accessory's seat,
	// and the li itself indents by the nested ul's padding-left.
	wrap := ui.Row(c).MinWidth(0).Margin(0, 0, 0, float32(depth*12)).AlignItems(ui.Center)
	wrap.Children(func() {
		row := ui.Row(c).Key(treeFileRowKey(m, rel)).Label("File "+rel).Focusable().Padding(0, 8).Gap(6).Grow(1).MinWidth(0).Height(28).Radius(8).AlignItems(ui.Center).ContextMenu(treeFileMenu(m, rel)).Drag(workspaceReference(m, rel))
		if selected {
			row.Background(accent.Alpha(0.12))
		} else if row.Hovered() || row.FocusWithin() {
			row.Background(chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Hover))
		}
		row.Children(func() {
			ui.Box(c).Width(12)
			paintTreeEntryIcon(c, false, muted)
			ui.Text(c, name).FontSize(theme.InterfaceSize(12.5)).FixedLineHeight(18).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
		})
		if row.Clicked() {
			openTreeFile(m, rel)
		}
		paintTreeReferenceButton(c, m, wrap, rel, muted)
	})
	_ = muted
}

func previewPane(c *ui.Context, m Model, fg, muted, border, field ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	paneBg := field
	if chrome.BackdropActive() {
		paneBg = ui.Transparent
	}
	ui.Column(c).Fill().MinHeight(0).Background(paneBg).Label(label("文件预览", "File preview")).Children(func() {
		ui.Box(c).FillWidth().Height(0.5).Background(border)
		if !m.ResourceOnly {
			paintTabs(c, m, fg, muted)
		} else if w := tabsFor(m); w != nil && w.PendingClose != "" {
			paintPendingClose(c, m, w, muted)
		}
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
		var mode EditViewMode

		paintPreviewToolbar(c, m, sel, muted, border)
		if paintDocumentSurface(c, m) {
			return
		}
		if tab := selectedTab(m); tab != nil && tab.readLoading {
			// A floating chip: a row in the flow would push the document down while it reads.
			chrome.FrostSurface(ui.Row(c).Absolute().Top(4).Right(12).Gap(8).AlignItems(ui.Center).Padding(2, 8), theme.Resolve(m.ThemeID).Active(m.SystemDark), m.SystemDark, c.Theme().Surface, 8).Children(func() {
				ui.Text(c, label("正在读取…", "Reading file…")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if widgets.Button(c, colors, label("取消读取", "Cancel read"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
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
				ui.Column(c).Padding(12, 16, 16, 16).FillWidth().Label(label("Markdown 内容", "Markdown content")).Children(func() {
					paintMarkdownResource(c, m, fg, muted)
				})
			})
			return
		}
		paintPreviewFindBar(c, m, sel, border, muted)
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
		CodeSurface: func(e ui.Element, radius float32) ui.Element {
			return chrome.FrostSurface(e, colors, m.SystemDark, chrome.Color(colors.Field), radius)
		},
		InlineCodeBG: chrome.FrostInset(colors, m.SystemDark, markdown.InlineCodeBase(m.SystemDark)),
		// The Original FilePreview passes no toolbarLabels, so its fences use
		// the plain .banner recipe (36px header, text copy action, 16px inset).
		PlainCodeBanner: true,
		CopyLabel:       FilesLocale("preview.copy"),
		CopiedLabel:     FilesLocale("preview.copied"),
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
// Unknown modes use the source editor alone, like desktop-file's textarea.
func resolveEditViewMode(m Model, canHighlight bool) EditViewMode {
	if !canHighlight {
		return EditViewEdit
	}
	switch mode := editViewModeOf(m); mode {
	case EditViewSplit, EditViewEdit, EditViewHighlight:
		return mode
	default:
		return EditViewEdit
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
		return "", errcopy.Sentence(err)
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
		return "", errcopy.Sentence(err), false
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
			hint = errcopy.Under(hint, xerr)
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
			return "", errcopy.Sentence(err), false
		default:
			return "", errcopy.Sentence(err), false
		}
	}
	return res.Text, "", res.Truncated
}
