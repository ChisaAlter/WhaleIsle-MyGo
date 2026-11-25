package files

import (
	"path/filepath"
	"strconv"
	"strings"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// desktop-files.ts canOpenDesktopFile chooses the component by resource
// ownership, not by its editor's current mode. Loading, images, a saved draft,
// or Keep draft must not turn the same resource into TextPreview.
func desktopFileToolbar(m Model, path string) bool {
	if m.SessionID == "" || strings.TrimSpace(m.WorkspaceRoot) == "" {
		return false
	}
	if _, ok := treePathCopies(m, path); !ok {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".doc", ".docx", ".ppt", ".pptx", ".xls", ".xlsx":
		return false
	}
	return true
}

// previewSupportsLineNavigation reports whether this preview has numbered text
// lines to jump to. Images, audio/video, PDF and workbook page previews have no
// line model, so offering "Go to line…" there would be a no-op control.
func previewSupportsLineNavigation(m Model, path string) bool {
	return isAsyncTextPath(path) && !imageMode(m, path) && !mediaMode(m, path)
}

func previewEditable(m Model, path string) bool {
	kind := workspacefs.ClassifyDoc(path)
	if tab := selectedTab(m); tab != nil && tab.deliverySource != nil {
		return false
	}
	return m.EditDraft != nil && m.Editing != nil && !textReadBlocked(m) && !previewTruncated(m) &&
		!imageMode(m, path) && !mediaMode(m, path) && (kind == workspacefs.DocNone || kind == workspacefs.DocHTML) &&
		(m.PreviewErr == nil || *m.PreviewErr == "") && (selectedTab(m) == nil || !selectedTab(m).readLoading)
}

func previewDirty(m Model) bool {
	return m.Preview != nil && m.EditDraft != nil && *m.Preview != *m.EditDraft
}

func explicitSaveDisabled(m Model) bool {
	if m.OnSaveEdit == nil || m.WorkspaceRoot == "" || !previewDirty(m) {
		return true
	}
	if tab := selectedTab(m); tab != nil {
		return tab.deliverySource != nil || tab.Remote || isTablePreviewPath(tab.Path) || tab.readLoading || tab.readResume || tab.hasConflict
	}
	return previewTruncated(m) || textReadBlocked(m)
}

type previewToolbarOwner struct {
	read textReadOwner
	tab  *FileTab
}

func previewToolbarOwnerFor(m Model, path string) previewToolbarOwner {
	owner := previewToolbarOwner{read: textReadOwner{binding: m.BindingID, root: m.WorkspaceRoot, session: m.SessionID, path: path}}
	if tab := selectedTab(m); tab != nil {
		owner.read, owner.tab = textReadOwnerFor(m, tab), tab
	}
	return owner
}

func previewTool(c *ui.Context, key, caption string, icon *ui.SVG, ink ui.Color) ui.Element {
	glyph := float32(15)
	if icon == previewFloatingIcon {
		glyph = 14
	} else if icon == previewOpenIcon {
		glyph = 13
	} else if icon == previewChevronIcon {
		glyph = 10
	}
	return coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonIconSM,
		Label:   caption,
		Key:     key,
		Content: func() {
			ui.Icon(c, icon).Size(glyph, glyph).TextColor(ink).PassThrough()
		},
	}).Tooltip(caption).Shrink(0).Radius(design.RadiusMD)
}

func previewTextTool(c *ui.Context, caption string) ui.Element {
	return coss.Button(c, caption, coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonSM,
	}).Shrink(0).Radius(design.RadiusMD)
}

func togglePreviewWrap(m Model) {
	prefs := previewPreferences(m)
	prefs.WordWrap = !prefs.WordWrap
	if m.Tabs != nil && m.Tabs.floating != nil && m.Tabs.floating.win != nil {
		m.Tabs.floating.win.Invalidate()
	}
}

