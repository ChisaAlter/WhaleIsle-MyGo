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

// This state remains visible after cancellation, a failed download, navigation
// away/back, or declining the installer handoff.
func nativeReleaseStatus(c *ui.Context, m Model) {
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
		if (m.ReleaseBusy || m.ReleasePaused) && m.ReleaseTotal > 0 {
			pct := float64(m.ReleaseReceived) * 100 / float64(m.ReleaseTotal)
			ui.Progress(c, min(1, max(0, float64(m.ReleaseReceived)/float64(m.ReleaseTotal)))).FillWidth().Height(4)
			ui.Text(c, fmt.Sprintf("%.1f / %.1f MiB (%.0f%%)", float64(m.ReleaseReceived)/(1<<20), float64(m.ReleaseTotal)/(1<<20), pct)).FontSize(fonttheme.InterfaceSize(12)).TextColor(colAccent)
		}

	})
}

func nativeReleaseOperation(c *ui.Context, m Model) {
	nativeReleaseStatus(c, m)
	nativeReleaseActions(c, m)
}

func nativeReleaseActions(c *ui.Context, m Model) {
	if m.ReleaseBusy && m.ReleasePhase != "installing" && m.OnCancelRelease != nil {
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if m.ReleasePhase == "downloading" && m.OnPauseRelease != nil && ghostSmall(c, releaseText("暂停下载", "Pause download")).Clicked() {
				m.OnPauseRelease()
			}
			if ghostSmall(c, releaseText("取消下载 / 读取", "Cancel download / lookup")).Clicked() {
				m.OnCancelRelease()
			}
		})
	} else if m.ReleasePaused {
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if m.OnResumeRelease != nil && primaryBtn(c, releaseText("继续下载", "Resume download")).Clicked() {
				m.OnResumeRelease()
			}
			if m.OnCancelRelease != nil && ghostSmall(c, releaseText("取消下载", "Cancel download")).Clicked() {
				m.OnCancelRelease()
			}
		})
	} else if !m.ReleaseBusy && m.ReleaseStagedTag != "" {
		if !m.ManagedDesktop {
			ui.Text(c, releaseText("点击安装即确认：启动已校验安装器、停止 Host 并退出桌面应用；运行中任务仍会单独提示。", "Clicking Install confirms: launch the verified installer, stop the Host and quit the desktop app. Active tasks still receive a separate warning.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			ui.Text(c, releaseText("已校验安装包：", "Verified installer: ")+m.ReleaseStagedTag).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
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
		})
	}
}
