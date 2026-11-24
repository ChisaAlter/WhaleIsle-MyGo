// Package launcher provides native runtime maintenance and update controls.
package launcher

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type launcherPalette struct {
	colors     theme.Colors
	primary    ui.Color
	secondary  ui.Color
	tertiary   ui.Color
	accent     ui.Color
	sidebar    ui.Color
	base       ui.Color
	borderL1   ui.Color
	borderL2   ui.Color
	navActive  ui.Color
	navHover   ui.Color
	primaryFg  ui.Color
	field      ui.Color
	successBG  ui.Color
	successFG  ui.Color
	warnBG     ui.Color
	warnBorder ui.Color
	kindBG     ui.Color
	kindFG     ui.Color
	officialBG ui.Color
	segActive  ui.Color
	danger     ui.Color
}

func resolveLauncherPalette(themeID string, dark bool) launcherPalette {
	th := theme.Resolve(themeID).Active(dark)
	accent := chrome.Color(th.Accent)
	border := chrome.Color(th.Border)
	hover := chrome.Color(th.Hover)
	muted := chrome.Color(th.Muted)
	p := launcherPalette{
		colors:     th,
		primary:    chrome.Color(th.FG),
		secondary:  muted,
		tertiary:   muted,
		accent:     accent,
		sidebar:    chrome.Color(th.Sidebar),
		base:       chrome.Color(th.BG),
		borderL1:   border,
		borderL2:   border,
		navActive:  hover,
		navHover:   hover,
		primaryFg:  chrome.Color(th.ButtonFG),
		field:      chrome.Color(th.Field),
		officialBG: accent.Alpha(0.14),
		segActive:  hover,
		danger:     chrome.Color(th.Danger),
	}
	if dark {
		p.successBG = design.TW.Green500.Alpha(0.18)
		p.successFG = design.TW.Green300
		p.warnBG = design.TW.Amber500.Alpha(0.16)
		p.warnBorder = design.TW.Amber500.Alpha(0.45)
		p.kindBG = design.TW.Orange400.Alpha(0.18)
		p.kindFG = design.TW.Orange300
	} else {
		p.navActive = accent.Alpha(0.12)
		p.successBG = design.TW.Green100
		p.successFG = design.TW.Green800
		p.warnBG = design.TW.Amber50
		p.warnBorder = design.TW.Amber300
		p.kindBG = design.TW.Orange100
		p.kindFG = design.TW.Orange800
	}
	return p
}

// Model wires launcher actions (version, import probe, update, start).
type Model struct {
	ManagedDesktop              bool // Standalone launcher remains open while installing another desktop.
	BridgeAlerts                BridgeAlertsModel
	ApplicationUninstall        ApplicationUninstallModel
	StartupDiagnostics          startuprec.Diagnostics
	StartupRepairBusy           bool
	OnRefreshStartupDiagnostics func()
	OnRepairStartupSuspects     func()
	OnSetStartupPluginEnabled   func(string, bool)
	OnRemoveStartupPlugin       func(string)
	ThemeID                     string
	SystemDark                  bool
	Version                     string
	VendorReady                 bool
	OnInstallUpdate             func()
	StartBusy                   bool
	StartError                  string
	ImportSummary               string
	ImportScanBusy              bool
	OnCancelImportScan          func()
	// ImportJournal is the per-item apply log (Electron runImport journal).
	ImportJournal                                                         []string
	UpdateStatus                                                          string
	UpdateAvailable                                                       bool // Electron syncUpdateNotice: only show #home-update when true
	DisabledPlugins                                                       []string
	SkipUserPlugins                                                       bool
	PluginRecoveryReason                                                  string
	StatusNote                                                            string
	OnChooseRepairRelease                                                 func()
	OnClose                                                               func()
	OnStart                                                               func()
	OnStop                                                                func()
	OnScanImport                                                          func() // scan official ~/.dsh
	OnPickImportSource                                                    func() // pick custom source home
	OnAddSkillImportDir                                                   func() // add skill import directory
	OnCopyImportJournal                                                   func(lines []string)
	OnCheckUpdate                                                         func()
	Releases                                                              []update.NativeRelease
	ReleaseBusy                                                           bool
	ReleasePaused, ReleaseAutoDownload                                    bool
	ReleasePhase                                                          string
	OnPauseRelease, OnResumeRelease                                       func()
	OnSetAutoDownload                                                     func(bool)
	ReleaseStatus, ReleaseError, ReleaseSelectedPackage, ReleaseStagedTag string
	ReleaseReceived, ReleaseTotal                                         int64
	OnRefreshReleases                                                     func()
	OnSelectRelease                                                       func(packageID string)
	OnInstallRuntime                                                      func()
	OnInstallRelease                                                      func(packageID string)
	OnCancelRelease                                                       func()
	OnDismissRelease                                                      func()
	OnStartStagedRelease                                                  func()
	OnEnableSkipUserPlugins                                               func()
	OnClearPluginRecovery                                                 func()
	OnRestartHarness                                                      func()
	AskOnUpdate                                                           *bool
	DownloadRoute                                                         *string // "", "github", "cnb"
	RouteUnavailable                                                      map[string]string
	OnSaveLauncherPrefs                                                   func()
	LauncherPrefsError                                                    string
	Components                                                            []components.Row
	OnRefreshComponents                                                   func()
	ComponentBusy                                                         bool
	ComponentPending                                                      map[string]string // component id -> in-flight action
	ComponentStatus                                                       string
	HostRuntimeState                                                      harness.State
	OnComponentAction                                                     func(id, action string) // install|start|stop|open|update|rollback|uninstall
	// ImportOverwrite / ImportAttachments mirror launcher.html #import-overwrite / #import-attachments.
	ImportOverwrite   *bool
	ImportAttachments *bool
	// ImportCategories maps chip id ??found (from importer.Report.CategoryFound).
	ImportCategories map[string]bool
	// ImportCatalog is the live scanImport inventory (item rows + selection).
	ImportCatalog  *importer.Catalog
	OnStartImport  func()
	ImportBusy     bool
	OnCancelImport func()
}

