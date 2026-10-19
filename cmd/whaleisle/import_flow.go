package main

import (
	"context"
	"errors"

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
		s.launcherImport = resumeText("内核正在启动，请稍后再确认导入。", "The runtime is starting. Retry import after startup settles.")
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
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: win, Type: mygo.MessageQuestion, Title: "确认导入数据",
		Message: importer.Confirmation(cat, sel) + "\n\n" + resumeText("导入前会检查当前任务并停止 Host 内核；如有运行中的工作，会继续显示现有任务确认。桌面窗口保持打开，导入结束后内核保持停止，可稍后手动启动。", "Before importing, current tasks are checked and the Host runtime is stopped. Active work uses the existing task confirmation. The desktop window stays open and the runtime stays stopped after import until you start it again."), Buttons: []string{"取消", "确认并继续"}, DefaultButton: 0, CancelButton: 0,
	})
	if err != nil || answer.Button != 1 {
		release()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.launcherImportCancel = cancel
	s.launcherImport = resumeText("正在检查当前任务，随后停止内核并导入；可取消后续导入。", "Checking current tasks before stopping the runtime and importing. You can cancel the remaining import.")
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
				return errors.New(resumeText("运行环境已改变，请重新确认导入。", "The runtime changed. Confirm the import again."))
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
			s.launcherImport = resumeText("内核已停止，正在导入；已完成项目将保留。", "The runtime is stopped. Importing; completed items will be retained.")
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
			result.Message = resumeText("已取消导入，未写入导入数据。", "Import cancelled. No import data was written.")
		} else {
			result.Message = resumeText("导入未开始：", "Import did not start: ") + err.Error()
		}
		s.mu.Lock()
		s.launcherImport = result.Message
		s.launcherImportJournal = result.Journal
		if err != nil && result.Message == "" {
			s.launcherImport = err.Error()
			s.launcherImportJournal = []string{err.Error()}
		}
		if kernelStopped {
			s.launcherImport += resumeText(" 内核保持停止，可在导入结果确认后手动启动。", " The runtime remains stopped; start it manually after checking the import result.")
		}
		s.launcherImportCancel = nil
		s.launcherImportBusy = false
		s.mu.Unlock()
		refresh()
	}()
}

// Native restarts preserve import admission and coordinate Host task protection.
func (s *mainState) restartUnlessImport(sup *harness.Supervisor) {
	if sup == nil {
		return
	}
	s.mu.Lock()
	if s.launcherImportBusy {
		s.launcherImport = "导入尚未结束，暂不能重启内核。"
		s.mu.Unlock()
		return
	}
	if s.taskRestartPending || s.quitting {
		s.mu.Unlock()
		return
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
}
