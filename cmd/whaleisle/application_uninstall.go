package main

import (
	"errors"
	"runtime"
	"whaleisle/internal/ui/i18n"

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
				Title:   i18n.T(i18n.KeyCmdCouldNotOpenInstalledApps),
				Message: i18n.T(i18n.KeyCmdOpenWindowsSettingsAppsInstalledAppsMa) + "\n\n" + err.Error(),
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
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: i18n.T(i18n.KeyCmdNoVerifiedUninstallerFound), Message: i18n.T(i18n.KeyCmdThisMayBeAPortableSourceBuildOrAnIn) + "\n\n" + err.Error()})
			return
		}
		approved := s.askConfirmBlocking(confirmAskRequest{
			Title:        i18n.T(i18n.KeyCmdUninstallCurrentNativeApp),
			Body:         i18n.T(i18n.KeyCmdContinuingChecksAndStopsTasksOpensThis) + plan.InstallLocation(),
			ConfirmLabel: i18n.T(i18n.KeyCmdStartUninstallerAndQuit),
			CancelLabel:  i18n.T(i18n.KeyCmdCancel),
			Danger:       true,
		})
		if !approved || !s.saveResumeBeforeClose(win) {
			return
		}
		did := s.withTaskProtection(win, sup, "quit", func() error {
			s.mu.Lock()
			changed := s.launcherImportBusy || s.quitting || s.sup != sup
			s.mu.Unlock()
			if changed {
				return errors.New(i18n.T(i18n.KeyCmdRuntimeChangedRetryUninstall))
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