var (
	importOverwriteFB   bool
	importAttachmentsFB bool
	importCatActive     = "sessions"
)

func componentsPanel(c *ui.Context, m Model) {
	pal := resolveLauncherPalette(m.ThemeID, m.SystemDark)
	ui.Column(c).Gap(12).Children(func() {
		bridgeAlertSummary(c, pal, m)
		if m.ComponentStatus != "" {
			ui.Text(c, m.ComponentStatus).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		}
		if !components.WhaleBridgeManagementSupported() {
			ui.Text(c, releaseText("鲸桥安装和组件管理仅支持 Windows x64。", "WhaleBridge installation and management require Windows x64.")).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		}
		// Electron: compact ????pill, not a full-width bar.
		ui.Row(c).AlignSelf(ui.Start).Children(func() {
			if ghostSmall(c, pal, i18n.T(i18n.KeyUsageRefresh)).Clicked() && m.OnRefreshComponents != nil {
				m.OnRefreshComponents()
			}
		})
		if len(m.Components) == 0 {
			ui.Text(c, i18n.T(i18n.KeyLauncherComponentsEmpty)).FontSize(theme.InterfaceSize(14)).TextColor(pal.secondary)
			return
		}
		for _, row := range m.Components {
			componentCard(c, pal, m, row, false)
		}
	})
}

