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
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type Model struct {
	DirectoryActions []DirectoryAction
	Store            *session.Store
	ThemeID          string
	SystemDark       bool
	Anchor           ui.Element
	OnBrowse         func()
	OnChoose         func(uint64, string)
	OnRegister       func(uint64, string)
	OnNoDirectory    func()
}

// DirectoryAction is an explicitly installed native directory capability.
// It receives the Host path, without disguising its owner as a workspace.
type DirectoryAction struct {
	ID, Label string
	Order     int
	Run       func(path string)
	Pick      func()
	// Optional projections use the contributor's cached state. Nil preserves
	// the existing static label and visibility in both picker and row menus.
	Visible      func(path string) bool
	LabelForPath func(path string) string
	Picker       *bool
}

func (a DirectoryAction) Presentation(path string, picker bool) (string, bool) {
	if a.ID == "" {
		return "", false
	}
	if picker {
		if a.Picker != nil && !*a.Picker {
			return "", false
		}
		if a.Pick != nil {
			return a.Label, true
		}
	}
	if a.Run == nil || a.Visible != nil && !a.Visible(path) {
		return "", false
	}
	label := a.Label
	if a.LabelForPath != nil {
		label = a.LabelForPath(path)
	}
	return label, true
}

func text(zh, en string) string {
	return i18n.Pick(zh, en)
}

// registrationFailureText names the cause and the next step. The store carries
// the Host's own wording, which is raw and English; that copy stays in the log.
func registrationFailureText(raw string) string {
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "workspace list is ready"), strings.Contains(lower, "no dsh client"), strings.Contains(lower, "connect a host"):
		return text("Host 尚未就绪，请先启动 Host，再选择工作区。", "The Host is not ready yet. Start it, then choose the workspace.")
	case strings.Contains(lower, "incomplete workspace"):
		return text("Host 没有返回完整的工作区信息，请重试。", "The Host did not return a complete workspace. Try again.")
	case strings.Contains(lower, "no longer registered"):
		return text("该工作区已不在列表中，请重新选择。", "That workspace is no longer listed. Choose another one.")
	case strings.Contains(lower, "config"), strings.Contains(lower, "persist"), strings.Contains(lower, "save"):
		return text("无法保存工作区设置，请检查配置目录是否可写后重试。", "The workspace setting could not be saved. Check that the config folder is writable and try again.")
	}
	return text("添加工作区没有完成，请重试。", "Adding the workspace did not finish. Try again.")
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
		} else {
			registrationDialog(c, m, request, colors)
		}
		return
	}
	managementDialog(c, m, snap, request, colors)
}

// Sidebar directory adoption must remain visible without the Hero chip anchor.
func registrationDialog(c *ui.Context, m Model, request *session.WorkspaceNavigationState, colors theme.Colors) {
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		chrome.FrostSurface(panel.Key("workspace-registration").Width(440).MaxWidthPercent(94).Padding(24).Gap(16), colors, m.SystemDark, chrome.Color(colors.Field), 24).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWSWorkspaceAdd)).FontSize(theme.InterfaceSize(17)).FontWeight(500)
			if request.Error != "" {
				ui.Text(c, i18n.T(i18n.KeyWSFolderErrorTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(chrome.Color(colors.Muted))
			}
			ui.Text(c, request.Draft).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(14))
			ui.Text(c, registrationFailureText(request.Error)).FillWidth().MinHeight(40).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG))
			ui.Row(c).FillWidth().Gap(design.Sp(2)).Children(func() {
				if coss.Button(c, text("取消", "Cancel"), coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonSM,
					Disabled: request.Busy,
					Key:      "ws-reg-cancel",
				}).Clicked() {
					m.Store.CancelWorkspaceNavigation(request.Token)
				}
				ui.Box(c).Grow(1)
				if m.OnBrowse != nil {
					if coss.Button(c, text("重新选择…", "Choose again…"), coss.ButtonOptions{
						Variant:  coss.ButtonOutline,
						Size:     coss.ButtonSM,
						Disabled: request.Busy,
						Key:      "ws-reg-browse",
					}).Tooltip(i18n.T(i18n.KeyWSFolderErrorRetry)).Clicked() {
						m.OnBrowse()
					}
				}
				label := text("重试", "Retry")
				if request.Busy {
					label = text("正在添加…", "Adding…")
				}
				if m.OnRegister != nil {
					regDisabled := request.Busy || strings.TrimSpace(request.Draft) == ""
					if coss.Button(c, label, coss.ButtonOptions{
						Variant:  coss.ButtonDefault,
						Size:     coss.ButtonSM,
						Disabled: regDisabled,
						Key:      "ws-reg-confirm",
					}).Clicked() {
						m.OnRegister(request.Token, request.Draft)
					}
				}
			})
		})
	})
	if !open && !request.Busy {
		m.Store.CancelWorkspaceNavigation(request.Token)
	}
}

