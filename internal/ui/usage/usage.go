// Package usage renders real token statistics alongside account and task activity.
package usage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
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
	// Preferences paints the original settingsRows below the toolbar and above statistics.
	Preferences func(c *ui.Context, fg, muted ui.Color)
}

// View paints the native usage panel.
func View(c *ui.Context, m Model) {
	if !m.Embedded && m.Store == nil {
		return
	}
	var snap usagestore.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot().Value
	}
	if m.Embedded {
		embeddedView(c, m)
		return
	}
	root := ui.Column(c).Key("usage-page").Grow(1).FillWidth().MinHeight(0).MinWidth(0)
	scroll := ui.Local(root, "scroll", func() *ui.ScrollState { return &ui.ScrollState{} })
	root.Children(func() {
		bar := c.TitleBar()
		if bar.Left > 0 || bar.Right > 0 {
			// Reserve the native window-control band above the scroll viewport so
			// neither the top-right page actions (even in narrow windows) nor
			// scrolled content can overlap Windows - □ × or macOS traffic lights.
			ui.Row(c.Key("usage-caption-safe")).FillWidth().Height(max(float32(44), bar.Height)).Shrink(0).DragWindow()
		}
		pagekit.Page(c, pagekit.PageOptions{Scroll: *scroll}, func() {
			usageHeader(c, m, i18n.T(settingsPass4UsageTitle))
			renderPreferences(c, m)
			if m.ActionError != "" {
				pagekit.FaultSentence(c, "", m.ActionError, "", nil)
			}
			statsView(c, m)
			if snap.Err != nil {
				pagekit.FaultSentence(c, "", snap.Err.SentenceText(), "", nil)
			}
			accountSection(c, m, snap)
			recentSection(c, m, snap)
		})
	})
}

func usageHeader(c *ui.Context, m Model, title string) {
	var actions func()
	if m.Store != nil {
		actions = func() {
			st := m.Store.Stats()
			exportDisabled := m.OnExportStats == nil
			exportBtn := coss.Button(c, i18n.T(settingsPass4UsageExport), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonSM,
				Icon:     coss.IconDownload,
				Disabled: exportDisabled,
			})
			if exportDisabled {
				exportBtn.Tooltip(i18n.T(settingsPass4UsageExportUnavailable))
			} else {
				exportOpen := ui.Local(exportBtn, "export-open", func() bool { return false })
				items := []coss.MenuItem{
					{ID: "json", Label: usageText("导出 JSON", "Export JSON")},
					{ID: "days", Label: usageText("导出每日 CSV", "Export daily CSV")},
					{ID: "models", Label: usageText("导出模型 CSV", "Export model CSV")},
				}
				if id := coss.Menu(c, exportBtn, exportOpen, items, coss.MenuOptions{
					Align: coss.AlignEnd,
					Label: i18n.T(settingsPass4UsageExport),
				}); id != "" {
					m.OnExportStats(id)
				}
			}
			pricingView(c, m)
			if coss.Button(c, i18n.T(settingsPass4UsageRefresh), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonSM,
				Icon:     coss.IconRefresh,
				Disabled: st.Loading,
			}).Clicked() {
				m.Store.RefreshStats(context.Background(), true)
				if !m.Embedded {
					if m.OnRefresh != nil {
						m.OnRefresh()
					} else {
						m.Store.Refresh(context.Background(), m.SessionID)
					}
				}
			}
			if !m.Embedded && m.OnClose != nil && coss.Button(c, i18n.T(i18n.KeyUsageBack), coss.ButtonOptions{
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonSM,
				Icon:    coss.IconArrowLeft,
			}).Clicked() {
				m.OnClose()
			}
		}
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
		pagekit.Header(c, title, "", actions)
		if m.Store != nil {
			st := m.Store.Stats()
			if st.UpdatedAt > 0 || st.Overview != nil {
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
					if st.UpdatedAt > 0 {
						pagekit.Caption(c, usageUpdatedLabel(st.UpdatedAt)).FillWidth()
					}
					if overview := st.Overview; overview != nil {
						label, source := usageSourceLabel(overview)
						pagekit.Caption(c, label).FillWidth()
						usageSourceLine(c, source)
					}
				})
			}
		}
	})
}

