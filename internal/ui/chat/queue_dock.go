package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

var queueIconGrip = ui.MustParseSVG([]byte(`<svg viewBox="0 0 12 12" fill="none"><circle cx="4" cy="3" r="1" fill="currentColor"/><circle cx="8" cy="3" r="1" fill="currentColor"/><circle cx="4" cy="6" r="1" fill="currentColor"/><circle cx="8" cy="6" r="1" fill="currentColor"/><circle cx="4" cy="9" r="1" fill="currentColor"/><circle cx="8" cy="9" r="1" fill="currentColor"/></svg>`))

// QueueDockView paints the compact queue dock that belongs above the composer:
// one row per queued prompt with its preview and icon-only actions, plus the
// inline queue editor. The expanded plan board renders the same rows from
// paintQueueRows, so the two surfaces never drift.
//
// The frame owns the seat; it only needs to call this when the dock has rows:
//
//	if snap.PlanBoard.HasQueue() {
//		chat.QueueDockView(c, chat.PlanBoardModel{...})
//	}
func QueueDockView(c *ui.Context, m PlanBoardModel) {
	if !queueDockHasRows(m.Board) && m.QueueEditor.ItemID == "" {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)
	root := ui.Column(c).Key("queue-dock").FillWidth().MinWidth(0).Gap(6).
		Background(field).Border(0.5, border).Radius(12).Padding(8, 10, 8, 10)
	collapsed := ui.Local(root, "collapsed", func() bool {
		if m.Open != nil {
			return !*m.Open
		}
		return true
	})
	root.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatQueueTitle)).FontSize(11).TextColor(muted).Grow(1)
			if n := len(m.Board.Queue) + len(m.Board.Pending); n > 0 {
				ui.Text(c, queueCountLabel(n)).FontSize(11).TextColor(muted)
			}
			if len(m.Board.Queue)+len(m.Board.Pending) > 1 {
				label := "▴"
				if *collapsed {
					label = "▾"
				}
				if ui.Button(c, label).Label(trajectoryText("展开或收起队列", "Expand or collapse queue")).Disabled(m.QueueEditor.ItemID != "").Clicked() {
					*collapsed = !*collapsed
				}
			}
		})
		if len(m.Board.Queue)+len(m.Board.Pending) > 1 && *collapsed && m.QueueEditor.ItemID == "" {
			return
		}
		if m.QueueEditor.ItemID != "" && !queueHasItemID(m.Board.Queue, m.QueueEditor.ItemID) {
			paintQueueEditor(c, m, fg, muted)
		}
		paintQueueRows(c, m, fg, muted, accent, border, true)
	})
}

