// Package launcher is the M11 native first-run / maintenance shell.
// Layout mirrors reference/electron launcher.html + launcher.css 1:1.
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
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Tab ids match Electron data-tab values.
const (
	TabHome       = "home"
	TabComponents = "components"
	TabVersions   = "versions"
	TabImport     = "import"
	TabPlugins    = "plugins"
	TabSettings   = "settings"
)

var (
	colPrimary   = chrome.Color("rgb(15, 17, 21)")
	colSecondary = chrome.Color("rgb(97, 102, 107)")
	colTertiary  = chrome.Color("rgb(129, 133, 140)")
	colAccent    = chrome.Color("rgb(65, 118, 230)")
	colSidebar   = chrome.Color("rgb(249, 250, 251)")
	colBase      = chrome.Color("rgb(255, 255, 255)")
	colBorderL1  = chrome.Color("rgba(0, 0, 0, 0.04)")
	colBorderL2  = chrome.Color("rgba(0, 0, 0, 0.10)")
	colNavActive = chrome.Color("rgb(228, 237, 253)")
	colNavHover  = chrome.Color("rgb(241, 243, 245)")
	colPrimaryFg = chrome.Color("rgb(255, 255, 255)")
	colField     = chrome.Color("rgb(245, 246, 247)")
	// Update-available chip (Electron: green pill on version rows).
	colSuccessBG  = chrome.Color("rgb(220, 252, 231)")
	colSuccessFG  = chrome.Color("rgb(22, 101, 52)")
	colWarnBG     = chrome.Color("rgb(255, 247, 230)")
	colWarnBorder = chrome.Color("rgb(230, 190, 120)")
	colKindBG     = chrome.Color("rgb(255, 237, 213)")
	colKindFG     = chrome.Color("rgb(154, 52, 18)")
	colOfficialBG = chrome.Color("rgb(228, 237, 253)")
	colSegActive  = chrome.Color("rgb(235, 238, 242)")
)

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
	Workspace                   string
	VendorReady                 bool
	LauncherPackage             bool // Electron: tray row visible when packaged (not only when desk running)
	DesktopRunning              bool
	RuntimeBusy                 bool
	OnInstallUpdate             func()
	StartBusy                   bool
	StartFailed                 bool
	StartError                  string
	BootLogs                    []string
	PluginLoad                  harness.PluginProgress
	ActiveTab                   string
	ImportSummary               string
	// ImportJournal is the per-item apply log (Electron runImport journal).
	ImportJournal                                                         []string
	UpdateStatus                                                          string
	UpdateAvailable                                                       bool // Electron syncUpdateNotice: only show #home-update when true
	DisabledPlugins                                                       []string
	SkipUserPlugins                                                       bool
	PluginRecoveryReason                                                  string
	StatusNote                                                            string
	OnTab                                                                 func(tab string)
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
	ReleaseStatus, ReleaseError, ReleaseSelectedPackage, ReleaseStagedTag string
	ReleaseReceived, ReleaseTotal                                         int64
	OnRefreshReleases                                                     func()
	OnSelectRelease                                                       func(packageID string)
	OnInstallRuntime                                                      func()
	OnInstallRelease                                                      func(packageID string)
	OnCancelRelease                                                       func()
	OnStartStagedRelease                                                  func()
	OnOpenSettings                                                        func()
	OnEnableSkipUserPlugins                                               func()
	OnClearPluginRecovery                                                 func()
	OnRestartHarness                                                      func()
	WorkspacePtr                                                          *string
	QuitAfterStart                                                        *bool
	CloseToTray                                                           *bool
	AutoStartDesktop                                                      *bool
	AskOnUpdate                                                           *bool
	DownloadRoute                                                         *string // "", "github", "cnb"
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
	// RoutePopOpen controls #rail-route-pop listbox visibility.
	RoutePopOpen *bool
}

var (
	diagOpen            bool
	importOverwriteFB   bool
	importAttachmentsFB bool
	importCatActive     = "sessions"
)

// View paints the Electron-parity launcher shell (rail + stage).
func View(c *ui.Context, m Model) {
	if m.SystemDark {
		applyDarkTokens()
	} else {
		applyLightTokens()
	}
	tab := m.ActiveTab
	if tab == "" {
		tab = TabHome
	}

	// launcher.css .shell: border-radius 20 + inset hairline on the window fill.
	ui.Column(c).Fill().Background(colBase).Radius(20).Border(1, colBorderL2).Children(func() {
		ui.Row(c).Fill().Children(func() {
			rail(c, m, tab)
			stage(c, m, tab)
		})
	})
	ui.Overlay(c, func() {
		// launcher.css .caption: 48px drag strip (trailing WCO reserve).
		ui.Box(c).Absolute().Left(0).Top(0).Right(120).Height(48).DragWindow()
	})
}

func applyLightTokens() {
	colPrimary = chrome.Color("rgb(15, 17, 21)")
	colSecondary = chrome.Color("rgb(97, 102, 107)")
	colTertiary = chrome.Color("rgb(129, 133, 140)")
	colAccent = chrome.Color("rgb(65, 118, 230)")
	colSidebar = chrome.Color("rgb(249, 250, 251)")
	colBase = chrome.Color("rgb(255, 255, 255)")
	colBorderL1 = chrome.Color("rgba(0, 0, 0, 0.04)")
	colBorderL2 = chrome.Color("rgba(0, 0, 0, 0.10)")
	colNavActive = chrome.Color("rgb(228, 237, 253)")
	colNavHover = chrome.Color("rgb(241, 243, 245)")
	colPrimaryFg = chrome.Color("rgb(255, 255, 255)")
	colField = chrome.Color("rgb(245, 246, 247)")
	colSuccessBG = chrome.Color("rgb(220, 252, 231)")
	colSuccessFG = chrome.Color("rgb(22, 101, 52)")
	colWarnBG = chrome.Color("rgb(255, 247, 230)")
	colWarnBorder = chrome.Color("rgb(230, 190, 120)")
	colKindBG = chrome.Color("rgb(255, 237, 213)")
	colKindFG = chrome.Color("rgb(154, 52, 18)")
	colOfficialBG = chrome.Color("rgb(228, 237, 253)")
	colSegActive = chrome.Color("rgb(235, 238, 242)")
}

