package frame

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// CordisPluginRow models a single plugin item in the Cordis runtime panel.
type CordisPluginRow struct {
	ID               string
	Name             string
	Purpose          string
	Status           string // idle | awaitingApproval | running | failed | clientPending | superseded
	CurrentPackageID string
	NextPackageID    string
	CurrentSession   bool
	CanStart         bool
	CanStop          bool
	CanRetry         bool
	CanRollback      bool
	CanApprove       bool
	Error            string
}

// CordisPanelModel drives the Cordis plugin control panel above Settings.
type CordisPanelModel struct {
	Open             *bool
	ThemeID          string
	SystemDark       bool
	Loading          bool
	Error            string
	RetryAvailable   bool
	Plugins          []CordisPluginRow
	OnRetryInventory func()
	OnStart          func(id string)
	OnStop           func(id string)
	OnRetryPlugin    func(id string)
	OnRollback       func(id string)
	OnApproveOnce    func(id string)
	OnApprovePlugin  func(id string)
}

// CordisDefineCardModel drives a Cordis plugin registration/update card in chat/workspace.
type CordisDefineCardModel struct {
	IsUpdate   bool
	State      string // preparing | defining | defined | failed | stopped
	Name       string
	Purpose    string
	Source     string
	Output     string
	Error      string
	Removed    bool
	Superseded bool
	ThemeID    string
	SystemDark bool
}

var cordisPanelOpenFallback bool

func cordisStatusLabel(status string) string {
	switch status {
	case "awaitingApproval", "awaiting_approval":
		return chat.CordisLocale("status.awaitingApproval")
	case "running":
		return chat.CordisLocale("status.running")
	case "failed", "error":
		return chat.CordisLocale("status.failed")
	case "clientPending", "client_pending":
		return chat.CordisLocale("status.clientPending")
	case "superseded":
		return chat.CordisLocale("status.superseded")
	default:
		return chat.CordisLocale("status.idle")
	}
}

