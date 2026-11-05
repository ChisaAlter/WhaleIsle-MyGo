package files

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/workspacefs"
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
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(chrome.Color(colors.Field)).Children(func() {
		if root == "" {
			ui.Text(c, i18n.T(i18n.KeyFilesNoWorkspace)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(12)
			return
		}
		ui.Row(c).FillWidth().MinWidth(0).Height(40).Shrink(0).Padding(6, 12).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, root).Key(m.BindingID + "\x00" + root).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(root).
				ContextMenu(func(menu *ui.Menu) {
					if menu.Item(label("复制工作区路径", "Copy workspace path")).Chosen() {
						mygo.Clipboard.WriteText(root)
					}
					if m.OnReveal != nil && menu.Item(label("在文件管理器中显示", "Show in folder")).Chosen() {
						m.OnReveal(root)
					}
				})
			if previewTool(c, "tree-refresh", label("刷新文件树", "Refresh files"), previewReloadIcon, muted).Disabled(m.Tabs == nil).Clicked() {
				m.Tabs.Tree.Refresh()
				m.Tabs.Search.Cancel()
				m.Tabs.explorerError = ""
			}
			previewTool(c, "tree-options", label("文件树选项", "Files options"), previewChevronIcon, muted).Menu(func(menu *ui.Menu) {
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
			ui.TextInput(c, m.Filter).FillWidth().Height(28).Margin(0, 12, 6, 12).Radius(6).Border(0.5, chrome.Color(colors.Border)).FontSize(theme.InterfaceSize(12)).Placeholder(i18n.T(i18n.KeyFilesFilter))
			filter = strings.TrimSpace(*m.Filter)
		}
		if m.Tabs != nil && m.Tabs.explorerError != "" {
			ui.Text(c, m.Tabs.explorerError).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Danger)).Padding(4, 12)
		}
		if m.Tabs != nil {
			m.Tabs.Search.EnsureHostOnly(m.BindingID, root, filter, m.ListRemoteContext, c.Invalidate, m.HostOnly)
		}
		ui.Scroll(c).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Padding(4).Gap(1).Children(func() {
				if filter != "" {
					paintFiltered(c, m, root, filter, fg, muted, chrome.Color(colors.Accent))
				} else {
					paintDir(c, m, root, "", 0, fg, muted, chrome.Color(colors.Accent))
				}
			})
		})
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
			problem = err.Error()
		} else {
			problem = ""
		}
	}
	if m.Tabs != nil {
		m.Tabs.explorerError = problem
	}
}