func applyDarkTokens() {
	th := theme.Resolve("deepseek.dark").Dark
	colPrimary = chrome.Color(th.FG)
	colSecondary = chrome.Color(th.Muted)
	colTertiary = chrome.Color(th.Muted)
	colAccent = chrome.Color(th.Accent)
	colSidebar = chrome.Color(th.Sidebar)
	colBase = chrome.Color(th.BG)
	colBorderL1 = chrome.Color(th.Border)
	colBorderL2 = chrome.Color(th.Border)
	colNavActive = chrome.Color(th.Hover)
	colNavHover = chrome.Color(th.Hover)
	colPrimaryFg = chrome.Color(th.ButtonFG)
	colField = chrome.Color(th.Field)
	// Dark: muted green chip on field (avoid light-only mint wash).
	colSuccessBG = chrome.Color("rgba(34, 197, 94, 0.18)")
	colSuccessFG = chrome.Color("rgb(134, 239, 172)")
	colWarnBG = chrome.Color("rgba(245, 158, 11, 0.16)")
	colWarnBorder = chrome.Color("rgba(245, 158, 11, 0.45)")
	colKindBG = chrome.Color("rgba(251, 146, 60, 0.18)")
	colKindFG = chrome.Color("rgb(253, 186, 116)")
	colOfficialBG = chrome.Color(th.Hover)
	colSegActive = chrome.Color(th.Hover)
}

func rail(c *ui.Context, m Model, tab string) {
	railPadTop := float32(56)
	railW := float32(188)
	ui.Column(c).Width(railW).FillHeight().Background(colSidebar).
		BorderWidth(0, 1, 0, 0).BorderColor(colBorderL1).
		Padding(railPadTop, 12, 16, 12).Gap(18).Children(func() {
		railHead(c)
		ui.Column(c).Gap(4).Grow(1).Role(ui.RoleTabList).FocusGroup(ui.Vertical).Children(func() {
			railTab(c, m, TabHome, i18n.T(i18n.KeyLauncherTabHome), tab, 0)
			compBadge := 0
			for _, row := range m.Components {
				// launcher-components.js syncComponentsBadge: updateAvailable || error only.
				if row.UpdateAvailable || row.State == "error" {
					compBadge++
				}
			}
			railTab(c, m, TabComponents, i18n.T(i18n.KeyLauncherTabComponents), tab, compBadge)
			railTab(c, m, TabVersions, i18n.T(i18n.KeyLauncherTabVersions), tab, 0)
			railTab(c, m, TabImport, i18n.T(i18n.KeyLauncherTabImport), tab, 0)
			railTab(c, m, TabPlugins, i18n.T(i18n.KeyLauncherTabPlugins), tab, 0)
			railTab(c, m, TabSettings, i18n.T(i18n.KeyLauncherTabSettings), tab, 0)
		})
		ui.Column(c).FillWidth().Padding(10, 12, 0, 12).
			BorderWidth(1, 0, 0, 0).BorderColor(colBorderL1).Children(func() {
			railRouteChip(c, m)
		})
	})
}

func railRouteLabel(m Model) string {
	route := "GitHub"
	if m.DownloadRoute != nil {
		switch strings.ToLower(strings.TrimSpace(*m.DownloadRoute)) {
		case "cnb":
			route = i18n.T(i18n.KeyLauncherRouteCNB)
		case "github", "":
			route = i18n.T(i18n.KeyLauncherRouteGitHub)
		default:
			route = strings.TrimSpace(*m.DownloadRoute)
		}
	}
	return route
}

func railRouteChip(c *ui.Context, m Model) {
	btn := ui.Button(c, "").FillWidth().Background(ui.Transparent).
		BorderWidth(0).BorderColor(ui.Transparent).
		Padding(4, 6, 4, 6).Radius(6).AlignItems(ui.Center)
	if btn.Hovered() {
		btn.Background(colNavHover)
	}
	btn.Children(func() {
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			globeIcon(c, colTertiary)
			ui.Text(c, i18n.T(i18n.KeyLauncherRouteLabel)).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
			ui.Text(c, railRouteLabel(m)).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(colSecondary)
			ui.Box(c).Grow(1)
			routeChevron(c, colTertiary, false)
		})
	})
	routePopup(c, m, btn, m.RoutePopOpen, true)
}

func railHead(c *ui.Context) {
	// Current launcher.html: a non-wrapping mixed-size wordmark on one baseline.
	ui.Row(c).Gap(8).Padding(0, 8, 0, 8).AlignItems(ui.Center).Children(func() {
		if bmp := brandAvatar(); bmp != nil {
			ui.Image(c, bmp).Width(36).Height(36).Shrink(0).Fit(ui.Contain)
		} else {
			ui.Box(c).Width(36).Height(36).Shrink(0).Radius(18).Background(colAccent.Alpha(0.2))
		}
		ui.Column(c).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, "").FontSize(theme.InterfaceSize(18)).FixedLineHeight(22).Height(22).NoWrap().AlignSelf(ui.Start).Shrink(0).Children(func() {
				ui.Text(c, "鲸").Label("鲸").FontSize(theme.InterfaceSize(18)).FontWeight(700).LetterSpacing(-0.72).TextColor(colPrimary)
				ui.Text(c, "屿").Label("屿").FontSize(theme.InterfaceSize(18)).FontWeight(700).LetterSpacing(-0.72).TextColor(colAccent)
				// An em-space at7 DIP preserves the CSS7px gap within one native text line.
				ui.Text(c, "\u2003").FontSize(theme.InterfaceSize(7))
				ui.Text(c, "WHALE ISLE").Label("WHALE ISLE").FontSize(theme.InterfaceSize(10)).FontWeight(600).LetterSpacing(1.2).TextColor(colPrimary)
			})
			ui.Text(c, i18n.T(i18n.KeyLauncherBrandRole)).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).
				FontWeight(500).LetterSpacing(0.72).TextColor(colTertiary).SingleLine()
		})
	})
}

func railTab(c *ui.Context, m Model, id, label, active string, badge int) {
	bg := ui.Transparent
	on := id == active
	if on {
		bg = colNavActive
	}
	// launcher.css .tab.is-active: wash only (no left accent bar).
	btn := ui.Button(c, "").Role(ui.RoleTab).Checked(on).Label(label).FillWidth().Height(40).Radius(12).Background(bg).
		BorderWidth(0).BorderColor(ui.Transparent).
		Padding(9, 16, 9, 12).AlignItems(ui.Center)
	if btn.Hovered() && !on {
		btn.Background(colNavHover)
	}
	btn.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			tabIcon(c, id, colPrimary)
			ui.Text(c, label).FontSize(theme.InterfaceSize(14)).TextColor(colPrimary).Grow(1)
			if badge > 0 {
				ui.Row(c).Background(colAccent).Padding(0, 5, 0, 5).Radius(999).
					MinWidth(18).Children(func() {
					ui.Text(c, fmt.Sprintf("%d", badge)).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(colPrimaryFg)
				})
			}
		})
	})
	if btn.Clicked() && m.OnTab != nil {
		m.OnTab(id)
	}
}