func componentCard(c *ui.Context, pal launcherPalette, m Model, row components.Row, compact bool) {
	pendingAction := m.ComponentPending[row.ID]
	componentBusy := m.ComponentBusy || pendingAction != ""
	statusText := stateLabel(row.State)
	if pendingAction != "" {
		statusText = componentPendingLabel(pendingAction)
	}
	if row.ID == components.HostComponentID {
		hostBundleCard(c, pal, m, row, compact)
		return
	}
	if compact {
		width, _ := c.Size()
		padding := float32(20)
		if width <= 760 {
			padding = 16
		}
		chrome.FrostSurface(ui.Column(c.Key("home-component-"+row.ID)).MinWidth(0).Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.sidebar, 12).
			Padding(padding).Gap(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
				componentAvatar(c, pal, row, 36)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, row.Name).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(pal.primary)
						ui.Text(c, "v"+nonempty(row.InstalledVersion, row.Version)).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
						componentStatusBadge(c, pal, row, statusText)
					})
					if row.Description != "" {
						ui.Text(c, row.Description).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
					}
				})
			})
			componentStatusDetail(c, pal, row)
			// launcher-components.js renderHomeRows keeps stop and settings
			// reachable from the home card while an installed component runs.
			if row.State == "running" {
				ui.Row(c).Wrap().Gap(8).Children(func() {
					label := fmt.Sprintf(releaseText("关闭%s", "Close %s"), row.Name)
					if ghostBtn(c, pal, label).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
						m.OnComponentAction(row.ID, "stop")
					}
					if row.ID == "whalebridge" && ghostBtn(c, pal, i18n.T(i18n.KeyLauncherCompOpenSettings)).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
						m.OnComponentAction(row.ID, "open")
					}
				})
			} else if row.State != "unsupported" {
				label := fmt.Sprintf(i18n.T(i18n.KeyLauncherCompStartNamed), row.Name)
				if row.State == "error" {
					label = i18n.T(i18n.KeyLauncherCompRetry)
				}
				if row.State == "available" {
					label = i18n.T(i18n.KeyLauncherCompInstall)
				}
				ui.Row(c).Children(func() {
					if primaryBtn(c, pal, label).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
						action := "start"
						if row.State == "available" {
							action = "install"
						}
						m.OnComponentAction(row.ID, action)
					}
				})
			}
		})
		return
	}

	// Electron L-COMP: divider row ??avatar | copy+tags | trailing column (status + ??/??).
	ui.Column(c.Key("component-"+row.ID)).FillWidth().Padding(12, 0, 14, 0).
		BorderWidth(0, 0, 1, 0).BorderColor(pal.borderL1).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			componentAvatar(c, pal, row, 40)
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, row.Name).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(pal.primary)
					if row.Kind != "" {
						ui.Row(c).Background(pal.kindBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, kindLabel(row.Kind)).FontSize(theme.InterfaceSize(11)).TextColor(pal.kindFG)
						})
					}
					if row.Source == "official" {
						ui.Row(c).Background(pal.officialBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyLauncherCompOfficial)).FontSize(theme.InterfaceSize(11)).TextColor(pal.accent)
						})
					} else if row.Source == "bundled" {
						ui.Row(c).Background(pal.officialBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyLauncherCompBundled)).FontSize(theme.InterfaceSize(11)).TextColor(pal.accent)
						})
					}
					if row.Orphan {
						ui.Row(c).Background(pal.warnBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, releaseText("目录外组件", "Outside catalog")).FontSize(theme.InterfaceSize(11)).TextColor(pal.secondary)
						})
					}
					if row.UpdateAvailable {
						ui.Row(c).Background(pal.field).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyLauncherCompUpdatable)).FontSize(theme.InterfaceSize(11)).TextColor(pal.secondary)
						})
					}
				})
				var meta []string
				if row.Description != "" {
					meta = append(meta, row.Description)
				}
				if row.State != "unsupported" {
					meta = append(meta, "v"+nonempty(row.Version, "?"))
				}
				if row.InstalledVersion != "" && row.InstalledVersion != row.Version {
					meta = append(meta, releaseText("\u5b89\u88c5", "Installed")+" v"+row.InstalledVersion)
				}
				if len(meta) > 0 {
					ui.Text(c, strings.Join(meta, " \u00b7 ")).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
				}
				if row.UpdateAvailable {
					ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyLauncherCompUpdateAvail), row.Version)).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
				}
			})
			ui.Column(c).MaxWidthPercent(55).MinWidth(0).Gap(8).AlignItems(ui.End).Children(func() {
				componentStatusBadge(c, pal, row, statusText)
				ui.Row(c).MaxWidthPercent(100).MinWidth(0).Wrap().Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
					if row.State == "unsupported" {
						return
					}
					compPrimary := func(label string) ui.Element {
						// Electron ?? ??72�42 device-px @150% ??~48�28 DIP.
						return ui.Button(c, label).Height(28).Padding(0, 10, 0, 10).
							Radius(14).Background(pal.primary).BorderWidth(0).
							TextColor(pal.primaryFg).FontSize(theme.InterfaceSize(12))
					}
					switch row.State {
					case "available":
						if compPrimary(i18n.T(i18n.KeyLauncherCompInstall)).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
							m.OnComponentAction(row.ID, "install")
						}
					case "running":
						if row.ID == "whalebridge" && compPrimary(i18n.T(i18n.KeyLauncherCompOpenSettings)).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
							m.OnComponentAction(row.ID, "open")
						}
						if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherCompStop)).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
							m.OnComponentAction(row.ID, "stop")
						}
					default:
						label := i18n.T(i18n.KeyLauncherCompStart)
						if row.State == "error" {
							label = i18n.T(i18n.KeyLauncherCompRetry)
						}
						if compPrimary(label).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
							m.OnComponentAction(row.ID, "start")
						}
					}
					if row.InstalledVersion != "" {
						componentManagement(c, pal, m, row, componentBusy)
					}
				})
			})
		})
		componentStatusDetail(c, pal, row)
	})
}

func componentAvatar(c *ui.Context, pal launcherPalette, row components.Row, size float32) {
	if row.ID == "whalebridge" {
		if bmp := chrome.BrandAvatar(); bmp != nil {
			ui.Image(c, bmp).Width(size).Height(size).Fit(ui.Contain)
			return
		}
	}
	name := []rune(strings.TrimSpace(nonempty(row.Name, row.ID)))
	glyph := "?"
	if len(name) > 0 {
		glyph = strings.ToUpper(string(name[0]))
	}
	ui.Box(c).Width(size).Height(size).Radius(8).Background(pal.field).Center().Children(func() {
		ui.Text(c, glyph).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(pal.secondary)
	})
}

