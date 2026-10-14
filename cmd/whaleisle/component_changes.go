package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/egoist/mygo"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
)

var bridgeChangeUI sync.Mutex

// Stage first, while the existing service keeps running. Only the short switch
// uses the Host admission lock; component maintenance independently refuses DSH
// work, including requests not visible to the Host.
func (s *mainState) changeWhaleBridge(win *mygo.Window, action string) {
	if !bridgeChangeUI.TryLock() {
		return
	}
	route := s.updateDownloadRoute()
	s.mu.Lock()
	s.launcherUpdate = resumeText("正在校验鲸桥版本；现有实例保持运行…", "Verifying WhaleBridge; the current instance stays running…")
	s.mu.Unlock()
	go func() {
		defer bridgeChangeUI.Unlock()
		var plan *desktopcomponents.BridgeChange
		var err error
		switch action {
		case "update":
			plan, err = desktopcomponents.PrepareWhaleBridgeUpdate(context.Background(), paths.UserDataDir(), route)
		case "rollback":
			plan, err = desktopcomponents.PrepareWhaleBridgeRollback(paths.UserDataDir())
		case "stop":
		default:
			err = desktopcomponents.ErrUnknownAction
		}
		if err == nil {
			s.mu.Lock()
			sup := s.sup
			busy := s.taskRestartPending || s.quitting || s.launcherImportBusy
			if !busy {
				s.taskRestartPending = true
			}
			s.mu.Unlock()
			if busy {
				err = fmt.Errorf("其他生命周期操作正在进行，已保留现有实例和已校验版本")
			} else {
				did := s.withTaskProtection(win, sup, "component", func() error {
					s.mu.Lock()
					changed := s.sup != sup || s.quitting || s.launcherImportBusy
					s.mu.Unlock()
					if changed {
						return fmt.Errorf("运行环境已改变，请重新操作")
					}
					if action == "stop" {
						err = desktopcomponents.RunAction(paths.UserDataDir(), "whalebridge", "stop")
					} else {
						err = plan.Apply(context.Background())
					}
					return err
				})
				s.mu.Lock()
				s.taskRestartPending = false
				s.mu.Unlock()
				if !did && err == nil {
					err = fmt.Errorf("操作未执行或任务保护未完成；请查看启动器提示后重试")
				}
			}
		}
		s.mu.Lock()
		s.launcherUpdate = desktopcomponents.FormatActionError("whalebridge", action, err)
		if err == nil && plan != nil {
			s.launcherUpdate = "鲸桥已切换至 v" + plan.Version() + "；用户数据和原运行状态已保留"
		}
		s.mu.Unlock()
		s.refreshLauncherComponents()
		if win != nil {
			win.Update(func() {})
		}
	}()
}