func stage(c *ui.Context, m Model, tab string) {
	// launcher.css: the padded stage stays fixed; main alone scrolls.
	ui.Column(c).Fill().MinWidth(0).Padding(56, 24, 24, 24).Children(func() {
		ui.Scroll(c).Fill().MinWidth(0).Children(func() {
			ui.Row(c).FillWidth().Children(func() {
				ui.Box(c).Grow(1).MinWidth(0)
				ui.Column(c).FillWidth().MaxWidth(1040).MinWidth(0).Shrink(1).Children(func() {
					stageBody(c, m, tab)
				})
				ui.Box(c).Grow(1).MinWidth(0)
			})
		})
	})
}

func stageBody(c *ui.Context, m Model, tab string) {
	ui.Column(c).FillWidth().Gap(0).Children(func() {
		if tab != TabComponents {
			bridgeAlertSummary(c, m, false)
		}
		switch tab {
		case TabComponents:
			routePageHead(c, m, i18n.T(i18n.KeyLauncherTabComponents), i18n.T(i18n.KeyLauncherComponentsLede))
			componentsPanel(c, m)
		case TabVersions:
			routePageHead(c, m, i18n.T(i18n.KeyLauncherTabVersions), "")
			versionsPanel(c, m)
		case TabImport:
			pageHead(c, i18n.T(i18n.KeyLauncherTabImport), i18n.T(i18n.KeyLauncherImportLede), "", nil)
			importPanel(c, m)
		case TabPlugins:
			pageHead(c, i18n.T(i18n.KeyLauncherTabPlugins),
				i18n.T(i18n.KeyLauncherPluginsUnknownHint), "", nil)
			pluginsPanel(c, m)
		case TabSettings:
			pageHead(c, i18n.T(i18n.KeyLauncherTabSettings), i18n.T(i18n.KeyLauncherSettingsLede), "", nil)
			settingsPanel(c, m)
		default:
			homePanel(c, m)
		}
	})
}

func pageHead(c *ui.Context, title, lede, action string, onAction func()) {
	// launcher.css .home-heading h2: 16/600; p: 12 tertiary; margin-bottom 16.
	titleSize := float32(16)
	headBot := float32(16)
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).Padding(0, 0, headBot, 0).Children(func() {
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(titleSize)).FontWeight(600).TextColor(colPrimary)
			if lede != "" {
				ui.Text(c, lede).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
			}
		})
		if action != "" {
			if onAction != nil {
				if ghostSmall(c, action).Clicked() {
					onAction()
				}
			}
		}
	})
}

func homePanel(c *ui.Context, m Model) {
	// Electron home header CTA jumps to the Versions tab.
	pageHead(c, i18n.T(i18n.KeyLauncherTabHome), i18n.T(i18n.KeyLauncherHomeLede), i18n.T(i18n.KeyLauncherGotoVersions), func() {
		if m.OnTab != nil {
			m.OnTab(TabVersions)
		}
	})

	// Electron home desktop card: text brand (no 48px avatar), version pill
	// top-right; status+CTAs one row; running note under the card actions.
	ui.Column(c).FillWidth().Background(colSidebar).Border(1, colBorderL2).Radius(16).
		Padding(20, 24, 20, 24).Gap(0).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(16).Children(func() {
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "WHALE ISLE").FontSize(theme.InterfaceSize(10)).FontWeight(600).TextColor(colTertiary)
				ui.Text(c, i18n.T(i18n.KeyLauncherDesktopTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(colPrimary)
			})
			badge := strings.TrimSpace(m.Version)
			if badge == "" {
				badge = i18n.T(i18n.KeyLauncherBadgeInstalled)
			} else if !strings.HasPrefix(badge, "v") && !strings.HasPrefix(badge, "V") {
				badge = "v" + badge
			}
			if !m.VendorReady {
				badge = i18n.T(i18n.KeyLauncherBadgeMissing)
			}
			ui.Row(c).Background(colNavActive).Padding(1, 8, 1, 8).Radius(999).Children(func() {
				ui.Text(c, badge).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(colSecondary)
			})
		})

		ui.Text(c, i18n.T(i18n.KeyLauncherDesktopDesc)).FontSize(theme.InterfaceSize(14)).TextColor(colSecondary).
			Padding(12, 0, 16, 0)

		status := i18n.T(i18n.KeyLauncherDesktopStopped)
		dot := colTertiary
		if m.StartBusy {
			status = i18n.T(i18n.KeyLauncherStarting)
			dot = colAccent
		} else if m.DesktopRunning {
			status = i18n.T(i18n.KeyLauncherDesktopRunning)
			dot = chrome.Color("rgb(34, 197, 94)")
		} else if m.StartError != "" {
			status = m.StartError
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Grow(1).Children(func() {
				ui.Box(c).Width(6).Height(6).Radius(3).Background(dot)
				ui.Text(c, status).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
			})
			ui.Row(c).Gap(8).Children(func() {
				if m.DesktopRunning && !m.StartBusy {
					if ghostBtn(c, i18n.T(i18n.KeyLauncherCloseDesktop)).Disabled(m.RuntimeBusy).Clicked() && m.OnStop != nil {
						m.OnStop()
					}
					if ghostBtn(c, i18n.T(i18n.KeyLauncherCheckUpdate)).Clicked() && m.OnCheckUpdate != nil {
						m.OnCheckUpdate()
					}
				} else {
					startLabel := i18n.T(i18n.KeyLauncherStartDesktop)
					if m.StartBusy {
						startLabel = i18n.T(i18n.KeyLauncherStarting)
					} else if m.StartFailed {
						startLabel = i18n.T(i18n.KeyLauncherRetryStart)
					}
					btn := primaryBtn(c, startLabel).Disabled(m.StartBusy || m.RuntimeBusy)
					if btn.Clicked() && m.OnStart != nil {
						m.OnStart()
					}
					if ghostBtn(c, i18n.T(i18n.KeyLauncherCheckUpdate)).Clicked() && m.OnCheckUpdate != nil {
						m.OnCheckUpdate()
					}
				}
			})
		})

		if m.DesktopRunning && !m.StartBusy {
			ui.Text(c, i18n.T(i18n.KeyLauncherRunningNote)).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary).
				Padding(12, 0, 0, 0)
		}
		// launcher.html #home-update + #btn-home-update / ????.
		if m.UpdateAvailable && m.UpdateStatus != "" {
			ui.Column(c).FillWidth().Padding(16, 0, 0, 0).
				BorderWidth(1, 0, 0, 0).BorderColor(colBorderL2).Gap(8).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherUpdateTitle)).FontSize(theme.InterfaceSize(10)).TextColor(colTertiary)
				ui.Text(c, m.UpdateStatus).FontSize(theme.InterfaceSize(14)).TextColor(colPrimary)
				ui.Row(c).Gap(8).Children(func() {
					if primaryBtn(c, i18n.T(i18n.KeyLauncherUpdateNow)).Clicked() {
						if m.OnInstallUpdate != nil {
							m.OnInstallUpdate()
						} else if m.OnCheckUpdate != nil {
							m.OnCheckUpdate()
						}
					}
					if ghostSmall(c, i18n.T(i18n.KeyLauncherViewVersions)).Clicked() && m.OnTab != nil {
						m.OnTab(TabVersions)
					}
				})
			})
		}
	})

	if !m.VendorReady {
		homeInstall(c, m)
	}
	nativeReleaseOperation(c, m)
	if r := strings.TrimSpace(m.PluginRecoveryReason); r != "" {
		ui.Column(c).FillWidth().Padding(24, 0, 0, 0).Children(func() {
			ui.Column(c).FillWidth().Background(colSidebar).Border(1, colBorderL2).Radius(16).
				Padding(16, 20, 16, 20).Gap(10).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherPluginFailTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(colPrimary)
				ui.Text(c, r).FontSize(theme.InterfaceSize(13)).TextColor(colSecondary)
				ui.Row(c).Gap(8).Children(func() {
					if primaryBtn(c, i18n.T(i18n.KeyLauncherPluginFailDisable)).Clicked() && m.OnEnableSkipUserPlugins != nil {
						m.OnEnableSkipUserPlugins()
					}
					if ghostBtn(c, i18n.T(i18n.KeyLauncherPluginFailOpen)).Clicked() && m.OnTab != nil {
						m.OnTab(TabPlugins)
					}
				})
			})
		})
	}
	homeComponents(c, m)
	diag(c, m)
}

