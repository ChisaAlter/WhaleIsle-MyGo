package pet

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"sync"
	"whaleisle/internal/store/storelocale"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

func quickChatName(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

var quickChatAvatarOnce sync.Once
var quickChatAvatar *ui.Bitmap

func (q *QuickChat) applyTheme(c *ui.Context) {
	t := *c.Theme()
	if q.owner != nil && q.owner.Store != nil {
		resolved := fonttheme.Resolve(q.owner.Store.Get().Theme)
		colors := resolved.Active(t.Dark)
		if resolved.Scheme == fonttheme.SchemeDark {
			t.Dark = true
		} else if resolved.Scheme == fonttheme.SchemeLight {
			t.Dark = false
		}
		t.Background = chrome.Color(colors.BG)
		t.Text = chrome.Color(colors.FG)
		t.TextMuted = chrome.Color(colors.Muted)
		t.Surface = chrome.Color(colors.Field)
		t.SurfaceHover = chrome.Color(colors.Hover)
		t.SurfacePressed = chrome.Color(colors.Pressed)
		t.Border = chrome.Color(colors.Border)
		t.Accent = chrome.Color(colors.Accent)
		t.AccentText = chrome.Color(colors.ButtonFG)
		t.Danger = chrome.Color(colors.Danger)
		c.SetTheme(&t)
	}
	fonttheme.ApplyInterface(c)
}
func (q *QuickChat) feedback() string {
	if q.readNote != "" {
		return q.readNote
	}
	if q.stale || q.options.Direct != nil {
		return q.status
	}
	// Successful state and shared-session explanations do not occupy a permanent
	// row. Every state is still available from the header's status menu.
	for _, term := range []string{"失败", "无法", "未确认", "不确定", "取消", "2000", "unavailable", "failed", "unconfirmed", "cancel", "changed before"} {
		if strings.Contains(strings.ToLower(q.status), term) {
			return q.status
		}
	}
	return ""
}
func (q *QuickChat) rowMetrics(c *ui.Context, message quickChatRow) (float32, float32) {
	w, h := c.MeasureText(178, ui.Span{Text: message.Text, Font: fonttheme.RenderPreferences().Chat.Family, Size: fonttheme.InterfaceSize(12)})
	if message.Image != nil {
		w = max(w, 128)
		h += 132
	}
	return min(float32(198), max(float32(32), w+18)), max(float32(28), h+14)
}

// Native fixed-width card; only the bounded thread and one-to-three-line draft
// determine its height. These are the same session, draft and painted-image
// receipts consumed by the main chat, not a second chat implementation.
func (q *QuickChat) view(c *ui.Context) {
	q.applyTheme(c)
	direct := q.options.Direct != nil
	active := q.win != nil && q.check()
	t := c.Theme()
	quickChatAvatarOnce.Do(func() { b, _ := assets.ReadFile("assets/pet-head.png"); quickChatAvatar, _ = ui.DecodeBitmap(b) })
	threadHeight := float32(44)
	if len(q.state.History) > 0 {
		threadHeight = 4
		for _, message := range q.state.History {
			_, h := q.rowMetrics(c, message)
			threadHeight += h + 6
		}
		threadHeight = min(float32(148), threadHeight)
	}
	_, draftH := c.MeasureText(184, ui.Span{Text: q.draft + " ", Font: fonttheme.RenderPreferences().Composer.Family, Size: fonttheme.InterfaceSize(12)})
	editorHeight := min(float32(66), max(float32(28), draftH+12))
	feedback := q.feedback()
	noteHeight := float32(0)
	if feedback != "" {
		_, h := c.MeasureText(220, ui.Span{Text: feedback, Size: fonttheme.InterfaceSize(10)})
		noteHeight = min(float32(32), h) + 4
	}
	wanted := int(20 + 28 + threadHeight + editorHeight + 24 + 18 + noteHeight)
	if feedback != "" {
		wanted += 6
	}
	if q.win != nil && !q.win.IsDestroyed() && q.height != wanted {
		q.height = wanted
		q.win.SetSize(quickChatWidth, wanted)
	}
	ui.Column(c).Fill().Padding(10, 12).Gap(6).Radius(12).Border(1, t.Border).Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Height(28).Gap(6).AlignItems(ui.Center).Children(func() {
			if quickChatAvatar != nil {
				ui.Image(c, quickChatAvatar).Size(24, 24).Fit(ui.Cover).Radius(12)
			}
			ui.Text(c, quickChatName(q.state.Name, storelocale.T(storelocale.KeyStorePetWhale))).FontWeight(600).FontSize(fonttheme.InterfaceSize(12)).Grow(1).MinWidth(0).SingleLine().Tooltip(q.status).DragWindow()
			ui.MenuButton(c, "⋯", func(menu *ui.Menu) {
				if q.status != "" {
					menu.Item(q.status).Disabled(true)
				}
				if q.readNote != "" {
					menu.Item(q.readNote).Disabled(true)
				}
				menu.Separator()
				if q.sending.busy() && menu.Item(storelocale.T(storelocale.KeyDesktopPetCancelWait)).Disabled(!active).Chosen() {
					q.cancelWait()
				}
				if !direct {
					if menu.Item(storelocale.T(storelocale.KeyDesktopPetStopTurn)).Disabled(!active || q.state.SessionID == "" || q.stopping.busy() || q.picking.busy()).Chosen() {
						q.stopTurn()
					}
					if menu.Item(storelocale.T(storelocale.KeyDesktopPetRefresh)).Disabled(!active || q.history.busy()).Chosen() {
						q.refreshHistory()
						q.refreshCatalog()
					}
				}
			}).Width(20).Height(24).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(t.TextMuted).Label(storelocale.T(storelocale.KeyDesktopPetStatus))
			if !direct && ui.Button(c, "↗").Width(20).Height(24).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(t.TextMuted).Label(storelocale.T(storelocale.KeyDesktopPetOpen)).Disabled(!active || q.state.SessionID == "" || q.jumping.busy()).Clicked() {
				q.openMain()
			}
			if ui.Button(c, "×").Width(20).Height(24).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(t.TextMuted).Label(storelocale.T(storelocale.KeyDesktopPetCloseQuickChat)).Clicked() {
				q.owner.CloseQuickChat()
			}
		})
		if len(q.state.History) == 0 {
			ui.Column(c).FillWidth().Height(threadHeight).Center().Children(func() {
				line := storelocale.T(storelocale.KeyDesktopPetLoadingSharedHistory)
				if q.state.SessionID != "" || direct {
					line = storelocale.T(storelocale.KeyDesktopPetNoMessagesYet)
				}
				if q.stale {
					line = storelocale.T(storelocale.KeyDesktopPetConnectionUnavailable)
				}
				ui.Text(c, line).FontSize(fonttheme.InterfaceSize(11)).TextColor(t.TextMuted)
			})
		} else {
			ui.List(c, &q.list, len(q.state.History)).Label(storelocale.T(storelocale.KeyDesktopPetSharedHistory)).FillWidth().Height(threadHeight).Rows(func(row ui.ListRow) {
				message := q.state.History[row.Index]
				mine := message.Role == "user"
				w, _ := q.rowMetrics(c, message)
				ui.Row(row.Context).FillWidth().Padding(0, 0, 6, 0).Children(func() {
					if mine {
						ui.Box(row.Context).Grow(1)
					}
					fill, color := t.Surface, t.Text
					if mine {
						fill = t.Accent.Alpha(.13)
					}
					if message.Role == "err" {
						color = t.Danger
					}
					bubble := ui.Column(row.Context).Width(w).MinWidth(0).Padding(7, 9).Gap(4).Radius(10).Background(fill)
					bubble.Children(func() {
						if message.Text != "" {
							ui.Text(row.Context, message.Text).FillWidth().Font(fonttheme.RenderPreferences().Chat.Family).FontSize(fonttheme.InterfaceSize(12)).TextColor(color).Selectable()
						}
						if message.Image != nil {
							ui.Image(row.Context, message.Image).Width(128).Height(128).Fit(ui.Contain).DrawOver(func(p *ui.Painter, r ui.Rect) {
								clip := p.ClipBounds()
								if message.Acknowledge != nil && r.W > 0 && r.H > 0 && r.X < clip.X+clip.W && r.X+r.W > clip.X && r.Y < clip.Y+clip.H && r.Y+r.H > clip.Y {
									message.Acknowledge()
								}
							})
						}
					})
					if !mine {
						ui.Box(row.Context).Grow(1)
					}
				})
			})
		}
		if feedback != "" {
			ui.Text(c, feedback).FillWidth().FontSize(fonttheme.InterfaceSize(10)).TextColor(t.TextMuted).MaxLines(2).Tooltip(feedback)
		}
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.End).Children(func() {
			input := ui.TextAreaBase(c, &q.draft).Grow(1).MinWidth(0).Height(editorHeight).MaxHeight(66).Padding(6, 8).Radius(8).Border(1, t.Border).Font(fonttheme.RenderPreferences().Composer.Family).FontSize(fonttheme.InterfaceSize(12)).Label(storelocale.T(storelocale.KeyDesktopPetQuickChatDraft)).Lines(1, 3).Placeholder(storelocale.T(storelocale.KeyDesktopPetEnterSendsShiftEnterNewline))
			composing := input.Composing()
			input.HandleInput(func(ev ui.InputEvent) bool {
				if !quickChatSendKey(ev, composing) {
					return false
				}
				if !ev.Repeat && active && !q.sending.busy() && !q.picking.busy() && !q.stopping.busy() && strings.TrimSpace(q.draft) != "" {
					q.send()
				}
				return true
			})
			if !c.Root().FocusWithin() {
				input.AutoFocus()
			}
			if q.sending.busy() {
				if ui.Button(c, "×").Label(storelocale.T(storelocale.KeyDesktopPetCancelWait)).Width(28).Height(28).Radius(14).Padding(0).Clicked() {
					q.cancelWait()
				}
			} else if ui.Button(c, "↑").Label(storelocale.T(storelocale.KeyDesktopPetSend)).Width(28).Height(28).Radius(14).Padding(0).BorderWidth(0).Background(t.Accent).TextColor(t.AccentText).Disabled(!active || (!direct && q.state.SessionID == "") || q.picking.busy() || q.stopping.busy() || strings.TrimSpace(q.draft) == "").Clicked() {
				q.send()
			}
		})
		if !direct {
			q.modelPicker(c, active)
		} else {
			ui.Text(c, storelocale.T(storelocale.KeyDesktopPetTemporaryOnlineApiChat)).FontSize(fonttheme.InterfaceSize(10)).TextColor(t.Warning).SingleLine()
		}
	})
	if c.Shortcut(0, ui.KeyEscape) {
		q.owner.CloseQuickChat()
	}
}
func (q *QuickChat) modelPicker(c *ui.Context, active bool) {
	label := storelocale.T(storelocale.KeyDesktopPetModel)
	if selected := q.state.Model; selected != nil {
		label = selected.Model
		if selected.ReasoningEffort != "" {
			label += " · " + selected.ReasoningEffort
		}
	}
	if ui.Button(c, label+" ▾").Label(storelocale.T(storelocale.KeyDesktopPetSelectSharedModel)).FillWidth().MinWidth(0).Height(24).Padding(0, 4).SingleLine().Tooltip(label).BorderWidth(0).Background(ui.Transparent).TextColor(c.Theme().TextMuted).FontSize(fonttheme.InterfaceSize(10)).Disabled(!active || q.sending.busy() || q.picking.busy() || q.stopping.busy() || len(q.state.Groups) == 0).Clicked() {
		q.openModelPicker()
	}
}
