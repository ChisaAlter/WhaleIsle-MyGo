package usage

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

// rankingView paints the two ranking cards: models in the current scope, then
// the session ranking. Each row keeps the original list's knowledge — rank,
// name, share, tokens and cost — but reads as a ranked row with a bar instead
// of one sentence per line.
func rankingView(c *ui.Context, m Model, st usagestore.StatsView, fg, muted, accent ui.Color) {
	seat := ui.Column(c.Key("usage-session-ranking-state/" + m.Store.StatisticsOwner()))
	open := ui.Local(seat, "more", func() bool { return false })
	sortBy := ui.Local(seat, "sort", func() string { return "tokens" })
	rows := append([]usagestore.SessionUsage(nil), st.Overview.TopSessions...)
	billing := m.Store.Billing()
	if *sortBy == "cost" {
		sort.SliceStable(rows, func(i, j int) bool {
			a, _ := usagestore.SessionEstimate(rows[i], billing)
			b, _ := usagestore.SessionEstimate(rows[j], billing)
			return a > b
		})
	}
	usageCard(c, "usage-session-ranking", i18n.T(i18n.KeySurfacesUsageRanking), fg, func() {
		rankingSort(c, sortBy, st.Loading, func() {})
		sessionRankRows(c, rows, billing, fg, muted)
		if len(rows) > 0 && ui.Button(c, usageText("更多", "More")).AlignSelf(ui.Start).BorderWidth(0).Background(ui.Transparent).Clicked() {
			*open = true
			m.Store.ResetSessionRanking(context.Background(), "tokens")
		}
	})
	rank := m.Store.SessionRanking()
	rankDialog(c, open, i18n.T(i18n.KeySurfacesUsageRanking), func() {
		value := rank.Sort
		rankingSort(c, &value, rank.Loading, func() { m.Store.ResetSessionRanking(context.Background(), value) })
		sessionRankRows(c, rank.Rows, billing, fg, muted)
		rankMore(c, rank.Loading, rank.Started, rank.HasMore, rank.Err, func() { m.Store.LoadMoreSessions(context.Background()) }, muted)
	})
}
func sessionRankRows(c *ui.Context, rows []usagestore.SessionUsage, billing usagestore.BillingView, fg, muted ui.Color) {
	rankTableHeader(c, true, muted)
	for i, entry := range rows {
		name := entry.Title
		if name == "" {
			name = usageText("未命名会话", "Untitled session")
		}
		price := "—"
		if cents, ok := usagestore.SessionEstimate(entry, billing); ok {
			price = fmt.Sprintf("¥%.2f", cents/100)
		}
		kind := usageText("主会话", "Main")
		if entry.Depth > 0 {
			kind = usageText("子代理", "Subagent")
		}
		date := time.UnixMilli(entry.LastActive).UTC().Format("2006-01-02")
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(6, 0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprint(i+1)).Width(22).TextColor(muted)
			ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(entry.ID).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
			ui.Text(c, kind).Width(52).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, date).Width(78).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Text(c, price).Width(62).TextAlign(ui.End).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Text(c, formatCompactTokens(entry.Totals.Total)).Width(62).TextAlign(ui.End).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		})
	}
	if len(rows) == 0 {
		ui.Text(c, usageText("暂无会话用量。", "No session usage.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
}

// usageScopeChip paints one of the mutually exclusive scope choices as a chip
// instead of prefixing the label with a bullet.
func usageScopeChip(c *ui.Context, m Model, label string, selected, disabled bool, onClick func()) {
	chip := ui.Button(c, label).Height(26).Padding(0, 10).Radius(6).BorderWidth(0).FontSize(fonttheme.InterfaceSize(12)).Disabled(disabled).Selected(selected).Tooltip(label)
	if selected {
		chip.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	} else {
		chip.Background(ui.Transparent).TextColor(c.Theme().TextMuted)
	}
	if chip.Hovered() && !selected && !disabled {
		chip.Background(ui.RGBA(c.Theme().TextMuted.R, c.Theme().TextMuted.G, c.Theme().TextMuted.B, .14))
	}
	if chip.Clicked() && onClick != nil {
		onClick()
	}
}