func homeInstall(c *ui.Context, m Model) {
	// launcher.html #home-install: route picker + install CTA when runtime missing.
	ui.Column(c).FillWidth().Padding(24, 0, 0, 0).Children(func() {
		ui.Column(c).FillWidth().Background(colSidebar).Border(1, colBorderL2).Radius(16).
			Padding(20, 24, 20, 24).Gap(12).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyLauncherInstallDesktopTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(colPrimary)
			ui.Text(c, i18n.T(i18n.KeyLauncherInstallDesktopDesc)).FontSize(theme.InterfaceSize(13)).TextColor(colSecondary)
			if m.DownloadRoute != nil {
				ui.Row(c).Gap(8).Children(func() {
					for _, id := range []string{"github", "cnb"} {
						label := i18n.T(i18n.KeyLauncherRouteGitHub)
						if id == "cnb" {
							label = i18n.T(i18n.KeyLauncherRouteCNB)
						}
						cur := strings.ToLower(strings.TrimSpace(*m.DownloadRoute))
						if cur == "" {
							cur = "github"
						}
						bg := colField
						if cur == id {
							bg = colNavActive
						}
						opt := ui.Button(c, label).Height(36).Radius(10).Padding(0, 14, 0, 14).
							Background(bg).Border(1, colBorderL2).TextColor(colPrimary).FontSize(theme.InterfaceSize(13))
						routeID := id
						if opt.Clicked() {
							*m.DownloadRoute = routeID
							if m.OnSaveLauncherPrefs != nil {
								m.OnSaveLauncherPrefs()
							}
						}
					}
				})
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if primaryBtn(c, i18n.T(i18n.KeyLauncherInstallDesktopCTA)).Disabled(m.ReleaseBusy).Clicked() && m.OnInstallRuntime != nil {
					m.OnInstallRuntime()
				}
				ui.Text(c, i18n.T(i18n.KeyLauncherInstallDesktopVerify)).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
			})
		})
	})
}

func homeComponents(c *ui.Context, m Model) {
	// Electron #home-components: visible only when installedVersion is set.
	var installed []components.Row
	for _, row := range m.Components {
		if strings.TrimSpace(row.InstalledVersion) != "" {
			installed = append(installed, row)
		}
	}
	if len(installed) == 0 {
		return
	}
	ui.Column(c).FillWidth().Padding(24, 0, 0, 0).Gap(8).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherMyComponents)).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(colPrimary)
				ui.Text(c, i18n.T(i18n.KeyLauncherMyComponentsLede)).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
			})
			if ghostSmall(c, i18n.T(i18n.KeyLauncherManageComponents)).Clicked() && m.OnTab != nil {
				m.OnTab(TabComponents)
			}
		})
		width, _ := c.Size()
		available := min(float32(1040), max(float32(0), width-188-48-2))
		columns := max(1, min(len(installed), int((available+16)/(280+16))))
		ui.Grid(c).FillWidth().Columns(columns).Gap(16).Children(func() {
			for _, row := range installed {
				componentCard(c, m, row, true)
			}
		})
	})
}

