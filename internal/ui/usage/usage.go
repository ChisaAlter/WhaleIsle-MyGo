// Package usage is the M9 native usage panel stub (account RPC + job list).
package usage

import (
	"context"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func jobKillable(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "active", "pending", "queued", "starting":
		return true
	default:
		return false
	}
}

// Model wires the usage panel to store state.
type Model struct {
	Store          *usagestore.Store
	SessionID      string
	ThemeID        string
	SystemDark     bool
	OnClose        func()
	OnRefresh      func()
	OnOpenUsageWeb func()
	OnOpenTopUp    func()
	OnCopyUsageURL func(url string)
	OnCopyTopUpURL func(url string)
	OnCopyBalance  func(text string)
}

// View paints the usage stub panel.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)
	snap := m.Store.Snapshot()

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyUsageTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
			if snap.Loading {
				ui.Text(c, i18n.T(i18n.KeyUsageLoading)).FontSize(12).TextColor(muted)
			}
			if ui.Button(c, i18n.T(i18n.KeyUsageRefresh)).Clicked() {
				if m.OnRefresh != nil {
					m.OnRefresh()
				} else {
					m.Store.Refresh(context.Background(), m.SessionID)
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyUsageBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Column(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Gap(10).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyUsagePluginNote)).FontSize(12).TextColor(muted)

			if snap.Err != "" {
				ui.Text(c, snap.Err).FontSize(12).TextColor(accent)
			}

			ui.Text(c, i18n.T(i18n.KeyUsageAccount)).FontSize(13).Bold().TextColor(fg)
			if snap.AccountStatus == "" && !snap.Loading {
				ui.Text(c, i18n.T(i18n.KeyUsageNoData)).FontSize(12).TextColor(muted)
			} else {
				ui.Text(c, snap.AccountStatus).FontSize(12).TextColor(fg)
			}
			if snap.UsageURL != "" {
				ui.Text(c, i18n.T(i18n.KeyUsageWebLink)+": "+snap.UsageURL).FontSize(11).TextColor(muted)
			}
			if snap.ExpiryNote != "" {
				ui.Text(c, i18n.T(i18n.KeyUsageExpiry)+": "+snap.ExpiryNote).FontSize(11).TextColor(muted)
			}

			ui.Text(c, i18n.T(i18n.KeyUsageBalance)).FontSize(13).Bold().TextColor(fg).Padding(8, 0, 0, 0)
			if snap.BalanceSummary != "" {
				ui.Text(c, snap.BalanceSummary).FontSize(11).TextColor(fg)
			}

			if len(snap.DayBuckets) > 0 {
				ui.Text(c, i18n.T(i18n.KeyUsageHeatmapTitle)).FontSize(13).Bold().TextColor(fg).Padding(8, 0, 0, 0)
				ui.Text(c, i18n.T(i18n.KeyUsageHeatmapNote)).FontSize(11).TextColor(muted)
				cells := make([]HeatCell, 0, len(snap.DayBuckets))
				for _, b := range snap.DayBuckets {
					cells = append(cells, HeatCell{Label: b.Label, Value: b.Count})
				}
				HeatmapView(c, cells, muted, accent, field)

				barsSrc := snap.DayBuckets
				if len(barsSrc) > 7 {
					barsSrc = barsSrc[len(barsSrc)-7:]
				}
				ui.Text(c, i18n.T(i18n.KeyUsageChartTitle)).FontSize(13).Bold().TextColor(fg).Padding(8, 0, 0, 0)
				ui.Text(c, i18n.T(i18n.KeyUsageChartNote)).FontSize(11).TextColor(muted)
				bars := make([]Bar, 0, len(barsSrc))
				for _, b := range barsSrc {
					bars = append(bars, Bar{Label: b.Label, Value: b.Count})
				}
				BarChartView(c, bars, fg, muted, accent, field)
			}
			ui.Row(c).Gap(8).Children(func() {
				if snap.UsageURL != "" && m.OnOpenUsageWeb != nil {
					if ui.Button(c, i18n.T(i18n.KeyUsageOpenWeb)).Clicked() {
						m.OnOpenUsageWeb()
					}
				}
				if snap.UsageURL != "" && m.OnCopyUsageURL != nil {
					if ui.Button(c, i18n.T(i18n.KeyUsageCopyURL)).FontSize(11).Clicked() {
						m.OnCopyUsageURL(snap.UsageURL)
					}
				}
				if snap.TopUpURL != "" && m.OnOpenTopUp != nil {
					if ui.Button(c, i18n.T(i18n.KeyUsageOpenTopUp)).Clicked() {
						m.OnOpenTopUp()
					}
				}
				if snap.TopUpURL != "" && m.OnCopyTopUpURL != nil {
					if ui.Button(c, i18n.T(i18n.KeyUsageCopyTopUp)).FontSize(11).Clicked() {
						m.OnCopyTopUpURL(snap.TopUpURL)
					}
				}
				if snap.BalanceSummary != "" && m.OnCopyBalance != nil {
					if ui.Button(c, i18n.T(i18n.KeyUsageCopyBalance)).FontSize(11).Clicked() {
						m.OnCopyBalance(snap.BalanceSummary)
					}
				}
			})

			ui.Text(c, i18n.T(i18n.KeyUsageRecent)).FontSize(13).Bold().TextColor(fg).Padding(8, 0, 0, 0)
			if len(snap.Recent) == 0 {
				ui.Text(c, i18n.T(i18n.KeyUsageRecentEmpty)).FontSize(12).TextColor(muted)
			} else {
				ui.Column(c).Fill().MinHeight(0).Background(field).Border(1, border).Radius(8).
					Padding(8, 10, 8, 10).Gap(6).Children(func() {
					for _, row := range snap.Recent {
						entry := row
						ui.Column(c).Gap(2).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Text(c, entry.Label).FontSize(12).Bold().TextColor(fg).Grow(1)
								body := strings.TrimSpace(entry.Detail)
								if body == "" {
									body = entry.Label
								}
								if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
									mygo.Clipboard.WriteText(body)
								}
								if entry.JobID != "" {
									if ui.Button(c, i18n.T(i18n.KeyUsageFollowJob)).FontSize(11).Clicked() {
										m.Store.FollowJob(context.Background(), m.SessionID, entry.JobID)
									}
								}
								if entry.JobID != "" && jobKillable(entry.Status) {
									if ui.Button(c, i18n.T(i18n.KeyUsageKillJob)).FontSize(11).
										TextColor(accent).Clicked() {
										m.Store.KillJob(context.Background(), m.SessionID, entry.JobID)
									}
								}
							})
							ui.Text(c, entry.Detail).FontSize(11).TextColor(muted)
						})
					}
				})
			}
		})
	})
}
