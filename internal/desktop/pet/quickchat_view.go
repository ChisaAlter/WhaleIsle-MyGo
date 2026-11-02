package pet

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/storelocale"
)

func quickChatName(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

// This is native MyGo content, not a page with general IPC/model credentials.
// Enter sends; Shift+Enter and IME composition remain with the editor.
func (q *QuickChat) view(c *ui.Context) {
	fonttheme.ApplyInterface(c)
	direct := q.options.Direct != nil
	active := q.win != nil && q.check()
	theme := c.Theme()
	_, height := c.Size()
	compact := height < 320
	ui.Column(c).Fill().Padding(8).Gap(4).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, quickChatName(q.state.Name, storelocale.T(storelocale.KeyStorePetWhale))+storelocale.T(storelocale.KeyStorePetChat)).FontWeight(600).FontSize(fonttheme.InterfaceSize(12)).Grow(1).SingleLine().DragWindow()
			if !direct && ui.Button(c, storelocale.T(storelocale.KeyDesktopPetOpen)).Disabled(!active || q.state.SessionID == "" || q.jumping.busy()).Clicked() {
				q.openMain()
			}
			if ui.Button(c, "×").Label(storelocale.T(storelocale.KeyDesktopPetCloseQuickChat)).Tooltip(storelocale.T(storelocale.KeyDesktopPetClosingOnlyCancelsTheLocalWait)).Clicked() {
				q.owner.CloseQuickChat()
			}
		})
		if !compact {
			if direct {
				ui.Text(c, storelocale.T(storelocale.KeyDesktopPetTemporaryOnlineApiChat)).FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.Warning).SingleLine()
			} else {
				ui.Text(c, storelocale.T(storelocale.KeyDesktopPetOneSharedConversation)).FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.TextMuted).SingleLine()
			}
		}
		if !direct {
			q.modelPicker(c, active)
		}
		if len(q.state.History) == 0 {
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				line := storelocale.T(storelocale.KeyDesktopPetLoadingSharedHistory)
				if direct {
					line = storelocale.T(storelocale.KeyDesktopPetClickSendToApiToSend)
				}
				if q.state.SessionID != "" {
					line = storelocale.T(storelocale.KeyDesktopPetNoMessagesYet)
				}
				if q.stale {
					line = storelocale.T(storelocale.KeyDesktopPetConnectionUnavailable)
				}
				ui.Text(c, line).TextColor(theme.TextMuted).FontSize(fonttheme.InterfaceSize(12))
			})
		} else {
			historyLabel := storelocale.T(storelocale.KeyDesktopPetSharedHistory)
			if direct {
				historyLabel = storelocale.T(storelocale.KeyDesktopPetTemporaryApiHistory)
			}
			ui.List(c, &q.list, len(q.state.History)).Label(historyLabel).Grow(1).FillWidth().MinHeight(0).Rows(func(row ui.ListRow) {
				message := q.state.History[row.Index]
				label, color := q.state.Name, theme.Text
				switch message.Role {
				case "user":
					label = storelocale.T(storelocale.KeyDesktopPetYou)
				case "her":
					label = quickChatName(label, storelocale.T(storelocale.KeyStorePetWhale))
				case "err":
					label, color = storelocale.T(storelocale.KeyDesktopPetStatus), theme.Danger
				default:
					label = storelocale.T(storelocale.KeyDesktopPetSession)
				}
				ui.Column(row.Context).Padding(8).Gap(3).FillWidth().Children(func() {
					ui.Text(row.Context, label).FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted)
					ui.Text(row.Context, message.Text).Font(fonttheme.RenderPreferences().Chat.Family).FontSize(fonttheme.InterfaceSize(12)).TextColor(color).Selectable()
				})
			})
		}
		if q.readNote != "" && !compact {
			ui.Text(c, q.readNote).FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.Warning).MaxLines(1).Tooltip(q.readNote)
		}
		ui.Text(c, q.status).FontSize(fonttheme.InterfaceSize(11)).TextColor(theme.TextMuted).MaxLines(1).Tooltip(q.status)
		minLines, maxLines := 2, 4
		if compact {
			minLines, maxLines = 1, 1
		}
		input := ui.TextArea(c, &q.draft).Font(fonttheme.RenderPreferences().Composer.Family).FontSize(fonttheme.InterfaceSize(12)).Label(storelocale.T(storelocale.KeyDesktopPetQuickChatDraft)).FillWidth().Lines(minLines, maxLines).Placeholder(storelocale.T(storelocale.KeyDesktopPetEnterSendsShiftEnterNewline))
		// Read the IME composition state during the build: input callbacks run
		// outside the build pass, where an Element query panics ("element from
		// pass N used in pass N"). The next frame refreshes it.
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
		ui.Row(c).FillWidth().Wrap().Gap(4).Children(func() {
			sendLabel := storelocale.T(storelocale.KeyDesktopPetSend)
			if direct {
				sendLabel = storelocale.T(storelocale.KeyDesktopPetSendToApi)
			}
			if ui.Button(c, sendLabel).Disabled(!active || (!direct && q.state.SessionID == "") || q.sending.busy() || q.picking.busy() || q.stopping.busy() || strings.TrimSpace(q.draft) == "").Clicked() {
				q.send()
			}
			if q.sending.busy() && ui.Button(c, storelocale.T(storelocale.KeyDesktopPetCancelWait)).Disabled(!active).Clicked() {
				q.cancelWait()
			}
			if !direct && ui.Button(c, storelocale.T(storelocale.KeyDesktopPetStopTurn)).Disabled(!active || q.state.SessionID == "" || q.stopping.busy() || q.picking.busy()).Clicked() {
				q.stopTurn()
			}
			if !direct && !compact && ui.Button(c, storelocale.T(storelocale.KeyDesktopPetRefresh)).Disabled(!active || q.history.busy()).Clicked() {
				q.refreshHistory()
				q.refreshCatalog()
			}
		})
		if !compact {
			if direct {
				ui.Text(c, storelocale.T(storelocale.KeyStorePetCancellingDoesNotGuaranteeTheApiReques)).FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted).MaxLines(2)
			} else {
				ui.Text(c, storelocale.T(storelocale.KeyStorePetCancellingTheWaitDoesNotWithdrawTheMe)).FontSize(fonttheme.InterfaceSize(10)).TextColor(theme.TextMuted).MaxLines(2)
			}
		}
	})
	if c.Shortcut(0, ui.KeyEscape) {
		q.owner.CloseQuickChat()
	}
}

