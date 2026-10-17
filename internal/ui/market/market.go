// Package market is the M9 native plugin marketplace stub panel.
package market

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/api"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the market panel to store, install state, and harness status.
type Model struct {
	Store                *marketstore.Store
	ThemeID              string
	SystemDark           bool
	HarnessReady         bool
	MuxConnected         bool
	HarnessMarketOverlay bool
	PluginDisabled       bool
	InstallReady         bool
	InstallSpec          *string
	InstallBusy          bool
	InstallMessage       string
	// InstallLog is the trailing installctrl.Result.Log (phase output).
	InstallLog       string
	NeedsAllowBuilds bool
	AllowBuilds      []string
	// CanCancelInstall is true when a pluginManager/installBundle requestId is active.
	CanCancelInstall bool
	Incompatible     []api.ChangeResultErrorIncompatibleItem

	OnClose       func()
	OnRefresh     func()
	OnInstallSpec func(spec string, allowBuilds []string)
	OnRemoveBundle   func(name string)
	OnSetBundleEnabled func(name string, enabled bool)
	OnCancelInstall    func()
	OnAcceptExemption  func(packageVersion, runtimeVersion string)
	OnCopyInstallLog func(log string)
	// OnOpenHarnessMarket opens the Electron ui-settings-market WebView when overlay is on.
	OnOpenHarnessMarket func()
}

