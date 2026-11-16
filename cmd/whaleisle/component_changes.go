package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
)

const (
	bridgePrepareTimeout = 2 * time.Minute
	bridgeApplyTimeout   = 5 * time.Minute
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
	s.mu.Unlock()
	verifyingMsg := i18n.T(i18n.KeyCmdVerifyingWhalebridgeVersionAndImpactTh)
	s.launcherController().SetComponentActivity(verifyingMsg, verifyingMsg, false)
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
			prepCtx, cancel := context.WithTimeout(s.lifecycleContext(), bridgePrepareTimeout)
			plan, err = desktopcomponents.PrepareWhaleBridgeUpdate(prepCtx, paths.UserDataDir(), route)
			cancel()
		case "rollback":
			if !desktopcomponents.WhaleBridgeManagementSupported() {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdComponentWindowsOnly))
			} else {
				plan, err = desktopcomponents.PrepareWhaleBridgeRollback(paths.UserDataDir())
			}
		case "uninstall":
			prepCtx, cancel := context.WithTimeout(s.lifecycleContext(), bridgePrepareTimeout)
			removal, err = desktopcomponents.PrepareWhaleBridgeUninstall(prepCtx, paths.UserDataDir())
			cancel()
		case "stop":
		default:
			err = desktopcomponents.ErrUnknownAction
		}
		if err == nil && !scopeValid() {
			err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdComponentRuntimeChanged))
		}
		cancelled := false
		removeData := false
		if err == nil && (plan != nil || removal != nil) {
			var approved bool
			approved, err = s.confirmBridgeManagement(action, plan, removal, win)
			cancelled = err == nil && !approved
			if err == nil && approved && removal != nil {
				removeData, err = s.confirmBridgeDataRemoval(removal, win)
			}
		}
		if err == nil && !cancelled && !scopeValid() {
			err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdComponentConfirmationChanged))
		}
		if err == nil && !cancelled {
			s.mu.Lock()
			busy := s.sup != sup || s.taskRestartPending || s.quitting || s.launcherImportBusy
			if !busy {
				s.taskRestartPending = true
			}
			s.mu.Unlock()
			if busy {
				err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdComponentLifecycleBusy))
			} else {
				did := s.withTaskProtection(win, sup, "component", func() error {
					if !scopeValid() {
						err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdRuntimeChangedRetry))
						return err
					}
					switch {
					case action == "stop":
						err = desktopcomponents.RunAction(paths.UserDataDir(), "whalebridge", "stop")
					case removal != nil:
						applyCtx, cancel := context.WithTimeout(s.lifecycleContext(), bridgeApplyTimeout)
						result, err = removal.ApplyWithDataRemoval(applyCtx, removeData)
						cancel()
					default:
						applyCtx, cancel := context.WithTimeout(s.lifecycleContext(), bridgeApplyTimeout)
						err = plan.Apply(applyCtx)
						cancel()
					}
					return err
				})
				s.mu.Lock()
				s.taskRestartPending = false
				s.mu.Unlock()
				if !did && err == nil {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdComponentTaskProtectionIncomplete))
				}
			}
		}
		msg := desktopcomponents.FormatActionError("whalebridge", action, err)
		if cancelled {
			msg = i18n.T(i18n.KeyCmdCancelledTheInstalledVersionConnection)
		} else if err == nil && removal != nil {
			msg = i18n.T(i18n.KeyCmdWhalebridgeV) + removal.Version() + i18n.T(i18n.KeyCmdWasUninstalledAndDisconnectedChatsAndO)
			if !removeData {
				msg += i18n.T(i18n.KeyCmdProviderSettingsKeysAccountsAndUsageWe)
			} else if result == "" {
				msg += i18n.T(i18n.KeyCmdWhalebridgeProviderSettingsKeysAccount)
			}
			if removal.DefaultModel() {
				msg += i18n.T(i18n.KeyCmdSelectANewDefaultModel)
			}
			if result != "" {
				msg += " " + result
			}
		} else if err == nil && plan != nil {
			msg = i18n.T(i18n.KeyCmdWhalebridgeSwitchedToV) + plan.Version() + i18n.T(i18n.KeyCmdUserDataAndTheOriginalRunningStateWer)
		}
		s.launcherController().SetComponentActivity(msg, msg, false)
		// A stopped, switched or uninstalled instance invalidates its management
		// page: the loopback key belongs to the old process. Close the embedded
		// window instead of leaving a dead document (reference ipc-components.js).
		if err == nil && !cancelled && win != nil {
			win.Update(func() { s.closeWhaleBridgeWindow() })
		}
		s.refreshLauncherComponents()
	}()
}

func (s *mainState) confirmBridgeManagement(action string, plan *desktopcomponents.BridgeChange, removal *desktopcomponents.BridgeUninstall, owners ...*mygo.Window) (bool, error) {
	if s.lifecycleDialogOwner(firstConfirmOwner(owners)) == nil {
		return false, fmt.Errorf("%s", i18n.T(i18n.KeyCmdComponentConfirmationUnavailable))
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
		Owner:        firstConfirmOwner(owners),
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
func (s *mainState) confirmBridgeDataRemoval(removal *desktopcomponents.BridgeUninstall, owners ...*mygo.Window) (bool, error) {
	approved := s.askConfirmBlocking(confirmAskRequest{
		Owner:        firstConfirmOwner(owners),
		Title:        i18n.T(i18n.KeyCmdAlsoDeleteWhalebridgeConfiguration),
		Body:         fmt.Sprintf(i18n.T(i18n.KeyCmdTargetNNdeletionPermanentlyRemovesWhal), removal.DataDirectory()),
		ConfirmLabel: i18n.T(i18n.KeyCmdDeleteConfiguration),
		CancelLabel:  i18n.T(i18n.KeyCmdKeepConfiguration),
		Danger:       true,
	})
	return approved, nil
}
