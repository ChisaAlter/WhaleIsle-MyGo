package usage

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

// rankingView paints the session ranking card and its full-list modal.
func rankingView(c *ui.Context, m Model, st usagestore.StatsView) {
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
	usageCard(c, "usage-session-ranking", i18n.T(i18n.KeySurfacesUsageRanking), func() {
		rankingSort(c, sortBy, st.Loading, func() {})
		sessionRankRows(c, rows, billing)
		if len(rows) > 0 && coss.Button(c, usageText("更多", "More"), coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonSM,
		}).AlignSelf(ui.Start).Clicked() {
			*open = true
			m.Store.ResetSessionRanking(context.Background(), "tokens")
		}
	})
	rank := m.Store.SessionRanking()
	rankDialog(c, open, i18n.T(i18n.KeySurfacesUsageRanking), func() {
		value := rank.Sort
		rankingSort(c, &value, rank.Loading, func() { m.Store.ResetSessionRanking(context.Background(), value) })
		sessionRankRows(c, rank.Rows, billing)
		rankMore(c, rank.Loading, rank.Started, rank.HasMore, rank.Err.SentenceText(), func() { m.Store.LoadMoreSessions(context.Background()) })
	})
}

func sessionRankRows(c *ui.Context, rows []usagestore.SessionUsage, billing usagestore.BillingView) {
	t := design.For(c)
	rankTableHeader(c, true)
	for i, entry := range rows {
		if i > 0 {
			coss.Separator(c, coss.Horizontal)
		}
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
		ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).PaddingY(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
			pagekit.Caption(c, fmt.Sprint(i+1)).Width(22).FontFeatures("tnum")
			ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(entry.ID).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.Foreground)
			ui.Row(c).Width(56).Shrink(0).Children(func() {
				coss.Badge(c, kind, coss.BadgeOptions{Variant: coss.BadgeSecondary, Size: coss.BadgeSM})
			})
			pagekit.Caption(c, date).Width(78).FontFeatures("tnum")
			pagekit.Caption(c, price).Width(62).TextAlign(ui.End).FontFeatures("tnum")
			ui.Text(c, formatCompactTokens(entry.Totals.Total)).Width(62).TextAlign(ui.End).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).FontFeatures("tnum").TextColor(t.Foreground)
		})
	}
	if len(rows) == 0 {
		pagekit.Muted(c, usageText("暂无会话用量。", "No session usage.")).FillWidth()
	}
}
