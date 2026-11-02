// Package workspacepicker presents Host-backed workspace selection and explicit
// registration management. It never reads or removes local filesystem paths.
package workspacepicker

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type Model struct {
	Store         *session.Store
	ThemeID       string
	SystemDark    bool
	Anchor        ui.Element
	OnBrowse      func()
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
	if m.Store == nil || !m.Store.WorkspaceNavigationPending() {
		return
	}
	snap := m.Store.Snapshot()
	request := snap.WorkspaceNavigation
	if request == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if request.Mode == "pick" {
		if m.Anchor.Valid() {
			pickMenu(c, m, snap, request, colors)
		}
		return
	}
	managementDialog(c, m, snap, request, colors)
}

// The conversation chip owns this anchor. Directory picking is a separate flow
// opened by the pinned add action, just as in WorkspacePickFlow.
func pickMenu(c *ui.Context, m Model, snap session.ViewState, request *session.WorkspaceNavigationState, colors theme.Colors) {
	open := true
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	_, height := c.Size()
	ui.PopoverBase(c, m.Anchor, &open, func(panel ui.Element) {
		panel = chrome.PopoverReveal(c, panel.Width(280).MaxWidth(360).MaxWidthPercent(94).Padding(4).Gap(0).Radius(12).
			Background(chrome.Color(colors.Field)).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, .18)))
		panel.Children(func() {
			count := len(snap.Workspaces)
			if m.OnNoDirectory != nil {
				count++
			}
			if m.OnBrowse != nil {
				count++
			}
			focused := ui.Local(panel, "menu-focus", func() int { return 0 })
			focusNext := false
			if panel.OverlayShortcut(0, ui.KeyDown) && count > 0 {
				*focused = (*focused + 1) % count
				focusNext = true
			}
			if panel.OverlayShortcut(0, ui.KeyUp) && count > 0 {
				*focused = (*focused + count - 1) % count
				focusNext = true
			}
			first := ui.Local(panel, "menu-first-focus", func() bool { return true })
			index := 0
			row := func(key, label, path string, selected bool, icon string, disabled bool, choose func()) {
				button := menuRow(c.Key(key), label, path, selected, icon, disabled, colors)
				if (*first || focusNext) && index == *focused {
					button.Focus().ScrollIntoView()
				}
				if !focusNext && !*first && button.Focused() {
					*focused = index
				}
				index++
				if button.Clicked() && choose != nil {
					choose()
				}
			}
			if len(snap.Workspaces) > 0 {
				ui.Scroll(c).FillWidth().MinWidth(0).Height(min(float32(len(snap.Workspaces))*34, max(34, height-180))).Children(func() {
					for _, workspace := range snap.Workspaces {
						ws := workspace
						title := ws.Title
						if title == "" {
							title = ws.Path
						}
						row(ws.WorkspaceID, title, ws.Path, ws.WorkspaceID == m.Store.WorkspaceID(), "folder", request.Busy || !snap.WorkspaceReady, func() {
							if m.OnChoose != nil {
								m.OnChoose(request.Token, ws.WorkspaceID)
							}
						})
					}
				})
				ui.Box(c).FillWidth().Height(.5).Margin(3, 2).Background(chrome.Color(colors.Border))
			}
			if !snap.WorkspaceReady {
				ui.Text(c, text("正在加载工作区…", "Loading workspaces…")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(6, 8)
			}
			if m.OnNoDirectory != nil {
				row("no-directory", text("无工作区文件夹", "No workspace folder"), "", m.Store.WorkspaceID() == "" && snap.ScratchCwd != "" && snap.SelectedCwd == snap.ScratchCwd, "chat", request.Busy, func() {
					m.Store.CancelWorkspaceNavigation(request.Token)
					m.OnNoDirectory()
				})
			}
			if m.OnBrowse != nil {
				row("add-workspace", text("添加工作区…", "Add workspace…"), "", false, "add", request.Busy, m.OnBrowse)
			}
			if request.Error != "" {
				ui.Text(c, request.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg).Padding(6, 8)
			}
			*first = false
		})
	})
	if !open && !request.Busy {
		m.Store.CancelWorkspaceNavigation(request.Token)
		m.Anchor.Focus()
	}
}

func menuRow(c *ui.Context, label, path string, selected bool, icon string, disabled bool, colors theme.Colors) ui.Element {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	button := ui.Button(c, "").Label(label).Tooltip(path).FillWidth().MinWidth(0).Height(34).Padding(6, 8).Radius(8).BorderWidth(0).Background(ui.Transparent).Disabled(disabled)
	if button.Pressed() {
		button.Background(chrome.Color(colors.Pressed))
	} else if button.Hovered() || button.FocusWithin() {
		button.Background(chrome.Color(colors.Hover))
	}
	button.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(16).Height(16).Draw(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, icon, muted) })
			ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
			if selected {
				ui.Box(c).Width(14).Height(14).Draw(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, "check", fg) })
			}
		})
	})
	return button
}