func componentsPanel(c *ui.Context, m Model) {
	ui.Column(c).Gap(12).Children(func() {
		bridgeAlertSummary(c, m, true)
		if m.ComponentStatus != "" {
			ui.Text(c, m.ComponentStatus).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if !components.WhaleBridgeManagementSupported() {
			ui.Text(c, releaseText("鲸桥安装和组件管理仅支持 Windows x64。", "WhaleBridge installation and management require Windows x64.")).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		}
		// Electron: compact ????pill, not a full-width bar.
		ui.Row(c).AlignSelf(ui.Start).Children(func() {
			if ghostSmall(c, i18n.T(i18n.KeyUsageRefresh)).Clicked() && m.OnRefreshComponents != nil {
				m.OnRefreshComponents()
			}
		})
		if len(m.Components) == 0 {
			ui.Text(c, i18n.T(i18n.KeyLauncherComponentsEmpty)).FontSize(theme.InterfaceSize(14)).TextColor(colSecondary)
			return
		}
		for _, row := range m.Components {
			componentCard(c, m, row, false)
		}
	})
}

func componentCard(c *ui.Context, m Model, row components.Row, compact bool) {
	pendingAction := m.ComponentPending[row.ID]
	componentBusy := m.ComponentBusy || pendingAction != ""
	statusText := stateLabel(row.State)
	if pendingAction != "" {
		statusText = componentPendingLabel(pendingAction)
	}
	if row.ID == components.HostComponentID {
		hostBundleCard(c, m, row, compact)
		return
	}
	if compact {
		width, _ := c.Size()
		padding := float32(20)
		if width <= 760 {
			padding = 16
		}
		ui.Column(c.Key("home-component-"+row.ID)).MinWidth(0).Background(colSidebar).Border(1, colBorderL2).Radius(12).
			Padding(padding).Gap(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
				componentAvatar(c, row, 36)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, row.Name).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(colPrimary)
						ui.Text(c, "v"+nonempty(row.InstalledVersion, row.Version)).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
						ui.Row(c).Background(colNavActive).Padding(1, 8).Radius(999).Children(func() {
							ui.Text(c, statusText).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
						})
					})
					if row.Description != "" {
						ui.Text(c, row.Description).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
					}
				})
			})
			if row.Message != "" {
				ui.Text(c, row.Message).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
			}
			// launcher-components.js renderHomeRows keeps stop and settings
			// reachable from the home card while an installed component runs.
			if row.State == "running" {
				ui.Row(c).Wrap().Gap(8).Children(func() {
					label := fmt.Sprintf(releaseText("关闭%s", "Close %s"), row.Name)
					if ghostBtn(c, label).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
						m.OnComponentAction(row.ID, "stop")
					}
					if row.ID == "whalebridge" && ghostBtn(c, i18n.T(i18n.KeyLauncherCompOpenSettings)).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
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
					if primaryBtn(c, label).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
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
		BorderWidth(0, 0, 1, 0).BorderColor(colBorderL1).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			componentAvatar(c, row, 40)
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, row.Name).FontSize(theme.InterfaceSize(15)).FontWeight(600).TextColor(colPrimary)
					if row.Kind != "" {
						ui.Row(c).Background(colKindBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, kindLabel(row.Kind)).FontSize(theme.InterfaceSize(11)).TextColor(colKindFG)
						})
					}
					if row.Source == "official" {
						ui.Row(c).Background(colOfficialBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyLauncherCompOfficial)).FontSize(theme.InterfaceSize(11)).TextColor(colAccent)
						})
					} else if row.Source == "bundled" {
						ui.Row(c).Background(colOfficialBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, releaseText("示例", "Sample")).FontSize(theme.InterfaceSize(11)).TextColor(colAccent)
						})
					}
					if row.Orphan {
						ui.Row(c).Background(colWarnBG).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, releaseText("目录外组件", "Outside catalog")).FontSize(theme.InterfaceSize(11)).TextColor(colSecondary)
						})
					}
					if row.UpdateAvailable {
						ui.Row(c).Background(colField).Padding(1, 6, 1, 6).Radius(6).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyLauncherCompUpdatable)).FontSize(theme.InterfaceSize(11)).TextColor(colSecondary)
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
					ui.Text(c, strings.Join(meta, " \u00b7 ")).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
				}
				if row.UpdateAvailable {
					ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyLauncherCompUpdateAvail), row.Version)).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
				}
				if row.Message != "" {
					ui.Text(c, row.Message).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
				}
			})
			ui.Column(c).MaxWidthPercent(55).MinWidth(0).Gap(8).AlignItems(ui.End).Children(func() {
				ui.Row(c).Background(colNavActive).Padding(1, 8, 1, 8).Radius(999).Children(func() {
					ui.Text(c, statusText).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(colSecondary)
				})
				ui.Row(c).MaxWidthPercent(100).MinWidth(0).Wrap().Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
					if row.State == "unsupported" {
						return
					}
					compPrimary := func(label string) ui.Element {
						// Electron ?? ??72�42 device-px @150% ??~48�28 DIP.
						return ui.Button(c, label).Height(28).Padding(0, 10, 0, 10).
							Radius(14).Background(colPrimary).BorderWidth(0).
							TextColor(colPrimaryFg).FontSize(theme.InterfaceSize(12))
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
						if ghostSmall(c, i18n.T(i18n.KeyLauncherCompStop)).Disabled(componentBusy).Clicked() && m.OnComponentAction != nil {
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
						componentManagement(c, m, row, componentBusy)
					}
				})
			})
		})
	})
}

func componentAvatar(c *ui.Context, row components.Row, size float32) {
	if row.ID == "whalebridge" {
		if bmp := brandAvatar(); bmp != nil {
			ui.Image(c, bmp).Width(size).Height(size).Fit(ui.Contain)
			return
		}
	}
	name := []rune(strings.TrimSpace(nonempty(row.Name, row.ID)))
	glyph := "?"
	if len(name) > 0 {
		glyph = strings.ToUpper(string(name[0]))
	}
	ui.Box(c).Width(size).Height(size).Radius(8).Background(colField).Center().Children(func() {
		ui.Text(c, glyph).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(colSecondary)
	})
}

