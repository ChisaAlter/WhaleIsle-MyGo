// Package usage renders real token statistics alongside account and task activity.
package usage

import (
	"time"
	"fmt"
	"context"
	"strings"
	"whaleisle/internal/ui/widgets"

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
	Embedded       bool
	ActionError    string
	Store          *usagestore.Store
	SessionID      string
	ThemeID        string
	SystemDark     bool
	OnExportStats  func(format string)
	OnOpenSession  func(id string)
	OnClose        func()
	OnRefresh      func()
	OnOpenUsageWeb func()
	OnOpenTopUp    func()
	OnCopyUsageURL func(url string)
	OnCopyTopUpURL func(url string)
	OnCopyBalance  func(text string)
}

// View paints the native usage panel.
func View(c *ui.Context, m Model) {
	if m.Embedded {
		embeddedView(c, m)
		return
	}
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
		ui.Column(c).FillWidth().MinWidth(0).Padding(14, 16, 8, 16).Gap(4).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, usageChartIcon).Size(16, 16).TextColor(fg)
				ui.Text(c, i18n.T(settingsPass4UsageTitle)).FontSize(theme.InterfaceSize(18)).FontWeight(600).TextColor(fg).Grow(1)
				if snap.Loading {
					ui.Text(c, i18n.T(i18n.KeyUsageLoading)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
				// The reference keeps export and refresh next to the title. Export
				// is a real action once the Host supplies it; the range dialog the
				// reference's gear opens does not exist in this build, so no second
				// dead button is drawn.
				if m.OnExportStats != nil {
					if widgets.Button(c, colors, i18n.T(settingsPass4UsageExport), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
						m.OnExportStats("json")
					}
				}
				if widgets.Button(c, colors, i18n.T(settingsPass4UsageRefresh), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
					m.Store.RefreshStats(context.Background(), true)
					if m.OnRefresh != nil {
						m.OnRefresh()
					} else {
						m.Store.Refresh(context.Background(), m.SessionID)
					}
				}
				if m.OnClose != nil && widgets.Button(c, colors, i18n.T(i18n.KeyUsageBack), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					m.OnClose()
				}
			})
			if at := m.Store.Stats().UpdatedAt; at > 0 {
				ui.Text(c, usageUpdatedLabel(at)).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Scroll(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				statsView(c, m, fg, muted, accent, field)

				if m.ActionError != "" {
					ui.Text(c, m.ActionError).FontSize(theme.InterfaceSize(12)).TextColor(accent)
				}
				if snap.Err != "" {
					ui.Text(c, snap.Err).FontSize(theme.InterfaceSize(12)).TextColor(accent)
				}

				ui.Text(c, i18n.T(i18n.KeyUsageAccount)).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg)
				if snap.AccountStatus == "" && !snap.Loading {
					ui.Text(c, i18n.T(i18n.KeyUsageNoData)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				} else {
					ui.Text(c, snap.AccountStatus).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				}
				if snap.UsageURL != "" {
					ui.Text(c, i18n.T(i18n.KeyUsageWebLink)+": "+snap.UsageURL).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
				if snap.ExpiryNote != "" {
					ui.Text(c, i18n.T(i18n.KeyUsageExpiry)+": "+snap.ExpiryNote).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}

				ui.Text(c, i18n.T(i18n.KeyUsageBalance)).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg).Padding(8, 0, 0, 0)
				if snap.BalanceSummary != "" {
					ui.Text(c, snap.BalanceSummary).FontSize(theme.InterfaceSize(11)).TextColor(fg)
				}

				ui.Row(c).Gap(8).Children(func() {
					if snap.UsageURL != "" && m.OnOpenUsageWeb != nil {
						if widgets.Button(c, colors, i18n.T(i18n.KeyUsageOpenWeb), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
							m.OnOpenUsageWeb()
						}
					}
					if snap.UsageURL != "" && m.OnCopyUsageURL != nil {
						if widgets.Button(c, colors, i18n.T(i18n.KeyUsageCopyURL), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
							m.OnCopyUsageURL(snap.UsageURL)
						}
					}
					if snap.TopUpURL != "" && m.OnOpenTopUp != nil {
						if widgets.Button(c, colors, i18n.T(i18n.KeyUsageOpenTopUp), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Clicked() {
							m.OnOpenTopUp()
						}
					}
					if snap.TopUpURL != "" && m.OnCopyTopUpURL != nil {
						if widgets.Button(c, colors, i18n.T(i18n.KeyUsageCopyTopUp), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
							m.OnCopyTopUpURL(snap.TopUpURL)
						}
					}
					if snap.BalanceSummary != "" && m.OnCopyBalance != nil {
						if widgets.Button(c, colors, i18n.T(i18n.KeyUsageCopyBalance), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
							m.OnCopyBalance(snap.BalanceSummary)
						}
					}
				})

				ui.Text(c, i18n.T(i18n.KeyUsageRecent)).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg).Padding(8, 0, 0, 0)
				if len(snap.Recent) == 0 {
					ui.Text(c, i18n.T(i18n.KeyUsageRecentEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				} else {
					ui.Column(c).Fill().MinHeight(0).Background(field).Border(1, border).Radius(8).
						Padding(8, 10, 8, 10).Gap(6).Children(func() {
						for _, row := range snap.Recent {
							entry := row
							ui.Column(c).Gap(2).Children(func() {
								ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
									ui.Text(c, entry.Label).FontSize(theme.InterfaceSize(12)).Bold().TextColor(fg).Grow(1)
									body := strings.TrimSpace(entry.Detail)
									if body == "" {
										body = entry.Label
									}
									if widgets.Button(c, colors, i18n.T(i18n.KeyChatCopy), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
										mygo.Clipboard.WriteText(body)
									}
									if entry.JobID != "" {
										if widgets.Button(c, colors, i18n.T(i18n.KeyUsageFollowJob), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
											m.Store.FollowJob(context.Background(), m.SessionID, entry.JobID)
										}
									}
									if entry.JobID != "" && jobKillable(entry.Status) {
										if widgets.Button(c, colors, i18n.T(i18n.KeyUsageKillJob), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Clicked() {
											m.Store.KillJob(context.Background(), m.SessionID, entry.JobID)
										}
									}
								})
								ui.Text(c, entry.Detail).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							})
						}
					})
				}
			})
		})
	})
}

// usageChartIcon is the bar-chart glyph beside the usage title.
var usageChartIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M3 13V8M8 13V3M13 13V6" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>`))
// usageUpdatedLabel renders the Host's snapshot time in UTC, matching the
// reference's "data as of HH:MM · UTC" line. Values above 1e12 are taken as
// milliseconds because older Host builds send both units.
func usageUpdatedLabel(at int64) string {
	if at <= 0 {
		return ""
	}
	if at > 1e12 {
		at /= 1000
	}
	return fmt.Sprintf(i18n.T(settingsPass4UsageUpdated), time.Unix(at, 0).UTC().Format("15:04"))
}