func activateFloatingPreview(m Model, path string) {
	if tab := selectedTab(m); tab != nil && tab.deliverySource != nil && isPreviewImage(path) {
		setFileStatus(m, label("本地交付图片已在只读预览中打开。", "The local delivered image is open in this read-only preview."))
		return
	}
	kind := workspacefs.ClassifyDoc(path)
	if mediaMode(m, path) && m.OnOpenMedia != nil {
		m.OnOpenMedia(path, true)
	} else if (isPreviewImage(path) || kind == workspacefs.DocPDF || kind == workspacefs.DocHTML) && m.OnOpenFloatingDoc != nil {
		m.OnOpenFloatingDoc(path)
	} else if err := openFloatingPreview(m); err != nil {
		setFileError(m, err)
	}
}

func paintPreviewToolbar(c *ui.Context, m Model, path string, muted, border ui.Color) {
	prepareDocumentViewer(m)
	desktop := desktopFileToolbar(m, path)
	height, gap := float32(38), float32(4)
	if desktop {
		height, gap = 36, 8 // FilePreview .toolbar: 8px top padding + 28px controls.
	}
	// Native menu selection returns in a later frame. Rekey the trigger with
	// the same resource/read owner used by async reads, so old choices cannot
	// execute against a replacement Host, selected tab or reopened resource.
	row := ui.Row(c).Key(previewToolbarOwnerFor(m, path)).Label("File preview toolbar").Height(height).Shrink(0).
		FillWidth().MinWidth(0).Gap(gap).AlignItems(ui.Center)
	if desktop {
		row.Padding(8, 16, 0, 16).DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 0.5, W: r.W, H: 0.5}, border, 0)
		})
	} else {
		row.Padding(0, 6, 0, 16).DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 0.5, W: r.W, H: 0.5}, border, 0)
		})
	}
	jump := ui.Local(row, "line-navigation", func() bool { return false })
	row.ContextMenu(func(menu *ui.Menu) {
		treeFileMenu(m, path)(menu)
		if previewSupportsLineNavigation(m, path) {
			if menu.Item(label("跳转到行…", "Go to line…")).Chosen() {
				*jump = true
			}
			if menu.Item(label("查找…", "Find…")).Chosen() {
				triggerPreviewFind(m)
			}
		}
	})
	previewFailure := paintPreviewFeedback(c, m, path, row)
	row.Children(func() {
		if desktop {
			paintDesktopFileCrumbs(c, m, path, muted)
		} else {
			paintToolbarPath(c, m, path, muted)
		}
		if tab := selectedTab(m); !desktop && tab != nil && tab.document.surface != nil && tab.document.surface.Alive() && m.OpenDocumentViewer != nil {
			caption := label("原生预览", "Native preview")
			if tab.document.enabled {
				caption = label("文档预览器", "Document viewer")
			}
			previewTextTool(c, caption).MaxWidth(160).Tooltip(label("选择预览器", "Choose viewer")).Disabled(m.Editing != nil && *m.Editing).
				Menu(func(menu *ui.Menu) { previewViewerChoices(menu, m, tab) })
		}
		if desktop {
			if previewEditable(m, path) || previewDirty(m) {
				if isMarkdown(path) {
					paintPreviewMarkdownToggle(c, m)
				}
				// FilePreview.tsx draws the wrap toggle as a ghost sm text
				// button (aria-pressed carries the state, no extra paint).
				if previewTextTool(c, label("自动换行", "Wrap lines")).Clicked() {
					togglePreviewWrap(m)
				}
			}
		} else if !imageMode(m, path) && !mediaMode(m, path) && isAsyncTextPath(path) {
			icon := previewWrapIcon
			caption := label("自动换行", "Wrap lines")
			if previewPreferences(m).WordWrap {
				icon, caption = previewNowrapIcon, label("关闭换行", "Unwrap lines")
			}
			if previewTool(c, "preview-wrap", caption, icon, muted).Clicked() {
				togglePreviewWrap(m)
			}
		}
		if m.Tabs != nil && !desktop {
			if previewTool(c, "preview-floating", label("独立窗口预览", "Preview in separate window"), previewFloatingIcon, muted).Clicked() {
				activateFloatingPreview(m, path)
			}
		}
		if desktop {
			paintPreviewSelectionActions(c, m, path)
			if previewEditable(m, path) || previewDirty(m) {
				caption, busy := previewSaveCaption(m)
				if coss.Button(c, caption, coss.ButtonOptions{
					Variant:  coss.ButtonDefault,
					Size:     coss.ButtonSM,
					Disabled: busy || explicitSaveDisabled(m),
				}).Shrink(0).Radius(design.RadiusMD).Clicked() {
					if err := requestSaveDraft(m); err != nil {
						setFileError(m, err)
					}
				}
			}
			// The original's toolbar ends at the save button; MyGo's extra
			// preview actions stay discoverable on one visible options button.
			previewTool(c, "preview-options", label("文件选项", "File options"), previewMoreIcon, muted).Menu(func(menu *ui.Menu) {
				if m.Tabs != nil && menu.Item(label("独立窗口预览", "Preview in separate window")).Chosen() {
					activateFloatingPreview(m, path)
				}
				treeFileMenu(m, path)(menu)
				if previewSupportsLineNavigation(m, path) && menu.Item(label("跳转到行…", "Go to line…")).Chosen() {
					*jump = true
				}
			})
		} else {
			if previewTool(c, "preview-reload", label("重新加载", "Reload"), previewReloadIcon, muted).Clicked() {
				reloadSelectedFile(m)
			}
			paintPreviewOpenTarget(c, m, path, muted)
		}
	})
	if *jump && previewSupportsLineNavigation(m, path) {
		paintPreviewLineNavigation(c, m, jump)
	}
	if previewFailure != "" {
		widgets.Alert(c, widgets.AlertOptions{ID: "file-preview/" + path, Message: previewFailure})
	}
}

