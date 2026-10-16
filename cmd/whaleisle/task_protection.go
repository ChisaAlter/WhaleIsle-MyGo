package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/desktop/taskguard"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
)

func (s *mainState) taskProtectionError(win *mygo.Window, err error) {
	if err == nil || err == taskcontrol.ErrCancelled {
		return
	}
	s.mu.Lock()
	s.launcherStartErr = resumeText("操作未完成，已保留当前窗口：", "Action did not complete; the window was kept open: ") + err.Error()
	s.launcherStartFailed = true
	s.launcherOpen = true
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) confirmHostTasks(win *mygo.Window, operation string, i taskcontrol.Inspection) (bool, error) {
	label := resumeText("退出", "Quit")
	if operation == "restart" {
		label = resumeText("重启", "Restart")
	} else if operation == "update" {
		label = resumeText("安装更新并退出", "Install update and quit")
	} else if operation == "component" {
		label = resumeText("更改鲸桥组件", "Change WhaleBridge component")
	} else if operation == "stop" {
		label = resumeText("停止内核", "Stop runtime")
	}
	lines := []string{}
	if operation == "stop" {
		lines = append(lines, resumeText("这会停止 Host 内核，桌面应用窗口将保持打开。", "This stops the Host runtime. The desktop application window will stay open."))
	}
	if i.Unknown != "" {
		lines = append(lines, resumeText("Host 任务保护不可用，无法确认后台任务、定时任务或在途请求是否已排空。继续将仅依赖本地会话信息，可能中断未显示的工作。", "Host task protection is unavailable. Background work, scheduled tasks and in-flight requests cannot be verified or drained. Continuing relies only on local session information and may interrupt unseen work."), i.Unknown)
	}
	if i.Unknown != "" {
		lines = append(lines, fmt.Sprintf(resumeText("本地已知运行会话：%d；Host 活动和计划工作：未知。", "Known local running sessions: %d; Host active and scheduled work: unknown."), i.LocalRunning))
	} else {
		lines = append(lines, fmt.Sprintf(resumeText("本地已知运行会话：%d；Host 活动工作：%d；计划工作：%d。", "Known local running sessions: %d; Host active work: %d; scheduled work: %d."), i.LocalRunning, len(i.ActiveWork), len(i.ScheduledWork)))
	}
	missing := []string{}
	for key, value := range i.Coverage {
		if value != "ok" && value != "intentional-disabled" {
			missing = append(missing, key+": "+value)
		}
	}
	if i.Unknown == "" && i.HostGeneration != "" {
		for _, key := range []string{"agents", "jobs", "schedule", "bots"} {
			if _, ok := i.Coverage[key]; !ok {
				missing = append(missing, key+": unknown")
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		lines = append(lines, resumeText("以下检查状态未知：", "These checks are unknown: ")+strings.Join(missing, ", "))
	}
	lines = append(lines, resumeText("继续可能中断任务；这不代表任务已经完成。是否仍要", "Continuing may interrupt work; this does not mean it has finished. Continue with ")+label+"?")
	result, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: label, Message: strings.Join(lines, "\n\n"), Buttons: []string{resumeText("取消", "Cancel"), resumeText("仍要", "Continue: ") + label}, DefaultButton: 0, CancelButton: 0})
	return err == nil && result.Button == 1, err
}

func (s *mainState) withTaskProtection(win *mygo.Window, sup *harness.Supervisor, operation string, commit func() error) bool {
	s.mu.Lock()
	busy := s.launcherImportBusy
	s.mu.Unlock()
	if busy {
		s.taskProtectionError(win, errors.New(resumeText("导入尚未结束，请等待完成或取消后重试。", "Import is still running; wait for completion or cancel it before retrying.")))
		return false
	}
	did, err := s.protectTaskAction(context.Background(), win, sup, operation, commit)
	s.taskProtectionError(win, err)
	return did && err == nil
}