func componentManagement(c *ui.Context, pal launcherPalette, m Model, row components.Row, busy bool) {
	busy = busy || (row.ID == "whalebridge" && !components.WhaleBridgeManagementSupported())
	trigger := ghostSmall(c, pal, i18n.T(i18n.KeyLauncherCompManage)).Disabled(busy)
	open := ui.Local(trigger, "component-manage-open", func() bool { return false })
	if trigger.Clicked() {
		*open = !*open
	}
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		chrome.PopoverReveal(c, chrome.FrostSurface(panel.MinWidth(168).Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.field, 12).Padding(8).Gap(4).
			AttachTo(trigger, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0))
		ui.Text(c, fmt.Sprintf(releaseText("当前 v%s", "Current v%s"), row.InstalledVersion)).
			FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary).Padding(6, 10)
		action := func(id, label string) {
			if ghostSmall(c, pal, label).FillWidth().Justify(ui.Start).Disabled(busy).Clicked() {
				*open = false
				if m.OnComponentAction != nil {
					m.OnComponentAction(row.ID, id)
				}
			}
		}
		if row.UpdateAvailable {
			action("update", fmt.Sprintf(releaseText("更新至 v%s", "Update to v%s"), row.Version))
		} else if row.ID == "whalebridge" && components.WhaleBridgeManagementSupported() {
			action("update", i18n.T(i18n.KeyLauncherCompCheckUpdate))
		}
		if row.PreviousVersion != "" && row.PreviousVersion != row.InstalledVersion {
			action("rollback", fmt.Sprintf(i18n.T(i18n.KeyLauncherCompRollback), row.PreviousVersion))
		}
		action("uninstall", releaseText("卸载", "Uninstall"))
	})
}

func kindLabel(kind string) string {
	switch kind {
	case "service":
		return i18n.T(i18n.KeyLauncherCompKindService)
	case "tool":
		return i18n.T(i18n.KeyLauncherCompKindTool)
	default:
		return kind
	}
}

func componentPendingLabel(action string) string {
	switch action {
	case "select":
		return releaseText("正在选择版本…", "Selecting version…")
	case "manage":
		return releaseText("正在打开…", "Opening…")
	case "install":
		return releaseText("正在安装…", "Installing…")
	case "start":
		return releaseText("正在启动…", "Starting…")
	case "stop":
		return releaseText("正在停止…", "Stopping…")
	case "update":
		return releaseText("正在更新…", "Updating…")
	case "rollback":
		return releaseText("正在回滚…", "Rolling back…")
	case "uninstall":
		return releaseText("正在卸载…", "Uninstalling…")
	case "open":
		return releaseText("正在打开…", "Opening…")
	default:
		return action
	}
}

func stateLabel(state string) string {
	switch state {
	case "running":
		return i18n.T(i18n.KeyLauncherCompStateRunning)
	case "stopped":
		return i18n.T(i18n.KeyLauncherCompStateStopped)
	case "installed":
		return releaseText("已安装", "Installed")
	case "unsupported":
		return releaseText("暂不支持", "Unsupported")
	case "available":
		return i18n.T(i18n.KeyLauncherCompStateAvailable)
	case "error":
		return i18n.T(i18n.KeyLauncherCompStateError)
	default:
		return state
	}
}

