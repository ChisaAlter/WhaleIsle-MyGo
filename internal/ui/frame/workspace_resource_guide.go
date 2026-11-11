package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Installed SidebarRight GuideBody: an actual page in the current pane, with
// two columns of resource doors. Picking a door replaces this guide occurrence.
func workspaceResourceGuide(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	entries := workspaceResourceEntries(m)
	if m.OnOpenMarket != nil || m.OnCreatePluginDraft != nil {
		entries = append(entries, workspaceResourceEntry{
			id:       "more",
			title:    i18n.T(i18n.KeySidebarRightTabGuideMore),
			icon:     workspaceGuideMoreIcon,
			disabled: m.ConnectionError != "",
		})
	}
	width := float32(320)
	if m.FilePaneWidth != nil && *m.FilePaneWidth > 0 {
		width = min(width, max(float32(0), *m.FilePaneWidth-32))
	}
	ui.Scroll(c.Key("workspace-resource-guide")).Label(sidebarRightText("tab.guide.heading")).Fill().MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().MinHeightPercent(100).Padding(16).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Margin(0, 0, 16, 0).AlignItems(ui.Center).Children(func() {
				ui.Text(c, sidebarRightText("tab.guide.heading")).FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg)
				ui.Text(c, paneLabel("选择要在右侧栏显示的内容。", "Choose what to show in the right sidebar.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			})
			ui.Column(c).Width(width).MaxWidthPercent(100).Gap(8).Children(func() {
				for i := 0; i < len(entries); i += 2 {
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						for j := i; j < min(i+2, len(entries)); j++ {
							entry := entries[j]
							button := ui.Button(c.Key("resource-guide:"+entry.id), "").Label(entry.title).Tooltip(entry.description).Width((width - 8) / 2).Height((width - 8) / 2).MinWidth(0).Padding(12).Radius(8).BorderWidth(0).Background(chrome.Color(colors.Field)).Disabled(entry.disabled)
							if !entry.disabled {
								button.Cursor(ui.CursorPointer)
							}
							if entry.id == "more" {
								button.Menu(func(menu *ui.Menu) {
									if m.OnOpenMarket != nil && menu.Item(paneLabel("浏览更多插件", "Browse other plugins")).Disabled(m.ConnectionError != "").Chosen() {
										m.OnOpenMarket()
									}
									if m.OnCreatePluginDraft != nil && menu.Item(paneLabel("创建你的插件", "Create your plugin")).Disabled(m.ConnectionError != "").Chosen() {
										m.OnCreatePluginDraft("rightbar")
									}
								})
							}
							button.Children(func() {
								ui.Column(c).Fill().MinWidth(0).AlignItems(ui.Center).Justify(ui.Center).Gap(4).Children(func() {
									ui.Icon(c, entry.icon).Size(20, 20).TextColor(muted)
									ui.Text(c, entry.title).FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(fg).SingleLine().Ellipsis("…").MaxWidthPercent(100)
									if entry.description != "" {
										ui.Text(c, entry.description).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(muted).TextAlign(ui.Center).MaxLines(2).MaxWidthPercent(100)
									}
								})
							})
							if entry.id != "more" && button.Clicked() && entry.open != nil {
								entry.open()
							}
						}
					})
				}
			})
		})
	})
}
