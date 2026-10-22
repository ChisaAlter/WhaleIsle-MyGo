package dirpick

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type Model struct {
	State                                       *BrowserState
	ThemeID                                     string
	SystemDark, Loading, Busy                   bool
	Err                                         string
	OnNavigate                                  func(string)
	OnSelect                                    func(api.DirectoryListingCrumbsItem, bool)
	OnPick                                      func(string)
	OnCreate                                    func(string, string)
	OnClose, OnRetry, OnBeginPath, OnCancelPath func()
	OnDismissError, OnDismissCreateError        func()
	OnRetryCreate                               func()
	OnEditPath, OnSubmitPath                    func(string)
}

type browseRecoveryFocus struct{ path, action ui.Handle }

// Header and footer never scroll; the Miller panes split evenly above 256 DIP.
func View(c *ui.Context, m Model) {
	if m.State == nil {
		return
	}
	s := m.State
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	inert := m.Busy || s.FolderDraft != nil
	requestedPathFocus := false
	pathFocused := false
	root := ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(chrome.Color(colors.Field)).Radius(28)
	recoveryFocus := ui.Local(root, "browse-recovery-focus", func() browseRecoveryFocus { return browseRecoveryFocus{} })
	if s.PathDraft != nil && s.FolderDraft == nil && m.OnCancelPath != nil {
		root.OnShortcut(0, ui.KeyEscape, m.OnCancelPath)
	}
	root.Children(func() {
		ui.Column(c).FillWidth().Shrink(0).Padding(16, 14, 8, 24).Gap(8).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Height(28).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, dirText("\u9009\u62e9\u5de5\u4f5c\u76ee\u5f55", "Select Workspace Directory")).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
				if ui.Button(c, dirText("主目录", "Home")).Height(28).Padding(2, 8).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(inert).Clicked() && m.OnNavigate != nil {
					path := ""
					if s.Parent != nil {
						path = s.Parent.Home
					}
					m.OnNavigate(path)
				}
			})
			bar := ui.Row(c).FillWidth().MinWidth(0).Height(24).AlignItems(ui.Center).Gap(4)
			if s.PathDraft != nil {
				bar.Border(1, border).Radius(6)
			}
			bar.Children(func() {
				if s.PathDraft != nil {
					input := ui.TextInputBase(c.Key("directory-path-input"), s.PathDraft).Label(dirText("编辑路径", "Edit path")).FillWidth().Height(22).FontSize(theme.InterfaceSize(13)).TextColor(fg).Disabled(inert).AutoFocus().Bind(&recoveryFocus.path)
					if s.FocusPath {
						input.Focus()
						s.FocusPath = false
						requestedPathFocus = true
					}
					pathFocused = input.Focused()
					if input.Changed() && m.OnEditPath != nil {
						m.OnEditPath(*s.PathDraft)
					}
					if input.Submitted() && !input.Composing() && strings.TrimSpace(*s.PathDraft) != "" && m.OnSubmitPath != nil {
						m.OnSubmitPath(*s.PathDraft)
					}
				} else {
					current := s.Current()
					ui.ScrollHorizontal(c.Key("directory-crumb-trail")).MinWidth(0).Shrink(1).Height(24).Children(func() {
						ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
							if current != nil {
								for index, crumb := range current.Crumbs {
									if index > 0 {
										ui.Text(c, "›").FontSize(12).TextColor(muted)
									}
									label := crumb.Name
									if crumb.Path == current.Home {
										label = dirText("主目录", "Home")
									}
									if !directorypicker.SelectablePath(crumb.Path) {
										label = dirText("\u6b64\u7535\u8111", "This PC")
									}
									button := ui.Button(c, "").Label(label).Tooltip(crumb.Path).Height(22).MaxWidth(160).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(inert)
									button.Children(func() { ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(muted).MinWidth(0).SingleLine() })
									if index == len(current.Crumbs)-1 && s.ScrolledCrumb != current.Path {
										button.ScrollIntoView()
										s.ScrolledCrumb = current.Path
									}
									if button.Clicked() && m.OnNavigate != nil {
										m.OnNavigate(crumb.Path)
									}
								}
							}
						})
					})
					edit := ui.Button(c, "").Label(dirText("编辑路径", "Edit path")).Tooltip(dirText("编辑路径", "Edit path")).MinWidth(34).Grow(1).Height(22).Padding(4).BorderWidth(0).Background(ui.Transparent).Disabled(inert)
					edit.Children(func() { ui.Row(c).FillWidth().Justify(ui.End).Children(func() { editGlyph(c, muted) }) })
					if edit.Hovered() || edit.FocusVisible() {
						bar.Border(1, border).Radius(6)
					}
					if s.FocusEdit {
						edit.Focus()
						s.FocusEdit = false
					}
					if edit.Clicked() && m.OnBeginPath != nil {
						m.OnBeginPath()
					}
				}
			})
		})
		ui.Box(c).FillWidth().Height(.5).Shrink(0).Background(border)
		ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(16, 16, 16, 24).Children(func() {
			ui.ScrollHorizontal(c.Key("directory-miller-scroll")).FillWidth().Grow(1).MinHeight(0).Children(func() {
				width, _ := c.Size()
				paneWidth := max(float32(256), min(float32(640), width-88))
				if s.Selected != nil {
					paneWidth = max(float32(256), (min(float32(680), width-48)-64.5)/2)
				}
				ui.Row(c).HeightPercent(100).Gap(12).Children(func() {
					prefix, _ := s.Prefix()
					if s.Parent != nil {
						selected := ""
						if s.Selected != nil {
							selected = s.Selected.Path
						}
						leftPrefix := prefix
						if s.Child != nil {
							leftPrefix = ""
						}
						levelColumn(c.Key("directory-parent"), m, *s.Parent, selected, leftPrefix, paneWidth, false, inert)
					}
					if s.Selected != nil {
						ui.Box(c).Width(.5).FillHeight().Background(border)
						if s.Child != nil {
							levelColumn(c.Key("directory-child"), m, *s.Child, "", prefix, paneWidth, true, inert)
						} else {
							ui.Box(c).Width(paneWidth).FillHeight()
						}
					}
				})
			})
			if s.Parent != nil && s.Parent.Truncated || s.Child != nil && s.Child.Truncated {
				ui.Text(c, dirText("\u6587\u4ef6\u5939\u8fc7\u591a\uff0c\u4ec5\u663e\u793a\u5f00\u5934\u90e8\u5206\u3002", "Too many folders to list; only the beginning is shown.")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Shrink(0).Padding(4, 120, 4, 4)
			}
			if m.Loading && s.SlowScan {
				ui.Text(c, dirText("\u52a0\u8f7d\u4e2d\u2026", "Loading\u2026")).Absolute().Right(16).Bottom(8).Padding(2, 8).Background(chrome.Color(colors.Field)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		})
		ui.Box(c).FillWidth().Height(.5).Shrink(0).Background(border)
		ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Wrap().Gap(8).Padding(16, 24).AlignItems(ui.Center).Children(func() {
			blocked := inert || m.Loading || s.PathDraft != nil || !directorypicker.SelectablePath(s.TargetPath())
			if ui.Button(c, dirText("新建文件夹", "New folder")).FontSize(theme.InterfaceSize(13)).Height(32).Disabled(blocked).Clicked() {
				draft := ""
				s.FolderDraft = &draft
				s.CreateError = ""
			}
			label := dirText("显示隐藏文件", "Show hidden files")
			if s.ShowHidden {
				label += "  ✓"
			}
			toggle := ui.Button(c, "").Label(dirText("显示隐藏文件", "Show hidden files")).Height(28).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(inert)
			toggle.Children(func() { ui.Text(c, label).FontSize(theme.InterfaceSize(13)).SingleLine() })
			if toggle.Clicked() {
				s.ShowHidden = !s.ShowHidden
			}
			ui.Box(c).Grow(1).MinWidth(0)
			if ui.Button(c, dirText("\u53d6\u6d88", "Cancel")).Height(32).MinWidth(72).Disabled(inert).Clicked() && m.OnClose != nil {
				m.OnClose()
			}
			if ui.Button(c, dirText("打开", "Open")).Height(32).MinWidth(72).Background(chrome.Color(colors.Accent)).TextColor(chrome.Color(colors.ButtonFG)).Disabled(blocked).Clicked() && m.OnPick != nil {
				m.OnPick(s.TargetPath())
			}
		})
	})
	if s.PathDraft != nil && s.FolderDraft == nil {
		// A newly mounted input can own native focus before this frame's
		// ancestor chain is committed. It is still inside this card.
		if root.FocusWithin() || pathFocused || recoveryFocus.action.Focused(c) {
			s.PathHadFocus = true
		} else if s.PathHadFocus && !requestedPathFocus && m.OnCancelPath != nil {
			m.OnCancelPath()
		}
	} else {
		s.PathHadFocus = false
	}
	createDialog(c, m)
	if m.Err != "" && s.FolderDraft == nil && s.FailureToast != m.Err {
		c.AddToast(ui.Toast{ID: "directory-browse-error", Description: m.Err, Type: "error", Timeout: -1, Action: dirText("重试", "Retry"), ActionHandle: &recoveryFocus.action,
			ActionReturnFocus: &recoveryFocus.path, ActionOnEscape: m.OnCancelPath,
			OnAction: m.OnRetry, OnClose: m.OnDismissError})
		s.FailureToast = m.Err
	} else if m.Err == "" && s.FailureToast != "" {
		c.CloseToast("directory-browse-error")
		s.FailureToast = ""
	}
}

