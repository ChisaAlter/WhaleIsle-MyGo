package settings

import (
	"fmt"
	"strings"

	"whaleisle/internal/credentials"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

// AboutAccess is the dedicated About settings page.
type AboutAccess struct {
	Loading, UpdateBusy, HomeBusy       bool
	Err, Status, UpdateStatus, HomePath string
	OnInstallUpdate                     func()
	OnRevealHome                        func() error
	ReadCredentialStorage               func() credentials.StorageStatus
	OnOpenLink                          func(string) error
	AppName                             string
	Version                             string
	Identifier                          string
	UserData                            string
	VendorRoot                          string
	OnRevealUserData                    func() error
	OnRevealLogs                        func() error
	OnRevealVendor                      func() error
	OnCheckUpdates                      func()
	OnCopyUserData                      func()
	OnCopyVersion                       func()
	AnalyticsEnabled                    *bool
	OnRefreshAnalytics                  func()
	UpdatePhase                         string
	UpdateVersion                       string
	UpdatePercent                       float64
	OnDownloadUpdate                    func()
	OnResumeUpdate                      func()
	OnOpenUpdateCenter                  func()
}

func aboutUpdatePhaseSummary(a *AboutAccess) string {
	if a == nil {
		return ""
	}
	switch a.UpdatePhase {
	case "checking":
		return providerCopy("正在检测更新…", "Checking for updates…")
	case "current":
		if latest := strings.TrimSpace(a.UpdateVersion); latest != "" {
			return providerCopy("已是最新版本 "+latest, "You are on the latest version "+latest)
		}
		return providerCopy("已是最新版本", "You are on the latest version")
	case "available":
		if latest := strings.TrimSpace(a.UpdateVersion); latest != "" {
			return providerCopy("发现新版本 "+latest, "New version "+latest+" is available")
		}
		return providerCopy("发现新版本", "New version available")
	case "downloading":
		return fmt.Sprintf(providerCopy("正在下载 %.0f%%", "Downloading %.0f%%"), a.UpdatePercent)
	case "paused":
		if a.UpdatePercent > 0 {
			return fmt.Sprintf("%s %.0f%%", providerCopy("下载已暂停", "Download paused"), a.UpdatePercent)
		}
		return providerCopy("下载已暂停", "Download paused")
	case "verifying":
		return providerCopy("正在校验更新包…", "Verifying update…")
	case "ready":
		return providerCopy("更新已就绪，重启后完成安装", "Update ready to install")
	case "installing":
		return providerCopy("已启动安装程序，请按向导完成安装。", "Installer launched. Follow the wizard to finish.")
	case "error":
		return providerCopy("检测或安装失败", "Update failed")
	default:
		return ""
	}
}

func aboutUpdateCenterSubtitle(a *AboutAccess) string {
	if a == nil {
		return ""
	}
	version := strings.TrimSpace(a.UpdateVersion)
	if version == "" {
		version = strings.TrimSpace(a.Version)
	}
	current := strings.TrimSpace(a.Version)
	percent := fmt.Sprintf("%.0f", a.UpdatePercent)
	switch a.UpdatePhase {
	case "idle":
		if current != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterIdle, "current", current)
		}
	case "current":
		if current != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterCurrent, "current", current)
		}
	case "available":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterAvailable, "version", version)
		}
	case "downloading":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterDownloading, "version", version, "percent", percent)
		}
	case "paused":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterPaused, "version", version, "percent", percent)
		}
	case "verifying":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterVerifying, "version", version)
		}
	case "ready":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterReady, "version", version)
		}
	case "installing":
		if version != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterInstalling, "version", version)
		}
	case "error":
		msg := strings.TrimSpace(a.UpdateStatus)
		if msg == "" {
			msg = strings.TrimSpace(a.Err)
		}
		if msg != "" {
			return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterError, "message", msg)
		}
	}
	return ""
}

