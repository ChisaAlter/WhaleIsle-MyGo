package main

// 功能优先；全部功能完成前不运行测试、不做独立审查、不新增防御性扩展

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// Host files use the component lane; actual execution stays with startHarness
// and the existing supervisor, Project bridge, installer and phone lifecycle.
func (s *mainState) changeHostBundle(win *mygo.Window, action string) {
	if !desktopcomponents.HostBundleSupported() || win == nil || !bridgeChangeUI.TryLock() {
		return
	}
	bridgeChangeActive.Store(true)
	go func() {
		defer func() {
			bridgeChangeActive.Store(false)
			bridgeChangeUI.Unlock()
			s.refreshLauncherComponents()
			win.Update(func() {})
		}()
		message := ""
		var err error
		switch action {
		case "install":
			var picked []string
			picked, err = mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent: win, Title: resumeText("选择 macOS Host 离线包", "Choose a macOS Host offline bundle"),
				Filters: []mygo.FileFilter{{Name: "WhaleIsle Host (.tar.gz)", Extensions: []string{"gz"}}},
			})
			if err != nil || len(picked) == 0 {
				break
			}
			var answer mygo.MessageResult
			answer, err = mygo.Dialog.Message(mygo.MessageOptions{
				Parent: win, Type: mygo.MessageQuestion,
				Title: resumeText("安装 Host 离线包", "Install Host offline bundle"),
				Message: fmt.Sprintf(resumeText("来源：%s\n目标：%s\n\n将解包并安装这个本地原生 Host 与配套 Node。请确认文件来源可信。安装不会运行包内程序，也不会停止当前 Host；安装后可选择版本并单独启动。", "Source: %s\nDestination: %s\n\nExtract and install this local native Host and its matching Node. Confirm that you trust the source. Installation does not execute the bundle or stop the current Host. Select a version and start it separately after installation."), picked[0], desktopcomponents.HostBundleVersionsDir(paths.UserDataDir())),
				Buttons: []string{resumeText("取消", "Cancel"), resumeText("安装", "Install")}, DefaultButton: 0, CancelButton: 0,
			})
			if err != nil || answer.Button != 1 {
				break
			}
			s.hostBundleMessage(win, resumeText("正在解包并校验 Host 与配套 Node；当前 Host 保持运行…", "Extracting and checking Host and its matching Node; the current Host stays running…"))
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			var bundle desktopcomponents.HostBundle
			bundle, err = desktopcomponents.InstallHostBundle(ctx, paths.UserDataDir(), picked[0])
			cancel()
			if err == nil {
				message = fmt.Sprintf(resumeText("Host v%s 已安装到 %s。点击“选择已安装版本”，确认选择后再启动。", "Host v%s was installed in %s. Choose “Select installed version”, confirm the selection, then start it."), bundle.Manifest.Version, bundle.Directory)
			}
		case "select":
			var picked []string
			picked, err = mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent: win, Title: resumeText("选择已安装的 Host 版本目录", "Select an installed Host version directory"),
				Directory: true, DefaultPath: desktopcomponents.HostBundleVersionsDir(paths.UserDataDir()),
			})
			if err != nil || len(picked) == 0 {
				break
			}
			s.hostBundleMessage(win, resumeText("正在读取所选 Host 版本…", "Reading the selected Host version…"))
			var bundle desktopcomponents.HostBundle
			bundle, err = desktopcomponents.LoadHostBundle(paths.UserDataDir(), picked[0])
			if err == nil {
				var selected bool
				selected, err = s.selectHostBundle(win, bundle)
				if !selected && err == nil {
					message = resumeText("Host 选择未更改；如操作被任务保护阻止，请查看启动器提示。", "Host selection was not changed. If task protection blocked the action, check the launcher notice.")
				}
				if selected {
					message = fmt.Sprintf(resumeText("已选择 Host v%s。内核保持停止；点击“启动 Host”使用该版本。", "Host v%s is selected. The runtime remains stopped; click “Start Host” to use this version."), bundle.Manifest.Version)
				}
			}
		case "open", "manage":
			err = desktopcomponents.OpenInFileManager(desktopcomponents.HostBundleVersionsDir(paths.UserDataDir()))
		default:
			err = desktopcomponents.ErrUnknownAction
		}
		if err != nil {
			message = resumeText("Host 操作未完成：", "Host action did not complete: ") + err.Error()
		}
		if message != "" {
			s.hostBundleMessage(win, message)
		}
	}()
}

