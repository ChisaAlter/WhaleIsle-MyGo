package chat

import (
	"context"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func prepareScheduleContext(m Model) map[float64][]schedstore.Entry {
	cards := session.ScheduleCards(m.Snapshot.Messages, m.Snapshot.SelectedID)
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
	if len(rows) == 0 {
		return
	}
	snap := schedstore.ViewState{}
	if m.ScheduleStore != nil {
		snap = m.ScheduleStore.Snapshot()
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
		ui.Column(c).Padding(8, 12).Gap(4).Border(1, c.Theme().Border).Radius(6).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatScheduleTask) + " · "+row.Title).Bold()
			ui.Text(c, row.Prompt).FontSize(12)
			ui.Text(c, schedstore.TimingLabel(row)).FontSize(11)
			if !found {
				label := i18n.T(i18n.KeyChatScheduleReceipt) + " · " + row.ID
				if deleted {
					label = i18n.T(i18n.KeyChatScheduleDeleted)
				}
				ui.Text(c, label).FontSize(11)
			}
			if ui.Button(c, i18n.T(i18n.KeyChatScheduleDetails)).Disabled(deleted || !snap.ClientReady || m.OnOpenScheduleTask == nil).Clicked() && m.OnOpenScheduleTask != nil {
				m.OnOpenScheduleTask(row)
			}
		})
	}
}
