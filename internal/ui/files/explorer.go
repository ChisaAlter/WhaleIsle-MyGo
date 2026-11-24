package files

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// viewExplorer is the ordinary Files resource. It intentionally does not prepare
// preview tabs, resume an editor or mount a second set of resource chips.
func viewExplorer(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	root := strings.TrimSpace(m.WorkspaceRoot)
	if m.Tabs != nil && root != "" {
		if m.Tabs.Tree == nil {
			m.Tabs.Tree = &workspacefs.DirectoryTree{}
		}
		if m.Tabs.Search == nil {
			m.Tabs.Search = &workspacefs.Search{}
		}
		m.Tabs.Tree.EnsureHostOnly(m.BindingID, root, m.HostOnly)
		m.Tabs.Tree.Observe(m.WatchRemoteContext, !tabsFor(m).AutoRefreshDisabled)
	}
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(ui.Transparent).Label(label("文件面板", "Files panel")).Children(func() {
		if root == "" {
			ui.Text(c, i18n.T(i18n.KeyFilesNoWorkspace)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(12)
			return
		}
		// FilesPanel.module.css `.toolbar`: the search field grows beside the
		// 28px refresh control, inset 8/12/0. MyGo keeps its extra tree
		// actions on a visible options button beside it — a desktop product
		// difference, not a hidden gesture.
		ui.Row(c).FillWidth().Shrink(0).Padding(8, 12, 0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			if m.Filter != nil {
				ui.TextInput(c, m.Filter).Grow(1).MinWidth(0).Height(33).Radius(12).Border(0.5, chrome.Color(colors.Border)).FontSize(theme.InterfaceSize(12)).Label(i18n.T(i18n.KeyFilesFilter)).Placeholder(i18n.T(i18n.KeyFilesFilter)).Background(chrome.FrostInset(colors, m.SystemDark, c.Theme().Surface))
			} else {
				ui.Box(c).Grow(1).MinWidth(0)
			}
			if previewTool(c, "tree-refresh", label("刷新文件树", "Refresh files"), previewReloadIcon, muted).Disabled(m.Tabs == nil).Clicked() {
				m.Tabs.Tree.Refresh()
				m.Tabs.Search.Cancel()
				m.Tabs.explorerError = ""
			}
			previewTool(c, "tree-options", label("文件树选项", "Files options"), previewChevronIcon, muted).Menu(func(menu *ui.Menu) {
				if menu.Item(label("复制工作区路径", "Copy workspace path")).Chosen() {
					mygo.Clipboard.WriteText(root)
				}
				if m.OnReveal != nil && menu.Item(label("在文件管理器中显示", "Show in folder")).Chosen() {
					m.OnReveal(root)
				}
				if w := tabsFor(m); w != nil {
					if menu.Item(label("自动刷新文件树", "Auto-refresh files")).Checked(!w.AutoRefreshDisabled).Chosen() {
						w.AutoRefreshDisabled = !w.AutoRefreshDisabled
						m.Tabs.Tree.Observe(m.WatchRemoteContext, !w.AutoRefreshDisabled)
					}
				}
			})
		})
		filter := ""
		if m.Filter != nil {
			filter = strings.TrimSpace(*m.Filter)
		}
		if m.Tabs != nil && m.Tabs.explorerError != "" {
			ui.Text(c, m.Tabs.explorerError).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Danger)).Padding(4, 12)
		}
		if m.Tabs != nil {
			m.Tabs.Search.EnsureHostOnly(m.BindingID, root, filter, m.ListRemoteContext, c.Invalidate, m.HostOnly)
		}
		paintVirtualizedFileTree(c, m, root, filter, false, fg, muted, chrome.Color(colors.Accent))
	})
}

func openTreeFile(m Model, path string) {
	if !m.TreeOnly {
		selectFile(m, path)
		return
	}
	problem := label("文件资源入口未连接", "File resource opener is unavailable")
	if m.OnOpenResource != nil {
		if err := m.OnOpenResource(path); err != nil {
			problem = errcopy.Sentence(err)
		} else {
			problem = ""
		}
	}
	if m.Tabs != nil {
		m.Tabs.explorerError = problem
	}
}
