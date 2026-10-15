package launcher

import "github.com/egoist/mygo/ui"

// ApplicationUninstallModel deliberately carries no installed/uninstalled state:
// the operating system owns the user's selection, confirmation and cancellation.
type ApplicationUninstallModel struct {
	Supported      bool
	OnOpenSettings func()
}

func applicationUninstallPanel(c *ui.Context, m ApplicationUninstallModel) {
	description := releaseText("打开 Windows 的“已安装应用”。请核对所选应用及安装位置，再由你完成卸载；此入口不会自动卸载、清理数据或退出当前应用。", "Open Windows Installed apps. Check the selected application and install location, then complete uninstall there. This entry does not automatically uninstall, delete data or quit the app.")
	if !m.Supported {
		description = releaseText("此入口仅支持 Windows。请使用当前系统的应用管理方式卸载。", "This entry is only available on Windows. Use your operating system's application management to uninstall.")
	} else if m.OnOpenSettings == nil {
		description = releaseText("暂时无法打开系统页面。请手动进入 Windows 设置 → 应用 → 已安装应用。", "The system page cannot be opened here. Open Windows Settings > Apps > Installed apps manually.")
	}
	switchRow(c, releaseText("卸载应用", "Uninstall application"), description, func() {
		if ghostSmall(c, releaseText("打开“已安装应用”", "Open Installed apps")).Disabled(!m.Supported || m.OnOpenSettings == nil).Clicked() && m.Supported && m.OnOpenSettings != nil {
			m.OnOpenSettings()
		}
	})
}