func importPanel(c *ui.Context, m Model) {
	pal := resolveLauncherPalette(m.ThemeID, m.SystemDark)
	if m.ImportBusy {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, releaseText("正在导入 · 操作结束前不能再次开始", "Importing · cannot start again until finished")).FontSize(theme.InterfaceSize(14)).TextColor(pal.primary)
			ui.Text(c, m.ImportSummary).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
			ui.Text(c, releaseText("取消会在当前项目完成后停止；已完成的数据保留，可安全重扫。", "Cancel stops after the current item finishes; already imported data is kept and can be safely rescanned.")).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
			if ghostSmall(c, pal, releaseText("取消后续导入", "Cancel remaining import")).Clicked() && m.OnCancelImport != nil {
				m.OnCancelImport()
			}
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(200).Children(func() {
				ui.Column(c).FillWidth().Gap(4).Children(func() {
					for _, line := range m.ImportJournal {
						ui.Text(c, line).FontSize(theme.InterfaceSize(11)).TextColor(pal.tertiary)
					}
				})
			})
		})
		return
	}
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyLauncherImportLede)).
			FontSize(12).TextColor(pal.secondary)
		ui.Text(c, i18n.T(i18n.KeyLauncherImportChecklist)).FontSize(theme.InterfaceSize(12)).TextColor(pal.accent)
		ui.Text(c, importer.CopyOnlyNotice()).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		ui.Text(c, releaseText("确认导入后会先检查当前任务并停止 Host 内核；窗口保持打开，导入后可手动启动内核。", "After confirmation, import checks current tasks and stops the Host runtime. The window stays open; start the runtime manually after importing.")).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)

		pathLine := i18n.T(i18n.KeyLauncherImportEmpty)
		if m.ImportSummary != "" {
			pathLine = m.ImportSummary
		}

		ui.Row(c).Gap(8).Children(func() {
			if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherImportPickDir)).Clicked() {
				if m.OnPickImportSource != nil {
					m.OnPickImportSource()
				} else if m.OnScanImport != nil {
					m.OnScanImport()
				}
			}
			if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherImportAddSkillDir)).Disabled(m.ImportScanBusy).Clicked() {
				if m.OnAddSkillImportDir != nil {
					m.OnAddSkillImportDir()
				} else if m.OnScanImport != nil {
					m.OnScanImport()
				}
			}
			if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherImportRescan)).Disabled(m.ImportScanBusy).Clicked() && m.OnScanImport != nil {
				m.OnScanImport()
			}
			if m.ImportScanBusy && ghostSmall(c, pal, releaseText("取消扫描", "Cancel scan")).Clicked() && m.OnCancelImportScan != nil {
				m.OnCancelImportScan()
			}
		})

		if m.ImportScanBusy {
			pathLine = releaseText("正在扫描，保留当前选择…", "Scanning; current selections are retained…")
		}
		ui.Text(c, pathLine).FillWidth().MinWidth(0).SingleLine().Ellipsis("…").Tooltip(pathLine).FontSize(theme.InterfaceSize(12)).TextColor(pal.tertiary)

		// Category chips: n/d from live catalog (fallback to probe CategoryFound).
		cats := []struct{ id, name string }{
			{"sessions", i18n.T(i18n.KeyLauncherImportCatSessions)},
			{"skills", i18n.T(i18n.KeyLauncherImportCatSkills)},
			{"plugins", i18n.T(i18n.KeyLauncherImportCatPlugins)},
			{"mcp", i18n.T(i18n.KeyLauncherImportCatMCP)},
			{"settings", i18n.T(i18n.KeyLauncherImportCatSettings)},
			{"presets", i18n.T(i18n.KeyLauncherImportCatPresets)},
		}
		counts := map[string][2]int{}
		if m.ImportCatalog != nil {
			counts = m.ImportCatalog.CategoryCounts()
		}
		ui.Row(c).FillWidth().Gap(6).Role(ui.RoleTabList).FocusGroup(ui.Horizontal).Children(func() {
			for _, cat := range cats {
				n, d := 0, 0
				if pair, ok := counts[cat.id]; ok {
					n, d = pair[0], pair[1]
				} else if m.ImportCategories != nil {
					d = 1
					if m.ImportCategories[cat.id] {
						n = 1
					}
				}
				label := fmt.Sprintf("%s %d/%d", cat.name, n, d)
				bg := pal.field
				if importCatActive == cat.id {
					bg = pal.navActive
				}
				id := cat.id
				btn := ui.Button(c, label).Grow(1).MinWidth(0).Height(32).Role(ui.RoleTab).Checked(importCatActive == id).Background(bg).Border(1, pal.borderL2).
					Padding(6, 4).Radius(8).TextColor(pal.primary).FontSize(theme.InterfaceSize(12))
				if btn.Clicked() {
					importCatActive = id
					c.Invalidate()
				}
			}
		})

		chrome.FrostSurface(ui.Column(c).FillWidth().Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.base, 12).
			Padding(12, 12, 12, 12).Gap(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherImportByDir)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(pal.primary).Grow(1)
				if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherImportSelectAll)).Clicked() && m.ImportCatalog != nil {
					if items := m.ImportCatalog.ItemsFor(importCatActive); items != nil {
						for i := range *items {
							if (*items)[i].Blocker.Importable() {
								(*items)[i].Selected = true
							}
						}
					}
					c.Invalidate()
				}
				if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherImportSelectNone)).Clicked() && m.ImportCatalog != nil {
					if items := m.ImportCatalog.ItemsFor(importCatActive); items != nil {
						for i := range *items {
							(*items)[i].Selected = false
						}
					}
					c.Invalidate()
				}
			})
			_, viewportHeight := c.Size()
			ui.Scroll(c).FillWidth().MinHeight(80).MaxHeight(max(120, min(360, viewportHeight-340))).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
					itemsPtr := (*[]importer.CatalogItem)(nil)
					if m.ImportCatalog != nil {
						itemsPtr = m.ImportCatalog.ItemsFor(importCatActive)
					}
					if importCatActive == "plugins" {
						ui.Text(c, i18n.T(i18n.KeyLauncherImportPluginsNote)).
							FontSize(12).TextColor(pal.accent)
					}
					if itemsPtr == nil || len(*itemsPtr) == 0 {
						empty := i18n.T(i18n.KeyLauncherImportEmptyItems)
						switch importCatActive {
						case "sessions":
							empty = i18n.T(i18n.KeyLauncherImportEmptySessions)
						case "skills":
							empty = i18n.T(i18n.KeyLauncherImportEmptySkills)
						case "plugins":
							empty = i18n.T(i18n.KeyLauncherImportEmptyPlugins)
						case "mcp":
							empty = i18n.T(i18n.KeyLauncherImportEmptyMCP)
						case "settings":
							empty = i18n.T(i18n.KeyLauncherImportEmptySettings)
						case "presets":
							empty = i18n.T(i18n.KeyLauncherImportEmptyPresets)
						}
						ui.Text(c, empty).FontSize(theme.InterfaceSize(13)).TextColor(pal.tertiary)
					} else if importCatActive == "sessions" {
						importSessionGroups(c, pal, *itemsPtr)
					} else {
						for i := range *itemsPtr {
							idx := i
							it := &(*itemsPtr)[idx]
							ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
								label := it.Label
								if it.Group != "" && importCatActive == "sessions" {
									label = it.Group + " / " + it.Label
								} else if it.Group != "" {
									label = it.Label + " / " + it.Group
								}
								if it.Conflict {
									label += i18n.T(i18n.KeyLauncherImportConflict)
								}
								if !it.Blocker.Importable() {
									label += i18n.T(i18n.KeyLauncherImportUnsupported)
								}
								if ui.Checkbox(c, &it.Selected, label).FontSize(theme.InterfaceSize(12)).FillWidth().MinWidth(0).Disabled(!it.Blocker.Importable()).Changed() {
									c.Invalidate()
								}

							})
						}
					}
				})
			})
		})

		overwrite := m.ImportOverwrite
		if overwrite == nil {
			overwrite = &importOverwriteFB
		}
		attachments := m.ImportAttachments
		if attachments == nil {
			attachments = &importAttachmentsFB
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Wrap().Children(func() {
			ui.Column(c).Gap(8).Grow(1).Children(func() {
				switchCheck(c, pal, i18n.T(i18n.KeyLauncherImportOverwrite), i18n.T(i18n.KeyLauncherImportOverwriteDesc), overwrite, nil)
				switchCheck(c, pal, i18n.T(i18n.KeyLauncherImportAttachments), i18n.T(i18n.KeyLauncherImportAttachmentsDesc), attachments, nil)
			})
			if m.ImportCatalog != nil {
				counts = m.ImportCatalog.CategoryCounts()
			}
			selLine := i18n.T(i18n.KeyLauncherImportSelected)
			for _, cat := range []struct{ id, name string }{
				{"sessions", i18n.T(i18n.KeyLauncherImportCatSessions)},
				{"skills", i18n.T(i18n.KeyLauncherImportCatSkills)},
				{"plugins", i18n.T(i18n.KeyLauncherImportCatPlugins)},
				{"mcp", i18n.T(i18n.KeyLauncherImportCatMCP)},
				{"settings", i18n.T(i18n.KeyLauncherImportCatSettings)},
				{"presets", i18n.T(i18n.KeyLauncherImportCatPresets)},
			} {
				n := 0
				if pair, ok := counts[cat.id]; ok {
					n = pair[0]
				} else if m.ImportCategories != nil && m.ImportCategories[cat.id] {
					n = 1
				}
				selLine += fmt.Sprintf(" %s %d ·", cat.name, n)
			}
			selLine = strings.TrimSuffix(selLine, " ·")
			ui.Text(c, selLine).FontSize(theme.InterfaceSize(11)).TextColor(pal.tertiary)
			if primaryBtn(c, pal, i18n.T(i18n.KeyLauncherImportStart)).Disabled(m.ImportScanBusy).Clicked() {
				if m.OnStartImport != nil {
					m.OnStartImport()
				} else if m.OnScanImport != nil {
					m.OnScanImport()
				}
			}
		})
		scope := releaseText("保留目标中的同名项及本地凭据，不重装已经安装的插件。", "Keep existing items and local credentials; keep installed plugins.")
		if *overwrite {
			scope = releaseText("覆盖所选中的同名会话、技能、MCP、预设和设置，包括全局指令；保留目标本地凭据，重新安装已勾选插件。", "Replace selected sessions, skills, MCP, presets and settings, including global instructions; retain destination local credentials and reinstall selected plugins.")
		}
		if *attachments {
			scope += releaseText(" 复制整个 attachments/ 目录，也包含未勾选会话的附件。", " Copy the entire attachments/ directory, including attachments of unselected sessions.")
			if *overwrite {
				scope += releaseText(" 同路径附件将被覆盖。", " Replace attachments with the same path.")
			} else {
				scope += releaseText(" 保留同路径已有附件。", " Keep existing attachments with the same path.")
			}
		} else {
			scope += releaseText(" 不导入附件。", " Do not import attachments.")
		}
		ui.Text(c, scope).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		if len(m.ImportJournal) > 0 {
			chrome.FrostSurface(ui.Column(c).FillWidth().Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.base, 12).
				Radius(12).Padding(10, 12, 10, 12).Gap(4).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyLauncherImportLog)).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(pal.primary).Grow(1)
					if m.OnCopyImportJournal != nil {
						if ghostSmall(c, pal, i18n.T(i18n.KeyLauncherImportCopy)).Clicked() {
							m.OnCopyImportJournal(append([]string(nil), m.ImportJournal...))
						}
					}
				})
				ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(148).Children(func() {
					ui.Column(c).FillWidth().Gap(4).Children(func() {
						for _, line := range m.ImportJournal {
							ui.Text(c, line).FontSize(theme.InterfaceSize(11)).TextColor(pal.tertiary)
						}
					})
				})
			})
		}
	})
}