// View paints the plugin market stub panel.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	var snap marketstore.ViewState
	if m.Store != nil {
		snap = m.Store.Snapshot()
	}

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMarketTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
			if snap.Loading || m.InstallBusy {
				ui.Text(c, i18n.T(i18n.KeyMarketLoading)).FontSize(12).TextColor(muted)
			}
			if m.HarnessReady && m.HarnessMarketOverlay && !m.PluginDisabled && m.OnOpenHarnessMarket != nil {
				if ui.Button(c, i18n.T(i18n.KeyMarketOpenHarness)).FontSize(12).Clicked() {
					m.OnOpenHarnessMarket()
				}
			}
			if m.Store != nil && ui.Button(c, i18n.T(i18n.KeyMarketRefresh)).Clicked() {
				if m.OnRefresh != nil {
					m.OnRefresh()
				} else {
					m.Store.Refresh(context.Background())
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyMarketBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Column(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Gap(10).Children(func() {
			if !snap.HasListRPC {
				ui.Text(c, i18n.T(i18n.KeyMarketStubNote)).FontSize(12).TextColor(muted)
			}

			rows := []struct{ label, value string }{
				{i18n.T(i18n.KeyMarketHarnessOverlay), onOff(m.HarnessMarketOverlay)},
				{i18n.T(i18n.KeyMarketPluginDisabled), onOff(m.PluginDisabled)},
				{i18n.T(i18n.KeyMarketMux), muxLabel(m.HarnessReady, m.MuxConnected)},
				{i18n.T(i18n.KeyMarketGeneratedRPC), rpcLabel(snap.HasListRPC, len(snap.Methods))},
				{i18n.T(i18n.KeyMarketInstallToolchain), onOff(m.InstallReady)},
			}
			if snap.RegistriesNote != "" {
				rows = append(rows, struct{ label, value string }{i18n.T(i18n.KeyMarketRegistry), snap.RegistriesNote})
			}
			if snap.ExemptionsNote != "" {
				rows = append(rows, struct{ label, value string }{i18n.T(i18n.KeyMarketExemptions), snap.ExemptionsNote})
			}
			ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
				for _, row := range rows {
					ui.Row(c).Gap(12).Children(func() {
						ui.Text(c, row.label).FontSize(12).TextColor(muted).Width(160)
						ui.Text(c, row.value).FontSize(12).TextColor(fg).Grow(1)
					})
				}
			})

			if m.PluginDisabled {
				ui.Text(c, i18n.T(i18n.KeyMarketPluginOff)).FontSize(12).TextColor(accent)
			}
			if snap.Err != "" {
				ui.Text(c, snap.Err).FontSize(12).TextColor(accent)
			}

			if snap.HasListRPC {
				ui.Text(c, i18n.T(i18n.KeyMarketListVia)+": "+snap.ListMethod).FontSize(12).Bold().TextColor(fg)
				if len(snap.Rows) == 0 {
					if snap.RawListPreview != "" {
						ui.Text(c, snap.RawListPreview).FontSize(11).TextColor(muted)
					} else {
						ui.Text(c, i18n.T(i18n.KeyMarketListEmpty)).FontSize(12).TextColor(muted)
					}
				} else {
					ui.Column(c).Fill().MinHeight(0).Background(field).Border(1, border).Radius(8).
						Padding(8, 10, 8, 10).Gap(6).Children(func() {
						for _, row := range snap.Rows {
							r := row
							ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
									ui.Text(c, r.Label).FontSize(12).Bold().TextColor(fg)
									if r.Detail != "" {
										ui.Text(c, r.Detail).FontSize(11).TextColor(muted)
									}
								})
								spec := marketInstallSpec(r)
								if spec != "" && m.OnInstallSpec != nil && !m.InstallBusy {
									if ui.Button(c, i18n.T(i18n.KeyMarketInstallButton)).FontSize(11).Clicked() {
										m.OnInstallSpec(spec, nil)
									}
								}
								if m.OnSetBundleEnabled != nil && r.Label != "" && !m.InstallBusy {
									enableLabel := i18n.T(i18n.KeySettingsPluginsEnable)
									disableLabel := i18n.T(i18n.KeySettingsPluginsDisable)
									if r.Enabled != nil {
										if *r.Enabled {
											if ui.Button(c, disableLabel).FontSize(11).Clicked() {
												m.OnSetBundleEnabled(r.Label, false)
											}
										} else {
											if ui.Button(c, enableLabel).FontSize(11).Clicked() {
												m.OnSetBundleEnabled(r.Label, true)
											}
										}
									} else {
										if ui.Button(c, disableLabel).FontSize(11).Clicked() {
											m.OnSetBundleEnabled(r.Label, false)
										}
										if ui.Button(c, enableLabel).FontSize(11).Clicked() {
											m.OnSetBundleEnabled(r.Label, true)
										}
									}
								}
								if m.OnRemoveBundle != nil && r.Label != "" && !m.InstallBusy && (r.Removable || r.Enabled == nil) {
									if ui.Button(c, i18n.T(i18n.KeyMarketRemove)).FontSize(11).Clicked() {
										m.OnRemoveBundle(r.Label)
									}
								}
							})
						}
					})
				}
			} else {
				ui.Text(c, i18n.T(i18n.KeyMarketNoRPC)).FontSize(12).TextColor(muted)
				ui.Text(c, i18n.T(i18n.KeyMarketInstallHint)).FontSize(12).TextColor(fg)
				installSection(c, m, fg, muted, field, border, accent)
			}

			if snap.HasListRPC {
				ui.Text(c, i18n.T(i18n.KeyMarketInstallAdd)).FontSize(12).TextColor(muted)
				installSection(c, m, fg, muted, field, border, accent)
			}

			if m.InstallMessage != "" {
				ui.Text(c, m.InstallMessage).FontSize(12).TextColor(accent)
			}
			if m.InstallBusy {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyMarketInstallRunning)).FontSize(12).TextColor(muted)
					if m.CanCancelInstall && m.OnCancelInstall != nil {
						if ui.Button(c, i18n.T(i18n.KeyMarketCancelInstall)).FontSize(11).Clicked() {
							m.OnCancelInstall()
						}
					}
				})
			}
			if log := strings.TrimSpace(m.InstallLog); log != "" {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyMarketInstallLog)).FontSize(11).Bold().TextColor(muted).Grow(1)
					if m.OnCopyInstallLog != nil {
						if ui.Button(c, i18n.T(i18n.KeyMarketCopyLog)).FontSize(11).Clicked() {
							m.OnCopyInstallLog(log)
						}
					}
				})
				lines := strings.Split(log, "\n")
				if len(lines) > 40 {
					lines = lines[len(lines)-40:]
				}
				ui.Column(c).Fill().MinHeight(0).MaxHeight(180).Background(field).
					Padding(8, 10, 8, 10).Gap(2).Children(func() {
					for _, line := range lines {
						ui.Text(c, line).FontSize(11).TextColor(fg)
					}
				})
			}
			if m.NeedsAllowBuilds && len(m.AllowBuilds) > 0 {
				ui.Text(c, i18n.T(i18n.KeyMarketAllowBuilds)+": "+strings.Join(m.AllowBuilds, ", ")).FontSize(12).TextColor(fg)
			}
			if len(m.Incompatible) > 0 {
				ui.Text(c, i18n.T(i18n.KeyMarketIncompatible)).FontSize(12).Bold().TextColor(accent)
				for _, item := range m.Incompatible {
					it := item
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						label := it.Name + "@" + it.Version
						if it.RuntimeVersion != "" {
							label += " (runtime " + it.RuntimeVersion + ")"
						}
						ui.Text(c, label).FontSize(11).TextColor(fg).Grow(1)
						if m.OnAcceptExemption != nil && it.Name != "" && it.Version != "" {
							if ui.Button(c, i18n.T(i18n.KeyMarketAcceptExemption)).FontSize(11).Clicked() {
								m.OnAcceptExemption(it.Name+"@"+it.Version, it.RuntimeVersion)
							}
						}
					})
				}
			}

			if len(snap.Methods) > 0 {
				ui.Text(c, i18n.T(i18n.KeyMarketMethods)).FontSize(11).TextColor(muted)
				ui.Text(c, strings.Join(snap.Methods, ", ")).FontSize(11).TextColor(fg)
			}
		})
	})
}