// The conversation chip owns this anchor. Directory picking is a separate flow
// opened by the pinned add action, just as in WorkspacePickFlow.
func pickMenu(c *ui.Context, m Model, snap session.ViewState, request *session.WorkspaceNavigationState, colors theme.Colors) {
	open := true
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	_, height := c.Size()
	bg := chrome.Color(colors.BG)
	dark := bg.R < 128 && bg.G < 128 && bg.B < 128
	ui.PopoverBase(c, m.Anchor, &open, func(panel ui.Element) {
		panel = chrome.PopoverReveal(c, chrome.MenuSurface(panel.Width(280).MaxWidth(360).Padding(4).Gap(0),
			chrome.MenuBaseFill(colors, dark), chrome.MenuBorderColor(colors, dark), 16))
		panel.Children(func() {
			actionPath := m.Store.WorkspaceCwd()
			count := len(snap.Workspaces)
			for _, action := range m.DirectoryActions {
				if _, visible := action.Presentation(actionPath, true); visible {
					count++
				}
			}
			if m.OnNoDirectory != nil {
				count++
			}
			if m.OnBrowse != nil {
				count++
			}
			focused := ui.Local(panel, "menu-focus", func() int { return 0 })
			hovered := ui.Local(panel, "menu-hovered", func() int { return -1 })
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
			hoveredItem := -1
			index := 0
			row := func(key, label, path string, selected bool, icon string, disabled bool, choose func()) {
				button := menuRow(c.Key(key), label, path, selected, icon, disabled, colors)
				if button.Hovered() && !disabled {
					hoveredItem = index
					if !focusNext && !*first && *hovered != index {
						*hovered = index
						*focused = index
						button.Focus()
					}
				}
				if (*first || focusNext) && index == *focused {
					button.Focus().ScrollIntoView()
				}
				if !focusNext && !*first && button.Focused() {
					*focused = index
				}
				if button.Pressed() {
					button.Background(chrome.Color(colors.Pressed))
				} else if index == *focused || button.FocusWithin() {
					button.Background(chrome.Color(colors.Hover))
				}
				index++
				if button.Clicked() && choose != nil {
					choose()
				}
			}
			if len(snap.Workspaces) > 0 {
				// Reserve a right scrollbar gutter so the vertical scrollbar never overlaps the selected checkmark.
				ui.Scroll(c.Key("workspace-picker-scroll")).FillWidth().MinWidth(0).Height(min(float32(len(snap.Workspaces))*34, max(34, height-180))).Padding(0, design.Sp(3), 0, 0).Children(func() {
					for _, workspace := range snap.Workspaces {
						ws := workspace
						title := session.WorkspaceDisplayTitle(ws.Title, i18n.T(i18n.KeyWSDefaultWorkspaceTitle))
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
				ui.Text(c, i18n.T(i18n.KeyWSPickerLoading)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(6, 8)
			}
			if m.OnNoDirectory != nil {
				row("no-directory", i18n.T(i18n.KeyWSMenuNoDirectory), "", m.Store.WorkspaceID() == "" && snap.ScratchCwd != "" && snap.SelectedCwd == snap.ScratchCwd, "chat", request.Busy, func() {
					m.Store.CancelWorkspaceNavigation(request.Token)
					m.OnNoDirectory()
				})
			}
			if m.OnBrowse != nil {
				row("add-workspace", i18n.T(i18n.KeyWSMenuAddWorkspace), "", false, "add", request.Busy, m.OnBrowse)
			}
			for _, action := range m.DirectoryActions {
				label, visible := action.Presentation(actionPath, true)
				if !visible {
					continue
				}
				row("directory-action:"+action.ID, label, "", false, "folder", request.Busy, func() {
					m.Store.CancelWorkspaceNavigation(request.Token)
					if action.Pick != nil {
						action.Pick()
					} else {
						action.Run(actionPath)
					}
				})
			}
			if request.Error != "" {
				ui.Text(c, request.Error).FontSize(theme.InterfaceSize(12)).TextColor(fg).Padding(6, 8)
			}
			*hovered = hoveredItem
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
	if !disabled {
		button.Cursor(ui.CursorPointer)
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
	borderL2 := chrome.Color(colors.Border)
	danger := design.For(c).Destructive
	title := text("工作区", "Workspace")
	switch request.Mode {
	case "rename":
		title = i18n.T(i18n.KeyWSRenameWorkspaceTitle)
	case "remove":
		title = i18n.T(i18n.KeyWSDeleteWorkspace)
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		panel.Key(request.Token).Width(380).MaxWidthPercent(94).Padding(0, 0, 24, 0).Gap(20).Radius(28).Clip().
			Background(chrome.Color(colors.Field)).Draw(chrome.Glass(chrome.Color(colors.Field), chrome.MaterialPanel, 28))
		chrome.MenuElevation(panel)
		panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if borderL2.A != 0 && r.W > 0 && r.H > 0 {
				p.Stroke(r, borderL2, 28, 0.5)
			}
		})
		panel.Children(func() {
			submitted := false
			blocked := request.Busy
			draft := request.Draft
			feedback := request.Error
			targetTitle := session.WorkspaceDisplayTitle(request.TargetTitle, i18n.T(i18n.KeyWSDefaultWorkspaceTitle))
			ui.Column(c).FillWidth().MinWidth(0).Gap(0).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Padding(22, 14, 12, 24).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0)
					if coss.Button(c, "", coss.ButtonOptions{
						Variant:  coss.ButtonGhost,
						Size:     coss.ButtonIconSM,
						Icon:     coss.IconX,
						Label:    text("关闭", "Close"),
						Disabled: request.Busy,
						Key:      "ws-mgmt-close",
					}).Shrink(0).Clicked() {
						m.Store.CancelWorkspaceNavigation(request.Token)
					}
				})
				if request.Mode == "remove" {
					desc := FormatWorkspaceDeleteDescription(targetTitle)
					if request.Busy {
						desc = WorkspaceLocaleText("delete.pending")
					}
					ui.Column(c).FillWidth().MinWidth(0).Padding(0, 24).Children(func() {
						ui.Text(c, desc).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
					})
					if feedback != "" {
						ui.Column(c).FillWidth().MinWidth(0).Margin(20, 0, 0, 0).Padding(0, 24).Children(func() {
							ui.Text(c, feedback).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(danger)
						})
					}
				} else {
					ui.Column(c).FillWidth().MinWidth(0).Margin(8, 0, 0, 0).Padding(0, 24).Gap(8).Children(func() {
						if request.Mode == "rename" {
							seeded := ui.Local(panel, "rename-seeded", func() bool { return false })
							if !*seeded {
								draft = session.WorkspaceDisplayTitle(draft, i18n.T(i18n.KeyWSDefaultWorkspaceTitle))
								if draft != request.Draft {
									m.Store.SetWorkspaceNavigationDraft(request.Token, draft)
								}
								*seeded = true
							}
							input := coss.Input(c, &draft, coss.InputOptions{
								Size:     coss.InputLG,
								Label:    i18n.T(i18n.KeyWSFieldWorkspaceName),
								Disabled: request.Busy,
								Invalid:  feedback != "",
								Key:      request.Token,
							}).Height(44).FixedLineHeight(44)
							first := ui.Local(input, "rename-focus", func() bool { return true })
							if *first {
								input.Focus().SetTextSelection(0, utf8.RuneCountInString(draft))
								*first = false
							}
							if input.Changed() {
								m.Store.SetWorkspaceNavigationDraft(request.Token, draft)
								feedback = ""
							}
							trimmed := strings.TrimSpace(draft)
							blocked = blocked || trimmed == ""
							found := false
							for _, ws := range snap.Workspaces {
								if ws.WorkspaceID == request.TargetID {
									found = true
									blocked = blocked || ws.Title == trimmed
									continue
								}
								if trimmed != "" && session.WorkspaceDisplayTitle(ws.Title, i18n.T(i18n.KeyWSDefaultWorkspaceTitle)) == trimmed {
									blocked = true
									feedback = i18n.WSf(i18n.KeyWSConflictNamed, "name", trimmed)
								}
							}
							blocked = blocked || !found
							submitted = input.Submitted() && !input.Composing()
						} else {
							ui.Text(c, targetTitle).FontSize(theme.InterfaceSize(14)).TextColor(fg)
							if coss.Button(c, i18n.T(i18n.KeyWSRenameWorkspaceTitle), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonMD, Key: "ws-mgmt-rename-entry"}).FillWidth().Clicked() {
								m.Store.BeginWorkspaceRename(request.TargetID)
							}
							if coss.Button(c, i18n.T(i18n.KeyWSDeleteWorkspace), coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonMD, Key: "ws-mgmt-delete-entry"}).FillWidth().Clicked() {
								m.Store.BeginWorkspaceRemoval(request.TargetID)
							}
						}
						if feedback != "" {
							ui.Text(c, feedback).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(danger)
						}
						if request.Busy {
							ui.Text(c, text("等待 Host 确认…", "Waiting for the Host…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						}
					})
				}
			})
			ui.Row(c).FillWidth().Padding(0, design.Sp(6)).Gap(design.Sp(2)).AlignItems(ui.Center).Justify(ui.End).Children(func() {
				cancelLabel := text("取消", "Cancel")
				if coss.Button(c, cancelLabel, coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonLG,
					Disabled: request.Busy,
					Key:      "ws-mgmt-cancel",
				}).Label(cancelLabel).Clicked() {
					m.Store.CancelWorkspaceNavigation(request.Token)
				}
				if request.Mode == "rename" {
					renameLabel := i18n.T(i18n.KeyWSRename)
					renameBtn := coss.Button(c, renameLabel, coss.ButtonOptions{
						Variant:  coss.ButtonDefault,
						Size:     coss.ButtonLG,
						Disabled: blocked,
						Key:      "ws-mgmt-rename",
					}).Label(renameLabel)
					if renameBtn.Clicked() || submitted && !blocked {
						m.Store.ConfirmWorkspaceRename(context.Background(), request.Token, draft)
					}
				} else if request.Mode == "remove" {
					deleteLabel := i18n.T(i18n.KeyWSDeleteWorkspace)
					if coss.Button(c, deleteLabel, coss.ButtonOptions{
						Variant:  coss.ButtonDestructiveOutline,
						Size:     coss.ButtonLG,
						Disabled: request.Busy,
						Key:      "ws-mgmt-delete",
					}).Label(deleteLabel).Clicked() {
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
