package main

import (
	"context"
	"errors"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

func (s *mainState) startImport(win *mygo.Window) {
	refresh := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	s.mu.Lock()
	if s.launcherImportBusy {
		s.mu.Unlock()
		return
	}
	if s.marketWorker != nil || s.marketInstallBusy || s.taskRestartPending || s.quitting {
		s.launcherImport = "插件操作或运行时切换尚未结束，请稍后再导入。"
		s.mu.Unlock()
		refresh()
		return
	}
	if s.launcherImportCatalog == nil {
		s.launcherImport = "请先扫描导入来源。"
		s.mu.Unlock()
		refresh()
		return
	}
	cat := s.launcherImportCatalog.Snapshot()
	sel := importer.SelectionFromCatalog(cat, s.launcherImportAttachments && cat.HasAttachments, s.launcherImportOverwrite)
	if sel.Count() == 0 {
		s.launcherImport = "未选择任何项，未写入。"
		s.mu.Unlock()
		refresh()
		return
	}
	if s.launcherStartBusy {
		s.launcherImport = i18n.T(i18n.KeyCmdTheRuntimeIsStartingRetryImportAfterS)
		s.mu.Unlock()
		refresh()
		return
	}
	sup := s.sup
	// Own the slot throughout confirmation, protected stop and import.
	s.launcherImportBusy = true
	s.mu.Unlock()
	release := func() { s.mu.Lock(); s.launcherImportBusy = false; s.mu.Unlock(); refresh() }
	if err := importer.ValidateRoots(cat.SourceHome, cat.DestHome); err != nil {
		s.mu.Lock()
		s.launcherImport = err.Error()
		s.mu.Unlock()
		release()
		return
	}
	s.askConfirm(confirmAskRequest{
		Title:        i18n.T(i18n.KeyCmdConfirmDataImport),
		Body:         importer.Confirmation(cat, sel) + "\n\n" + i18n.T(i18n.KeyCmdBeforeImportingCurrentTasksAreCheckedA),
		ConfirmLabel: i18n.T(i18n.KeyCmdConfirmAndContinue),
		CancelLabel:  i18n.T(i18n.KeyCmdCancel),
		Danger:       true,
		Run:          func() { s.continueImport(win, sup, cat, sel) },
		Cancel:       release,
	})
}

// continueImport performs the protected stop and the import after the in-app
// confirmation; it owns the import slot until the goroutine below releases it.
func (s *mainState) continueImport(win *mygo.Window, sup *harness.Supervisor, cat importer.Catalog, sel importer.ApplySelection) {
	refresh := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.launcherImportCancel = cancel
	s.launcherImport = i18n.T(i18n.KeyCmdCheckingCurrentTasksBeforeStoppingTheR)
	s.launcherImportJournal = nil
	node := ""
	if s.cfgStore != nil {
		node = s.cfgStore.Get().NodeBin
	}
	s.mu.Unlock()
	refresh()
	go func() {
		defer cancel()
		var result importer.ApplyResult
		var applyErr error
		kernelStopped := false
		did, err := s.protectTaskAction(ctx, win, sup, "stop", func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			s.mu.Lock()
			changed := s.sup != sup || s.quitting || s.launcherStartBusy
			s.mu.Unlock()
			if changed {
				return errors.New(i18n.T(i18n.KeyCmdTheRuntimeChangedConfirmTheImportAgain))
			}
			if err := s.closeTaskProtectedResources(); err != nil {
				return err
			}
			if sup != nil {
				kernelStopped = sup.Status().State != harness.Stopped
				sup.Stop()
			}
			s.mu.Lock()
			s.launcherStartBusy = false
			s.launcherStartFailed = false
			s.launcherStartErr = ""
			s.launcherImport = i18n.T(i18n.KeyCmdTheRuntimeIsStoppedImportingCompletedI)
			s.mu.Unlock()
			refresh()
			result, applyErr = importer.ApplyCatalogWithPlugins(ctx, cat, sel, s.importPluginInstaller(win, cat.DestHome, node), func(progress importer.ApplyResult) {
				s.mu.Lock()
				s.launcherImport = progress.Message
				s.launcherImportJournal = progress.Journal
				s.mu.Unlock()
				refresh()
			})
			// The protected stop committed even if a selected import item failed.
			// Keep its import result separate from the stopped Host's admission.
			return nil
		})
		if !did && err == nil {
			err = taskcontrol.ErrCancelled
		}
		if err == nil {
			err = applyErr
		} else if errors.Is(err, context.Canceled) || errors.Is(err, taskcontrol.ErrCancelled) {
			result.Message = i18n.T(i18n.KeyCmdImportCancelledNoImportDataWasWritten)
		} else {
			result.Message = i18n.T(i18n.KeyCmdImportDidNotStart) + err.Error()
		}
		s.mu.Lock()
		s.launcherImport = result.Message
		s.launcherImportJournal = result.Journal
		if err != nil && result.Message == "" {
			s.launcherImport = err.Error()
			s.launcherImportJournal = []string{err.Error()}
		}
		if kernelStopped {
			s.launcherImport += i18n.T(i18n.KeyCmdTheRuntimeRemainsStoppedStartItManuall)
		}
		s.launcherImportCancel = nil
		s.launcherImportBusy = false
		s.mu.Unlock()
		refresh()
	}()
}

// Native restarts preserve import admission and coordinate Host task protection.
func (s *mainState) restartUnlessImport(sup *harness.Supervisor) bool {
	if sup == nil {
		return false
	}
	s.mu.Lock()
	if s.launcherImportBusy {
		s.launcherImport = "导入尚未结束，暂不能重启内核。"
		s.mu.Unlock()
		return false
	}
	if s.taskRestartPending || s.quitting {
		s.mu.Unlock()
		return false
	}
	s.taskRestartPending = true
	win := s.win
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
		if err := importer.CheckImportReady(paths.DshHome()); err != nil {
			s.taskProtectionError(win, err)
			return
		}
		s.withTaskProtection(win, sup, "restart", func() error {
			// Recheck after Host inspection/confirmation and before the first side effect.
			s.mu.Lock()
			if s.launcherImportBusy || s.quitting || s.sup != sup {
				s.mu.Unlock()
				return errors.New("runtime or import state changed; retry restart")
			}
			s.launcherStartBusy = true
			s.mu.Unlock()
			if err := s.closeTaskProtectedResources(); err != nil {
				return err
			}
			sup.Restart()
			return nil
		})
	}()
	if win != nil {
		win.Update(func() {})
	}
	return true
}