func queueHasItemID(items []session.PlanQueueItem, id string) bool {
	if id == "" {
		return false
	}
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

// queueDockHasRows reports whether the dock has anything worth mounting.
func queueDockHasRows(board session.PlanBoard) bool {
	return len(board.Queue) > 0 || len(board.Pending) > 0
}

func queueCountLabel(n int) string {
	if i18n.Locale() == i18n.LocaleZH {
		return queueCountNumber(n) + " 条排队消息"
	}
	return queueCountNumber(n) + " queued messages"
}

func queueSendingCopy() string {
	return referenceCopy("Sending…", "发送中…")
}

func queueReorderLabel(n, total int) string {
	return i18n.ChatParityf(i18n.KeyParityConvQueueReorder, "n", queueCountNumber(n), "total", queueCountNumber(total))
}

func queueMoveCopy() string {
	return i18n.T(i18n.KeyParityConvQueueReorderHint)
}

func queueMoveFailedCopy() string {
	return i18n.T(i18n.KeyParityConvQueueMoveFailed)
}

func queueEditCopy() string {
	return referenceCopy("Edit queued message", "编辑排队消息")
}

func queueEditUnsupportedCopy() string {
	return referenceCopy("Contains non-text content; editing is not supported yet", "包含非文本内容，暂不支持编辑")
}

func queueSaveEditCopy() string {
	return referenceCopy("Save queued message", "保存排队消息")
}

func queueCancelEditCopy() string {
	return referenceCopy("Cancel editing", "取消编辑")
}

func queueRemoveCopy() string {
	return referenceCopy("Remove queued message", "删除排队消息")
}

func queueSteerCopy() string {
	return referenceCopy("Steer queued message", "插话发送")
}

func queueSteerUnavailableCopy() string {
	return referenceCopy("Steering is available only while the agent is running", "仅运行中可插话发送")
}

func sessionInUseNotice() string {
	return referenceCopy(
		"This session is already in use, possibly by another running DSH instance (such as dsh web or the desktop app). Quit other running DSH instances and try again.",
		"当前会话已被占用，可能是其他正在运行的 DSH 导致的（如其他 dsh web、桌面端），请退出其他正在运行的 DSH 后重试。",
	)
}

func parentOfflineNotice() string {
	return referenceCopy(
		"Parent session offline; sending is unavailable but you can still stop the run",
		"父会话已离线，无法继续发送；仍可停止当前运行",
	)
}

func readOnlyChildNotice() string {
	return referenceCopy(
		"Read-only child session; messages cannot be sent",
		"只读子会话，无法发送消息",
	)
}

func normalizeConversationNotice(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "session_in_use") || strings.Contains(lower, "already in use") || strings.Contains(trimmed, "当前会话已被占用") {
		return sessionInUseNotice()
	}
	if lower == "parent_offline" || lower == "parentoffline" || strings.Contains(lower, "parent session offline") || strings.Contains(trimmed, "父会话已离线") {
		return parentOfflineNotice()
	}
	if lower == "read_only" || lower == "readonly" || strings.Contains(lower, "read-only child session") || strings.Contains(trimmed, "只读子会话") {
		return readOnlyChildNotice()
	}
	return trimmed
}

func conversationSessionNotice(snap session.ViewState) string {
	if notice := normalizeConversationNotice(snap.HistoryError); notice != "" && notice != strings.TrimSpace(snap.HistoryError) {
		return notice
	}
	if notice := normalizeConversationNotice(snap.LastAction); notice != "" && notice != strings.TrimSpace(snap.LastAction) {
		return notice
	}
	if snap.SelectedID == "" {
		return ""
	}
	var selected *session.Item
	for i := range snap.Items {
		if snap.Items[i].SessionID == snap.SelectedID {
			selected = &snap.Items[i]
			break
		}
	}
	if selected == nil {
		return ""
	}
	if selected.SubagentIdentity != nil && selected.SubagentIdentity.Mode == "one-shot" {
		return readOnlyChildNotice()
	}
	if selected.ParentID != "" {
		parentOnline := false
		parentFound := false
		for i := range snap.Items {
			if snap.Items[i].SessionID == selected.ParentID {
				parentFound = true
				parentOnline = snap.Items[i].AgentAvailable || snap.Items[i].Running
				break
			}
		}
		if (parentFound && !parentOnline) || (!parentFound && !selected.AgentAvailable && selected.ContactError != "") {
			return parentOfflineNotice()
		}
	}
	return ""
}

func queueItemEditUnsupported(item session.PlanQueueItem) bool {
	return len(item.Attachments) > 0 || (item.ContentKnown && !item.TextOnly) || strings.HasPrefix(strings.TrimSpace(item.Preview), "[image]")
}

type queueInlineEditState struct {
	editing bool
	draft   string
}