// CordisPanelView renders the Cordis plugin trigger and popover/panel.
func CordisPanelView(c *ui.Context, m CordisPanelModel) {
	t := tk(c)
	fg := t.Foreground
	muted := t.MutedForeground
	border := t.Border
	accent := t.Primary

	openPtr := &cordisPanelOpenFallback
	if m.Open != nil {
		openPtr = m.Open
	}

	runningCount := 0
	var currentGroup, otherGroup []CordisPluginRow
	for _, p := range m.Plugins {
		if p.Status == "running" {
			runningCount++
		}
		if p.CurrentSession {
			currentGroup = append(currentGroup, p)
		} else {
			otherGroup = append(otherGroup, p)
		}
	}

	ui.Column(c.Key("cordis-panel-root")).FillWidth().Gap(design.Sp(2)).Children(func() {
		triggerLabel := chat.CordisLocale("panel.trigger")
		if runningCount > 0 {
			triggerLabel += " · " + chat.FormatCordisRunningCount(runningCount)
		}
		trigger := coss.Button(c, triggerLabel, coss.ButtonOptions{
			Variant: coss.ButtonOutline,
			Size:    coss.ButtonSM,
			Label:   chat.CordisLocale("panel.plugins.aria"),
			Key:     "cordis-panel-trigger",
		}).Tooltip(chat.CordisLocale("panel.hint"))
		if trigger.Clicked() {
			*openPtr = !*openPtr
		}

		if !*openPtr {
			return
		}

		ui.Column(c.Key("cordis-panel-body")).
			Label(chat.CordisLocale("panel.plugins.aria")).
			FillWidth().Padding(design.Sp(2.5), design.Sp(3)).Gap(design.Sp(2)).
			Radius(design.RadiusLG).Border(design.BorderWidth/2, border).Background(t.Popover).
			Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.Sp(2)).Children(func() {
					ui.Text(c, chat.CordisLocale("panel.title")).FontSize(theme.InterfaceSize(13)).FontWeight(design.WeightSemibold).TextColor(fg).Grow(1)
					if m.Loading {
						ui.Text(c, chat.CordisLocale("panel.loading")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					} else if runningCount > 0 {
						ui.Text(c, chat.FormatCordisRunningCount(runningCount)).FontSize(theme.InterfaceSize(11)).TextColor(accent)
					}
				})
				ui.Text(c, chat.CordisLocale("panel.hint")).FontSize(theme.InterfaceSize(11)).TextColor(muted)

				if m.Error != "" {
					ui.Text(c, chat.FormatCordisPanelReadFailed(m.Error)).FontSize(theme.InterfaceSize(12)).TextColor(t.DestructiveForeground)
				}
				if m.RetryAvailable || m.Error != "" {
					ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, chat.CordisLocale("panel.retryAvailable")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						if coss.Button(c, chat.CordisLocale("action.retry"), coss.ButtonOptions{
							Variant: coss.ButtonOutline,
							Size:    coss.ButtonXS,
							Key:     "cordis-retry-inventory",
						}).Clicked() && m.OnRetryInventory != nil {
							m.OnRetryInventory()
						}
					})
				}

				if len(m.Plugins) == 0 && m.Error == "" {
					if m.Loading {
						ui.Text(c, chat.CordisLocale("panel.loading")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					} else {
						ui.Text(c, chat.CordisLocale("panel.empty")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
					return
				}

				renderGroup := func(title string, items []CordisPluginRow) {
					if len(items) == 0 {
						return
					}
					ui.Text(c, title).FontSize(theme.InterfaceSize(11)).FontWeight(design.WeightSemibold).TextColor(muted)
					for _, item := range items {
						row := item
						ui.Column(c.Key("cordis-plugin:"+row.ID)).FillWidth().Padding(design.Sp(2), design.Sp(2.5)).Gap(design.Sp(1)).Radius(design.RadiusMD).Border(design.BorderWidth/2, border).Background(frameInsetBg(t)).Children(func() {
							ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.Sp(1.5)).Children(func() {
								name := strings.TrimSpace(row.Name)
								if name == "" {
									name = row.ID
								}
								ui.Text(c, name).FontSize(theme.InterfaceSize(12)).FontWeight(design.WeightSemibold).TextColor(fg).Grow(1)
								ui.Text(c, cordisStatusLabel(row.Status)).FontSize(theme.InterfaceSize(11)).TextColor(accent)
							})
							purpose := strings.TrimSpace(row.Purpose)
							if purpose == "" {
								purpose = chat.CordisLocale("purpose.missing")
							}
							ui.Text(c, purpose).FontSize(theme.InterfaceSize(11)).TextColor(muted)

							if row.CurrentPackageID != "" {
								ui.Text(c, chat.FormatCordisPanelCurrent(row.CurrentPackageID)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							}
							if row.NextPackageID != "" {
								ui.Text(c, chat.FormatCordisPanelNext(row.NextPackageID)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
							}
							if row.Error != "" {
								ui.Text(c, chat.CordisLocale("body.error")+": "+row.Error).FontSize(theme.InterfaceSize(11)).TextColor(t.DestructiveForeground)
							}

							if row.CanApprove {
								ui.Row(c.Key("cordis-approvals:" + row.ID)).Label(chat.CordisLocale("panel.approvals.aria")).Gap(design.Sp(1.5)).Wrap().Children(func() {
									if coss.Button(c, chat.CordisLocale("action.approveOnce"), coss.ButtonOptions{
										Variant: coss.ButtonOutline,
										Size:    coss.ButtonXS,
										Key:     "cordis-approve-once:" + row.ID,
									}).Clicked() && m.OnApproveOnce != nil {
										m.OnApproveOnce(row.ID)
									}
									if coss.Button(c, chat.CordisLocale("action.approvePlugin"), coss.ButtonOptions{
										Variant: coss.ButtonOutline,
										Size:    coss.ButtonXS,
										Key:     "cordis-approve-plugin:" + row.ID,
									}).Clicked() && m.OnApprovePlugin != nil {
										m.OnApprovePlugin(row.ID)
									}
								})
							}

							ui.Row(c.Key("cordis-actions:" + row.ID)).Gap(design.Sp(1.5)).Wrap().Children(func() {
								if row.CanStart {
									if coss.Button(c, chat.CordisLocale("action.start"), coss.ButtonOptions{
										Variant: coss.ButtonOutline,
										Size:    coss.ButtonXS,
										Key:     "cordis-start:" + row.ID,
									}).Clicked() && m.OnStart != nil {
										m.OnStart(row.ID)
									}
								}
								if row.CanStop {
									if coss.Button(c, chat.CordisLocale("action.stop"), coss.ButtonOptions{
										Variant: coss.ButtonOutline,
										Size:    coss.ButtonXS,
										Key:     "cordis-stop:" + row.ID,
									}).Clicked() && m.OnStop != nil {
										m.OnStop(row.ID)
									}
								}
								if row.CanRetry {
									if coss.Button(c, chat.CordisLocale("action.retry"), coss.ButtonOptions{
										Variant: coss.ButtonOutline,
										Size:    coss.ButtonXS,
										Key:     "cordis-retry:" + row.ID,
									}).Clicked() && m.OnRetryPlugin != nil {
										m.OnRetryPlugin(row.ID)
									}
								}
								if row.CanRollback {
									if coss.Button(c, chat.CordisLocale("action.rollback"), coss.ButtonOptions{
										Variant: coss.ButtonOutline,
										Size:    coss.ButtonXS,
										Key:     "cordis-rollback:" + row.ID,
									}).Clicked() && m.OnRollback != nil {
										m.OnRollback(row.ID)
									}
								}
							})
						})
					}
				}

				renderGroup(chat.CordisLocale("panel.group.current"), currentGroup)
				renderGroup(chat.CordisLocale("panel.group.others"), otherGroup)
			})
	})
}

