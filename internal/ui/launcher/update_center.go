package launcher

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/theme"
)

// UpdateCenter is shared by the real update modal and Versions. Release rows
// stay compact; notes and package details expand only for the selected row.
func UpdateCenter(c *ui.Context, m Model) {
	updateCenter(c, m, false)
}

// UpdateCenterBody leaves download/install actions in the modal's fixed footer.
func UpdateCenterBody(c *ui.Context, m Model) {
	updateCenter(c, m, true)
}

func UpdateCenterFooter(c *ui.Context, m Model) {
	if m.SystemDark {
		applyDarkTokens()
	} else {
		applyLightTokens()
	}
	releaseStrip(c, m)
}

func updateCenter(c *ui.Context, m Model, fixedFooter bool) {
	if m.SystemDark {
		applyDarkTokens()
	} else {
		applyLightTokens()
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		ui.Text(c, releaseText("当前版本：", "Current version: ")+nonempty(m.Version, "dev")).FontSize(theme.InterfaceSize(14)).TextColor(colPrimary)
		ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
			enabled := m.ReleaseAutoDownload
			if ui.Checkbox(c, &enabled, releaseText("有新版本时在后台自动下载", "Download new versions automatically in the background")).Grow(1).Changed() && m.OnSetAutoDownload != nil {
				m.OnSetAutoDownload(enabled)
			}
			if ghostSmall(c, releaseText("检查更新", "Check for updates")).Disabled(m.ReleaseBusy).Clicked() && m.OnRefreshReleases != nil {
				m.OnRefreshReleases()
			}
		})
		if !fixedFooter {
			releaseStrip(c, m)
		}
		latest := ""
		for _, release := range m.Releases {
			if !release.Prerelease && release.UnavailableReason == "" {
				latest = release.Version
				break
			}
		}
		for _, release := range m.Releases {
			r := release
			row := ui.Column(c.Key("release-"+r.PackageKey())).FillWidth().MinWidth(0).Border(1, colBorderL2).Radius(12).Clip()
			expanded := ui.Local(row, "expanded", func() bool { return false })
			row.Children(func() {
				header := ui.Button(c, "").FillWidth().MinWidth(0).Background(colSidebar).BorderWidth(0).Padding(12).Label(r.Tag).Expanded(*expanded)
				header.Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, r.Tag).FontSize(theme.InterfaceSize(14)).FontWeight(600).TextColor(colPrimary)
						if update.CompareVersions(r.Version, m.Version) == 0 {
							releaseBadge(c, releaseText("当前", "Current"))
						}
						if r.Version == latest {
							releaseBadge(c, releaseText("最新", "Latest"))
						}
						if r.Prerelease {
							releaseBadge(c, releaseText("预发布", "Prerelease"))
						}
						ui.Box(c).Grow(1)
						if r.PublishedAt != "" {
							ui.Text(c, releaseDate(r.PublishedAt)).FontSize(theme.InterfaceSize(11)).TextColor(colTertiary)
						}
						arrow := "⌄"
						if *expanded {
							arrow = "⌃"
						}
						ui.Text(c, arrow).TextColor(colSecondary)
					})
				})
				if header.Clicked() {
					*expanded = !*expanded
				}
				if !*expanded {
					return
				}
				ui.Column(c).FillWidth().MinWidth(0).Padding(12).Gap(8).Children(func() {
					variant := releaseText("完整安装包", "Full installer")
					if r.IsDelta() {
						variant = releaseText("差量更新", "Delta update")
					}
					if r.RuntimeIncluded {
						variant += releaseText(" · 包含 Host 运行时", " · Includes Host runtime")
					}
					ui.Text(c, fmt.Sprintf("%s · %s/%s · %.1f MiB", variant, r.Platform, r.Arch, float64(r.DownloadBytes())/(1<<20))).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
					if r.Delta != nil {
						ui.Text(c, releaseText("基于版本：", "Base version: ")+r.Delta.BaseVersion).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
					}
					releaseNotes(c, r)
					unavailable := r.UnavailableReason
					if unavailable == "" && !m.VendorReady && !r.RuntimeIncluded {
						unavailable = releaseText("此发布不包含 Host 运行时。", "This release does not contain the Host runtime.")
					}
					if unavailable != "" {
						ui.Text(c, unavailable).FontSize(theme.InterfaceSize(12)).TextColor(colSecondary)
					}
					if primaryBtn(c, releaseText("下载此版本", "Download this version")).Disabled(m.ReleaseBusy || unavailable != "").Clicked() && m.OnInstallRelease != nil {
						if m.OnSelectRelease != nil {
							m.OnSelectRelease(r.PackageKey())
						}
						m.OnInstallRelease(r.PackageKey())
					}
				})
			})
		}
		if len(m.Releases) == 0 && !m.ReleaseBusy {
			ui.Text(c, releaseText("暂无兼容发布。", "No compatible releases available.")).TextColor(colTertiary)
		}
	})
}

func releaseBadge(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(theme.InterfaceSize(11)).TextColor(colAccent).Background(colNavActive).Padding(2, 6).Radius(4)
}
func releaseDate(raw string) string {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	return raw
}
func releaseNotes(c *ui.Context, r update.NativeRelease) {
	seat := ui.Column(c).FillWidth().MinWidth(0).Gap(8)
	lang := ui.Local(seat, "notes-language", func() string {
		if releaseText("zh", "en") == "zh" {
			return "zh"
		}
		return "en"
	})
	seat.Children(func() {
		if r.NotesZH != "" && r.NotesEN != "" {
			ui.Row(c).Gap(8).Role(ui.RoleTabList).FocusGroup(ui.Horizontal).Children(func() {
				for _, option := range []struct{ id, label string }{{"zh", "中文"}, {"en", "English"}} {
					if ghostSmall(c, option.label).Role(ui.RoleTab).Checked(*lang == option.id).Clicked() {
						*lang = option.id
					}
				}
			})
		}
		notes := r.Notes
		if *lang == "zh" && r.NotesZH != "" {
			notes = r.NotesZH
		}
		if *lang == "en" && r.NotesEN != "" {
			notes = r.NotesEN
		}
		if notes != "" {
			ui.Text(c, notes).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(colSecondary).Selectable()
		}
	})
}
