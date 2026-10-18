package chat

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
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
func ScheduleHeaderView(c *ui.Context, m Model) {
	if m.ScheduleStore == nil || m.Snapshot.SelectedID == "" {
		return
	}
	snap := m.ScheduleStore.Snapshot()
	rows := schedstore.ActiveForSession(snap, m.Snapshot.SelectedID)
	ui.Row(c).Gap(6).Padding(4, 12).Children(func() {
		ui.Text(c, fmt.Sprintf("会话任务 / Tasks (%d)", len(rows))).FontSize(11)
		for _, row := range rows {
			if ui.Button(c, row.Title).Tooltip(schedstore.TimingLabel(row)).FontSize(11).Clicked() && m.OnOpenScheduleTask != nil {
				m.OnOpenScheduleTask(row)
			}
		}
		if snap.Err != "" {
			ui.Text(c, snap.Err).FontSize(11)
		}
		if ui.Button(c, "刷新任务 / Refresh tasks").Disabled(snap.Loading).FontSize(11).Clicked() {
			m.ScheduleStore.Refresh(context.Background(), m.Snapshot.SelectedID)
		}
	})
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
		row := receipt
		found := false
		for _, live := range snap.Rows {
			if live.ID == row.ID && live.SessionID == m.Snapshot.SelectedID {
				row = live
				found = true
				break
			}
		}
		ui.Column(c).Padding(8, 12).Gap(4).Border(1, c.Theme().Border).Radius(6).Children(func() {
			ui.Text(c, "定时任务 / Scheduled task · "+row.Title).Bold()
			ui.Text(c, row.Prompt).FontSize(12)
			ui.Text(c, schedstore.TimingLabel(row)).FontSize(11)
			if !found {
				ui.Text(c, "创建时记录 / Creation receipt · "+row.ID).FontSize(11)
			}
			if ui.Button(c, "任务详情 / Task details").Disabled(!found || snap.Err != "" || !snap.ClientReady).Clicked() && m.OnOpenScheduleTask != nil {
				m.OnOpenScheduleTask(row)
			}
		})
	}
}