func previewSaveCaption(m Model) (string, bool) {
	if tab := selectedTab(m); tab != nil && tab.save != nil {
		state := tab.save.Snapshot()
		if state.Running || state.Pending {
			return label("保存中…", "Saving…"), true
		}
		if state.Revision > 0 && state.Error == "" && !previewDirty(m) {
			return "✓ " + i18n.T(i18n.KeyFilesSaved), false
		}
	}
	return i18n.T(i18n.KeyFilesSave), false
}

// Feedback belongs to the fixed toolbar or overlay, never an inserted editor
// row. The resource owner keys the receipt so another file cannot consume it.
// feedbackEntry is the line the preview toolbar reports: the edit status, or
// the save queue's failure in copy a person can act on.
func feedbackEntry(editStatus string, state workspacefs.SaveSnapshot) (text, kind string) {
	text, kind = editStatus, "info"
	if state.Error != "" {
		text, kind = errcopy.SentenceText(state.Error), "error"
	}
	if state.Running || state.Pending {
		text = ""
	}
	return text, kind
}

// paintPreviewFeedback hands a save's outcome back to the pane, which writes it
// under its own toolbar; there is no floating acknowledgement.
func paintPreviewFeedback(c *ui.Context, m Model, path string, owner ui.Element) string {
	last := ui.Local(owner, "preview-feedback", func() string { return "" })
	status := ""
	if m.EditStatus != nil {
		status = *m.EditStatus
	}
	if tab := selectedTab(m); tab != nil && tab.save != nil {
		status, _ = feedbackEntry(status, tab.save.Snapshot())
	}
	if status == i18n.T(i18n.KeyFilesSaved) || status == label("正在保存…", "Saving…") {
		status = ""
	}
	// Success and failure land in the pane's status line alike; the model
	// clears the outcome on the next edit.
	if status != "" && status != *last {
		*last = status
		return status
	}
	*last = status
	return ""
}