func pluginsPanel(c *ui.Context, m Model) {
	pal := resolveLauncherPalette(m.ThemeID, m.SystemDark)
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		startupDiagnosticsPanel(c, pal, m)
		if !m.VendorReady {
			chrome.FrostSurface(ui.Column(c).FillWidth().Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.sidebar, 12).
				Padding(14, 16, 14, 16).Gap(6).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherPluginsNeedRuntime)).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(pal.primary)
				ui.Text(c, i18n.T(i18n.KeyLauncherPluginsNeedRuntimeDesc)).FontSize(theme.InterfaceSize(12)).TextColor(pal.tertiary)
			})
		}
		if len(m.DisabledPlugins) == 0 && !m.SkipUserPlugins {
			ui.Text(c, i18n.T(i18n.KeyLauncherPluginsEmpty)).FontSize(theme.InterfaceSize(13)).TextColor(pal.tertiary)
			ui.Box(c).FillWidth().Height(1).Background(pal.borderL1)
		} else {
			ui.Text(c, i18n.T(i18n.KeyLauncherPluginsTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(pal.primary)
			ui.Text(c, joinPlugins(m.DisabledPlugins)).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
		}
		recoveryLine := i18n.T(i18n.KeyLauncherSkipUserPluginsOff)
		if m.SkipUserPlugins {
			recoveryLine = i18n.T(i18n.KeyLauncherSkipUserPluginsOn)
			if r := strings.TrimSpace(m.PluginRecoveryReason); r != "" {
				recoveryLine += " (" + r + ")"
			}
		}
		chrome.FrostSurface(ui.Column(c).FillWidth().Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.field, 12).
			Padding(14, 14, 14, 14).Gap(8).Children(func() {
			kv(c, pal, i18n.T(i18n.KeyLauncherSkipUserPlugins), recoveryLine)
			canRestore := m.SkipUserPlugins
			canSkip := !canRestore && m.StartupDiagnostics.CanRepair && len(m.StartupDiagnostics.RepairNames) > 0
			if !canRestore && !canSkip {
				return
			}
			busy := m.StartBusy || m.ImportBusy || m.StartupRepairBusy
			ui.Row(c).Gap(8).Children(func() {
				if canSkip && ghostSmall(c, pal, i18n.T(i18n.KeyLauncherEnableSkipUserPlugins)).Disabled(busy).Clicked() && m.OnEnableSkipUserPlugins != nil {
					m.OnEnableSkipUserPlugins()
				}
				if canRestore && ghostSmall(c, pal, i18n.T(i18n.KeyLauncherClearPluginRecovery)).Disabled(busy).Clicked() && m.OnClearPluginRecovery != nil {
					m.OnClearPluginRecovery()
				}
				if canRestore && ghostSmall(c, pal, i18n.T(i18n.KeyLauncherRestartHarness)).Disabled(busy).Clicked() && m.OnRestartHarness != nil {
					m.OnRestartHarness()
				}
			})
		})
	})
}

