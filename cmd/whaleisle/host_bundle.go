package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/i18n"

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
	release, claimed := s.claimLauncherComponentAction(win, desktopcomponents.HostComponentID, action)
	if !claimed {
		bridgeChangeUI.Unlock()
		return
	}
	bridgeChangeActive.Store(true)
	go func() {
		defer release()
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
				Parent: win, Title: i18n.T(i18n.KeyCmdChooseAMacosHostOfflineBundle),
				Filters: []mygo.FileFilter{{Name: "WhaleIsle Host (.tar.gz)", Extensions: []string{"gz"}}},
			})
			if err != nil || len(picked) == 0 {
				break
			}
			approved := s.askConfirmBlocking(confirmAskRequest{Owner: win,
				Title:       i18n.T(i18n.KeyCmdInstallHostOfflineBundle),
				Body:        fmt.Sprintf(i18n.T(i18n.KeyCmdSourceNdestinationNNextractAndInstallT), picked[0], desktopcomponents.HostBundleVersionsDir(paths.UserDataDir())),
				CancelLabel: i18n.T(i18n.KeyCmdCancel), ConfirmLabel: i18n.T(i18n.KeyCmdInstall),
			})
			if !approved {
				break
			}
			s.hostBundleMessage(win, i18n.T(i18n.KeyCmdExtractingAndCheckingHostAndItsMatchin))
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			var bundle desktopcomponents.HostBundle
			bundle, err = desktopcomponents.InstallHostBundle(ctx, paths.UserDataDir(), picked[0])
			cancel()
			if err == nil {
				message = fmt.Sprintf(i18n.T(i18n.KeyCmdHostVWasInstalledInChooseSelectInstal), bundle.Manifest.Version, bundle.Directory)
			}
		case "select":
			var picked []string
			picked, err = mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent: win, Title: i18n.T(i18n.KeyCmdSelectAnInstalledHostVersionDirectory),
				Directory: true, DefaultPath: desktopcomponents.HostBundleVersionsDir(paths.UserDataDir()),
			})
			if err != nil || len(picked) == 0 {
				break
			}
			s.hostBundleMessage(win, i18n.T(i18n.KeyCmdReadingTheSelectedHostVersion))
			var bundle desktopcomponents.HostBundle
			bundle, err = desktopcomponents.LoadHostBundle(paths.UserDataDir(), picked[0])
			if err == nil {
				var selected bool
				selected, err = s.selectHostBundle(win, bundle)
				if !selected && err == nil {
					message = i18n.T(i18n.KeyCmdHostSelectionWasNotChangedIfTaskProte)
				}
				if selected {
					message = fmt.Sprintf(i18n.T(i18n.KeyCmdHostVIsSelectedTheRuntimeRemainsStopp), bundle.Manifest.Version)
				}
			}
		case "open", "manage":
			err = desktopcomponents.OpenInFileManager(desktopcomponents.HostBundleVersionsDir(paths.UserDataDir()))
		default:
			err = desktopcomponents.ErrUnknownAction
		}
		if err != nil {
			message = i18n.T(i18n.KeyCmdHostActionDidNotComplete) + errcopy.Sentence(err)
		}
		if message != "" {
			s.hostBundleMessage(win, message)
		}
	}()
}

func (s *mainState) hostBundleMessage(win *mygo.Window, message string) {
	s.launcherController().SetComponentActivity(message, message, false)
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) selectHostBundle(win *mygo.Window, bundle desktopcomponents.HostBundle) (bool, error) {
	startBusy := s.launcherController().StartBusy()
	marketBusy := s.marketCoordinator().IsBusy()
	s.mu.Lock()
	store := s.cfgStore
	sup := s.sup
	busy := s.taskRestartPending || s.launcherImportBusy || startBusy || marketBusy || s.quitting
	if !busy {
		s.taskRestartPending = true
	}
	s.mu.Unlock()
	if busy {
		return false, fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostSelectionBusy))
	}
	defer func() { s.mu.Lock(); s.taskRestartPending = false; s.mu.Unlock() }()
	if store == nil {
		return false, fmt.Errorf("%s", i18n.T(i18n.KeyCmdConfigStoreNotReady))
	}
	cfg := store.Get()
	if os.Getenv(paths.EnvVendor) != "" || strings.TrimSpace(cfg.DshBin) != "" || strings.TrimSpace(cfg.NodeBin) != "" {
		return false, fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostExternalOverride))
	}
	generation := 0
	if sup != nil {
		generation = sup.Status().Generation
	}
	approved := s.askConfirmBlocking(confirmAskRequest{
		Owner:        win,
		Title:        i18n.T(i18n.KeyCmdSelectHostVersion),
		Body:         fmt.Sprintf(i18n.T(i18n.KeyCmdTargetHostVNNtheExistingTaskProtectio), bundle.Manifest.Version, bundle.Directory),
		ConfirmLabel: i18n.T(i18n.KeyCmdStopAndSelect),
		CancelLabel:  i18n.T(i18n.KeyCmdCancel),
		Danger:       true,
	})
	if !approved {
		return false, nil
	}
	var applyErr error
	did := s.withTaskProtection(win, sup, "stop", func() error {
		curStartBusy := s.launcherController().StartBusy()
		s.mu.Lock()
		changed := s.sup != sup || s.launcherImportBusy || curStartBusy || s.quitting
		s.mu.Unlock()
		current := store.Get()
		if changed || (sup != nil && sup.Status().Generation != generation) ||
			os.Getenv(paths.EnvVendor) != "" || current.DshBin != cfg.DshBin || current.NodeBin != cfg.NodeBin {
			applyErr = fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostSelectionChanged))
			return applyErr
		}
		s.stopPhoneRuntime()
		if sup != nil {
			sup.Stop()
		}
		applyErr = bundle.Select(paths.UserDataDir())
		s.mu.Lock()
		if applyErr == nil {
			s.sup = nil
		}
		s.mu.Unlock()
		s.launcherController().SetStartError(false, false, "")
		s.refreshBoot(harness.Status{State: harness.Stopped})
		return applyErr
	})
	if !did && applyErr == nil {
		return false, nil
	}
	return did, applyErr
}