func paintDesktopFileCrumbs(c *ui.Context, m Model, path string, muted ui.Color) {
	root := strings.TrimRight(strings.ReplaceAll(m.WorkspaceRoot, `\`, "/"), "/")
	project := root[strings.LastIndex(root, "/")+1:]
	parts := append([]string{project}, strings.Split(path, "/")...)
	ui.Row(c).Key("preview-crumbs").Label("File breadcrumbs").Grow(1).MinWidth(0).Gap(0).ClipX().AlignItems(ui.Center).
		Tooltip(previewPathTooltip(m, path)).Children(func() {
		for index, part := range parts {
			if index > 0 {
				ui.Box(c).Width(22).Height(14).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).
					Children(func() { ui.Icon(c, previewCrumbIcon).Size(14, 14).TextColor(muted) })
			}
			ink := muted
			if index == len(parts)-1 {
				ink = c.Theme().Text
			}
			ui.Text(c, part).FontSize(12).FixedLineHeight(18).TextColor(ink).SingleLine().MaxWidth(120).Shrink(0).Tooltip(part)
		}
	})
}

func paintPreviewOpenTarget(c *ui.Context, m Model, path string, muted ui.Color) {
	if m.OnOpenDoc == nil && m.OnReveal == nil && m.OnOpenWith == nil && !previewHasEditorApps(m) {
		return
	}
	border := c.Theme().Border
	ui.Row(c).Gap(0).Shrink(0).Height(24).Radius(6).BorderWidth(0.5).BorderColor(border).Clip().Children(func() {
		caption := i18n.T(i18n.KeyFilesOpenExternal)
		if m.OnOpenDoc == nil {
			caption = i18n.T(i18n.KeyFilesReveal)
		}
		if previewTool(c, "preview-open", caption, previewOpenIcon, muted).Width(23).Height(23).Radius(0).Clicked() {
			if m.OnOpenDoc != nil {
				withToolbarPath(m, path, textPreviewPath(m, path), func(full string) { m.OnOpenDoc(full, workspacefs.ClassifyDoc(path), false) })
			} else {
				revealToolbarPath(m, path, textPreviewPath(m, path))
			}
		}
		if m.OnReveal != nil || (m.OnOpenWith != nil && len(m.OpenWithApps) > 0) || previewHasEditorApps(m) {
			previewTool(c, "preview-open-menu", label("更多打开方式", "More open options"), previewChevronIcon, muted).Width(17).Height(23).Radius(0).
				DrawOver(func(p *ui.Painter, r ui.Rect) { p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 0.5, H: r.H}, border, 0) }).
				Menu(func(menu *ui.Menu) { previewOpenMenu(menu, m, path) })
		}
	})
}

// previewHasEditorApps reports whether the probed external editors are usable
// from this pane (Electron's editors.js menu next to the harness apps).
func previewHasEditorApps(m Model) bool {
	return m.OnOpenWithEditor != nil && len(m.EditorApps) > 0
}

// previewOpenLine is the 1-based line last requested for path, or 0 when the
// editor should just open the file. revealLine is the pending jump target;
// lineDraft keeps the last explicitly requested line after it is consumed.
func previewOpenLine(m Model, path string) int {
	t := selectedTab(m)
	if t == nil || t.Path != path {
		return 0
	}
	if t.revealLine > 0 {
		return t.revealLine
	}
	if line, err := strconv.Atoi(strings.TrimSpace(t.lineDraft)); err == nil && line > 0 {
		return line
	}
	return 0
}

func previewEditorMenu(menu *ui.Menu, m Model, path string) {
	if previewHasEditorApps(m) {
		for _, app := range m.EditorApps {
			caption := strings.TrimSpace(app.Name)
			if caption == "" {
				caption = app.ID
			}
			editorID := app.ID
			if menu.Item(caption).Chosen() {
				line := previewOpenLine(m, path)
				withToolbarPath(m, path, textPreviewPath(m, path), func(full string) { m.OnOpenWithEditor(full, editorID, line) })
			}
		}
	}
}

func previewOpenMenu(menu *ui.Menu, m Model, path string) {
	previewEditorMenu(menu, m, path)
	if state := m.OpenWithState; state != nil {
		if state.Loading {
			menu.Item(label("正在查找应用…", "Finding applications…")).Disabled(true)
		}
		if state.Error != "" {
			menu.Item(state.Error).Disabled(true)
		}
	}
	if m.OnRefreshOpenWith != nil && menu.Item(label("刷新应用列表", "Refresh applications")).Chosen() {
		m.OnRefreshOpenWith()
	}
	if m.OnOpenWith != nil {
		for _, app := range m.OpenWithApps {
			caption := strings.TrimSpace(app.Name)
			if caption == "" {
				caption = app.ID
			}
			if menu.Item(caption).Chosen() {
				withToolbarPath(m, path, textPreviewPath(m, path), func(full string) { m.OnOpenWith(full, app.ID) })
			}
		}
	}
	if m.OnReveal != nil && menu.Item(i18n.T(i18n.KeyFilesReveal)).Chosen() {
		revealToolbarPath(m, path, textPreviewPath(m, path))
	}
}

func paintPreviewMarkdownToggle(c *ui.Context, m Model) {
	t := selectedTab(m)
	if t == nil {
		return
	}
	caption := label("渲染", "Rendered")
	if t.RenderedMarkdown {
		caption = label("源码", "Source")
	}
	if previewTextTool(c, caption).Clicked() {
		t.RenderedMarkdown = !t.RenderedMarkdown
		t.selectionText, t.revealLine = "", 0
	}
}

func previewSelection(m Model, path string) (string, bool) {
	t := selectedTab(m)
	if t == nil || t.RenderedMarkdown || t.selectionText != previewText(m) {
		return "", false
	}
	return FormatSelectedLines(path, previewText(m), t.selectionStart, t.selectionEnd)
}

func paintPreviewSelectionActions(c *ui.Context, m Model, path string) {
	if m.Editing == nil || !*m.Editing {
		return
	}
	if fragment, ok := previewSelection(m, path); ok {
		if previewTextTool(c, label("添加到对话", "Add to chat")).Disabled(m.OnAddToChat == nil || !m.CanAddToChat).Clicked() {
			if err := m.OnAddToChat(fragment); err != nil {
				setFileError(m, err)
			} else {
				setFileStatus(m, label("所选行已添加到对话草稿", "Selected lines added to the chat draft"))
			}
		}
	}
}

func paintPreviewLineNavigation(c *ui.Context, m Model, open *bool) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	t := selectedTab(m)
	if t == nil {
		*open = false
		return
	}
	ui.Row(c).Height(32).Gap(6).Padding(2, 16, 2, 16).Children(func() {
		var input ui.Element
		ui.Box(c).Width(80).Children(func() {
			input = coss.Input(c, &t.lineDraft, coss.InputOptions{
				Key:         "preview-line",
				Size:        coss.InputSM,
				Label:       "Line number",
				Placeholder: label("行号", "Line"),
			})
		})
		focus := ui.Local(input, "focus-once", func() bool { return false })
		if !*focus {
			input.Focus()
			*focus = true
		}
		if input.Shortcut(0, ui.KeyEscape) || widgets.Button(c, colors, label("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
			*open = false
			*focus = false
			return
		}
		if input.Submitted() && !input.Composing() || widgets.Button(c, colors, label("跳转行", "Go to line"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
			line, err := strconv.Atoi(strings.TrimSpace(t.lineDraft))
			if err != nil || line < 1 {
				setFileStatus(m, label("请输入正整数行号", "Enter a positive line number"))
			} else {
				RequestLine(m, line)
				*open = false
				*focus = false
			}
		}
	})
}

// The existing tree/header context owns occasional native-only operations;
// none of these consumes a permanent row above the resource body.
func previewContextActions(menu *ui.Menu, m Model, path string) {
	if m.SelectedFile == nil || *m.SelectedFile != path {
		return
	}
	menu.Separator()
	if menu.Item(label("重新加载", "Reload")).Chosen() {
		reloadSelectedFile(m)
	}
	previewOpenMenu(menu, m, path)
	if mediaMode(m, path) && m.OnOpenMedia != nil && menu.Item(label("音视频预览", "Media preview")).Chosen() {
		m.OnOpenMedia(path, false)
	}
	kind := workspacefs.ClassifyDoc(path)
	if m.OnOpenDoc != nil && (kind == workspacefs.DocPDF || kind == workspacefs.DocHTML) && menu.Item(i18n.T(i18n.KeyFilesOpenWebView)).Chosen() {
		m.OnOpenDoc(path, kind, true)
	}
	if m.OnRenderPDF != nil && (kind == workspacefs.DocOther || kind == workspacefs.DocXLSX) && menu.Item(i18n.T(i18n.KeyFilesRenderPDF)).Chosen() {
		m.OnRenderPDF(path)
	}
	editing := m.Editing != nil && *m.Editing
	if !editing && previewEditable(m, path) && menu.Item(i18n.T(i18n.KeyFilesEdit)).Chosen() {
		*m.Editing = true
		setFileStatus(m, "")
	}
	if !editing && previewEditable(m, path) && m.OnShowDiff != nil && menu.Item(i18n.T(i18n.KeyFilesShowDiff)).Chosen() {
		m.OnShowDiff(path)
	}
	if editing && m.EditDraft != nil && menu.Item(label("复制草稿", "Copy draft")).Chosen() {
		mygo.Clipboard.WriteText(*m.EditDraft)
	}
	if !desktopFileToolbar(m, path) && editing && m.OnSaveEdit != nil && m.EditDraft != nil && menu.Item(i18n.T(i18n.KeyFilesSave)).Disabled(explicitSaveDisabled(m)).Chosen() {
		if err := requestSaveDraft(m); err != nil {
			setFileError(m, err)
		}
	}
	if editing && menu.Item(label("保留草稿并预览", "Keep draft and preview")).Chosen() {
		*m.Editing = false
		setFileStatus(m, "")
	}
	if editing && markdown.CanSourceHighlight(path, previewText(m)) {
		mode := resolveEditViewMode(m, true)
		if menu.Item(editViewModeCycleLabel(mode)).Chosen() {
			setEditViewMode(m, cycleEditViewMode(mode))
		}
	}
	if fragment, ok := previewSelection(m, path); ok {
		if menu.Item(label("复制所选行", "Copy selected lines")).Chosen() {
			mygo.Clipboard.WriteText(fragment)
		}
		if menu.Item(label("添加到对话", "Add to chat")).Disabled(m.OnAddToChat == nil || !m.CanAddToChat).Chosen() {
			if err := m.OnAddToChat(fragment); err != nil {
				setFileError(m, err)
			} else {
				setFileStatus(m, label("所选行已添加到对话草稿", "Selected lines added to the chat draft"))
			}
		}
	}
	if tab := selectedTab(m); tab != nil {
		if isMarkdown(path) {
			if menu.Item(label("源码", "Source")).Checked(!tab.RenderedMarkdown).Chosen() {
				tab.RenderedMarkdown, tab.selectionText = false, ""
			}
			if menu.Item(label("渲染", "Rendered")).Checked(tab.RenderedMarkdown).Chosen() {
				tab.RenderedMarkdown, tab.selectionText, tab.revealLine = true, "", 0
			}
		}
		if isAsyncPreviewPath(path) && menu.Item(label("自动刷新此文件", "Auto-refresh this file")).Checked(!tab.AutoRefreshDisabled).Chosen() {
			tab.AutoRefreshDisabled = !tab.AutoRefreshDisabled
			if !tab.AutoRefreshDisabled && m.Tabs.resourceObserver != nil {
				o := m.Tabs.resourceObserver
				o.queue.request(o.owner, o.read)
			}
		}
		if tab.deliverySource == nil && m.OpenDocumentViewer != nil {
			menu.Submenu(label("选择预览器", "Choose viewer"), func(sub *ui.Menu) {
				previewViewerChoices(sub, m, tab)
			})
		}
	}
}

func previewViewerChoices(menu *ui.Menu, m Model, tab *FileTab) {
	if tab.deliverySource != nil {
		return
	}
	editing := m.Editing != nil && *m.Editing
	if menu.Item(label("原生预览", "Native preview")).Checked(tab.document.choice == 1).Disabled(editing).Chosen() {
		tab.document.choice = 1
	}
	if menu.Item(label("文档预览器", "Document viewer")).Checked(tab.document.choice == 2).Disabled(editing).Chosen() {
		if tab.document.surface == nil || !tab.document.surface.Alive() {
			surface, err := m.OpenDocumentViewer(tab.Path)
			if err != nil {
				setFileError(m, err)
				return
			}
			tab.document.surface = surface
			tab.document.attempted = true
		}
		tab.document.choice = 2
		tab.document.workbook = isTablePreviewPath(tab.Path) && workbookViewerAvailable(tab.document.surface)
	}
	if menu.Item(label("自动选择", "Auto viewer")).Checked(tab.document.choice == 0).Disabled(editing).Chosen() {
		tab.document.choice = 0
	}
	tab.document.enabled = tabDocumentSurfaceEnabled(tab)
}