func (s *mainState) hostBundleMessage(win *mygo.Window, message string) {
	s.mu.Lock()
	s.launcherUpdate = message
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) selectHostBundle(win *mygo.Window, bundle desktopcomponents.HostBundle) (bool, error) {
	s.mu.Lock()
	store := s.cfgStore
	sup := s.sup
	busy := s.taskRestartPending || s.launcherImportBusy || s.launcherStartBusy || s.marketWorker != nil || s.marketInstallBusy || s.quitting
	if !busy {
		s.taskRestartPending = true
	}
	s.mu.Unlock()
	if busy {
		return false, fmt.Errorf("其他生命周期操作正在进行，请稍后选择 Host")
	}
	defer func() { s.mu.Lock(); s.taskRestartPending = false; s.mu.Unlock() }()
	if store == nil {
		return false, fmt.Errorf("config store is not ready")
	}
	cfg := store.Get()
	if os.Getenv(paths.EnvVendor) != "" || strings.TrimSpace(cfg.DshBin) != "" || strings.TrimSpace(cfg.NodeBin) != "" {
		return false, fmt.Errorf("请先在启动设置中清空显式 dshBin/nodeBin，并移除 WHALEISLE_VENDOR 覆盖；现有外部运行时未更改")
	}
	generation := 0
	if sup != nil {
		generation = sup.Status().Generation
	}
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: win, Type: mygo.MessageWarning,
		Title: resumeText("选择 Host 版本", "Select Host version"),
		Message: fmt.Sprintf(resumeText("目标：Host v%s (%s)\n\n会通过现有任务保护停止当前 Host 和手机连接，再保存该版本供下次启动使用。旧运行时、会话及配置会保留。选择完成后内核保持停止，需单独点击“启动 Host”。", "Target: Host v%s (%s)\n\nThe existing task protection will stop the current Host and phone connection before saving this version for the next start. Previous runtimes, sessions and settings are retained. The runtime stays stopped until you click “Start Host”."), bundle.Manifest.Version, bundle.Directory),
		Buttons: []string{resumeText("取消", "Cancel"), resumeText("停止并选择", "Stop and select")}, DefaultButton: 0, CancelButton: 0,
	})
	if err != nil || answer.Button != 1 {
		return false, err
	}
	var applyErr error
	did := s.withTaskProtection(win, sup, "stop", func() error {
		s.mu.Lock()
		changed := s.sup != sup || s.launcherImportBusy || s.launcherStartBusy || s.quitting
		s.mu.Unlock()
		current := store.Get()
		if changed || (sup != nil && sup.Status().Generation != generation) ||
			os.Getenv(paths.EnvVendor) != "" || current.DshBin != cfg.DshBin || current.NodeBin != cfg.NodeBin {
			applyErr = fmt.Errorf("运行环境已改变，请重新选择 Host")
			return applyErr
		}
		s.stopPhoneRuntime()
		if sup != nil {
			sup.Stop()
		}
		applyErr = bundle.Select(paths.UserDataDir())
		s.mu.Lock()
		if applyErr == nil {
			// The next normal Start must recreate all runtime-dependent bridges,
			// rather than restarting a supervisor bound to the previous vendor.
			s.sup = nil
		}
		s.launcherStartBusy = false
		s.launcherStartFailed = false
		s.launcherStartErr = ""
		s.mu.Unlock()
		s.refreshBoot(harness.Status{State: harness.Stopped})
		return applyErr
	})
	if !did && applyErr == nil {
		return false, nil
	}
	return did, applyErr
}