// CordisDefineCardView renders a Cordis plugin registration or update card.
func CordisDefineCardView(c *ui.Context, m CordisDefineCardModel) {
	t := tk(c)
	fg := t.Foreground
	muted := t.MutedForeground
	border := t.Border

	title := chat.CordisLocale("row.defineTitle")
	if m.IsUpdate {
		title = chat.CordisLocale("row.updateTitle")
	}
	a11y := chat.CordisLocale("a11y.defined")
	switch m.State {
	case "preparing":
		a11y = chat.CordisLocale("a11y.preparing")
	case "defining":
		a11y = chat.CordisLocale("a11y.defining")
	case "failed":
		a11y = chat.CordisLocale("a11y.failed")
	case "stopped":
		a11y = chat.CordisLocale("a11y.stopped")
	}

	ui.Column(c.Key("cordis-define-card:"+m.Name)).Label(a11y).FillWidth().Padding(design.Sp(2.5), design.Sp(3)).Gap(design.Sp(1.5)).
		Radius(design.RadiusMD).Border(design.BorderWidth/2, border).Background(frameInsetBg(t)).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.Sp(1.5)).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(12)).FontWeight(design.WeightSemibold).TextColor(fg).Grow(1)
			ui.Text(c, a11y).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		})
		purpose := strings.TrimSpace(m.Purpose)
		if purpose == "" {
			purpose = chat.CordisLocale("purpose.missing")
		}
		ui.Text(c, purpose).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		if m.Removed {
			ui.Text(c, chat.CordisLocale("run.removed")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if m.Superseded {
			ui.Text(c, chat.CordisLocale("run.superseded")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if m.Source != "" {
			ui.Text(c, chat.CordisLocale("body.source")).FontSize(theme.InterfaceSize(11)).FontWeight(design.WeightSemibold).TextColor(muted)
			ui.Text(c, m.Source).FontSize(theme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		if m.Output != "" {
			ui.Text(c, chat.CordisLocale("body.output")).FontSize(theme.InterfaceSize(11)).FontWeight(design.WeightSemibold).TextColor(muted)
			ui.Text(c, m.Output).FontSize(theme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		if m.Error != "" {
			ui.Text(c, chat.CordisLocale("body.error")).FontSize(theme.InterfaceSize(11)).FontWeight(design.WeightSemibold).TextColor(t.DestructiveForeground)
			ui.Text(c, m.Error).FontSize(theme.InterfaceSize(11)).TextColor(t.DestructiveForeground).Selectable()
		}
	})
}