func installSection(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(10, 12, 10, 12).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMarketInstallSpecLabel)).FontSize(12).TextColor(fg)
		spec := ""
		if m.InstallSpec != nil {
			ui.TextInput(c, m.InstallSpec).FillWidth().Placeholder(i18n.T(i18n.KeyMarketInstallPlaceholder))
			spec = strings.TrimSpace(*m.InstallSpec)
		}
		var allow []string
		if m.NeedsAllowBuilds {
			allow = append([]string(nil), m.AllowBuilds...)
		}
		btnLabel := i18n.T(i18n.KeyMarketInstallButton)
		if m.NeedsAllowBuilds {
			btnLabel = i18n.T(i18n.KeyMarketAllowBuildsButton)
		}
		if !m.InstallBusy && ui.Button(c, btnLabel).Clicked() && m.OnInstallSpec != nil && spec != "" {
			m.OnInstallSpec(spec, allow)
		}
		if !m.InstallReady {
			ui.Text(c, i18n.T(i18n.KeyMarketInstallNotReady)).FontSize(11).TextColor(accent)
		}
	})
}

// marketInstallSpec returns a github:/name@ver candidate from a list row.
func marketInstallSpec(row marketstore.Entry) string {
	d := strings.TrimSpace(row.Detail)
	if strings.HasPrefix(d, "github:") || strings.Contains(d, "@") {
		return d
	}
	l := strings.TrimSpace(row.Label)
	if strings.HasPrefix(l, "github:") {
		return l
	}
	return ""
}

func onOff(v bool) string {
	if v {
		return i18n.T(i18n.KeyMarketOn)
	}
	return i18n.T(i18n.KeyMarketOff)
}

func muxLabel(ready, connected bool) string {
	if !ready {
		return i18n.T(i18n.KeyMarketMuxWaiting)
	}
	if connected {
		return i18n.T(i18n.KeyMarketMuxConnected)
	}
	return i18n.T(i18n.KeyMarketMuxDisconnected)
}

func rpcLabel(hasRPC bool, count int) string {
	if hasRPC {
		return i18n.T(i18n.KeyMarketRPCYes)
	}
	if count > 0 {
		return i18n.T(i18n.KeyMarketRPCPartial)
	}
	return i18n.T(i18n.KeyMarketRPCNo)
}