func levelColumn(c *ui.Context, m Model, listing api.DirectoryListing, selected, prefix string, width float32, right, inert bool) {
	s := m.State
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	column := ui.Scroll(c).Width(width).MinWidth(256).FillHeight().Gap(2).Padding(0, 8, 0, 0)
	if right && s.ScrolledChild != listing.Path {
		column.ScrollIntoView()
		s.ScrolledChild = listing.Path
	}
	column.Children(func() {
		for _, entry := range VisibleEntries(listing.Entries, selected, s.ShowHidden, prefix) {
			entry := entry
			button := ui.Button(c.Key(entry.Path), "").Label(entry.Name).Tooltip(entry.Path).FillWidth().Height(28).Padding(4).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(inert)
			if entry.Path == selected {
				button.Background(chrome.Color(colors.Pressed))
				if s.FocusSelection {
					button.Focus()
					s.FocusSelection = false
				}
			} else if button.Hovered() || button.FocusWithin() {
				button.Background(chrome.Color(colors.Hover))
			}
			button.Children(func() {
				ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
					icon := chrome.Color(colors.Muted)
					if entry.Path == selected {
						icon = chrome.Color(colors.ButtonInfoFill)
					}
					folderIcon(c, icon, entry.Path == selected)
					ui.Text(c, entry.Name).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(chrome.Color(colors.FG)).Grow(1).MinWidth(0).SingleLine()
					ui.Text(c, "›").FontSize(12).TextColor(chrome.Color(colors.Muted))
				})
			})
			if button.Clicked() && m.OnSelect != nil {
				m.OnSelect(entry, right)
			}
		}
	})
}

