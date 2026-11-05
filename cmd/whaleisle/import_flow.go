package main

import (
	"context"
	"errors"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

func (s *mainState) startImport(win *mygo.Window) {
	s.cancelImportCatalogScan()
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
		s.launcherImport = i18n.T(i18n.KeyCmdImportBusy)
		s.mu.Unlock()
		refresh()
		return
	}
	if s.launcherImportCatalog == nil {
		s.launcherImport = i18n.T(i18n.KeyCmdImportScanFirst)
		s.mu.Unlock()
		refresh()
		return
	}
	cat := s.launcherImportCatalog.Snapshot()
	sel := importer.SelectionFromCatalog(cat, s.launcherImportAttachments && cat.HasAttachments, s.launcherImportOverwrite)
	if sel.Count() == 0 {
		s.launcherImport = i18n.T(i18n.KeyCmdImportNothingSelected)
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
		Owner:        win,
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
	return s.restartUnlessImportWithResult(sup, false, nil)
}

// External feature writers keep their maintenance lease until complete settles
// or rolls back their fields. Accepted work calls complete before releasing the
// pending slot; a synchronous admission refusal returns false without a call.
func (s *mainState) restartUnlessImportWithResult(sup *harness.Supervisor, maintenanceOwned bool, complete func(bool, error)) bool {
	if sup == nil {
		return false
	}
	s.mu.Lock()
	if s.launcherImportBusy {
		s.launcherImport = i18n.T(i18n.KeyCmdImportRestartBlocked)
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
		var did bool
		var resultErr error
		defer func() {
			if complete != nil {
				complete(did, resultErr)
			}
			s.mu.Lock()
			s.taskRestartPending = false
			s.mu.Unlock()
			if win != nil {
				win.Update(func() {})
			}
		}()
		ctx := context.Background()
		if complete != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, 4*time.Minute)
			defer cancel()
		}
		if !maintenanceOwned {
			if resultErr = importer.CheckImportReady(paths.DshHome()); resultErr != nil {
				if complete == nil {
					s.taskProtectionError(win, resultErr)
				}
				return
			}
		}
		s.mu.Lock()
		busy := s.launcherImportBusy
		s.mu.Unlock()
		if busy {
			resultErr = errors.New(i18n.T(i18n.KeyCmdImportIsStillRunningWaitForCompletion))
			if complete == nil {
				s.taskProtectionError(win, resultErr)
			}
			return
		}
		did, resultErr = s.protectTaskAction(ctx, win, sup, "restart", func() error {
			// Recheck after Host inspection/confirmation and before the first side effect.
			if err := ctx.Err(); err != nil {
				return err
			}
			s.mu.Lock()
			if s.launcherImportBusy || s.quitting || s.sup != sup {
				s.mu.Unlock()
				return errors.New(i18n.T(i18n.KeyCmdRuntimeChangedRetryRestart))
			}
			s.launcherStartBusy = true
			s.mu.Unlock()
			if complete != nil {
				// Publish an external writer's pending overlay only after task
				// confirmation accepted this exact runtime's restart.
				s.syncHarnessFromConfig()
			}
			if err := s.closeTaskProtectedResources(); err != nil {
				s.mu.Lock()
				s.launcherStartBusy = false
				s.mu.Unlock()
				return err
			}
			if complete != nil {
				return sup.RestartAndWait(ctx)
			}
			sup.Restart()
			return nil
		})
		if !did && resultErr == nil {
			resultErr = taskcontrol.ErrCancelled
		}
		if complete == nil {
			s.taskProtectionError(win, resultErr)
		}
	}()
	if win != nil {
		win.Update(func() {})
	}
	return true
}