// Imports already own the import slot. Reuse the same task inspection, drain
// and confirmation without rejecting that owner as a competing operation.
func (s *mainState) protectTaskAction(ctx context.Context, win *mygo.Window, sup *harness.Supervisor, operation string, commit func() error) (bool, error) {
	var did bool
	var err error
	if sup != nil {
		did, err = sup.ProtectTaskAction(ctx, operation, s.sessionRunningCount, func(i taskcontrol.Inspection) (bool, error) { return s.confirmHostTasks(win, operation, i) }, commit)
	} else {
		// No owned Host exists. Preserve the original local-session confirmation.
		label := resumeText("退出", "Quit")
		if operation == "restart" {
			label = resumeText("重启", "Restart")
		} else if operation == "update" {
			label = resumeText("安装更新并退出", "Install update and quit")
		} else if operation == "component" {
			label = resumeText("更改鲸桥组件", "Change WhaleBridge component")
		} else if operation == "stop" {
			label = resumeText("停止内核", "Stop runtime")
		}
		did, err = taskguard.ConfirmAction(win, s.sessionRunningCount(), label, resumeText("当前有 %d 个会话仍在运行。继续会中断任务，仍要继续吗？", "%d sessions are still running. Continuing will interrupt work. Continue?"), label, resumeText("取消", "Cancel"))
		if did && err == nil {
			err = commit()
			did = err == nil
		}
	}
	return did, err
}

func (s *mainState) quitProtected(win *mygo.Window, sup *harness.Supervisor) bool {
	s.mu.Lock()
	busy := s.launcherImportBusy
	s.mu.Unlock()
	if busy {
		s.taskProtectionError(win, errors.New("import is still running; finish or cancel it before quitting"))
		return false
	}
	// Save before acquiring the Host admission lock: a disk-recovery dialog must
	// not leave a lock held indefinitely or let its TTL expire before shutdown.
	if !s.saveResumeBeforeClose(win) {
		return false
	}
	return s.withTaskProtection(win, sup, "quit", func() error {
		s.mu.Lock()
		busy := s.launcherImportBusy
		same := s.sup == sup
		s.mu.Unlock()
		if busy || !same {
			return errors.New("runtime or import state changed; retry quit")
		}
		s.finalizeQuit(sup)
		return nil
	})
}

// stopProtected is the launcher's explicit Stop entry. Keep the window and
// running Host intact until the same admission checks used by restart succeed.
func (s *mainState) stopProtected(win *mygo.Window, sup *harness.Supervisor) {
	if sup == nil {
		return
	}
	s.mu.Lock()
	if s.taskRestartPending || s.quitting {
		s.mu.Unlock()
		return
	}
	s.taskRestartPending = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.taskRestartPending = false; s.mu.Unlock() }()
	s.withTaskProtection(win, sup, "stop", func() error {
		s.mu.Lock()
		if s.launcherImportBusy || s.quitting || s.sup != sup {
			s.mu.Unlock()
			return errors.New("runtime or import state changed; retry stop")
		}
		s.mu.Unlock()
		s.stopPhoneRuntime()
		sup.Stop()
		s.mu.Lock()
		s.launcherStartBusy = false
		s.launcherStartFailed = false
		s.launcherStartErr = ""
		s.mu.Unlock()
		return nil
	})
	if win != nil {
		win.Update(func() {})
	}
}

// installStagedUpdate hands off only verified bytes and uses the already-held
// lifecycle lock for final cleanup, without reentering quitProtected.
func (s *mainState) installStagedUpdate(win *mygo.Window, staged desktopupdate.ApplyResult) bool {
	s.mu.Lock()
	if s.taskRestartPending || s.quitting || s.launcherImportBusy {
		s.mu.Unlock()
		return false
	}
	s.taskRestartPending = true
	sup := s.sup
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.taskRestartPending = false; s.mu.Unlock() }()
	choice, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning,
		Title:   resumeText("安装更新", "Install update"),
		Message: resumeText("更新已下载并校验。继续会启动安装器，停止 Host 并退出桌面应用。", "The update has been downloaded and verified. Continuing starts the installer, stops the Host and quits the desktop application."),
		Buttons: []string{resumeText("稍后", "Later"), resumeText("安装并退出", "Install and quit")}, DefaultButton: 0, CancelButton: 0})
	if err != nil {
		s.taskProtectionError(win, err)
		return false
	}
	if choice.Button != 1 || !s.saveResumeBeforeClose(win) {
		return false
	}
	did := s.withTaskProtection(win, sup, "update", func() error {
		s.mu.Lock()
		changed := s.launcherImportBusy || s.quitting || s.sup != sup
		s.mu.Unlock()
		if changed {
			return errors.New("runtime or import state changed; retry update")
		}
		if _, err := desktopupdate.StartStaged(staged); err != nil {
			return err
		}
		s.finalizeQuit(sup)
		return nil
	})
	if did {
		mygo.App.Quit()
	}
	return did
}
