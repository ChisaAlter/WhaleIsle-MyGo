package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
	"whaleisle/internal/startuprec"
)

type startupRecoveryState struct {
	store              *startuprec.Store
	loadError          string
	saveError          string
	recordedGeneration int
	recordedState      harness.State
	recorded           bool
	updatePromptBusy   bool
	updateSeen         map[string]bool
	updateHold         bool
}

func (s *mainState) openStartupRecovery(dir string, cfg *config.Store) {
	store, err := startuprec.Open(dir)
	if err == nil {
		err = cfg.ReconcilePluginRecovery(buildVersion)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startupRecovery.store = store
	if err != nil {
		s.startupRecovery.loadError = startuprec.RedactLine(err.Error())
	}
}

func (s *mainState) coldStartupAllowed(cfg config.Config, importReady bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &s.startupRecovery
	failed := r.loadError != "" || r.store == nil
	if r.store != nil {
		failed = failed || r.store.Last().Failed()
	}
	if failed || cfg.SkipUserPlugins {
		s.launcherStartFailed = failed
		s.launcherStartErr = resumeText("上次启动未完整恢复，已暂停自动启动。请查看诊断后重试。", "The previous start did not fully recover. Auto-start is paused; review diagnostics and retry.")
		if r.loadError != "" {
			s.launcherStartErr = r.loadError + "\n" + resumeText("恢复文件已保留，请修复后重新打开应用。", "The recovery file was preserved; repair it and reopen the app.")
		}
		if cfg.SkipUserPlugins {
			s.launcherNote = resumeText("已保留跳过用户插件的恢复模式。可继续以恢复模式启动，或清除恢复标记后重试全部插件。", "Skip-user-plugins recovery is saved. Start in recovery mode, or clear recovery and retry all plugins.")
		}
	}
	return startuprec.ShouldAutoStart(cfg.Workspace, cfg.AutoStartDesktop, importReady, failed, cfg.SkipUserPlugins)
}

// Called with the main state lock after accepting the current supervisor's
// generation/state. A recovery Ready is deliberately not recorded as healthy.
func (s *mainState) recordStartupStatusLocked(st harness.Status) bool {
	r := &s.startupRecovery
	if st.State != harness.Ready && st.State != harness.Failed {
		return r.loadError != "" || r.saveError != ""
	}
	if r.recorded && r.recordedGeneration == st.Generation && r.recordedState == st.State {
		return r.loadError != "" || r.saveError != ""
	}
	r.recorded = true
	r.recordedGeneration = st.Generation
	r.recordedState = st.State
	logs := []string(nil)
	if s.sup != nil {
		logs = s.sup.CurrentStartLogs()
	}
	reason := ""
	if st.Err != nil {
		reason = st.Err.Error()
	}
	fullReady := st.State == harness.Ready && !st.Recovery
	if st.Recovery && reason == "" {
		reason = resumeText("仅恢复模式就绪；用户插件仍被跳过。", "Recovery mode is ready; user plugins remain skipped.")
	}
	var err error
	if r.store == nil {
		err = fmt.Errorf("startup recovery storage is unavailable")
	} else {
		err = r.store.Record(fullReady, reason, logs, buildVersion)
	}
	r.saveError = ""
	if err != nil {
		r.saveError = startuprec.RedactLine(err.Error())
		s.launcherNote = resumeText("启动结果未能保存：", "Could not save startup outcome: ") + r.saveError
	}
	if st.Recovery {
		s.launcherNote = resumeText("恢复模式已就绪；用户插件仍被跳过。请在插件页检查原因并明确恢复。", "Recovery mode is ready; user plugins are still skipped. Review the cause on the Plugins page before restoring them.")
	}
	return r.loadError != "" || r.saveError != ""
}

// This consumes the existing pinned native-release catalog. It never starts a
// second check, delays the local start decision or opens/focuses a hidden window.
func (s *mainState) maybeAskStartupUpdate(win *mygo.Window, snapshot nativeReleaseState, vendorReady bool) {
	if win == nil || snapshot.busy || snapshot.err != "" || snapshot.catalogRoute != snapshot.route {
		return
	}
	var release update.NativeRelease
	for _, row := range snapshot.rows {
		if !row.Prerelease && row.Validate(runtime.GOOS, runtime.GOARCH, !vendorReady) == nil && update.CompareVersions(row.Version, buildVersion) > 0 {
			release = row
			break
		}
	}
	if release.Tag == "" {
		return
	}
	key := snapshot.catalogRoute + "\x00" + release.Tag
	visible := win.IsVisible() && !win.IsMinimized()
	s.mu.Lock()
	r := &s.startupRecovery
	if r.updateSeen == nil {
		r.updateSeen = map[string]bool{}
	}
	ask := s.cfgStore != nil && s.cfgStore.Get().AskOnUpdate
	if !ask {
		r.updateSeen[key] = true
	}
	can := startuprec.CanPromptUpdate(ask, !r.updateSeen[key], visible, s.launcherOpen, s.launcherStartBusy, r.updatePromptBusy || s.nativeReleases.busy, s.quitting)
	if can {
		r.updatePromptBusy = true
	}
	s.mu.Unlock()
	if !can {
		return
	}
	win.Update(func() {
		// A queued callback may run after Hide, navigation, a restart or route change.
		visible := win.IsVisible() && !win.IsMinimized()
		s.mu.Lock()
		r := &s.startupRecovery
		ask := s.cfgStore != nil && s.cfgStore.Get().AskOnUpdate
		can := startuprec.CanPromptUpdate(ask, !r.updateSeen[key], visible, s.launcherOpen, s.launcherStartBusy, s.nativeReleases.busy, s.quitting) && s.win == win && s.nativeReleases.catalogRoute == snapshot.catalogRoute && s.nativeReleases.route == snapshot.route
		if !can {
			r.updatePromptBusy = false
			s.mu.Unlock()
			return
		}
		r.updateSeen[key] = true
		s.mu.Unlock()
		choice, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageInfo,
			Title:   resumeText("MyGo 更新可用", "MyGo update available"),
			Message: fmt.Sprintf(resumeText("发现 %s。现在下载并校验该版本？安装仍需单独确认，可稍后重试。", "%s is available. Download and verify this version now? Installation needs a separate confirmation; you can retry later."), release.Version),
			Buttons: []string{resumeText("稍后", "Later"), resumeText("下载更新", "Download update")}, DefaultButton: 0, CancelButton: 0})
		visible = win.IsVisible() && !win.IsMinimized()
		s.mu.Lock()
		r = &s.startupRecovery
		r.updatePromptBusy = false
		allowed := err == nil && choice.Button == 1 && visible && s.win == win && s.launcherOpen && !s.launcherStartBusy && !s.quitting && !s.launcherImportBusy && !s.nativeReleases.busy && s.nativeReleases.route == snapshot.route && s.nativeReleases.catalogRoute == snapshot.catalogRoute
		if allowed {
			r.updateHold = true
			s.launcherOpen = true
		}
		if err != nil {
			s.launcherNote = startuprec.RedactLine(err.Error())
		}
		s.mu.Unlock()
		if allowed {
			s.downloadNativeRelease(win, release.Tag, !vendorReady)
		}
	})
}

func recoveryReason(reason string) string { return startuprec.RedactLine(strings.TrimSpace(reason)) }
