package frame

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
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
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

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

	ui.Column(c.Key("cordis-panel-root")).FillWidth().Gap(8).Children(func() {
		triggerLabel := chat.CordisLocale("panel.trigger")
		if runningCount > 0 {
			triggerLabel += " · " + chat.FormatCordisRunningCount(runningCount)
		}
		trigger := ui.Button(c.Key("cordis-panel-trigger"), triggerLabel).
			Label(chat.CordisLocale("panel.plugins.aria")).
			Tooltip(chat.CordisLocale("panel.hint")).
			Height(28).Padding(0, 10).Radius(6).Border(0.5, border).
			Background(field).TextColor(fg).FontSize(theme.InterfaceSize(12)).
			Cursor(ui.CursorPointer)
		if trigger.Clicked() {
			*openPtr = !*openPtr
		}

		if !*openPtr {
			return
		}

		ui.Column(c.Key("cordis-panel-body")).
			Label(chat.CordisLocale("panel.plugins.aria")).
			FillWidth().Padding(10, 12).Gap(8).Radius(10).Border(0.5, border).Background(chrome.Color(colors.BG)).
			Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, chat.CordisLocale("panel.title")).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
					if m.Loading {
						ui.Text(c, chat.CordisLocale("panel.loading")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					} else if runningCount > 0 {
						ui.Text(c, chat.FormatCordisRunningCount(runningCount)).FontSize(theme.InterfaceSize(11)).TextColor(accent)
					}
				})
				ui.Text(c, chat.CordisLocale("panel.hint")).FontSize(theme.InterfaceSize(11)).TextColor(muted)

				if m.Error != "" {
					ui.Text(c, chat.FormatCordisPanelReadFailed(m.Error)).FontSize(theme.InterfaceSize(12)).TextColor(chrome.Color(colors.Danger))
				}
				if m.RetryAvailable || m.Error != "" {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, chat.CordisLocale("panel.retryAvailable")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						if ui.Button(c.Key("cordis-retry-inventory"), chat.CordisLocale("action.retry")).
							Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(field).
							FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnRetryInventory != nil {
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
					ui.Text(c, title).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(muted)
					for _, item := range items {
						row := item
						ui.Column(c.Key("cordis-plugin:"+row.ID)).FillWidth().Padding(8, 10).Gap(4).Radius(8).Border(0.5, border).Background(field).Children(func() {
							ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
								name := strings.TrimSpace(row.Name)
								if name == "" {
									name = row.ID
								}
								ui.Text(c, name).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(fg).Grow(1)
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
								ui.Text(c, chat.CordisLocale("body.error")+": "+row.Error).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Danger))
							}

							if row.CanApprove {
								ui.Row(c.Key("cordis-approvals:" + row.ID)).Label(chat.CordisLocale("panel.approvals.aria")).Gap(6).Wrap().Children(func() {
									if ui.Button(c.Key("cordis-approve-once:"+row.ID), chat.CordisLocale("action.approveOnce")).
										Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(ui.Transparent).
										FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnApproveOnce != nil {
										m.OnApproveOnce(row.ID)
									}
									if ui.Button(c.Key("cordis-approve-plugin:"+row.ID), chat.CordisLocale("action.approvePlugin")).
										Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(ui.Transparent).
										FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnApprovePlugin != nil {
										m.OnApprovePlugin(row.ID)
									}
								})
							}

							ui.Row(c.Key("cordis-actions:" + row.ID)).Gap(6).Wrap().Children(func() {
								if row.CanStart {
									if ui.Button(c.Key("cordis-start:"+row.ID), chat.CordisLocale("action.start")).
										Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(ui.Transparent).
										FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnStart != nil {
										m.OnStart(row.ID)
									}
								}
								if row.CanStop {
									if ui.Button(c.Key("cordis-stop:"+row.ID), chat.CordisLocale("action.stop")).
										Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(ui.Transparent).
										FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnStop != nil {
										m.OnStop(row.ID)
									}
								}
								if row.CanRetry {
									if ui.Button(c.Key("cordis-retry:"+row.ID), chat.CordisLocale("action.retry")).
										Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(ui.Transparent).
										FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnRetryPlugin != nil {
										m.OnRetryPlugin(row.ID)
									}
								}
								if row.CanRollback {
									if ui.Button(c.Key("cordis-rollback:"+row.ID), chat.CordisLocale("action.rollback")).
										Height(24).Padding(0, 8).Radius(4).Border(0.5, border).Background(ui.Transparent).
										FontSize(theme.InterfaceSize(11)).Cursor(ui.CursorPointer).Clicked() && m.OnRollback != nil {
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
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)

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

	ui.Column(c.Key("cordis-define-card:"+m.Name)).Label(a11y).FillWidth().Padding(10, 12).Gap(6).Radius(8).Border(0.5, border).Background(field).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(fg).Grow(1)
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
			ui.Text(c, chat.CordisLocale("body.source")).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(muted)
			ui.Text(c, m.Source).FontSize(theme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		if m.Output != "" {
			ui.Text(c, chat.CordisLocale("body.output")).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(muted)
			ui.Text(c, m.Output).FontSize(theme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		if m.Error != "" {
			ui.Text(c, chat.CordisLocale("body.error")).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(chrome.Color(colors.Danger))
			ui.Text(c, m.Error).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Danger)).Selectable()
		}
	})
}