func componentManagement(c *ui.Context, m Model, row components.Row, busy bool) {
	busy = busy || (row.ID == "whalebridge" && !components.WhaleBridgeManagementSupported())
	trigger := ghostSmall(c, i18n.T(i18n.KeyLauncherCompManage)).Disabled(busy)
	open := ui.Local(trigger, "component-manage-open", func() bool { return false })
	if trigger.Clicked() {
		*open = !*open
	}
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		panel.MinWidth(220).Background(colBase).Border(1, colBorderL2).Radius(10).Padding(6).
			AttachTo(trigger, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0)
		ui.Text(c, fmt.Sprintf(releaseText("当前 v%s", "Current v%s"), row.InstalledVersion)).
			FontSize(theme.InterfaceSize(12)).TextColor(colSecondary).Padding(6, 10)
		action := func(id, label string) {
			if ghostSmall(c, label).FillWidth().Justify(ui.Start).Disabled(busy).Clicked() {
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

func pluginHint(p harness.PluginProgress) string {
	if p.Total > 0 {
		line := fmt.Sprintf(i18n.T(i18n.KeyBootPlugins), p.Ready, p.Total)
		if p.Detail != "" {
			line += " · " + p.Detail
		}
		return line
	}
	if p.Phase == "runtime" {
		return i18n.T(i18n.KeyBootPluginsIndeterminate)
	}
	return ""
}

func diag(c *ui.Context, m Model) {
	// Electron #home-recovery stays hidden unless recovery/start failure needs it.
	// StartBusy alone must not open the accordion (live Electron keeps it collapsed).
	show := m.StartFailed || m.StartError != "" || m.StatusNote != "" || m.SkipUserPlugins
	if !show {
		return
	}
	if m.StartFailed {
		diagOpen = true
	}
	ui.Column(c).FillWidth().Padding(24, 0, 0, 0).Gap(8).Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(colBorderL1)
		label := "? " + i18n.T(i18n.KeyLauncherDiagTitle)
		if diagOpen {
			label = "? " + i18n.T(i18n.KeyLauncherDiagTitle)
		}
		sum := ui.Button(c, label).Background(ui.Transparent).BorderWidth(0).
			BorderColor(ui.Transparent).TextColor(colSecondary).FontSize(theme.InterfaceSize(16)).FontWeight(500).
			Padding(0, 0, 0, 0).AlignSelf(ui.Start)
		if sum.Clicked() {
			diagOpen = !diagOpen
		}
		if !diagOpen {
			return
		}
		if m.StatusNote != "" {
			ui.Text(c, m.StatusNote).FontSize(theme.InterfaceSize(13)).TextColor(colAccent)
		}
		ui.Text(c, i18n.T(i18n.KeyLauncherWorkspace)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(colPrimary)
		if m.WorkspacePtr != nil {
			ui.TextInput(c, m.WorkspacePtr).FillWidth().Placeholder(i18n.T(i18n.KeyLauncherWorkspaceHint))
		}
		kv(c, i18n.T(i18n.KeyLauncherVersion), nonempty(m.Version, "dev"))
		kv(c, i18n.T(i18n.KeyLauncherVendor), onOff(m.VendorReady))
		startupDiagnosticsPanel(c, m)
	})
}

func versionsPanel(c *ui.Context, m Model) {
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyLauncherLocalVersion)+": "+nonempty(m.Version, "dev")).FontSize(theme.InterfaceSize(16)).FontWeight(600).TextColor(colPrimary)
		if !m.VendorReady {
			ui.Text(c, releaseText("尚未安装 Host 运行时；请选择包含运行时的 MyGo 发布。", "Host runtime is missing; choose a MyGo release that includes it.")).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
			if primaryBtn(c, i18n.T(i18n.KeyLauncherInstallDesktopCTA)).Disabled(m.ReleaseBusy).Clicked() && m.OnInstallRuntime != nil {
				m.OnInstallRuntime()
			}
		}
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyLauncherVerAllList)).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(colPrimary).Grow(1)
			if ghostSmall(c, i18n.T(i18n.KeyLauncherVerRefresh)).Disabled(m.ReleaseBusy).Clicked() && m.OnRefreshReleases != nil {
				m.OnRefreshReleases()
			}
		})
		ui.Text(c, releaseText("按版本、完整包/差量和运行时选择原生安装包；下载校验后再确认安装。", "Choose a version, full/delta package and runtime variant; confirm installation after verification.")).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
		for _, release := range m.Releases {
			r := release
			unavailable := r.UnavailableReason
			if unavailable == "" && !m.VendorReady && !r.RuntimeIncluded {
				unavailable = releaseText("此发布不包含 Host 运行时。", "This release does not contain the Host runtime.")
			}
			bg := colSidebar
			if m.ReleaseSelectedPackage == r.PackageKey() && unavailable == "" {
				bg = colNavActive
			}
			ui.Column(c).FillWidth().Background(bg).Border(1, colBorderL2).Radius(10).Padding(12).Gap(6).Children(func() {
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, r.Tag).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(colPrimary).Grow(1)
					ui.Text(c, r.Platform+"/"+r.Arch).FontSize(theme.InterfaceSize(11)).TextColor(colTertiary)
					if r.Version == m.Version || "v"+r.Version == m.Version {
						ui.Text(c, i18n.T(i18n.KeyLauncherVerCurrent)).FontSize(theme.InterfaceSize(11)).TextColor(colAccent)
					}
					if r.Prerelease {
						ui.Text(c, releaseText("预发布", "Prerelease")).FontSize(theme.InterfaceSize(11)).TextColor(colSecondary)
					}
					if ghostSmall(c, releaseText("选择", "Select")).Disabled(m.ReleaseBusy || unavailable != "").Clicked() && m.OnSelectRelease != nil {
						m.OnSelectRelease(r.PackageKey())
					}
				})
				label := releaseText("仅桌面端", "Desktop only")
				if r.RuntimeIncluded {
					label = releaseText("包含 Host 与 Node 运行时", "Includes Host and Node runtime")
				}
				if r.IsDelta() && r.Delta != nil {
					label = releaseText("差量更新", "Delta update") + " · " + label
					ui.Text(c, fmt.Sprintf(releaseText("基于 %s · 需下载约 %.1f MiB · 重建 %.1f MiB", "From %s · approximately %.1f MiB to download · %.1f MiB reconstructed"), r.Delta.BaseVersion, float64(r.DownloadBytes())/(1<<20), float64(r.Size)/(1<<20))).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
				} else {
					label = releaseText("完整安装包", "Full installer") + " · " + label
				}
				ui.Text(c, fmt.Sprintf("%s · %.1f MiB", label, float64(r.Size)/(1<<20))).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
				if r.DownloadRoute != "" {
					ui.Text(c, releaseText("下载源：", "Download source: ")+r.DownloadRoute).FontSize(theme.InterfaceSize(11)).TextColor(colTertiary)
				}
				if r.Notes != "" {
					ui.Text(c, r.Notes).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
				}
				if unavailable != "" {
					ui.Text(c, unavailable).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
				}
				if m.ReleaseSelectedPackage == r.PackageKey() && unavailable == "" {
					if primaryBtn(c, releaseText("下载所选发布", "Download selected release")).Disabled(m.ReleaseBusy).Clicked() && m.OnInstallRelease != nil {
						m.OnInstallRelease(r.PackageKey())
					}
				}
			})
		}
		if len(m.Releases) == 0 && !m.ReleaseBusy {
			ui.Text(c, releaseText("暂无兼容发布目录。不会改用 Electron 安装包。", "No compatible release catalog is available.")).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
		}
		nativeReleaseOperation(c, m)
	})
}

