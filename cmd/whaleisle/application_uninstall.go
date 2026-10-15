package main

import (
	"runtime"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/appuninstall"
	"whaleisle/internal/ui/launcher"
)

func withApplicationUninstall(win *mygo.Window, m launcher.Model) launcher.Model {
	m.ApplicationUninstall = launcher.ApplicationUninstallModel{Supported: runtime.GOOS == "windows"}
	if !m.ApplicationUninstall.Supported {
		return m
	}
	m.ApplicationUninstall.OnOpenSettings = func() {
		// Fixed OS URI through MyGo's native ShellExecuteW path, never a command
		// interpreter or an installation/registry-derived executable.
		if err := appuninstall.OpenInstalledApps(runtime.GOOS, mygo.Shell.OpenExternal); err != nil {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{
				Parent:  win,
				Type:    mygo.MessageWarning,
				Title:   resumeText("无法打开已安装应用", "Could not open Installed apps"),
				Message: resumeText("请手动进入 Windows 设置 → 应用 → 已安装应用，核对应用后再自行卸载。当前应用和安装状态未更改。", "Open Windows Settings > Apps > Installed apps manually, check the application, then uninstall it there. The current app and installation state are unchanged.") + "\n\n" + err.Error(),
			})
		}
	}
	return m
}
