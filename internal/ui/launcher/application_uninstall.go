package launcher

import "github.com/egoist/mygo/ui"

// Discovery runs only after an explicit click. Starting an uninstaller is not
// an uninstall-completion receipt. The OS-settings fallback remains available.
type ApplicationUninstallModel struct {
	Supported          bool
	OnOpenSettings     func()
	OnUninstallCurrent func()
}

func applicationUninstallPanel(c *ui.Context, m ApplicationUninstallModel) {
	description := releaseText("可检测当前原生安装并确认启动卸载器；这会停止任务并退出应用。也可打开“已安装应用”手动核对并卸载。", "Detect the current native installation and confirm starting its uninstaller; this stops tasks and quits the app. Alternatively, open Installed apps to check and uninstall manually.")
	if !m.Supported {
		description = releaseText("此入口仅支持 Windows。请使用当前系统的应用管理方式卸载。", "This entry is only available on Windows. Use your operating system's application management to uninstall.")
	} else if m.OnOpenSettings == nil {
		description = releaseText("暂时无法打开系统页面。请手动进入 Windows 设置 → 应用 → 已安装应用。", "The system page cannot be opened here. Open Windows Settings > Apps > Installed apps manually.")
	}
	switchRow(c, releaseText("卸载应用", "Uninstall application"), description, func() {
		if m.OnUninstallCurrent != nil {
			if ghostSmall(c, releaseText("检测并卸载当前应用", "Detect and uninstall current app")).Disabled(!m.Supported).Clicked() && m.Supported {
				m.OnUninstallCurrent()
			}
		}
		if ghostSmall(c, releaseText("打开“已安装应用”", "Open Installed apps")).Disabled(!m.Supported || m.OnOpenSettings == nil).Clicked() && m.Supported && m.OnOpenSettings != nil {
			m.OnOpenSettings()
		}
	})
}