func aboutUpdateCenterErrorAlt(a *AboutAccess) string {
	if a == nil || a.UpdatePhase != "error" {
		return ""
	}
	msg := strings.TrimSpace(a.UpdateStatus)
	if msg == "" {
		msg = strings.TrimSpace(a.Err)
	}
	if msg == "" {
		return ""
	}
	return i18n.SettingsParityFormat(i18n.KeySettingsParityAboutCenterErrorAlt, "message", msg)
}

func aboutPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.About == nil {
		return
	}
	a := m.About
	t := tk(c)
	anchor := ui.Column(c.Key("about-heading")).FillWidth().MaxWidth(560).MinWidth(0).Gap(design.Sp(3))
	storageState := ui.Local(anchor, "metadata-async", func() credentials.StorageStatusProbe {
		return credentials.StorageStatusProbe{}
	})
	started, storageLoading, storageLoaded, storage := storageState.State()
	if !started && a.ReadCredentialStorage != nil {
		storageState.Start(a.ReadCredentialStorage, c.Invalidate)
		_, storageLoading, storageLoaded, storage = storageState.State()
	}
	linkError := ui.Local(anchor, "error", func() settingsRefusal { return settingsRefusal{} })
	revealResult := ui.Local(anchor, "home-result", func() string { return "" })
	meta := func(text string) {
		kHint(c, text)
	}
	link := func(link aboutLink) {
		button := coss.Button(c, link.Label, coss.ButtonOptions{
			Variant:  coss.ButtonLink,
			Size:     coss.ButtonSM,
			Disabled: a.OnOpenLink == nil,
		}).AlignSelf(ui.Start).Padding(0).TextColor(t.Primary).Tooltip(link.URL)
		if button.Clicked() {
			linkError.refuse(nil)
			if err := a.OnOpenLink(link.URL); err != nil {
				linkError.refuse(err)
			}
		}
	}
	anchor.Children(func() {
		// Keep the primary update status and next action directly under the
		// desktop version. Only local diagnostics belong in the disclosure.
		kPageTitle(c, i18n.T(KeyPass4AboutTitle))
		kBody(c, providerCopy("本桌面应用与官方 Harness。", "This desktop app and official Harness.")).TextColor(muted)
		settingsAlert(c, "about:118", a.Err != "", providerCopy("没能读取更新信息，请检查网络后重试。", "Update information could not be read. Check the connection and retry."), a.Err)
		links := aboutLinks()
		ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3.5)).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1)).Children(func() {
				kText(c, emptyDash(a.AppName), design.TextBase, design.WeightSemibold, fg).FillWidth()
				meta(providerCopy("桌面版本 ", "Desktop version ") + emptyDash(a.Version))
				aboutUpdateRow(c, a, fg, muted)
				meta("Based on DeepSeek Harness")
				storageSummary := credentialStorageSummary(storage)
				if storageLoading && !storageLoaded {
					storageSummary = providerCopy("正在检查…", "Checking…")
				}
				meta(providerCopy("凭据存储：", "Credential storage: ") + storageSummary)
				if a.HomePath != "" {
					meta(providerCopy("运行目录 ", "Runtime folder ") + a.HomePath)
				}
				if a.OnRevealHome != nil {
					ui.Row(c).FillWidth().Margin(design.Sp(2.5), 0, design.Sp(2)).Children(func() {
						label := i18n.T(KeyPass4OpenRuntime)
						folder := kButtonDisabled(c, label, a.HomeBusy).Label(label).AlignSelf(ui.Start)
						if folder.Clicked() {
							if err := a.OnRevealHome(); err != nil {
								*revealResult = errcopy.Sentence(err)
							} else {
								*revealResult = ""
							}
						}
					})
				}
				if *revealResult != "" {
					meta(*revealResult)
				}
				link(links[0])
			})
			link(links[1])
		})
		settingsAlert(c, "about:154", linkError.failed(), linkError.sentence, linkError.detail)
		settingsNotice(c, "about:157", a.Status != "", func() string { return a.Status })
		bodyDisclosure(c, "about-local-details", providerCopy("应用数据与诊断", "Application data and diagnostics"), func() {
			storageDesc := credentialStorageDescription(storage)
			if storageLoading && !storageLoaded {
				storageDesc = providerCopy("正在检查本机凭据存储状态…", "Checking local credential storage status…")
			}
			kHint(c, storageDesc)
			for _, row := range []struct{ id, label, value string }{{"about-detail-id", i18n.T(i18n.KeySettingsAboutID), a.Identifier}, {"about-detail-user-data", i18n.T(i18n.KeySettingsAboutUserData), a.UserData}, {"about-detail-vendor", i18n.T(i18n.KeySettingsAboutVendor), a.VendorRoot}} {
				if row.value != "" {
					element := bodyPreferenceRow(c, "about-detail-"+row.label, row.label, "", fg, muted, func() {
						ui.Text(c, row.value).FillWidth().SingleLine().Tooltip(row.value).Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					})
					settingsSearchTarget(c, row.id, row.label, element)
				}
			}
			if a.AnalyticsEnabled != nil {
				value := i18n.T(i18n.KeySettingsAboutAnalyticsOff)
				if *a.AnalyticsEnabled {
					value = i18n.T(i18n.KeySettingsAboutAnalyticsOn)
				}
				kBody(c, i18n.T(i18n.KeySettingsAboutAnalytics)+": "+value).TextColor(muted)
			}
			if a.ReadCredentialStorage != nil {
				refreshLabel := providerCopy("刷新凭据存储状态", "Refresh credential storage status")
				if kButtonDisabled(c, refreshLabel, storageLoading).AlignSelf(ui.Start).Clicked() {
					storageState.Start(a.ReadCredentialStorage, c.Invalidate)
				}
			}
			ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
				for _, action := range []struct {
					label  string
					run    func()
					reveal func() error
				}{{i18n.T(i18n.KeySettingsAboutRevealData), nil, a.OnRevealUserData}, {i18n.T(i18n.KeySettingsAboutRevealLogs), nil, a.OnRevealLogs}, {i18n.T(i18n.KeySettingsAboutRevealVendor), nil, a.OnRevealVendor}, {i18n.T(i18n.KeySettingsAboutCopyData), a.OnCopyUserData, nil}, {i18n.T(i18n.KeySettingsRefresh), a.OnRefreshAnalytics, nil}} {
					if action.run == nil && action.reveal == nil {
						continue
					}
					if !kButton(c, action.label).Clicked() {
						continue
					}
					if action.reveal != nil {
						if err := action.reveal(); err != nil {
							*revealResult = errcopy.Sentence(err)
						} else {
							*revealResult = ""
						}
						continue
					}
					action.run()
				}
			})
			if hasDesktopDiagnostics(m.Desktop) {
				desktopDiagnostics(c, m.Desktop, fg, muted)
			}
			if m.Store != nil {
				advancedNavigation(c, m, m.withVersion().snapshot(), fg, muted)
			}
		})
	})
}

