package files

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
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

func previewEditable(m Model, path string) bool {
	kind := workspacefs.ClassifyDoc(path)
	return m.EditDraft != nil && m.Editing != nil && !textReadBlocked(m) && !previewTruncated(m) &&
		!imageMode(m, path) && !mediaMode(m, path) && (kind == workspacefs.DocNone || kind == workspacefs.DocHTML) &&
		(m.PreviewErr == nil || *m.PreviewErr == "") && (selectedTab(m) == nil || !selectedTab(m).readLoading)
}

func previewDirty(m Model) bool {
	return m.Preview != nil && m.EditDraft != nil && *m.Preview != *m.EditDraft
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
	b := ui.Button(c.Key(key), "").Label(caption).Tooltip(caption).Width(28).Height(28).Shrink(0).Padding(0).
		Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(ink).AlignItems(ui.Center).Justify(ui.Center)
	if b.Hovered() {
		b.Background(c.Theme().SurfaceHover)
	}
	glyph := float32(15)
	if icon == previewFloatingIcon {
		glyph = 14
	}
	b.Children(func() { ui.Icon(c, icon).Size(glyph, glyph).PassThrough() })
	return b
}

func previewTextTool(c *ui.Context, caption string) ui.Element {
	b := ui.Button(c, caption).Height(28).Shrink(0).FontSize(12).Padding(0, 10, 0, 10).
		BorderWidth(0).Background(ui.Transparent)
	if b.Hovered() {
		b.Background(c.Theme().SurfaceHover)
	}
	return b
}

func togglePreviewWrap(m Model) {
	prefs := previewPreferences(m)
	prefs.WordWrap = !prefs.WordWrap
	if m.Tabs != nil && m.Tabs.floating != nil && m.Tabs.floating.win != nil {
		m.Tabs.floating.win.Invalidate()
	}
}

func activateFloatingPreview(m Model, path string) {
	kind := workspacefs.ClassifyDoc(path)
	if mediaMode(m, path) && m.OnOpenMedia != nil {
		m.OnOpenMedia(path, true)
	} else if (isPreviewImage(path) || kind == workspacefs.DocPDF || kind == workspacefs.DocHTML) && m.OnOpenFloatingDoc != nil {
		m.OnOpenFloatingDoc(path)
	} else if err := openFloatingPreview(m); err != nil {
		setFileStatus(m, err.Error())
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
		row.Padding(8, 16, 0, 16)
	} else {
		row.Padding(0, 6, 0, 16).DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 0.5, W: r.W, H: 0.5}, border, 0)
		})
	}
	jump := ui.Local(row, "line-navigation", func() bool { return false })
	row.ContextMenu(func(menu *ui.Menu) {
		treeFileMenu(m, path)(menu)
		if menu.Item(label("跳转到行…", "Go to line…")).Chosen() {
			*jump = true
		}
	})
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
				if previewTextTool(c, label("自动换行", "Word wrap")).Tooltip(label("切换自动换行", "Toggle word wrap")).Clicked() {
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
		if m.Tabs != nil {
			if previewTool(c, "preview-floating", label("独立窗口预览", "Preview in separate window"), previewFloatingIcon, muted).Clicked() {
				activateFloatingPreview(m, path)
			}
		}
		if desktop {
			paintPreviewSelectionActions(c, m, path)
			if previewEditable(m, path) || previewDirty(m) {
				if previewTextTool(c, i18n.T(i18n.KeyFilesSave)).Background(c.Theme().Accent).TextColor(c.Theme().AccentText).
					Disabled(m.OnSaveEdit == nil || !previewDirty(m) || previewTruncated(m) || textReadBlocked(m)).Clicked() {
					if err := requestSaveDraft(m); err != nil {
						setFileStatus(m, err.Error())
					}
				}
			}
		} else {
			if previewTool(c, "preview-reload", label("重新加载", "Reload"), previewReloadIcon, muted).Clicked() {
				reloadSelectedFile(m)
			}
			paintPreviewOpenTarget(c, m, path, muted)
		}
	})
	if *jump {
		paintPreviewLineNavigation(c, m, jump)
	}
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
			ui.Text(c, part).FontSize(12).TextColor(ink).SingleLine().MaxWidth(120).Shrink(0).Tooltip(part)
		}
	})
}

