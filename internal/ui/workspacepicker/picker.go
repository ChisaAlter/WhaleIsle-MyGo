// Package workspacepicker presents Host-backed workspace selection and explicit
// registration management. It never reads or removes local filesystem paths.
package workspacepicker

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type Model struct {
	Store         *session.Store
	ThemeID       string
	SystemDark    bool
	OnChoose      func(uint64, string)
	OnRegister    func(uint64, string)
	OnNoDirectory func()
}

func text(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	snap := m.Store.Snapshot()
	request := snap.WorkspaceNavigation
	if request == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(520).MaxWidthPercent(94).Padding(20).Background(chrome.Color(colors.BG)).Radius(14).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				switch request.Mode {
				case "pick":
					ui.Text(c, text("选择工作区", "Choose workspace")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
					if !snap.WorkspaceReady {
						ui.Text(c, text("正在等待 Host 工作区列表…", "Waiting for the Host workspace list…")).TextColor(muted)
					}
					ui.Scroll(c).Height(180).MaxHeight(260).Gap(4).Children(func() {
						for _, row := range snap.Workspaces {
							ws := row
							title := ws.Title
							if title == "" {
								title = ws.Path
							}
							if ui.Button(c, title).FillWidth().Disabled(request.Busy || !snap.WorkspaceReady).Tooltip(ws.Path).Clicked() && m.OnChoose != nil {
								m.OnChoose(request.Token, ws.WorkspaceID)
							}
						}
					})
					draft := request.Draft
					ui.TextInput(c.Key("workspace-path"), &draft).FillWidth().Placeholder(text("Host 上的目录路径", "Directory path on this Host")).Disabled(request.Busy).OnChange(func() { m.Store.SetWorkspaceNavigationDraft(request.Token, draft) })
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, text("浏览 Host 目录", "Browse Host folders")).Disabled(request.Busy || request.ListBusy).Clicked() {
							m.Store.BrowseWorkspaceDirectories(context.Background(), request.Token, "")
						}
						if ui.Button(c, text("登记并切换", "Register and switch")).Disabled(request.Busy || !snap.WorkspaceReady).Clicked() && m.OnRegister != nil {
							m.OnRegister(request.Token, draft)
						}
						if m.OnNoDirectory != nil && ui.Button(c, text("无目录会话", "No workspace folder")).Disabled(request.Busy).Clicked() {
							m.Store.CancelWorkspaceNavigation(request.Token)
							m.OnNoDirectory()
						}
					})
					if request.ListBusy {
						ui.Text(c, text("正在读取目录…", "Loading folders…")).TextColor(muted)
					}
					if request.Listing != nil && !request.Busy {
						dirpick.View(c, dirpick.Model{Listing: *request.Listing, ThemeID: m.ThemeID, SystemDark: m.SystemDark, OnNavigate: func(path string) { m.Store.BrowseWorkspaceDirectories(context.Background(), request.Token, path) }, OnPick: func(path string) { m.Store.SetWorkspaceNavigationDraft(request.Token, path) }})
					}
				case "manage":
					ui.Text(c, request.TargetTitle).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
					if ui.Button(c, text("重命名工作区", "Rename workspace")).Clicked() {
						m.Store.BeginWorkspaceRename(request.TargetID)
					}
					if ui.Button(c, text("移除工作区注册…", "Remove workspace registration…")).Clicked() {
						m.Store.BeginWorkspaceRemoval(request.TargetID)
					}
				case "rename":
					ui.Text(c, text("重命名工作区", "Rename workspace")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
					ui.Text(c, request.TargetTitle).TextColor(muted)
					draft := request.Draft
					ui.TextInput(c.Key("workspace-name"), &draft).FillWidth().Disabled(request.Busy).OnChange(func() { m.Store.SetWorkspaceNavigationDraft(request.Token, draft) })
					if ui.Button(c, text("保存名称", "Save name")).Disabled(request.Busy).Clicked() {
						m.Store.ConfirmWorkspaceRename(context.Background(), request.Token, draft)
					}
				case "remove":
					ui.Text(c, text("移除工作区注册？", "Remove workspace registration?")).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
					ui.Text(c, request.TargetTitle).TextColor(fg)
					ui.Text(c, text("仅从工作区列表中移除注册。磁盘文件不会删除。", "This removes the workspace registration from the list. Files on disk are not deleted.")).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					if ui.Button(c, text("确认移除注册", "Remove registration")).Disabled(request.Busy).Clicked() {
						m.Store.ConfirmWorkspaceRemoval(context.Background(), request.Token)
					}
				}
				if request.Error != "" {
					ui.Text(c, request.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				}
				if request.Busy {
					ui.Text(c, text("等待 Host 确认…", "Waiting for the Host…")).TextColor(muted)
				}
				if ui.Button(c, text("取消", "Cancel")).Disabled(request.Busy).Clicked() {
					m.Store.CancelWorkspaceNavigation(request.Token)
				}
			})
		})
	})
	if !open && !request.Busy {
		m.Store.CancelWorkspaceNavigation(request.Token)
	}
}