func settingsPanel(c *ui.Context, m Model) {
	pal := resolveLauncherPalette(m.ThemeID, m.SystemDark)
	ui.Column(c).Gap(0).Children(func() {
		if m.LauncherPrefsError != "" {
			ui.Text(c, m.LauncherPrefsError).FontSize(theme.InterfaceSize(12)).TextColor(pal.danger).Padding(0, 0, 12, 0)
		}

		switchRow(c, pal, i18n.T(i18n.KeyLauncherOptRouteTitle), i18n.T(i18n.KeyLauncherOptRouteDesc), func() {
			route := ""
			if m.DownloadRoute != nil {
				route = strings.ToLower(strings.TrimSpace(*m.DownloadRoute))
			}
			if route == "" {
				route = "github"
			}
			// Electron: white bordered capsule; selected chip is light grey-blue (not white-on-grey).
			chrome.FrostSurface(ui.Row(c).Border(1, pal.borderL2), pal.colors, m.SystemDark, pal.base, 14).
				Padding(2, 2, 2, 2).Gap(0).Children(func() {
				seg := func(id, label string) {
					bg := ui.Transparent
					fg := pal.tertiary
					if route == id {
						bg = pal.segActive
						fg = pal.primary
					}
					b := ui.Button(c, label).Disabled(m.RouteUnavailable[id] != "").Height(26).Padding(0, 14, 0, 14).Radius(theme.ContextRadii(c).MD).
						Background(bg).BorderWidth(0).TextColor(fg).FontSize(theme.InterfaceSize(12))
					if b.Clicked() && m.DownloadRoute != nil {
						*m.DownloadRoute = id
						if m.OnSaveLauncherPrefs != nil {
							m.OnSaveLauncherPrefs()
						}
					}
				}
				seg("github", i18n.T(i18n.KeyLauncherRouteGitHub))
				seg("cnb", i18n.T(i18n.KeyLauncherRouteCNB))
			})
		})
		if reason := m.RouteUnavailable["cnb"]; reason != "" {
			ui.Text(c, reason).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary).Padding(0, 0, 12, 0)
		}

		if m.AskOnUpdate != nil {
			switchCheck(c, pal, i18n.T(i18n.KeyLauncherOptAskTitle), i18n.T(i18n.KeyLauncherOptAskDesc), m.AskOnUpdate, m.OnSaveLauncherPrefs)
		}
		applicationUninstallPanel(c, pal, m.ApplicationUninstall)
	})
}

