package terminal

import (
	"fmt"

	"github.com/egoist/mygo"
	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// HostResourceMenu is consumed both by the DockKit resource title and by the
// terminal viewport. Store owner coordinates bind delayed menu choices.
func HostResourceMenu(menu *ui.Menu, h *HostState, store *hostterminal.Store, session string) bool {
	if h == nil || store == nil || session == "" {
		return false
	}
	v := store.Snapshot(session)
	selected := false
	if menu.Item(terminalLabel("新建 Host 终端", "New Host terminal")).Disabled(!v.Ready || v.Busy).Chosen() {
		store.Create(v.Owner, session)
		selected = true
	}
	menu.Submenu(terminalLabel("选择终端", "Choose terminal"), func(sub *ui.Menu) {
		if sub.Item(terminalLabel("刷新列表", "Refresh list")).Disabled(!v.ClientReady || v.Busy).Chosen() {
			store.Refresh(v.Owner, session)
			selected = true
		}
		sub.Separator()
		for _, item := range v.Items {
			caption := item.Title
			if caption == "" {
				caption = item.ID
			}
			if sub.Item(caption).Checked(item.ID == v.ActiveID).Disabled(!v.Ready || v.Busy).Chosen() {
				store.Connect(v.Owner, session, item.ID)
				selected = true
			}
		}
	})
	if menu.Item(terminalLabel("终端管理…", "Terminal management…")).Chosen() {
		h.manageOpen = true
		selected = true
	}
	return selected
}

// Terminal output owns the pane. Management is a bounded overlay, not a stack
// of permanent rows that competes with the terminal's viewport.
func hostSurface(c *ui.Context, h *HostState, store *hostterminal.Store, session string, colors theme.Colors, v hostterminal.View, actions HostActions) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	surface := ui.Column(c).Fill().MinWidth(0).MinHeight(0).Gap(0).Background(chrome.Color(colors.BG))
	surface.ContextMenu(func(menu *ui.Menu) { HostResourceMenu(menu, h, store, session) })
	surface.Children(func() {
		anchor := ui.Box(c).Key("host-management-anchor").Absolute().Right(8).Top(0).Size(1, 1)
		hostManagement(c, h, store, session, colors, v, actions, surface, anchor)
		if v.Phase != "connected" || !v.Writable || v.Info.State == "exited" {
			ui.Row(c).FillWidth().Height(28).Shrink(0).MinWidth(0).Gap(8).Padding(0, 8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, hostStatus(v)).FontSize(12).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").TextColor(muted)
				hostRecoveryAction(c, store, v, session, colors)
				ui.Button(c, "•••").Size(28, 28).Label(terminalLabel("终端选项", "Terminal options")).Menu(func(menu *ui.Menu) { HostResourceMenu(menu, h, store, session) })
			})
		}

		if v.Error != "" {
			ui.Text(c, v.Error).FillWidth().FontSize(12).MaxLines(2).Tooltip(v.Error).TextColor(chrome.Color(colors.Danger))
		}
		if h.renderError != "" && h.renderError != v.Error {
			ui.Text(c, h.renderError).FillWidth().FontSize(12).MaxLines(2).Tooltip(h.renderError).TextColor(chrome.Color(colors.Danger))
		}
		if h.terminal != nil {
			h.terminal.SetReadOnly(!v.Writable)
			myterm.View(c, h.terminal).FillWidth().Grow(1).MinWidth(0).MinHeight(0)
		} else {
			ui.Column(c).Label("Terminal").FillWidth().Grow(1).MinWidth(0).MinHeight(0).Padding(20, 12).Gap(8).Children(func() {
				message := terminalLabel("选择或新建一个终端", "Choose or create a terminal")
				if !v.ClientReady || session == "" {
					message = terminalLabel("请选择已连接 Host 的会话。", "Select a session on a connected Host.")
				} else if v.Busy || v.Phase == "connecting" || v.Phase == "retaining" {
					message = terminalLabel("正在连接…", "Connecting to terminal…")
				} else if v.Info.State == "exited" || v.Phase == "closed" {
					message = terminalLabel("终端已关闭。", "This terminal ended. Choose New to start another.")
				}
				ui.Text(c, message).FontSize(14).TextColor(fg)
				ui.Row(c).Wrap().Gap(6).Children(func() {
					if ui.PrimaryButton(c, terminalLabel("新建终端", "New terminal")).Disabled(!v.Ready || v.Busy || session == "").Clicked() {
						store.Create(v.Owner, session)
					}
					ui.Button(c, terminalLabel("选择终端", "Choose terminal")).Disabled(!v.ClientReady || session == "").Menu(func(menu *ui.Menu) { HostResourceMenu(menu, h, store, session) })
				})
				ui.Text(c, terminalLabel("终端在当前 Host 和会话中保留；隐藏面板不会关闭进程。", "Terminals stay with this Host and session. Hiding the pane leaves the process running.")).FontSize(12).TextColor(muted)
			})
		}
	})
}