func queueCountNumber(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 6)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// paintQueueRows is the shared row list: pending submissions first, then queued
// prompts. Compact rows keep the preview on one line and put the actions in
// icon seats; the expanded board keeps the same order with more room.
func paintQueueRows(c *ui.Context, m PlanBoardModel, fg, muted, accent, border ui.Color, compact bool) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	rowsOwner := ui.Box(c.Key("queue-rows-owner")).Size(0, 0).PassThrough()
	moveFailed := ui.Local(rowsOwner, "queue-move-failed", func() bool { return false })
	dispatchMove := func(itemID string, toIndex int) {
		*moveFailed = false
		if m.OnQueueMove != nil {
			m.OnQueueMove(itemID, toIndex)
		}
		if m.OnQueueMoveResult != nil && !m.OnQueueMoveResult(itemID, toIndex) {
			*moveFailed = true
		}
	}
	errNotice := strings.TrimSpace(m.QueueMoveError)
	if errNotice == "" && (m.QueueMoveFailed || *moveFailed) {
		errNotice = queueMoveFailedCopy()
	}
	if errNotice != "" {
		ui.Text(c, errNotice).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(palette.Danger))
	}
	for _, pending := range m.Board.Pending {
		paintPendingSubmission(c, pending, fg, muted)
	}
	for index, q := range m.Board.Queue {
		index := index
		item := q
		itemBox := ui.Column(c).Key("queue-item:" + item.ID).FillWidth().MinWidth(0).Gap(4)
		localEdit := ui.Local(itemBox, "inline-edit", func() queueInlineEditState {
			return queueInlineEditState{draft: item.Preview}
		})
		storeEditing := m.QueueEditor.ItemID != "" && m.QueueEditor.ItemID == item.ID
		unsupported := queueItemEditUnsupported(item)
		can := m.Board.QueueActions
		moveEnabled := can.Can(session.QueueActionMove, item) && !storeEditing && !localEdit.editing
		editEnabled := can.Can(session.QueueActionEdit, item) && !unsupported
		steerEnabled := can.Can(session.QueueActionSteer, item)
		removeEnabled := can.Can(session.QueueActionRemove, item)

		itemBox.Children(func() {
			row := ui.Row(c).Key("queue-row:"+item.ID).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).
				Padding(4, 6, 4, 6).Radius(8)
			if moveEnabled && (m.OnQueueMove != nil || m.OnQueueMoveResult != nil) {
				queueDropTarget(row, m, item.ID, accent, dispatchMove)
			}
			row.Children(func() {
				if (m.OnQueueMove != nil || m.OnQueueMoveResult != nil) && len(m.Board.Queue) > 1 && item.ID != "" {
					reorderLabel := queueReorderLabel(index+1, len(m.Board.Queue))
					moveTip := queueMoveCopy()
					grip := ui.Button(c, "").Key("queue-grip:"+item.ID).Label(reorderLabel).Tooltip(moveTip).Description(moveTip).Width(20).Height(24).Radius(6).
						Padding(0, 0, 0, 0).Background(ui.Transparent).BorderWidth(0)
					grip.Children(func() {
						ui.Box(c).Fill().PassThrough().Label(moveTip)
						ui.Icon(c, queueIconGrip).Width(12).Height(12).TextColor(muted)
					})
					if !moveEnabled {
						grip.Disabled(true).Opacity(.35)
					} else {
						grip.Drag(queueDrag{Scope: can.Scope, ID: item.ID})
						ui.DragEnd[queueDrag](grip)
						grip.HandleInput(func(ev ui.InputEvent) bool {
							if ev.Kind != ui.InputKeyDown || ev.Mods != 0 {
								return false
							}
							switch ev.Key {
							case ui.KeyUp:
								if index > 0 {
									dispatchMove(item.ID, index-1)
								}
								return true
							case ui.KeyDown:
								if index+1 < len(m.Board.Queue) {
									dispatchMove(item.ID, index+1)
								}
								return true
							}
							return false
						})
						if grip.Hovered() {
							grip.Background(muted.Alpha(.12))
						}
					}
				}
				ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
					paintQueueAttachments(c, m, item, fg, muted)
					line := item.Preview
					if item.Source != "" && !compact {
						line = item.Source + ": " + line
					}
					if strings.TrimSpace(line) == "" {
						line = i18n.T(i18n.KeyChatQueueEmptyPreview)
					}
					ui.Text(c, maskedUserText(line)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(fg).SingleLine().MinWidth(0)
					if unsupported {
						ui.Text(c, queueEditUnsupportedCopy()).FontSize(11).TextColor(muted)
					}
				})
				action := func(icon *ui.SVG, label, tip string, enabled bool, tone ui.Color, run func()) {
					if run == nil {
						return
					}
					if tip == "" {
						tip = label
					}
					button := ui.Button(c, "").Label(label).Tooltip(tip).Description(tip).Width(24).Height(24).Radius(6).
						Padding(0, 0, 0, 0).Background(ui.Transparent).BorderWidth(0)
					button.Children(func() { ui.Icon(c, icon).Width(13).Height(13).TextColor(tone) })
					if !enabled {
						button.Disabled(true).Opacity(.35)
						return
					}
					if button.Hovered() {
						button.Background(tone.Alpha(.12))
					}
					if button.Clicked() {
						run()
					}
				}
				if (m.OnQueueEdit != nil || m.OnSaveQueueEdit != nil) && item.ID != "" {
					editLabel := i18n.T(i18n.KeyPlanBoardQueueEdit)
					editTip := editLabel
					if unsupported {
						editTip = queueEditUnsupportedCopy()
					}
					action(messageIconEditOutline, editLabel, editTip, editEnabled, muted, func() {
						localEdit.editing = true
						localEdit.draft = item.Preview
						if m.OnQueueEdit != nil {
							m.OnQueueEdit(item.ID, item.Preview)
						}
					})
				}
				if m.OnQueueSteer != nil && item.ID != "" {
					steerLabel := i18n.T(i18n.KeyPlanBoardQueueSteer)
					steerTip := steerLabel
					if !steerEnabled && !can.Running {
						steerTip = queueSteerUnavailableCopy()
					}
					action(queueIconSteer, steerLabel, steerTip, steerEnabled, accent, func() {
						m.OnQueueSteer(item.ID)
					})
				}
				if m.OnQueueRemove != nil && item.ID != "" {
					removeLabel := i18n.T(i18n.KeyPlanBoardQueueRemove)
					action(messageIconCloseOutline, removeLabel, removeLabel, removeEnabled, muted, func() {
						m.OnQueueRemove(item.ID)
					})
				}
			})
			if storeEditing || (localEdit.editing && !unsupported) {
				draft := localEdit.draft
				saving := false
				ready := true
				errText := ""
				if storeEditing {
					draft = m.QueueEditor.Text
					saving = m.QueueEditor.Saving
					ready = m.QueueEditor.Ready
					errText = m.QueueEditor.Error
				}
				ui.Column(c).Key("queue-inline-editor:"+item.ID).FillWidth().MinWidth(0).Gap(6).Padding(4, 6, 4, 6).Children(func() {
					if ready {
						ui.TextArea(c.Key("queue-edit-"+item.ID), &draft).Label(queueEditCopy()).FillWidth().Lines(2, 6).ReadOnly(saving).OnChange(func() {
							localEdit.draft = draft
							if storeEditing && m.OnChangeQueueEdit != nil {
								m.OnChangeQueueEdit(m.QueueEditor.Token, draft)
							}
						})
					}
					if errText != "" {
						ui.Text(c, errText).FontSize(theme.InterfaceSize(11)).TextColor(fg)
					}
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						saveLabel := queueSaveEditCopy()
						if ui.Button(c, saveLabel).Label(saveLabel).Tooltip(saveLabel).Disabled(!ready || saving || strings.TrimSpace(draft) == "").Clicked() {
							if storeEditing && m.OnSaveQueueEdit != nil {
								m.OnSaveQueueEdit(m.QueueEditor.Token)
							} else if m.OnQueueEdit != nil {
								m.OnQueueEdit(item.ID, strings.TrimSpace(draft))
							}
							localEdit.editing = false
						}
						cancelLabel := queueCancelEditCopy()
						if stdButton(c, palette, cancelLabel, buttonGhost, saving).Label(cancelLabel).Tooltip(cancelLabel).Clicked() {
							if storeEditing && m.OnCancelQueueEdit != nil {
								m.OnCancelQueueEdit(m.QueueEditor.Token)
							}
							localEdit.editing = false
							localEdit.draft = item.Preview
						}
					})
				})
			}
		})
	}
}