func folderIcon(c *ui.Context, color ui.Color, open bool) {
	ui.Box(c).Width(16).Height(16).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		path.MoveTo(r.X+1, r.Y+4)
		path.LineTo(r.X+6, r.Y+4)
		path.LineTo(r.X+8, r.Y+6)
		path.LineTo(r.X+14, r.Y+6)
		path.LineTo(r.X+14, r.Y+13)
		path.LineTo(r.X+1, r.Y+13)
		path.Close()
		p.StrokePath(path, 1.2, color)
		if open {
			path = &ui.Path{}
			path.MoveTo(r.X+2, r.Y+9)
			path.LineTo(r.X+13, r.Y+9)
			p.StrokePath(path, 1, color)
		}
	})
}

func createDialog(c *ui.Context, m Model) {
	s := m.State
	if s.FolderDraft == nil {
		c.CloseToast("directory-create-error")
		s.CreateToast = ""
		return
	}
	open := true
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	w, h := c.Size()
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		panel.Key("directory-create-dialog").Width(min(float32(380), max(float32(0), w-48))).MaxHeight(max(float32(0), h-48)).Padding(22, 24, 20).Radius(28).Background(chrome.Color(colors.Field)).Draw(chrome.Glass(chrome.Color(colors.Field), chrome.MaterialPanel, 28)).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
				ui.Text(c, dirText("新建文件夹", "New folder")).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(chrome.Color(colors.FG))
				name := s.TargetName(dirText("\u4e3b\u76ee\u5f55", "Home"))
				ui.Text(c, dirText("\u5728\"", "New folder in \"")+name+dirText("\"\u4e2d\u65b0\u5efa\u6587\u4ef6\u5939", "\"")).FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.Muted))
				input := ui.TextInput(c.Key("directory-folder-name"), s.FolderDraft).Label(dirText("文件夹名称", "Folder name")).Placeholder(dirText("未命名文件夹", "Untitled folder")).FillWidth().Height(44).Padding(7, 14).Radius(12).FontSize(theme.InterfaceSize(14)).Disabled(s.Creating).AutoFocus()
				ui.Row(c).FillWidth().Gap(8).Margin(8, 0, 0, 0).Justify(ui.End).Children(func() {
					if ui.Button(c, dirText("取消", "Cancel")).Disabled(s.Creating).Clicked() {
						s.FolderDraft = nil
						s.CreateError = ""
					}
					create := ui.Button(c, dirText("创建", "Create")).Background(chrome.Color(colors.Accent)).TextColor(chrome.Color(colors.ButtonFG)).Disabled(s.Creating || s.FolderDraft == nil || strings.TrimSpace(*s.FolderDraft) == "")
					if !s.Creating && s.FolderDraft != nil && strings.TrimSpace(*s.FolderDraft) != "" && (create.Clicked() || input.Submitted() && !input.Composing()) && m.OnCreate != nil {
						m.OnCreate(s.TargetPath(), *s.FolderDraft)
					}
				})
			})
		})
	})
	if !open && !s.Creating {
		s.FolderDraft = nil
		s.CreateError = ""
	}
	if s.CreateError != "" && s.CreateToast != s.CreateError {
		c.AddToast(ui.Toast{ID: "directory-create-error", Description: s.CreateError, Type: "error", Timeout: -1, Action: dirText("重试", "Retry"), OnAction: m.OnRetryCreate, OnClose: m.OnDismissCreateError})
		s.CreateToast = s.CreateError
	} else if s.CreateError == "" && s.CreateToast != "" {
		c.CloseToast("directory-create-error")
		s.CreateToast = ""
	}
}

func dirText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

func editGlyph(c *ui.Context, color ui.Color) {
	ui.Box(c).Width(14).Height(14).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		path.MoveTo(r.X+3, r.Y+9)
		path.LineTo(r.X+10, r.Y+2)
		path.LineTo(r.X+12, r.Y+4)
		path.LineTo(r.X+5, r.Y+11)
		path.LineTo(r.X+2, r.Y+12)
		path.Close()
		p.StrokePath(path, 1.1, color)
	})
}
