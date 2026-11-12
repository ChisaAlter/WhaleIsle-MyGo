package launcher

import (
	"fmt"
	"runtime"
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

// releaseIdle reports whether no download, install or result state is active.
func releaseIdle(m Model) bool {
	return m.ReleaseStatus == "" && m.ReleaseError == "" && m.ReleaseStagedTag == "" && !m.ReleaseBusy && !m.ReleasePaused
}

// releaseStrip is the single slot for download/install state: one bordered
// strip — status on the left, controls on the right. State transitions swap
// strip content in place; they never add or remove sibling blocks that shift
// surrounding text. Nothing renders while idle.
// This state remains visible after cancellation, a failed download, navigation
// away/back, or declining the installer handoff.
func releaseStrip(c *ui.Context, m Model) {
	if releaseIdle(m) {
		return
	}
	ui.Column(c).FillWidth().Padding(12, 0, 0, 0).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).
			Background(colSidebar).Border(1, colBorderL2).Radius(12).Padding(10, 14).Children(func() {
			releaseStripContent(c, m)
		})
	})
}

// releaseStripContent draws the status column and the controls row. It is also
// used inside the install card, where it swaps with the idle call to action in
// the same slot.
func releaseStripContent(c *ui.Context, m Model) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
		if (m.ReleaseBusy || m.ReleasePaused) && m.ReleaseTotal > 0 {
			pct := float64(m.ReleaseReceived) * 100 / float64(m.ReleaseTotal)
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Progress(c, min(1, max(0, float64(m.ReleaseReceived)/float64(m.ReleaseTotal)))).Grow(1).Height(4)
				ui.Text(c, fmt.Sprintf("%.1f/%.1f MiB (%.0f%%)", float64(m.ReleaseReceived)/(1<<20), float64(m.ReleaseTotal)/(1<<20), pct)).FontSize(fonttheme.InterfaceSize(12)).TextColor(colAccent)
			})
		}
		if m.ReleaseStatus != "" {
			ui.Text(c, m.ReleaseStatus).FillWidth().MinWidth(0).MaxLines(2).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if m.ReleaseError != "" {
			ui.Text(c, m.ReleaseError).FillWidth().MinWidth(0).MaxLines(3).FontSize(fonttheme.InterfaceSize(12)).TextColor(colDanger)
		}
		if !m.ReleaseBusy && !m.ReleasePaused && m.ReleaseStagedTag != "" {
			ui.Text(c, releaseText("已校验安装包：", "Verified installer: ")+m.ReleaseStagedTag).FillWidth().MinWidth(0).MaxLines(2).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		if !m.ReleaseBusy && !m.ReleasePaused && m.ReleaseStagedTag != "" && !m.ManagedDesktop {
			ui.Text(c, releaseText("点击安装即确认：启动已校验安装器、停止 Host 并退出桌面应用；运行中任务仍会单独提示。", "Clicking Install confirms: launch the verified installer, stop the Host and quit the desktop app. Active tasks still receive a separate warning.")).FillWidth().MinWidth(0).MaxLines(3).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
	})
	ui.Row(c).Gap(8).Wrap().Children(func() {
		switch {
		case m.ReleasePaused:
			if m.OnResumeRelease != nil && primaryBtn(c, releaseText("继续下载", "Resume download")).Clicked() {
				m.OnResumeRelease()
			}
			if m.OnCancelRelease != nil && ghostSmall(c, releaseText("取消", "Cancel")).Clicked() {
				m.OnCancelRelease()
			}
		case m.ReleaseBusy && m.ReleasePhase != "installing":
			if m.ReleasePhase == "downloading" && m.OnPauseRelease != nil && ghostSmall(c, releaseText("暂停", "Pause")).Clicked() {
				m.OnPauseRelease()
			}
			if m.OnCancelRelease != nil && ghostSmall(c, releaseText("取消", "Cancel")).Clicked() {
				m.OnCancelRelease()
			}
		case !m.ReleaseBusy && m.ReleaseStagedTag != "":
			action := releaseText("安装并退出", "Install and quit")
			if m.ManagedDesktop {
				action = releaseText("安装并等待完成", "Install and wait")
			}
			if runtime.GOOS == "darwin" {
				action = releaseText("打开磁盘映像并退出", "Open disk image and quit")
			}
			if primaryBtn(c, action).Clicked() && m.OnStartStagedRelease != nil {
				m.OnStartStagedRelease()
			}
		case !m.ReleaseBusy && (m.ReleaseError != "" || m.ReleaseStatus != ""):
			if m.OnInstallRelease != nil && m.ReleaseSelectedPackage != "" {
				pkg := m.ReleaseSelectedPackage
				if ghostSmall(c, releaseText("重试", "Retry")).Clicked() {
					m.OnInstallRelease(pkg)
				}
			} else if m.OnInstallRuntime != nil && ghostSmall(c, releaseText("重试", "Retry")).Clicked() {
				m.OnInstallRuntime()
			}
			if m.OnDismissRelease != nil && ghostSmall(c, releaseText("清除", "Dismiss")).Clicked() {
				m.OnDismissRelease()
			}
		}
	})
}
