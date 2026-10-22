package chat

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func historyNavigationText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

func historyTurnCaption(turn session.HistoryTurn) string {
	caption := fmt.Sprintf(historyNavigationText("回合 %.0f", "Turn %.0f"), turn.Turn)
	if turn.Prompt != "" {
		caption += " · " + turn.Prompt
	}
	return caption
}

func prepareHistoryNavigation(m Model, msgs []session.Message, list *ui.ListState) {
	if m.HistoryStore == nil {
		return
	}
	// Stable identities let the native virtual list retain its visible row when
	// older records are prepended. A new Host/session cannot reuse old row state.
	list.Key = func(i int) any {
		if i < 0 || i >= len(msgs) {
			return i
		}
		msg := msgs[i]
		return struct {
			Scope, Session, ID, Kind, Event string
			Seq                             float64
		}{m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, msg.ID, string(msg.Kind), msg.EventType, msg.Seq}
	}
	v := m.Snapshot.HistoryNavigation
	if !v.Ready {
		return
	}
	for _, turn := range v.Turns {
		if turn.Seq != v.TargetSeq {
			continue
		}
		row := session.HistoryTurnMessageIndex(msgs, turn)
		if m.HistoryStore.ConsumeHistoryTurnLanding(m.Snapshot.SelectedID, v.Request) && row >= 0 {
			list.ScrollTo(row, ui.Start)
		}
		return
	}
	// A concurrent outline replacement may remove the target; never keep a latent
	// scroll that can revive when a later unrelated projection happens to match.
	m.HistoryStore.ConsumeHistoryTurnLanding(m.Snapshot.SelectedID, v.Request)
}

// HistoryNavigationView is the native equivalent of the original turn rail:
// a keyboard-accessible directory with Host previews and explicit jump/cancel.
func HistoryNavigationView(c *ui.Context, m Model, list *ui.ListState, muted ui.Color) {	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if m.HistoryStore == nil {
		return
	}
	v := m.Snapshot.HistoryNavigation
	// The reference keeps turn navigation out of the transcript header. Show
	// the native directory only while a jump is pending, running or failed.
	if len(v.Turns) == 0 || (!v.HasTarget && !v.Busy && v.Err == "") {
		return
	}
	ui.Column(c).FillWidth().Gap(4).Padding(4, 12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			if selected, changed := historyTurnPicker(c, v, m.Snapshot.ClientReady && (!m.Snapshot.HistoryPageBusy || v.Busy)); changed {
				m.HistoryStore.JumpToHistoryTurn(context.Background(), m.Snapshot.SelectedID, v.Request, selected)
			}
			if v.Busy {
				ui.Text(c, historyNavigationText("正在加载回合…", "Loading turn…")).FontSize(11).TextColor(muted)
				if stdButton(c, palette, historyNavigationText("取消跳转", "Cancel jump"), buttonGhost, false).Clicked() {
					m.HistoryStore.CancelHistoryTurnJump(m.Snapshot.SelectedID, v.Request)
				}
			}
			if v.Err != "" && v.HasTarget {
				if stdButton(c, palette, historyNavigationText("重试跳转", "Retry jump"), buttonGhost, false).Clicked() {
					m.HistoryStore.JumpToHistoryTurn(context.Background(), m.Snapshot.SelectedID, v.Request, v.TargetSeq)
				}
			}
			if stdButton(c, palette, historyNavigationText("回到最新", "Latest"), buttonGhost, false).Clicked() {
				if m.HistoryStore.CancelHistoryTurnJump(m.Snapshot.SelectedID, v.Request) {
					list.ScrollToEnd()
				}
			}
		})
		if v.Err != "" {
			ui.Text(c, v.Err).FontSize(11).TextColor(muted)
		}
	})
}

func historyTurnPicker(c *ui.Context, v session.HistoryNavigationView, enabled bool) (float64, bool) {
	caption := historyNavigationText("回合目录", "Turn directory")
	for _, turn := range v.Turns {
		if v.HasTarget && turn.Seq == v.TargetSeq {
			caption = historyTurnCaption(turn)
			break
		}
	}
	// No selected value is retained in the widget: clicking the same turn is an
	// explicit new jump even after the reader has manually scrolled elsewhere.
	selected := float64(-1)
	selectParts := ui.SelectBase(c, &selected)
	trigger := selectParts.Trigger.Label(historyNavigationText("回合目录", "Turn directory")).Tooltip(caption).Grow(1).MinWidth(0).Padding(5, 8).Border(1, c.Theme().Border).Radius(c.Theme().Radius)
	trigger.Disabled(!enabled).Children(func() {
		ui.Text(c, caption+" ▾").SingleLine().MinWidth(0).Grow(1).FontSize(12)
	})
	selectParts.Popup(func(panel ui.Element) {
		panel.Padding(4).MaxWidth(440).MaxWidthPercent(100).Background(c.Theme().Background).Border(1, c.Theme().Border).Radius(c.Theme().Radius)
		ui.Scroll(c).MaxHeight(260).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for _, turn := range v.Turns {
					item := selectParts.Item(turn.Seq).Label(historyTurnCaption(turn)).Tooltip(turn.Response).Padding(6, 8).Radius(c.Theme().Radius)
					if item.Highlighted() {
						item.Background(c.Theme().Accent).TextColor(c.Theme().AccentText)
					}
					item.Children(func() {
						ui.Column(c).Gap(2).Children(func() {
							ui.Text(c, historyTurnCaption(turn)).FontSize(12).SingleLine()
							if turn.Response != "" {
								ui.Text(c, turn.Response).FontSize(11).SingleLine()
							}
						})
					})
				}
			})
		})
	})

	return selected, trigger.Changed() && selected >= 0
}
