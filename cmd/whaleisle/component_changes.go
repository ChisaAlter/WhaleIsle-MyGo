package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
)

var bridgeChangeUI sync.Mutex
var bridgeChangeActive atomic.Bool

// Stage and inspect first without stopping the existing service. Review the
// exact version/impact, then use the existing Host admission and maintenance
// guards for the short lifecycle mutation.
func (s *mainState) changeWhaleBridge(win *mygo.Window, action string) {
	if !bridgeChangeUI.TryLock() {
		return
	}
	release, claimed := s.claimLauncherComponentAction(win, "whalebridge", action)
	if !claimed {
		bridgeChangeUI.Unlock()
		return
	}
	bridgeChangeActive.Store(true)
	route := s.updateDownloadRoute()
	s.mu.Lock()
	sup := s.sup
	s.launcherUpdate = i18n.T(i18n.KeyCmdVerifyingWhalebridgeVersionAndImpactTh)
	s.launcherComponentStatus = s.launcherUpdate
	s.mu.Unlock()
	generation := 0
	if sup != nil {
		generation = sup.Status().Generation
	}
	scopeValid := func() bool {
		s.mu.Lock()
		valid := s.sup == sup && !s.quitting && !s.launcherImportBusy
		s.mu.Unlock()
		return valid && (sup == nil || sup.Status().Generation == generation)
	}
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		defer release()
		defer func() {
			bridgeChangeActive.Store(false)
			bridgeChangeUI.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		var plan *desktopcomponents.BridgeChange
		var removal *desktopcomponents.BridgeUninstall
		var err error
		var result string
		switch action {
		case "update":
			plan, err = desktopcomponents.PrepareWhaleBridgeUpdate(context.Background(), paths.UserDataDir(), route)
		case "rollback":
			if !desktopcomponents.WhaleBridgeManagementSupported() {
				err = fmt.Errorf("鲸桥组件管理仅支持 Windows x64")
			} else {
				plan, err = desktopcomponents.PrepareWhaleBridgeRollback(paths.UserDataDir())
			}
		case "uninstall":
			removal, err = desktopcomponents.PrepareWhaleBridgeUninstall(context.Background(), paths.UserDataDir())
		case "stop":
		default:
			err = desktopcomponents.ErrUnknownAction
		}
		if err == nil && !scopeValid() {
			err = fmt.Errorf("运行环境已改变，请重新检查组件并确认")
		}
		cancelled := false
		removeData := false
		if err == nil && (plan != nil || removal != nil) {
			var approved bool
			approved, err = s.confirmBridgeManagement(action, plan, removal)
			cancelled = err == nil && !approved
			if err == nil && approved && removal != nil {
				removeData, err = s.confirmBridgeDataRemoval(removal)
			}
		}
		if err == nil && !cancelled && !scopeValid() {
			err = fmt.Errorf("确认期间运行环境已改变，请重新检查组件并确认")
		}
		if err == nil && !cancelled {
			s.mu.Lock()
			busy := s.sup != sup || s.taskRestartPending || s.quitting || s.launcherImportBusy
			if !busy {
				s.taskRestartPending = true
			}
			s.mu.Unlock()
			if busy {
				err = fmt.Errorf("其他生命周期操作正在进行，已保留现有实例和已校验版本")
			} else {
				did := s.withTaskProtection(win, sup, "component", func() error {
					if !scopeValid() {
						err = fmt.Errorf("运行环境已改变，请重新操作")
						return err
					}
					switch {
					case action == "stop":
						err = desktopcomponents.RunAction(paths.UserDataDir(), "whalebridge", "stop")
					case removal != nil:
						result, err = removal.ApplyWithDataRemoval(context.Background(), removeData)
					default:
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
		if cancelled {
			s.launcherUpdate = i18n.T(i18n.KeyCmdCancelledTheInstalledVersionConnection)
		} else if err == nil && removal != nil {
			s.launcherUpdate = i18n.T(i18n.KeyCmdWhalebridgeV) + removal.Version() + i18n.T(i18n.KeyCmdWasUninstalledAndDisconnectedChatsAndO)
			if !removeData {
				s.launcherUpdate += i18n.T(i18n.KeyCmdProviderSettingsKeysAccountsAndUsageWe)
			} else if result == "" {
				s.launcherUpdate += i18n.T(i18n.KeyCmdWhalebridgeProviderSettingsKeysAccount)
			}
			if removal.DefaultModel() {
				s.launcherUpdate += i18n.T(i18n.KeyCmdSelectANewDefaultModel)
			}
			if result != "" {
				s.launcherUpdate += " " + result
			}
		} else if err == nil && plan != nil {
			s.launcherUpdate = i18n.T(i18n.KeyCmdWhalebridgeSwitchedToV) + plan.Version() + i18n.T(i18n.KeyCmdUserDataAndTheOriginalRunningStateWer)
		}
		s.launcherComponentStatus = s.launcherUpdate
		s.mu.Unlock()
		// A stopped, switched or uninstalled instance invalidates its management
		// page: the loopback key belongs to the old process. Close the embedded
		// window instead of leaving a dead document (reference ipc-components.js).
		if err == nil && !cancelled && win != nil {
			win.Update(func() { s.closeWhaleBridgeWindow() })
		}
		s.refreshLauncherComponents()
	}()
}

func (s *mainState) confirmBridgeManagement(action string, plan *desktopcomponents.BridgeChange, removal *desktopcomponents.BridgeUninstall) (bool, error) {
	s.mu.Lock()
	hasWindow := s.win != nil
	s.mu.Unlock()
	if !hasWindow {
		return false, fmt.Errorf("无法显示组件操作确认；未更改组件")
	}
	title, button, message := "", "", ""
	if removal != nil {
		title = i18n.T(i18n.KeyCmdConfirmWhalebridgeUninstall)
		button = i18n.T(i18n.KeyCmdContinueUninstall)
		message = fmt.Sprintf(i18n.T(i18n.KeyCmdTargetWhalebridgeVThisStopsTheService), removal.Version())
		if removal.DefaultModel() {
			message += "\n\n" + i18n.T(i18n.KeyCmdYourCurrentDefaultModelUsesWhalebridge)
		}
	} else if plan != nil {
		title, button = i18n.T(i18n.KeyCmdConfirmWhalebridgeVersionChange), i18n.T(i18n.KeyCmdSwitchVersion)
		if action == "rollback" {
			title, button = i18n.T(i18n.KeyCmdConfirmWhalebridgeRollback), i18n.T(i18n.KeyCmdRollBack)
		}
		message = fmt.Sprintf(i18n.T(i18n.KeyCmdTargetWhalebridgeVARunningServiceWill), plan.Version())
	} else {
		return false, desktopcomponents.ErrUnknownAction
	}
	approved := s.askConfirmBlocking(confirmAskRequest{
		Title:        title,
		Body:         message,
		ConfirmLabel: button,
		CancelLabel:  i18n.T(i18n.KeyCmdCancel),
		Danger:       true,
	})
	return approved, nil
}

// The original launcher asks about configuration separately after approving
// uninstall. Closing this choice or selecting the default keeps all data.
func (s *mainState) confirmBridgeDataRemoval(removal *desktopcomponents.BridgeUninstall) (bool, error) {
	approved := s.askConfirmBlocking(confirmAskRequest{
		Title:        i18n.T(i18n.KeyCmdAlsoDeleteWhalebridgeConfiguration),
		Body:         fmt.Sprintf(i18n.T(i18n.KeyCmdTargetNNdeletionPermanentlyRemovesWhal), removal.DataDirectory()),
		ConfirmLabel: i18n.T(i18n.KeyCmdDeleteConfiguration),
		CancelLabel:  i18n.T(i18n.KeyCmdKeepConfiguration),
		Danger:       true,
	})
	return approved, nil
}
