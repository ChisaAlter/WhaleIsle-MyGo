package chat

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func historyNavigationText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func historyTurnCaption(turn session.HistoryTurn) string {
	caption := fmt.Sprintf(historyNavigationText("回合 %.0f", "Turn %.0f"), turn.Turn)
	if turn.Prompt != "" {
		caption += " · " + turn.Prompt
	}
	return caption
}

func prepareHistoryNavigation(m Model, msgs []session.Message, list *ui.ListState) {
	// Keys belong to every transcript, including read-only/test surfaces without
	// a history Store. Navigation uses the same optional-prefix translation.
	prepareTranscriptList(m, msgs, list)
	prepareHistoryNavigationLanding(m, msgs, list)
}

func prepareHistoryNavigationLanding(m Model, msgs []session.Message, list *ui.ListState) {
	if m.HistoryStore == nil {
		return
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
			list.ScrollTo(transcriptMessageRow(m, row), ui.Start)
		}
		return
	}
	// A concurrent outline replacement may remove the target; never keep a latent
	// scroll that can revive when a later unrelated projection happens to match.
	m.HistoryStore.ConsumeHistoryTurnLanding(m.Snapshot.SelectedID, v.Request)
}

// HistoryNavigationView is the native equivalent of the original turn rail:
// a keyboard-accessible directory with Host previews and explicit jump/cancel.
func HistoryNavigationView(c *ui.Context, m Model, list *ui.ListState, _ ui.Color) {
	t := design.For(c)
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if m.HistoryStore == nil {
		return
	}
	v := m.Snapshot.HistoryNavigation
	// The reference keeps turn navigation out of the transcript header. Show
	// the native directory only while a jump is pending, running or failed.
	if len(v.Turns) == 0 || (!v.HasTarget && !v.Busy && v.Err == "") {
		return
	}
	ui.Column(c).FillWidth().Gap(design.Sp(1)).Padding(design.Sp(1), design.Sp(3)).Children(func() {
		ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			if selected, changed := historyTurnPicker(c, v, m.Snapshot.ClientReady && (!m.Snapshot.HistoryPageBusy || v.Busy)); changed {
				m.HistoryStore.JumpToHistoryTurn(context.Background(), m.Snapshot.SelectedID, v.Request, selected)
			}
			if v.Busy {
				ui.Text(c, historyNavigationText("正在加载回合…", "Loading turn…")).FontSize(11).TextColor(t.MutedForeground)
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
			ui.Text(c, v.Err).FontSize(11).TextColor(t.DestructiveForeground)
		}
	})
}

func historyTurnPicker(c *ui.Context, v session.HistoryNavigationView, enabled bool) (float64, bool) {
	t := design.For(c)
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
	trigger := selectParts.Trigger.Label(historyNavigationText("回合目录", "Turn directory")).Tooltip(caption).
		Grow(1).MinWidth(0).Height(28).Padding(0, design.Sp(2.5)-1).Border(design.BorderWidth, t.Input).
		Background(t.Background).Radius(design.RadiusLG).Cursor(ui.CursorPointer).FocusRing(false)
	if t.IsDark() {
		trigger.Background(t.Input.Alpha(0.32))
	}
	trigger.Disabled(!enabled).Children(func() {
		ui.Text(c, caption).SingleLine().MinWidth(0).Grow(1).FontSize(design.TextXS.Size).TextColor(t.Foreground)
		ui.Icon(c, coss.IconChevronDown.SVG()).Size(design.IconXS, design.IconXS).TextColor(t.MutedForeground).Shrink(0)
	})
	selectParts.Popup(func(panel ui.Element) {
		chatPopupSurface(c, panel, design.RadiusLG).Padding(design.Sp(1)).MaxWidth(440).MaxWidthPercent(100)
		coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
			ui.Column(c).Gap(design.Sp(0.5)).Children(func() {
				for _, turn := range v.Turns {
					item := selectParts.Item(turn.Seq).Label(historyTurnCaption(turn)).Tooltip(turn.Response).
						Padding(design.Sp(1.5), design.Sp(2)).Radius(design.RadiusSM).Cursor(ui.CursorPointer)
					fg := t.PopoverForeground
					if item.Highlighted() {
						item.Background(t.Accent)
						fg = t.AccentForeground
					}
					item.Children(func() {
						ui.Column(c).Gap(design.Sp(0.5)).Children(func() {
							ui.Text(c, historyTurnCaption(turn)).FontSize(design.TextXS.Size).TextColor(fg).SingleLine()
							if turn.Response != "" {
								ui.Text(c, turn.Response).FontSize(11).TextColor(t.MutedForeground).SingleLine()
							}
						})
					})
				}
			})
		}).MaxHeight(260)
	})

	return selected, trigger.Changed() && selected >= 0
}