// Recovery is part of the current terminal state, not a hidden management task.
// Reconnect uses the retained Host terminal; it never launches a local shell.
func hostRecoveryAction(c *ui.Context, store *hostterminal.Store, v hostterminal.View, session string, colors theme.Colors) {
	if !v.ClientReady || session == "" || v.Busy || v.Info.State == "exited" || v.Phase == "closing" || v.Phase == "closed" {
		return
	}
	caption := ""
	if v.Phase == "connected" && !v.Writable {
		caption = terminalLabel("接管输入", "Take control")
	} else if v.Phase == "failed" || v.Phase == "disconnected" {
		caption = terminalLabel("重新连接", "Reconnect")
	}
	if caption == "" {
		return
	}
	if widgets.Button(c, colors, caption, widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(!v.Ready).Clicked() {
		if v.ActiveID == "" {
			store.Refresh(v.Owner, session)
		} else {
			store.Connect(v.Owner, session, v.ActiveID)
		}
	}
}

func hostStatus(v hostterminal.View) string {
	if v.Info.State == "exited" {
		text := terminalLabel("进程已退出", "Process exited")
		if v.Info.ExitCode != nil {
			text = fmt.Sprintf(terminalLabel("进程已退出（%d）", "Process exited (%d)"), *v.Info.ExitCode)
		}
		return text
	}
	if !v.ClientReady {
		return terminalLabel("Host 未连接", "Host disconnected")
	}
	status := map[string]string{
		"connected": terminalLabel("已连接", "Connected"), "connecting": terminalLabel("正在连接", "Connecting"),
		"retaining": terminalLabel("正在保留", "Retaining"), "retained": terminalLabel("已保留", "Retained"),
		"failed": terminalLabel("连接失败", "Connection failed"), "disconnected": terminalLabel("连接中断", "Disconnected"),
		"closing": terminalLabel("正在关闭", "Closing"), "closed": terminalLabel("已关闭", "Closed"),
	}[v.Phase]
	if status == "" {
		return terminalLabel("尚未连接终端", "No terminal connected")
	}
	if v.Writable {
		return status + " · " + terminalLabel("可输入", "Writable")
	}
	return status + " · " + terminalLabel("只读", "Read-only")
}

func hostManagement(c *ui.Context, h *HostState, store *hostterminal.Store, session string, colors theme.Colors, v hostterminal.View, actions HostActions, surface, anchor ui.Element) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	width, height := c.Size()
	if b := surface.Bounds(); b.W > 0 {
		width = min(width, b.W)
		height = min(height, b.H)
	}
	maxHeight := max(float32(0), min(float32(360), height-64))
	ui.PopoverBase(c.Key("host-terminal-management"), anchor, &h.manageOpen, func(panel ui.Element) {
		panel = chrome.MenuSurface(panel.Width(max(float32(0), min(float32(360), width-16))).MaxHeight(maxHeight).Padding(12), chrome.Color(colors.Field), chrome.Color(colors.BorderL3), 10)
		panel.Children(func() {
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(float32(0), maxHeight-24)).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
					ui.Text(c, terminalLabel("终端管理", "Terminal management")).FontSize(13).Bold().TextColor(fg)
					if widgets.Button(c, colors, terminalLabel("刷新列表", "Refresh list"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(!v.ClientReady || v.Busy).Clicked() {
						store.Refresh(v.Owner, session)
					}
					if v.ActiveID != "" {
						ui.Row(c).FillWidth().Wrap().Gap(4).Children(func() {
							if widgets.Button(c, colors, terminalLabel("重连并接管输入", "Reconnect and take control"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(!v.Ready || v.Busy || v.Phase == "closing" || v.Phase == "closed").Clicked() {
								store.Connect(v.Owner, session, v.ActiveID)
							}
							if widgets.Button(c, colors, terminalLabel("关闭终端", "Close terminal"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Disabled(!v.Ready || v.Busy || v.Phase == "closing" || v.Phase == "closed").Clicked() {
								h.confirmClose = true
							}
						})
					}
					if h.confirmClose {
						ui.Text(c, terminalLabel("将终止此 Host 会话中选中的终端进程。", "This terminates the selected process in this Host session.")).FontSize(12).TextColor(fg)
						ui.Row(c).Wrap().Gap(4).Children(func() {
							if widgets.Button(c, colors, terminalLabel("确认关闭", "Confirm close"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Clicked() {
								h.confirmClose = false
								store.Close(v.Owner, session, v.ActiveID)
							}
							if widgets.Button(c, colors, terminalLabel("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
								h.confirmClose = false
							}
						})
					}
					if v.ActiveID != "" {
						hostRenameView(c, store, v, session, muted)
						ui.Text(c, v.Info.Cwd).FillWidth().FontSize(12).Selectable().TextColor(muted)
						ui.Text(c, v.ActiveID).FillWidth().FontSize(12).Selectable().TextColor(muted)
					}
					hostShellView(c, store, v, session, muted)
					ui.Text(c, terminalLabel("新建会以 Host 系统用户权限启动 shell。", "New starts a shell with the Host system user's permissions.")).FontSize(12).TextColor(muted)
					if h.terminal != nil {
						selection, selected := h.terminal.SelectionText()
						if widgets.Button(c, colors, terminalLabel("复制选区", "Copy selection"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Disabled(!selected || selection == "").Clicked() {
							if store.CanCopySelection(v.Owner, session, v.ActiveID, h.attachment) {
								if text, ok := h.terminal.SelectionText(); ok && text != "" {
									mygo.Clipboard.WriteText(text)
								}
							}
						}
						h.selectionActions(c, store, v, session, actions)
					}
				})
			})
		})
	})
}
