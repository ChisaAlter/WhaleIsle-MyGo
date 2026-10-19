package main

import (
	"errors"
	"runtime"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/appuninstall"
	"whaleisle/internal/ui/launcher"
)

func (s *mainState) withApplicationUninstall(win *mygo.Window, m launcher.Model) launcher.Model {
	m.ApplicationUninstall = launcher.ApplicationUninstallModel{Supported: runtime.GOOS == "windows"}
	if !m.ApplicationUninstall.Supported {
		return m
	}
	m.ApplicationUninstall.OnUninstallCurrent = func() { s.uninstallCurrentApplication(win) }
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

// Mirror the staged-update lifecycle: explicit confirmation, resume save,
// admission/drain checks, launch, then final cleanup. Never claim removal.
func (s *mainState) uninstallCurrentApplication(win *mygo.Window) {
	s.mu.Lock()
	if s.taskRestartPending || s.quitting || s.launcherImportBusy {
		s.mu.Unlock()
		return
	}
	s.taskRestartPending = true
	sup := s.sup
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.taskRestartPending = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		plan, err := appuninstall.DetectCurrent()
		if err != nil {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: resumeText("未检测到可安全启动的卸载器", "No verified uninstaller found"), Message: resumeText("当前程序可能为便携版、源码构建或没有新版原生安装标记。未启动任何卸载器。请使用“已安装应用”并核对安装位置。", "This may be a portable/source build or an installation without the native marker. No uninstaller was started. Use Installed apps and check the installation location.") + "\n\n" + err.Error()})
			return
		}
		choice, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: resumeText("卸载当前原生应用", "Uninstall current native app"), Message: resumeText("继续将检查并停止任务、启动此安装的卸载器并退出应用。卸载器仍会要求确认；在其中取消不会恢复已退出的应用。不会清理用户数据目录。安装位置：", "Continuing checks and stops tasks, opens this installation's uninstaller and quits the app. The uninstaller still asks for confirmation; cancelling there will not reopen the app. User data directories are retained. Install location: ") + plan.InstallLocation(), Buttons: []string{resumeText("取消", "Cancel"), resumeText("启动卸载器并退出", "Start uninstaller and quit")}, DefaultButton: 0, CancelButton: 0})
		if err != nil {
			s.taskProtectionError(win, err)
			return
		}
		if choice.Button != 1 || !s.saveResumeBeforeClose(win) {
			return
		}
		did := s.withTaskProtection(win, sup, "quit", func() error {
			s.mu.Lock()
			changed := s.launcherImportBusy || s.quitting || s.sup != sup
			s.mu.Unlock()
			if changed {
				return errors.New("runtime or import state changed; retry uninstall")
			}
			if err := appuninstall.Start(plan); err != nil {
				return err
			}
			s.finalizeQuit(sup)
			return nil
		})
		if did {
			s.quitAfterCleanup()
		}
	}()
	if win != nil {
		win.Update(func() {})
	}
}