// aboutUpdateRow is the primary version/update surface immediately below the version.
func aboutUpdateRow(c *ui.Context, a *AboutAccess, fg, muted ui.Color) {
	t := tk(c)
	ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Margin(design.Sp(2.5), 0, design.Sp(2)).AlignItems(ui.Center).Children(func() {
		// The version is data, not a choice: it paints as read-only text
		// instead of a disabled picker with one option.
		ui.Text(c, emptyDash(a.Version)).Label(i18n.T(i18n.KeySettingsAboutCopyVersion)).
			Height(28).Padding(0, design.Sp(2.5)).Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).
			FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Tooltip(a.Version)
		installLabel := i18n.T(KeyPass4OnlineInstall)
		enabled := a.OnInstallUpdate != nil && !a.UpdateBusy
		install := kPrimary(c, installLabel, !enabled).AlignSelf(ui.Start)
		if !enabled {
			install = install.Tooltip(i18n.T(KeyPass4InstallUnavailable))
		}
		if install.Clicked() && enabled {
			a.OnInstallUpdate()
		}
		if a.OnCheckUpdates != nil && kButtonDisabled(c, i18n.T(i18n.KeySettingsDesktopCheckUpdates), a.UpdateBusy).Clicked() {
			a.OnCheckUpdates()
		}
		if a.UpdatePhase == "available" && a.OnDownloadUpdate != nil && kPrimary(c, providerCopy("下载更新", "Download update"), a.UpdateBusy).Clicked() {
			a.OnDownloadUpdate()
		}
		if (a.UpdatePhase == "paused" || a.UpdatePhase == "error") && a.OnResumeUpdate != nil && kPrimary(c, providerCopy("继续下载", "Resume download"), a.UpdateBusy).Clicked() {
			a.OnResumeUpdate()
		}
		if a.UpdatePhase == "ready" && a.OnInstallUpdate != nil && kPrimary(c, providerCopy("安装并重启", "Install and restart"), a.UpdateBusy).Clicked() {
			a.OnInstallUpdate()
		}
		if a.OnOpenUpdateCenter != nil && kButton(c, providerCopy("打开更新中心", "Open Update Center")).Clicked() {
			a.OnOpenUpdateCenter()
		}
		if a.OnOpenUpdateCenter != nil && kButton(c, i18n.T(i18n.KeySettingsParityAboutCenterOpen)).Clicked() {
			a.OnOpenUpdateCenter()
		}
	})
	if summary := aboutUpdatePhaseSummary(a); summary != "" && summary != a.UpdateStatus && !(a.UpdateBusy && a.UpdatePhase == "checking") {
		kHint(c, summary)
	}
	if center := aboutUpdateCenterSubtitle(a); center != "" && center != aboutUpdatePhaseSummary(a) && !(a.UpdateBusy && a.UpdatePhase == "checking") {
		kHint(c, center)
	}
	if altErr := aboutUpdateCenterErrorAlt(a); altErr != "" && altErr != aboutUpdateCenterSubtitle(a) && !(a.UpdateBusy && a.UpdatePhase == "checking") {
		kHint(c, altErr)
	}
	if a.UpdateStatus != "" {
		kHint(c, a.UpdateStatus)
	}
}