func (q *QuickChat) modelPicker(c *ui.Context, active bool) {
	blocked := !active || q.sending.busy() || q.picking.busy() || q.stopping.busy()
	selected := q.state.Model
	modelLabel := storelocale.T(storelocale.KeyDesktopPetModel)
	if selected != nil {
		modelLabel = selected.Provider + " / " + selected.Model
	}
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		ui.MenuButton(c, modelLabel, func(menu *ui.Menu) {
			for _, group := range q.state.Groups {
				menu.Submenu(quickChatName(group.Name, group.ID)+" ["+group.ID+"]", func(sub *ui.Menu) {
					for _, model := range group.Models {
						choice := quickChatModel{Provider: group.ID, Model: model.ID, ReasoningEffort: model.DefaultEffort}
						if !q.state.allows(choice) {
							choice.ReasoningEffort = ""
						}
						picked := selected != nil && selected.Provider == group.ID && selected.Model == model.ID
						if sub.Item(quickChatName(model.Name, model.ID)+" ["+model.ID+"]").Checked(picked).Disabled(blocked).Chosen() && !picked {
							q.choose(choice)
						}
					}
				})
			}
		}).Label(storelocale.T(storelocale.KeyDesktopPetSelectSharedModel)).MaxWidthPercent(100).MinWidth(0).Disabled(blocked || len(q.state.Groups) == 0)
		if selected != nil {
			for _, group := range q.state.Groups {
				if group.ID != selected.Provider {
					continue
				}
				for _, model := range group.Models {
					if model.ID != selected.Model || len(model.Efforts) == 0 {
						continue
					}
					label := storelocale.T(storelocale.KeyDesktopPetReasoning) + quickChatName(selected.ReasoningEffort, storelocale.T(storelocale.KeyDesktopPetDefault))
					ui.MenuButton(c, label, func(menu *ui.Menu) {
						if menu.Item(storelocale.T(storelocale.KeyDesktopPetDefault)).Checked(selected.ReasoningEffort == "").Disabled(blocked).Chosen() {
							q.choose(quickChatModel{Provider: selected.Provider, Model: selected.Model})
						}
						for _, effort := range model.Efforts {
							if menu.Item(fmt.Sprintf("%s [%s]", quickChatName(effort.Name, effort.ID), effort.ID)).Checked(selected.ReasoningEffort == effort.ID).Disabled(blocked).Chosen() {
								q.choose(quickChatModel{Provider: selected.Provider, Model: selected.Model, ReasoningEffort: effort.ID})
							}
						}
					}).Label(storelocale.T(storelocale.KeyDesktopPetSelectReasoningEffort)).Disabled(blocked)
				}
			}
		}
	})
	if q.catalogNote != "" {
		ui.Text(c, q.catalogNote).FontSize(fonttheme.InterfaceSize(10)).TextColor(c.Theme().TextMuted)
	}
}