func accountSection(c *ui.Context, m Model, snap usagestore.ViewState) {
	pagekit.Section(c, i18n.T(i18n.KeyUsageAccount), nil, func() {
		coss.Card(c, coss.CardOptions{}, func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
				if snap.AccountStatus == "" && !snap.Loading {
					pagekit.Muted(c, i18n.T(i18n.KeyUsageNoData)).FillWidth()
				} else {
					pagekit.Body(c, snap.AccountStatus).FillWidth()
				}
				if snap.UsageURL != "" {
					pagekit.Caption(c, i18n.T(i18n.KeyUsageWebLink)+": "+snap.UsageURL).FillWidth()
				}
				if snap.ExpiryNote != "" {
					pagekit.Caption(c, i18n.T(i18n.KeyUsageExpiry)+": "+snap.ExpiryNote).FillWidth()
				}
				if snap.BalanceSummary != "" {
					coss.Separator(c, coss.Horizontal)
					pagekit.Strong(c, i18n.T(i18n.KeyUsageBalance)).FillWidth()
					pagekit.Body(c, snap.BalanceSummary).FillWidth()
				}
				hasAction := (snap.UsageURL != "" && (m.OnOpenUsageWeb != nil || m.OnCopyUsageURL != nil)) ||
					(snap.TopUpURL != "" && (m.OnOpenTopUp != nil || m.OnCopyTopUpURL != nil)) ||
					(snap.BalanceSummary != "" && m.OnCopyBalance != nil)
				if hasAction {
					ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
						if snap.UsageURL != "" && m.OnOpenUsageWeb != nil {
							if coss.Button(c, i18n.T(i18n.KeyUsageOpenWeb), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconExternalLink}).Clicked() {
								m.OnOpenUsageWeb()
							}
						}
						if snap.UsageURL != "" && m.OnCopyUsageURL != nil {
							if coss.Button(c, i18n.T(i18n.KeyUsageCopyURL), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconCopy}).Clicked() {
								m.OnCopyUsageURL(snap.UsageURL)
							}
						}
						if snap.TopUpURL != "" && m.OnOpenTopUp != nil {
							if coss.Button(c, i18n.T(i18n.KeyUsageOpenTopUp), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Icon: coss.IconExternalLink}).Clicked() {
								m.OnOpenTopUp()
							}
						}
						if snap.TopUpURL != "" && m.OnCopyTopUpURL != nil {
							if coss.Button(c, i18n.T(i18n.KeyUsageCopyTopUp), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconCopy}).Clicked() {
								m.OnCopyTopUpURL(snap.TopUpURL)
							}
						}
						if snap.BalanceSummary != "" && m.OnCopyBalance != nil {
							if coss.Button(c, i18n.T(i18n.KeyUsageCopyBalance), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconCopy}).Clicked() {
								m.OnCopyBalance(snap.BalanceSummary)
							}
						}
					})
				}
			})
		})
	})
}

func recentSection(c *ui.Context, m Model, snap usagestore.ViewState) {
	pagekit.Section(c, i18n.T(i18n.KeyUsageRecent), nil, func() {
		if len(snap.Recent) == 0 {
			coss.Card(c, coss.CardOptions{}, func() {
				pagekit.Muted(c, i18n.T(i18n.KeyUsageRecentEmpty)).FillWidth()
			})
			return
		}
		coss.Card(c, coss.CardOptions{}, func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Children(func() {
				for i, row := range snap.Recent {
					if i > 0 {
						coss.Separator(c, coss.Horizontal)
					}
					entry := row
					ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1)).Children(func() {
						ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
							pagekit.Strong(c, entry.Label).Grow(1).MinWidth(0)
							body := strings.TrimSpace(entry.Detail)
							if body == "" {
								body = entry.Label
							}
							if coss.Button(c, i18n.T(i18n.KeyChatCopy), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconCopy}).Clicked() {
								mygo.Clipboard.WriteText(body)
							}
							if entry.JobID != "" {
								if coss.Button(c, i18n.T(i18n.KeyUsageFollowJob), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
									m.Store.FollowJob(context.Background(), m.SessionID, entry.JobID)
								}
							}
							if entry.JobID != "" && jobKillable(entry.Status) {
								if coss.Button(c, i18n.T(i18n.KeyUsageKillJob), coss.ButtonOptions{Variant: coss.ButtonDestructive, Size: coss.ButtonSM}).Clicked() {
									m.Store.KillJob(context.Background(), m.SessionID, entry.JobID)
								}
							}
						})
						if entry.Detail != "" {
							pagekit.Caption(c, entry.Detail).FillWidth()
						}
					})
				}
			})
		})
	})
}

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