func importPanel(c *ui.Context, m Model) {
	if m.ImportBusy {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, "正在导入 · 操作结束前不能再次开始").FontSize(theme.InterfaceSize(14)).TextColor(colPrimary)
			ui.Text(c, m.ImportSummary).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
			ui.Text(c, "取消会在当前项目完成后停止；已完成的数据保留，可安全重扫。 ").FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
			if ghostSmall(c, "取消后续导入").Clicked() && m.OnCancelImport != nil {
				m.OnCancelImport()
			}
			for _, line := range m.ImportJournal {
				ui.Text(c, line).FontSize(theme.InterfaceSize(11)).TextColor(colTertiary)
			}
		})
		return
	}
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyLauncherImportLede)).
			FontSize(12).TextColor(colSecondary)
		ui.Text(c, i18n.T(i18n.KeyLauncherImportChecklist)).FontSize(theme.InterfaceSize(12)).TextColor(colAccent)
		ui.Text(c, importer.CopyOnlyNotice).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		ui.Text(c, releaseText("确认导入后会先检查当前任务并停止 Host 内核；窗口保持打开，导入后可手动启动内核。", "After confirmation, import checks current tasks and stops the Host runtime. The window stays open; start the runtime manually after importing.")).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)

		pathLine := i18n.T(i18n.KeyLauncherImportEmpty)
		if m.ImportSummary != "" {
			pathLine = m.ImportSummary
		}

		ui.Row(c).Gap(8).Children(func() {
			if ghostSmall(c, i18n.T(i18n.KeyLauncherImportPickDir)).Clicked() {
				if m.OnPickImportSource != nil {
					m.OnPickImportSource()
				} else if m.OnScanImport != nil {
					m.OnScanImport()
				}
			}
			if ghostSmall(c, i18n.T(i18n.KeyLauncherImportAddSkillDir)).Clicked() {
				if m.OnAddSkillImportDir != nil {
					m.OnAddSkillImportDir()
				} else if m.OnScanImport != nil {
					m.OnScanImport()
				}
			}
			if ghostSmall(c, i18n.T(i18n.KeyLauncherImportRescan)).Clicked() && m.OnScanImport != nil {
				m.OnScanImport()
			}
		})

		ui.Text(c, pathLine).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)

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
		ui.Row(c).Gap(6).Role(ui.RoleTabList).FocusGroup(ui.Horizontal).Children(func() {
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
				bg := colField
				if importCatActive == cat.id {
					bg = colNavActive
				}
				id := cat.id
				btn := ui.Button(c, label).Role(ui.RoleTab).Checked(importCatActive == id).Background(bg).Border(1, colBorderL2).
					Padding(4, 10, 4, 10).Radius(8).TextColor(colPrimary).FontSize(theme.InterfaceSize(12))
				if btn.Clicked() {
					importCatActive = id
					c.Invalidate()
				}
			}
		})

		ui.Column(c).FillWidth().Background(colBase).Border(1, colBorderL2).Radius(12).
			Padding(12, 12, 12, 12).Gap(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherImportByDir)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(colPrimary).Grow(1)
				if ghostSmall(c, i18n.T(i18n.KeyLauncherImportSelectAll)).Clicked() && m.ImportCatalog != nil {
					if items := m.ImportCatalog.ItemsFor(importCatActive); items != nil {
						for i := range *items {
							if !(*items)[i].Unsupported {
								(*items)[i].Selected = true
							}
						}
					}
					c.Invalidate()
				}
				if ghostSmall(c, i18n.T(i18n.KeyLauncherImportSelectNone)).Clicked() && m.ImportCatalog != nil {
					if items := m.ImportCatalog.ItemsFor(importCatActive); items != nil {
						for i := range *items {
							(*items)[i].Selected = false
						}
					}
					c.Invalidate()
				}
			})
			itemsPtr := (*[]importer.CatalogItem)(nil)
			if m.ImportCatalog != nil {
				itemsPtr = m.ImportCatalog.ItemsFor(importCatActive)
			}
			if importCatActive == "plugins" {
				ui.Text(c, i18n.T(i18n.KeyLauncherImportPluginsNote)).
					FontSize(12).TextColor(colAccent)
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
				ui.Text(c, empty).FontSize(theme.InterfaceSize(13)).TextColor(colTertiary)
			} else if importCatActive == "sessions" {
				importSessionGroups(c, *itemsPtr)
			} else {
				for i := range *itemsPtr {
					idx := i
					it := &(*itemsPtr)[idx]
					ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
						mark := "□"
						if it.Selected {
							mark = "✓"
						}
						ink := colPrimary
						if it.Unsupported {
							ink = colTertiary
						}
						label := it.Label
						if it.Group != "" && importCatActive == "sessions" {
							label = it.Group + " / " + it.Label
						} else if it.Group != "" {
							label = it.Label + " / " + it.Group
						}
						if it.Conflict {
							label += i18n.T(i18n.KeyLauncherImportConflict)
						}
						if it.Unsupported {
							label += i18n.T(i18n.KeyLauncherImportUnsupported)
						}
						if ui.Button(c, mark+"  "+label).FontSize(theme.InterfaceSize(12)).BorderWidth(0).
							Background(ui.Transparent).TextColor(ink).FillWidth().
							Disabled(it.Unsupported).Clicked() {
							it.Selected = !it.Selected
							c.Invalidate()
						}
					})
				}
			}
		})

		overwrite := m.ImportOverwrite
		if overwrite == nil {
			overwrite = &importOverwriteFB
		}
		attachments := m.ImportAttachments
		if attachments == nil {
			attachments = &importAttachmentsFB
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Column(c).Gap(8).Grow(1).Children(func() {
				switchCheck(c, i18n.T(i18n.KeyLauncherImportOverwrite), i18n.T(i18n.KeyLauncherImportOverwriteDesc), overwrite, nil)
				switchCheck(c, i18n.T(i18n.KeyLauncherImportAttachments), i18n.T(i18n.KeyLauncherImportAttachmentsDesc), attachments, nil)
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
			ui.Text(c, selLine).FontSize(theme.InterfaceSize(11)).TextColor(colTertiary)
			if primaryBtn(c, i18n.T(i18n.KeyLauncherImportStart)).Clicked() {
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
		ui.Text(c, scope).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		if len(m.ImportJournal) > 0 {
			ui.Column(c).FillWidth().Background(colBase).Border(1, colBorderL2).
				Radius(12).Padding(10, 12, 10, 12).Gap(4).MaxHeight(200).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyLauncherImportLog)).FontSize(theme.InterfaceSize(12)).FontWeight(600).TextColor(colPrimary).Grow(1)
					if m.OnCopyImportJournal != nil {
						if ghostSmall(c, i18n.T(i18n.KeyLauncherImportCopy)).Clicked() {
							m.OnCopyImportJournal(append([]string(nil), m.ImportJournal...))
						}
					}
				})
				for _, line := range m.ImportJournal {
					ui.Text(c, line).FontSize(theme.InterfaceSize(11)).TextColor(colTertiary)
				}
			})
		}
	})
}