// credentialStorageSummary is the one-line value after "凭据存储：".
func credentialStorageSummary(s credentials.StorageStatus) string {
	switch s.Mode {
	case "encrypted":
		return providerCopy("系统钥匙串加密（DPAPI）", "Encrypted by the system keychain (DPAPI)")
	case "plaintext":
		return providerCopy("明文文件（不安全）", "Plaintext file (unsafe)")
	case "missing":
		return providerCopy("尚未创建，保存时使用系统钥匙串加密", "Not created yet; saved credentials use the system keychain")
	default:
		return providerCopy("无法确认", "Unknown")
	}
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

type aboutLink struct{ Label, URL string }

func aboutLinks() []aboutLink {
	return []aboutLink{
		{i18n.T(KeyPass4GitHubReleases), "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases"},
		{providerCopy("官方 Harness", "Official Harness"), "https://github.com/deepseek-ai/deepseek-harness"},
	}
}
func credentialStorageDescription(s credentials.StorageStatus) string {
	text := ""
	switch s.Mode {
	case "plaintext":
		text = providerCopy("警告：现有凭据文件为明文，能访问该文件的人可读取凭据。", "Warning: the existing credential file is plaintext. Anyone with file access can read its credentials.")
	case "encrypted":
		text = providerCopy("已检测到加密封装；尚未验证系统密钥能否解锁。", "Encrypted envelope detected; OS key unlock has not been verified.")
	case "missing":
		text = providerCopy("尚未创建凭据文件。保存需要可用的 Windows DPAPI 密钥；不会回退为明文。", "No credential file exists. Saving requires an available Windows DPAPI key; there is no plaintext fallback.")
	default:
		text = providerCopy("无法确认当前凭据存储状态。", "Current credential storage status is unknown.")
	}
	if !s.SupportedPlatform {
		text += providerCopy(" 当前平台不支持此 DPAPI 写入后端。", " This platform does not support this DPAPI write backend.")
	}
	return text
}
