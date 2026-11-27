package dirpick

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	OnNativePicker                              func()
	OnNativeMode                                func(bool)
	OnEditPath, OnSubmitPath                    func(string)
}

type browseRecoveryFocus struct {
	path       ui.Handle
	retryFocus bool
}

// Header and footer never scroll; the Miller panes split evenly above 256 DIP.
func View(c *ui.Context, m Model) {
	if m.State == nil {
		return
	}
	s := m.State
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)
	inert := m.Busy || s.NativeBusy || s.FolderDraft != nil
	requestedPathFocus := false
	pathFocused := false
	root := chrome.FrostSurface(ui.Column(c).Fill().MinWidth(0).MinHeight(0), colors, m.SystemDark, chrome.Color(colors.Field), 28)
	recoveryFocus := ui.Local(root, "browse-recovery-focus", func() browseRecoveryFocus { return browseRecoveryFocus{} })
	if !s.UseNative && s.PathDraft != nil && s.FolderDraft == nil && m.OnCancelPath != nil {
		root.OnShortcut(0, ui.KeyEscape, m.OnCancelPath)
	}
	root.Children(func() {
		ui.Column(c).FillWidth().Shrink(0).Padding(16, 14, 8, 24).Gap(8).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Height(28).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, dirText("\u9009\u62e9\u5de5\u4f5c\u76ee\u5f55", "Select Workspace Directory")).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
				if m.OnNativeMode == nil && m.OnNativePicker != nil && coss.Button(c, dirText("系统选择器…", "System picker…"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: inert || m.Loading}).Clicked() {
					m.OnNativePicker()
				}
				if !s.UseNative && coss.Button(c, dirText("主目录", "Home"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: inert}).Clicked() && m.OnNavigate != nil {
					path := ""
					if s.Parent != nil {
						path = s.Parent.Home
					}
					m.OnNavigate(path)
				}
			})
			if s.UseNative {
				return
			}
			bar := ui.Row(c).FillWidth().MinWidth(0).Height(24).AlignItems(ui.Center).Gap(4).Border(1, ui.Transparent).Radius(6)
			if s.PathDraft != nil {
				bar.BorderColor(border)
			}
			bar.Children(func() {
				if s.PathDraft != nil {
					input := ui.TextInputBase(c.Key("directory-path-input"), s.PathDraft).Label(dirText("编辑路径", "Edit path")).FillWidth().Height(22).FontSize(theme.InterfaceSize(13)).TextColor(fg).Disabled(inert).AutoFocus().Bind(&recoveryFocus.path)
					if recoveryFocus.retryFocus {
						input.Focus()
						recoveryFocus.retryFocus = false
					}
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
									button := coss.Button(c, "", coss.ButtonOptions{
										Variant:  coss.ButtonGhost,
										Size:     coss.ButtonXS,
										Disabled: inert,
										Label:    label,
										Content: func() {
											ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(muted).MinWidth(0).SingleLine()
										},
									}).Tooltip(crumb.Path).Height(22).MaxWidth(160).Padding(0)
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
					edit := coss.Button(c, "", coss.ButtonOptions{
						Variant:  coss.ButtonGhost,
						Size:     coss.ButtonXS,
						Disabled: inert,
						Label:    dirText("编辑路径", "Edit path"),
						Content: func() {
							ui.Row(c).FillWidth().Justify(ui.End).Children(func() { editGlyph(c, muted) })
						},
					}).Tooltip(dirText("编辑路径", "Edit path")).MinWidth(34).Grow(1).Height(22).Padding(design.Sp(1))
					if edit.Hovered() || edit.FocusVisible() {
						bar.BorderColor(border)
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
			if s.UseNative {
				ui.Column(c).FillWidth().Grow(1).MinHeight(0).Justify(ui.Center).AlignItems(ui.Center).Gap(12).Children(func() {
					ui.Icon(c, nativeFolderArtwork).Size(24, 24).TextColor(muted)
					ui.Text(c, dirText("使用 Windows 默认窗口选择本机工作区目录。关闭下方开关可返回应用内浏览。", "Use the Windows folder picker to select a local workspace directory. Turn off the switch below to return to the in-app browser.")).MaxWidth(360).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(muted).TextAlign(ui.Center)
					label := dirText("选择文件夹", "Choose folder")
					if s.NativeBusy {
						label = dirText("选择中…", "Choosing…")
					}
					if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonMD, Disabled: inert}).Clicked() && m.OnNativePicker != nil {
						m.OnNativePicker()
					}
				})
				return
			}
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
				ui.Text(c, dirText("\u52a0\u8f7d\u4e2d\u2026", "Loading\u2026")).Absolute().Right(16).Bottom(8).Padding(2, 8).Background(chrome.FrostInset(colors, m.SystemDark, chrome.Color(colors.Field))).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		})
		ui.Box(c).FillWidth().Height(.5).Shrink(0).Background(border)
		ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Wrap().Gap(8).Padding(16, 24).AlignItems(ui.Center).Children(func() {
			blocked := inert || m.Loading || s.PathDraft != nil || !directorypicker.SelectablePath(s.TargetPath())
			if !s.UseNative {
				if coss.Button(c, dirText("新建文件夹", "New folder"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonMD, Disabled: blocked}).Clicked() {
					draft := ""
					s.FolderDraft = &draft
					s.CreateError = ""
				}
				label := dirText("显示隐藏文件", "Show hidden files")
				if s.ShowHidden {
					label += "  ✓"
				}
				toggle := coss.Button(c, "", coss.ButtonOptions{
					Variant:  coss.ButtonGhost,
					Size:     coss.ButtonSM,
					Disabled: inert,
					Label:    dirText("显示隐藏文件", "Show hidden files"),
					Content: func() {
						ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(muted).SingleLine()
					},
				})
				if toggle.Clicked() {
					s.ShowHidden = !s.ShowHidden
				}
			}
			if m.OnNativeMode != nil {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					label := dirText("Windows 系统选择器", "Windows folder picker")
					ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(muted).SingleLine()
					native := s.UseNative
					if ui.Switch(c, &native).Label(label).Disabled(inert).Changed() {
						m.OnNativeMode(native)
					}
				})
			}
			ui.Box(c).Grow(1).MinWidth(0)
			if coss.Button(c, dirText("\u53d6\u6d88", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonMD, Disabled: inert}).MinWidth(72).Clicked() && m.OnClose != nil {
				m.OnClose()
			}
			if !s.UseNative && coss.Button(c, dirText("打开", "Open"), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonMD, Disabled: blocked}).MinWidth(72).Clicked() && m.OnPick != nil {
				m.OnPick(s.TargetPath())
			}
		})
		if m.Err != "" && s.FolderDraft == nil {
			widgets.Alert(c, widgets.AlertOptions{ID: "directory-browse-error", Message: m.Err,
				Action: dirText("重试", "Retry"), OnAction: func() {
					recoveryFocus.retryFocus = true
					if m.OnRetry != nil {
						m.OnRetry()
					}
				},
				Dismiss: dirText("关闭", "Dismiss"), OnDismiss: m.OnDismissError})
		}
	})
	if !s.UseNative && s.PathDraft != nil && s.FolderDraft == nil {
		// A newly mounted input can own native focus before this frame's
		// ancestor chain is committed. It is still inside this card.
		if root.FocusWithin() || pathFocused {
			s.PathHadFocus = true
		} else if s.PathHadFocus && !requestedPathFocus && m.OnCancelPath != nil {
			m.OnCancelPath()
		}
	} else {
		s.PathHadFocus = false
	}
	createDialog(c, m)
}

var nativeFolderArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7h6l2 2h10v11H3z"/><path d="M3 13h18"/></svg>`))

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
			button := coss.Button(c.Key(entry.Path), "", coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonSM,
				Disabled: inert,
				Pressed:  entry.Path == selected,
				Label:    entry.Name,
				Content: func() {
					ui.Row(c).FillWidth().Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
						icon := chrome.Color(colors.Muted)
						if entry.Path == selected {
							icon = chrome.Color(colors.ButtonInfoFill)
						}
						folderIcon(c, icon, entry.Path == selected)
						ui.Text(c, entry.Name).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(chrome.Color(colors.FG)).Grow(1).MinWidth(0).SingleLine()
						ui.Text(c, "›").FontSize(12).TextColor(chrome.Color(colors.Muted))
					})
				},
			}).Tooltip(entry.Path).FillWidth().Padding(design.Sp(1)).Radius(design.RadiusMD)
			if entry.Path == selected {
				button.Background(chrome.Color(colors.Pressed))
				if s.FocusSelection {
					button.Focus()
					s.FocusSelection = false
				}
			} else if button.Hovered() || button.FocusWithin() {
				button.Background(chrome.Color(colors.Hover))
			}
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
		return
	}
	open := true
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	w, h := c.Size()
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		chrome.DialogReveal(c, backdrop, panel)
		chrome.FrostSurface(panel.Key("directory-create-dialog").Width(min(float32(380), max(float32(0), w-48))).MaxHeight(max(float32(0), h-48)).Padding(22, 24, 20), colors, m.SystemDark, chrome.Color(colors.Field), 28).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
				ui.Text(c, dirText("新建文件夹", "New folder")).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(chrome.Color(colors.FG))
				name := s.TargetName(dirText("\u4e3b\u76ee\u5f55", "Home"))
				ui.Text(c, dirText("\u5728\"", "New folder in \"")+name+dirText("\"\u4e2d\u65b0\u5efa\u6587\u4ef6\u5939", "\"")).FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.Muted))
				if s.CreateError != "" {
					widgets.Alert(c, widgets.AlertOptions{ID: "directory-create-error", Message: s.CreateError,
						Action: dirText("重试", "Retry"), OnAction: m.OnRetryCreate,
						Dismiss: dirText("关闭", "Dismiss"), OnDismiss: m.OnDismissCreateError})
				}
				input := coss.Input(c.Key("directory-folder-name"), s.FolderDraft, coss.InputOptions{
					Size:        coss.InputLG,
					Label:       dirText("文件夹名称", "Folder name"),
					Placeholder: dirText("未命名文件夹", "Untitled folder"),
					Disabled:    s.Creating,
				}).Height(44).FixedLineHeight(44).AutoFocus()
				ui.Row(c).FillWidth().Gap(8).Margin(8, 0, 0, 0).Justify(ui.End).Children(func() {
					if coss.Button(c, dirText("取消", "Cancel"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonMD, Disabled: s.Creating}).Clicked() {
						s.FolderDraft = nil
						s.CreateError = ""
					}
					create := coss.Button(c, dirText("创建", "Create"), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonMD, Disabled: s.Creating || s.FolderDraft == nil || strings.TrimSpace(*s.FolderDraft) == ""})
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
}

func dirText(zh, en string) string {
	return i18n.Pick(zh, en)
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
