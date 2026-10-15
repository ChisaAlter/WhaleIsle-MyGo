package main

import (
	"context"
	"errors"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/importer"
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
	if s.launcherStartBusy || (s.sup != nil && s.sup.Status().State != harness.Stopped) {
		s.launcherImport = "请先在启动器停止内核，再确认导入。"
		s.mu.Unlock()
		refresh()
		return
	}
	// Own the slot throughout confirmation, so start/repeated import cannot race it.
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
		Message: importer.Confirmation(cat, sel), Buttons: []string{"取消", "确认导入"}, DefaultButton: 0, CancelButton: 0,
	})
	if err != nil || answer.Button != 1 {
		release()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.launcherImportCancel = cancel
	s.launcherImport = "正在导入；可取消后续项目，已完成项目将保留。"
	s.launcherImportJournal = nil
	node := ""
	if s.cfgStore != nil {
		node = s.cfgStore.Get().NodeBin
	}
	s.mu.Unlock()
	refresh()
	go func() {
		defer cancel()
		result, err := importer.ApplyCatalogWithPlugins(ctx, cat, sel, s.importPluginInstaller(win, cat.DestHome, node), func(progress importer.ApplyResult) {
			s.mu.Lock()
			s.launcherImport = progress.Message
			s.launcherImportJournal = progress.Journal
			s.mu.Unlock()
			refresh()
		})
		s.mu.Lock()
		s.launcherImport = result.Message
		s.launcherImportJournal = result.Journal
		if err != nil && result.Message == "" {
			s.launcherImport = err.Error()
			s.launcherImportJournal = []string{err.Error()}
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
	defer func() { s.mu.Lock(); s.taskRestartPending = false; s.mu.Unlock() }()
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
		s.stopPhoneRuntime()
		sup.Restart()
		return nil
	})
}