func drawIcon(p *ui.Painter, r ui.Rect, kind string, ink ui.Color) {
	path := &ui.Path{}
	switch kind {
	case "add":
		path.MoveTo(r.X+8, r.Y+3)
		path.LineTo(r.X+8, r.Y+13)
		path.MoveTo(r.X+3, r.Y+8)
		path.LineTo(r.X+13, r.Y+8)
	case "check":
		path.MoveTo(r.X+2, r.Y+7)
		path.LineTo(r.X+5, r.Y+10)
		path.LineTo(r.X+12, r.Y+3)
	case "chat":
		path.MoveTo(r.X+2, r.Y+3)
		path.LineTo(r.X+14, r.Y+3)
		path.LineTo(r.X+14, r.Y+11)
		path.LineTo(r.X+6, r.Y+11)
		path.LineTo(r.X+3, r.Y+14)
		path.LineTo(r.X+3, r.Y+11)
		path.LineTo(r.X+2, r.Y+11)
		path.Close()
	default:
		path.MoveTo(r.X+2, r.Y+4)
		path.LineTo(r.X+6, r.Y+4)
		path.LineTo(r.X+8, r.Y+6)
		path.LineTo(r.X+14, r.Y+6)
		path.LineTo(r.X+14, r.Y+13)
		path.LineTo(r.X+2, r.Y+13)
		path.Close()
	}
	p.StrokePath(path, 1.2, ink)
}

func managementDialog(c *ui.Context, m Model, snap session.ViewState, request *session.WorkspaceNavigationState, colors theme.Colors) {
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	title := text("工作区", "Workspace")
	switch request.Mode {
	case "rename":
		title = text("重命名工作区", "Rename workspace")
	case "remove":
		title = text("移除工作区注册？", "Remove workspace registration?")
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		panel.Width(380).MaxWidthPercent(94).Padding(0, 0, 24, 0).Gap(20).Radius(28).
			Background(chrome.Color(colors.Field)).Draw(chrome.Glass(chrome.Color(colors.Field), chrome.MaterialPanel, 28)).
			Shadow(0, 12, 32, 0, ui.RGBA(0, 0, 0, .2)).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Padding(22, 14, 0, 24).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, title).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0)
				close := ui.Button(c, "").Label(text("关闭", "Close")).Width(28).Height(28).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(request.Busy)
				close.Draw(func(p *ui.Painter, r ui.Rect) {
					path := &ui.Path{}
					path.MoveTo(r.X+9, r.Y+9)
					path.LineTo(r.X+19, r.Y+19)
					path.MoveTo(r.X+19, r.Y+9)
					path.LineTo(r.X+9, r.Y+19)
					p.StrokePath(path, 1.2, muted)
				})
				if close.Clicked() {
					m.Store.CancelWorkspaceNavigation(request.Token)
				}
			})
			submitted := false
			blocked := request.Busy
			ui.Column(c).FillWidth().MinWidth(0).Padding(0, 24).Gap(12).Children(func() {
				if request.Mode == "rename" {
					draft := request.Draft
					input := ui.TextInput(c.Key(request.Token), &draft).FillWidth().MinWidth(0).Height(44).Padding(7, 14).Radius(12).FontSize(theme.InterfaceSize(14)).Label(text("工作区名称", "Workspace name")).Disabled(request.Busy)
					first := ui.Local(input, "rename-focus", func() bool { return true })
					if *first {
						input.Focus().SetTextSelection(0, utf8.RuneCountInString(draft))
						*first = false
					}
					if input.Changed() {
						m.Store.SetWorkspaceNavigationDraft(request.Token, draft)
					}
					trimmed := strings.TrimSpace(draft)
					blocked = blocked || trimmed == ""
					for _, ws := range snap.Workspaces {
						if ws.Title == trimmed {
							blocked = true
						}
					}
					submitted = input.Submitted()
				} else if request.Mode == "remove" {
					ui.Text(c, request.TargetTitle).FontSize(theme.InterfaceSize(14)).TextColor(fg)
					ui.Text(c, text("仅从工作区列表移除注册，磁盘文件将保留。", "This removes the workspace registration from the list. Files on disk are retained.")).FontSize(theme.InterfaceSize(14)).TextColor(fg)
				} else {
					ui.Text(c, request.TargetTitle).FontSize(theme.InterfaceSize(14)).TextColor(fg)
					if ui.Button(c, text("重命名工作区", "Rename workspace")).FillWidth().Clicked() {
						m.Store.BeginWorkspaceRename(request.TargetID)
					}
					if ui.Button(c, text("移除工作区注册…", "Remove workspace registration…")).FillWidth().Clicked() {
						m.Store.BeginWorkspaceRemoval(request.TargetID)
					}
				}
				if request.Error != "" {
					ui.Text(c, request.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				}
				if request.Busy {
					ui.Text(c, text("等待 Host 确认…", "Waiting for the Host…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
			})
			ui.Row(c).FillWidth().Padding(0, 24).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Grow(1)
				if ui.Button(c, text("取消", "Cancel")).MinWidth(72).Disabled(request.Busy).Clicked() {
					m.Store.CancelWorkspaceNavigation(request.Token)
				}
				if request.Mode == "rename" {
					if ui.PrimaryButton(c, text("重命名", "Rename")).MinWidth(72).Disabled(blocked).Clicked() || submitted && !blocked {
						m.Store.ConfirmWorkspaceRename(context.Background(), request.Token, request.Draft)
					}
				} else if request.Mode == "remove" {
					if ui.Button(c, text("移除注册", "Remove registration")).MinWidth(72).Disabled(request.Busy).Clicked() {
						m.Store.ConfirmWorkspaceRemoval(context.Background(), request.Token)
					}
				}
			})
		})
	})
	if !open && !request.Busy {
		m.Store.CancelWorkspaceNavigation(request.Token)
	}
}
