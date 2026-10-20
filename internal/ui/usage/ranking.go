package usage

import (
	"context"
	"fmt"
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
	models := st.Overview.AllTime.ByModel
	scope := i18n.T(i18n.KeySurfacesUsageAllTime)
	if st.Range.Active() {
		models = st.Selection().Models
		scope = fmt.Sprintf("%s → %s", st.Range.Start, st.Range.End)
	}
	usageCard(c, "usage-model-ranking", i18n.T(i18n.KeySurfacesUsageModelRank), fg, func() {
		ui.Text(c, scope).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		limit := min(len(models), m.Store.ModelRankLimit())
		if limit == 0 {
			ui.Text(c, i18n.T(i18n.KeySurfacesUsageEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		maximum := 1.0
		for _, row := range models[:limit] {
			maximum = max(maximum, row.Total)
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			for index, row := range models[:limit] {
				entry := row
				ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprint(index+1)).Width(18).Shrink(0).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).TextAlign(ui.End)
						ui.Text(c, entry.Model).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).SingleLine().Ellipsis("…").Tooltip(entry.Provider + " / " + entry.Model)
						ui.Text(c, formatCompactTokens(entry.Total)).FontSize(fonttheme.InterfaceSize(11)).FontFeatures("tnum").TextColor(muted)
					})
					providerUsageBar(c, entry.Total, maximum, accent, c.Theme().SurfaceHover)
				})
			}
		})
		if limit < len(models) {
			if ui.Button(c, i18n.T(i18n.KeySurfacesUsageShowMore)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				m.Store.ShowMoreModels()
			}
		}
	})
	if st.Range.Active() {
		return
	}
	rank := m.Store.SessionRanking()
	billing := m.Store.Billing()
	usageCard(c, "usage-session-ranking", i18n.T(i18n.KeySurfacesUsageRanking), fg, func() {
		ui.Row(c).Gap(4).Children(func() {
			for _, sort := range []string{"tokens", "cost"} {
				key := sort
				label := usageText("按 Token", "By tokens")
				if key == "cost" {
					label = usageText("按费用", "By cost")
				}
				usageScopeChip(c, m, label, rank.Sort == key, st.Loading || rank.Loading, func() {
					m.Store.ResetSessionRanking(context.Background(), key)
				})
			}
			if rank.Sort == "cost" && !rank.Started && !rank.Loading && rank.Err == "" && !st.Loading {
				m.Store.LoadMoreSessions(context.Background())
			}
		})
		if rank.Loading {
			ui.Text(c, i18n.T(i18n.KeySurfacesUsageLoading)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		rows := st.Overview.TopSessions
		if rank.Started || rank.Sort == "cost" {
			rows = rank.Rows
		}
		if len(rows) == 0 {
			ui.Text(c, i18n.T(i18n.KeySurfacesUsageEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		maximum := 1.0
		for _, row := range rows {
			maximum = max(maximum, row.Totals.Total)
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			for index, row := range rows {
				entry := row
				name := entry.Title
				if name == "" {
					name = entry.ID
				}
				cost := usageText("费用未完整计价", "Cost not fully priced")
				if cents, ok := usagestore.SessionEstimate(entry, billing); ok {
					cost = fmt.Sprintf("≈ ¥%.2f", cents/100)
				}
				meta := usageText("主会话", "Main session")
				if entry.Depth > 0 {
					meta = usageText("子代理会话", "Subagent session")
				}
				if entry.LastActive > 0 {
					meta += " · " + time.UnixMilli(entry.LastActive).UTC().Format("2006-01-02 15:04 UTC")
				}
				ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprint(index+1)).Width(18).Shrink(0).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).TextAlign(ui.End)
						ui.Text(c, name).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).SingleLine().Ellipsis("…").Tooltip(name)
						ui.Text(c, formatCompactTokens(entry.Totals.Total)).FontSize(fonttheme.InterfaceSize(11)).FontFeatures("tnum").TextColor(muted)
						if m.OnOpenSession != nil && ui.Button(c, usageText("打开", "Open")).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
							m.OnOpenSession(entry.ID)
						}
					})
					ui.Text(c, meta+" · "+cost).FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).SingleLine().Ellipsis("…")
					providerUsageBar(c, entry.Totals.Total, maximum, accent, c.Theme().SurfaceHover)
				})
			}
		})
		if rank.Err != "" {
			ui.Text(c, usageText("加载排行失败：", "Could not load ranking: ")+rank.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			return
		}
		if rank.Started && !rank.HasMore {
			return
		}
		label := usageText("显示更多会话", "Show more sessions")
		if rank.Err != "" {
			label = usageText("重试加载排行", "Retry ranking")
		}
		if !st.Loading && ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			m.Store.LoadMoreSessions(context.Background())
		}
	})
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