func paintPreviewOpenTarget(c *ui.Context, m Model, path string, muted ui.Color) {
	if m.OnOpenDoc == nil && m.OnReveal == nil && m.OnOpenWith == nil {
		return
	}
	ui.Row(c).Gap(0).Shrink(0).Children(func() {
		caption := i18n.T(i18n.KeyFilesOpenExternal)
		if m.OnOpenDoc == nil {
			caption = i18n.T(i18n.KeyFilesReveal)
		}
		if previewTool(c, "preview-open", caption, previewOpenIcon, muted).Width(24).Height(24).Clicked() {
			if m.OnOpenDoc != nil {
				withToolbarPath(m, path, textPreviewPath(m, path), func(full string) { m.OnOpenDoc(full, workspacefs.ClassifyDoc(path), false) })
			} else {
				revealToolbarPath(m, path, textPreviewPath(m, path))
			}
		}
		if m.OnReveal != nil || (m.OnOpenWith != nil && len(m.OpenWithApps) > 0) {
			previewTool(c, "preview-open-menu", label("更多打开方式", "More open options"), previewChevronIcon, muted).Width(18).Height(24).
				Menu(func(menu *ui.Menu) { previewOpenMenu(menu, m, path) })
		}
	})
}

func previewOpenMenu(menu *ui.Menu, m Model, path string) {
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
				setFileStatus(m, err.Error())
			} else {
				setFileStatus(m, label("所选行已添加到对话草稿", "Selected lines added to the chat draft"))
			}
		}
	}
}

func paintPreviewLineNavigation(c *ui.Context, m Model, open *bool) {
	t := selectedTab(m)
	if t == nil {
		*open = false
		return
	}
	ui.Row(c).Height(32).Gap(6).Padding(2, 16, 2, 16).Children(func() {
		input := ui.TextInput(c.Key("preview-line"), &t.lineDraft).Label("Line number").Width(80).Placeholder(label("行号", "Line"))
		focus := ui.Local(input, "focus-once", func() bool { return false })
		if !*focus {
			input.Focus()
			*focus = true
		}
		if input.Shortcut(0, ui.KeyEscape) || ui.Button(c, label("取消", "Cancel")).Clicked() {
			*open = false
			*focus = false
			return
		}
		if input.Submitted() && !input.Composing() || ui.Button(c, label("跳转行", "Go to line")).Clicked() {
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
	if !desktopFileToolbar(m, path) && editing && m.OnSaveEdit != nil && m.EditDraft != nil && menu.Item(i18n.T(i18n.KeyFilesSave)).Disabled(previewTruncated(m) || textReadBlocked(m)).Chosen() {
		if err := requestSaveDraft(m); err != nil {
			setFileStatus(m, err.Error())
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
				setFileStatus(m, err.Error())
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
		if m.OpenDocumentViewer != nil {
			menu.Submenu(label("选择预览器", "Choose viewer"), func(sub *ui.Menu) {
				previewViewerChoices(sub, m, tab)
			})
		}
	}
}

func previewViewerChoices(menu *ui.Menu, m Model, tab *FileTab) {
	editing := m.Editing != nil && *m.Editing
	if menu.Item(label("原生预览", "Native preview")).Checked(tab.document.choice == 1).Disabled(editing).Chosen() {
		tab.document.choice = 1
	}
	if menu.Item(label("文档预览器", "Document viewer")).Checked(tab.document.choice == 2).Disabled(editing).Chosen() {
		if tab.document.surface == nil || !tab.document.surface.Alive() {
			surface, err := m.OpenDocumentViewer(tab.Path)
			if err != nil {
				setFileStatus(m, err.Error())
				return
			}
			tab.document.surface = surface
		}
		tab.document.choice = 2
		tab.document.workbook = isTablePreviewPath(tab.Path) && workbookViewerAvailable(tab.document.surface)
	}
	if menu.Item(label("自动选择", "Auto viewer")).Checked(tab.document.choice == 0).Disabled(editing).Chosen() {
		tab.document.choice = 0
	}
	tab.document.enabled = documentSurfaceEnabled(tab.document)
}