func pluginsPanel(c *ui.Context, m Model) {
	ui.Column(c).FillWidth().Gap(12).Children(func() {
		startupDiagnosticsPanel(c, m)
		if !m.VendorReady {
			ui.Column(c).FillWidth().Background(colSidebar).Border(1, colBorderL2).Radius(12).
				Padding(14, 16, 14, 16).Gap(6).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyLauncherPluginsNeedRuntime)).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(colPrimary)
				ui.Text(c, i18n.T(i18n.KeyLauncherPluginsNeedRuntimeDesc)).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary)
			})
		}
		// Electron: compact ????pill (not a full-width bar).
		ui.Row(c).AlignSelf(ui.Start).Children(func() {
			if ghostSmall(c, i18n.T(i18n.KeyLauncherVerRefresh)).Disabled(m.StartupRepairBusy).Clicked() && m.OnRefreshStartupDiagnostics != nil {
				m.OnRefreshStartupDiagnostics()
			}
		})
		if len(m.DisabledPlugins) == 0 && !m.SkipUserPlugins {
			ui.Text(c, i18n.T(i18n.KeyLauncherPluginsEmpty)).FontSize(theme.InterfaceSize(13)).TextColor(colTertiary)
			ui.Box(c).FillWidth().Height(1).Background(colBorderL1)
		} else {
			ui.Text(c, i18n.T(i18n.KeyLauncherPluginsTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(colPrimary)
			ui.Text(c, joinPlugins(m.DisabledPlugins)).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
		}
		recoveryLine := i18n.T(i18n.KeyLauncherSkipUserPluginsOff)
		if m.SkipUserPlugins {
			recoveryLine = i18n.T(i18n.KeyLauncherSkipUserPluginsOn)
			if r := strings.TrimSpace(m.PluginRecoveryReason); r != "" {
				recoveryLine += " (" + r + ")"
			}
		}
		ui.Column(c).FillWidth().Background(colField).Border(1, colBorderL2).Radius(12).
			Padding(14, 14, 14, 14).Gap(8).Children(func() {
			kv(c, i18n.T(i18n.KeyLauncherSkipUserPlugins), recoveryLine)
			canRestore := m.SkipUserPlugins
			canSkip := !canRestore && m.StartupDiagnostics.CanRepair && len(m.StartupDiagnostics.RepairNames) > 0
			if !canRestore && !canSkip {
				return
			}
			busy := m.StartBusy || m.ImportBusy || m.StartupRepairBusy
			ui.Row(c).Gap(8).Children(func() {
				if canSkip && ghostSmall(c, i18n.T(i18n.KeyLauncherEnableSkipUserPlugins)).Disabled(busy).Clicked() && m.OnEnableSkipUserPlugins != nil {
					m.OnEnableSkipUserPlugins()
				}
				if canRestore && ghostSmall(c, i18n.T(i18n.KeyLauncherClearPluginRecovery)).Disabled(busy).Clicked() && m.OnClearPluginRecovery != nil {
					m.OnClearPluginRecovery()
				}
				if canRestore && ghostSmall(c, i18n.T(i18n.KeyLauncherRestartHarness)).Disabled(busy).Clicked() && m.OnRestartHarness != nil {
					m.OnRestartHarness()
				}
			})
		})
	})
}

func settingsPanel(c *ui.Context, m Model) {
	ui.Column(c).Gap(0).Children(func() {
		hintBot := float32(16)
		ui.Text(c, i18n.T(i18n.KeyLauncherSettingsOnlyHint)).FontSize(theme.InterfaceSize(13)).TextColor(colSecondary).
			Padding(0, 0, hintBot, 0)
		if m.LauncherPrefsError != "" {
			ui.Text(c, m.LauncherPrefsError).FontSize(theme.InterfaceSize(12)).TextColor(colPrimary).Padding(0, 0, 12, 0)
		}

		switchRow(c, i18n.T(i18n.KeyLauncherOptRouteTitle), i18n.T(i18n.KeyLauncherOptRouteDesc), func() {
			route := ""
			if m.DownloadRoute != nil {
				route = strings.ToLower(strings.TrimSpace(*m.DownloadRoute))
			}
			if route == "" {
				route = "github"
			}
			// Electron: white bordered capsule; selected chip is light grey-blue (not white-on-grey).
			ui.Row(c).Background(colBase).Border(1, colBorderL2).Radius(14).
				Padding(2, 2, 2, 2).Gap(0).Children(func() {
				seg := func(id, label string) {
					bg := ui.Transparent
					fg := colTertiary
					if route == id {
						bg = colSegActive
						fg = colPrimary
					}
					b := ui.Button(c, label).Height(26).Padding(0, 14, 0, 14).Radius(12).
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

		if m.QuitAfterStart != nil {
			switchCheck(c, i18n.T(i18n.KeyLauncherOptQuitTitle), i18n.T(i18n.KeyLauncherOptQuitDesc), m.QuitAfterStart, m.OnSaveLauncherPrefs)
		}
		// launcher.js #row-opt-tray: hidden unless launcherPackage === true.
		if m.CloseToTray != nil && m.LauncherPackage {
			switchCheck(c, i18n.T(i18n.KeyLauncherOptTrayTitle), i18n.T(i18n.KeyLauncherOptTrayDesc), m.CloseToTray, m.OnSaveLauncherPrefs)
		}
		if m.AutoStartDesktop != nil {
			switchCheck(c, i18n.T(i18n.KeyLauncherOptAutoTitle), i18n.T(i18n.KeyLauncherOptAutoDesc), m.AutoStartDesktop, m.OnSaveLauncherPrefs)
		}
		if m.AskOnUpdate != nil {
			switchCheck(c, i18n.T(i18n.KeyLauncherOptAskTitle), i18n.T(i18n.KeyLauncherOptAskDesc), m.AskOnUpdate, m.OnSaveLauncherPrefs)
		}
		applicationUninstallPanel(c, m.ApplicationUninstall)
	})
}

func switchRow(c *ui.Context, title, desc string, trailing func()) {
	// Electron settings rows: copy on the left, control on the trailing edge.
	padY := float32(14)
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).
		Padding(padY, 0, padY, 0).
		BorderWidth(0, 0, 1, 0).BorderColor(colBorderL1).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(colPrimary).FillWidth()
			ui.Text(c, desc).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary).FillWidth()
		})
		if trailing != nil {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(trailing)
		}
	})
}

func switchCheck(c *ui.Context, title, desc string, checked *bool, onSave func()) {
	before := false
	if checked != nil {
		before = *checked
	}
	switchRow(c, title, desc, func() {
		// Electron uses an iOS-style switch; approximate with a track + Absolute knob.
		on := checked != nil && *checked
		track := colField
		if on {
			track = colAccent
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
				Background(colBase)
		})
		if btn.Clicked() && checked != nil {
			*checked = !*checked
		}
	})
	if checked != nil && *checked != before && onSave != nil {
		onSave()
	}
}

func primaryBtn(c *ui.Context, label string) ui.Element {
	return ui.Button(c, label).Height(36).Padding(0, 14, 0, 14).
		Radius(18).Background(colPrimary).BorderWidth(0).
		TextColor(colPrimaryFg).FontSize(theme.InterfaceSize(14))
}

func ghostBtn(c *ui.Context, label string) ui.Element {
	return ui.Button(c, label).Height(36).Padding(0, 14, 0, 14).
		Radius(18).Border(1, colBorderL2).Background(ui.Transparent).
		TextColor(colPrimary).FontSize(theme.InterfaceSize(14))
}

func ghostSmall(c *ui.Context, label string) ui.Element {
	return ui.Button(c, label).Height(28).Padding(0, 12, 0, 12).
		Radius(14).Border(1, colBorderL2).Background(ui.Transparent).
		TextColor(colPrimary).FontSize(theme.InterfaceSize(12))
}

func kv(c *ui.Context, label, value string) {
	ui.Row(c).Gap(12).Children(func() {
		ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(colTertiary).Width(140)
		ui.Text(c, value).FontSize(theme.InterfaceSize(12)).TextColor(colPrimary).Grow(1)
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
