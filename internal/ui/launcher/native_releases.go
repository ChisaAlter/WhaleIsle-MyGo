package launcher

import (
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func releaseText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

// This state remains visible after cancellation, a failed download, navigation
// away/back, or declining the installer handoff.
func nativeReleaseOperation(c *ui.Context, m Model) {
	if m.ReleaseStatus == "" && m.ReleaseError == "" && m.ReleaseStagedTag == "" {
		return
	}
	ui.Column(c).FillWidth().Padding(12, 0, 0, 0).Gap(8).Children(func() {
		if m.ReleaseStatus != "" {
			ui.Text(c, m.ReleaseStatus).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if m.ReleaseError != "" {
			ui.Text(c, m.ReleaseError).FontSize(fonttheme.InterfaceSize(12)).TextColor(colPrimary)
		}
		if m.ReleaseBusy && m.ReleaseTotal > 0 {
			pct := float64(m.ReleaseReceived) * 100 / float64(m.ReleaseTotal)
			ui.Text(c, fmt.Sprintf("%.1f / %.1f MiB (%.0f%%)", float64(m.ReleaseReceived)/(1<<20), float64(m.ReleaseTotal)/(1<<20), pct)).FontSize(fonttheme.InterfaceSize(12)).TextColor(colAccent)
		}
		if m.ReleaseBusy {
			if ghostSmall(c, releaseText("取消下载 / 读取", "Cancel download / lookup")).Clicked() && m.OnCancelRelease != nil {
				m.OnCancelRelease()
			}
		} else if m.ReleaseStagedTag != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, releaseText("已校验安装包：", "Verified installer: ")+m.ReleaseStagedTag).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
				if primaryBtn(c, releaseText("安装并退出", "Install and quit")).Clicked() && m.OnStartStagedRelease != nil {
					m.OnStartStagedRelease()
				}
			})
		}
	})
}
