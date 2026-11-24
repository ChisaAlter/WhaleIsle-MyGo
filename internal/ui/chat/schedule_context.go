package chat

import (
	"context"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func prepareScheduleContext(m Model, memo *session.HistoryDerivedMemo) map[float64][]schedstore.Entry {
	cards := memo.ScheduleCards(m.Snapshot)
	var ids []string
	for _, rows := range cards {
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
	}
	if m.ScheduleStore != nil {
		m.ScheduleStore.EnsureContext(context.Background(), m.Snapshot.SelectedID, ids)
	}
	return cards
}
func ScheduleTurnCardsView(c *ui.Context, m Model, rows []schedstore.Entry) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if len(rows) == 0 {
		return
	}
	snap := schedstore.ViewState{}
	if m.ScheduleStore != nil {
		snap = m.ScheduleStore.Snapshot().Value
	}
	for _, receipt := range rows {
		receipt.Origin = snap.Origin
		row := receipt
		found := false
		for _, live := range snap.Rows {
			if live.ID == row.ID && live.SessionID == m.Snapshot.SelectedID {
				row = live
				found = true
				break
			}
		}
		settled := false
		if m.ScheduleStore != nil {
			_, settled = m.ScheduleStore.ReceiptCatalogState(receipt)
		}
		deleted := settled && !found
		chatCardSurface(c, ui.Column(c).Padding(design.Sp(2), design.Sp(3)).Gap(design.Sp(1)), design.RadiusSM).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatScheduleTask)+" · "+row.Title).Bold()
			ui.Text(c, row.Prompt).FontSize(12)
			ui.Text(c, schedstore.TimingLabel(row)).FontSize(11)
			if !found {
				label := i18n.T(i18n.KeyChatScheduleReceipt) + " · " + row.ID
				if deleted {
					label = i18n.T(i18n.KeyChatScheduleDeleted)
				}
				ui.Text(c, label).FontSize(11)
			}
			disabled := deleted || !snap.ClientReady || m.OnOpenScheduleTask == nil
			if stdButton(c, palette, i18n.T(i18n.KeyChatScheduleDetails), buttonGhost, disabled).Clicked() && !disabled {
				m.OnOpenScheduleTask(row)
			}
		})
	}
}