func switchRow(c *ui.Context, pal launcherPalette, title, desc string, trailing func()) {
	// Electron settings rows: copy on the left, control on the trailing edge.
	padY := float32(14)
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).
		Padding(padY, 0, padY, 0).
		BorderWidth(0, 0, 1, 0).BorderColor(pal.borderL1).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(pal.primary).FillWidth()
			ui.Text(c, desc).FontSize(theme.InterfaceSize(12)).TextColor(pal.tertiary).FillWidth()
		})
		if trailing != nil {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(trailing)
		}
	})
}

func switchCheck(c *ui.Context, pal launcherPalette, title, desc string, checked *bool, onSave func()) {
	before := false
	if checked != nil {
		before = *checked
	}
	switchRow(c, pal, title, desc, func() {
		// Electron uses an iOS-style switch; approximate with a track + Absolute knob.
		on := checked != nil && *checked
		track := pal.field
		if on {
			track = pal.accent
		}
		swW, swH := float32(48), float32(28)
		knob := float32(22)
		knobOn := float32(23)
		btn := ui.Button(c, "").Width(swW).Height(swH).Radius(swH / 2).
			Background(track).BorderWidth(0).Clip()
		btn.Children(func() {
			left := float32(3)
			if on {
				left = knobOn
			}
			ui.Box(c).Absolute().Left(left).Top((swH - knob) / 2).Width(knob).Height(knob).Radius(knob / 2).
				Background(pal.base)
		})
		if btn.Clicked() && checked != nil {
			*checked = !*checked
		}
	})
	if checked != nil && *checked != before && onSave != nil {
		onSave()
	}
}

func primaryBtn(c *ui.Context, pal launcherPalette, label string) ui.Element {
	return widgets.Button(c, pal.colors, label, widgets.ButtonSpec{Variant: widgets.ButtonPrimary})
}

func ghostBtn(c *ui.Context, pal launcherPalette, label string) ui.Element {
	return widgets.Button(c, pal.colors, label, widgets.ButtonSpec{Variant: widgets.ButtonOutline})
}

func ghostSmall(c *ui.Context, pal launcherPalette, label string) ui.Element {
	return widgets.Button(c, pal.colors, label, widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall})
}

func kv(c *ui.Context, pal launcherPalette, label, value string) {
	ui.Row(c).Gap(12).Children(func() {
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(pal.tertiary).Width(140)
		ui.Text(c, value).FontSize(theme.InterfaceSize(12)).TextColor(pal.primary).Grow(1)
	})
}

func onOff(v bool) string {
	if v {
		return i18n.T(i18n.KeyLauncherOn)
	}
	return i18n.T(i18n.KeyLauncherOff)
}

func nonempty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func joinPlugins(names []string) string {
	if len(names) == 0 {
		return ""
	}
	out := names[0]
	for i := 1; i < len(names); i++ {
		out += ", " + names[i]
	}
	return out
}

// componentStatusBadge draws the state pill. A row with a message carries it in
// componentStatusDetail, in the row that says so, instead of a toast over the
// launcher.
func componentStatusBadge(c *ui.Context, pal launcherPalette, row components.Row, label string) {
	ui.Row(c).Background(pal.navActive).Padding(1, 8).Radius(999).Children(func() {
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary)
	})
}

// componentStatusDetail is the row's own status message: why the component is
// not running, or what the last action did. It sits under the row it describes.
func componentStatusDetail(c *ui.Context, pal launcherPalette, row components.Row) {
	if row.Message == "" {
		return
	}
	ui.Text(c, row.Message).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(12)).TextColor(pal.secondary).Selectable()
}
