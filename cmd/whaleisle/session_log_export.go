package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type sessionLogExportKey struct {
	client *dsh.Client
	id     string
}

type sessionLogExportOperation struct {
	cancel context.CancelFunc
}

func sessionLogFilename(id string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, id)
	return "dsh-session-" + safe + ".zip"
}

func sessionHasExportCommand(v session.ViewState) bool {
	for _, command := range v.SlashCommands {
		if command.Name == "export" && command.DefinitionId != nil && *command.DefinitionId == "@deepseek-ai/dsh-session-log-export" {
			return true
		}
	}
	return false
}

func (s *mainState) sessionLogDownloading(id string) bool {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionLogDownloads[sessionLogExportKey{client, id}] != nil
}

// Called while s.mu is held as the Host generation retires.
func (s *mainState) cancelSessionLogExportsLocked() {
	for key, operation := range s.sessionLogDownloads {
		operation.cancel()
		delete(s.sessionLogDownloads, key)
	}
}

func (s *mainState) sessionLogExportAction(win *mygo.Window, sessions *session.Store, v session.ViewState) func(string) {
	if sessions == nil || !v.ClientReady || v.SelectedID == "" {
		return nil
	}
	return func(id string) {
		if id == v.SelectedID && sessions.SelectedSessionID() == id {
			s.requestSessionLogExport(win, sessions, id)
		}
	}
}

func (s *mainState) sessionFeedbackAction(win *mygo.Window, sessions *session.Store, v session.ViewState) func(string) {
	if sessions == nil || !v.NativeContributions.Known || !v.NativeContributions.FeedbackCommand {
		return nil
	}
	return func(id string) {
		s.mu.Lock()
		current := s.sessions == sessions && !s.quitting
		s.mu.Unlock()
		live := sessions.Snapshot()
		if current && id == v.SelectedID && live.SelectedID == id && live.ComposerHostGeneration == v.ComposerHostGeneration && live.NativeContributions.FeedbackCommand {
			sessions.OpenSessionFeedback(id)
			if win != nil {
				win.Invalidate()
			}
		}
	}
}

// A browser /export opens its download controller, not commands/execute alone.
// Consume only the exact bare invocation of the published command and preserve
// every attachment and the draft when the native Save dialog is cancelled.
func (s *mainState) handleSessionLogExportCommand(win *mygo.Window, sessions *session.Store, d *composer.DraftState, v session.ViewState) bool {
	if d.EditSeq > 0 || strings.TrimSpace(d.Text) != "/export" {
		return false
	}
	if !v.SlashCommandsKnown {
		d.Error = v.SlashCommandsErr
		if d.Error == "" {
			d.Error = resumeText("正在加载命令目录，请稍后重试。", "Loading the command catalog. Try again shortly.")
		}
		return true
	}
	if !sessionHasExportCommand(v) {
		return false
	}
	d.Error = ""
	if s.requestSessionLogExport(win, sessions, v.SelectedID) {
		if d.Client != nil {
			d.Client.ReplaceText(&ui.TextInputRange{Start: 0, End: ui.UTF16Len(d.Text)}, "")
		} else {
			d.Text = ""
		}
	}
	return true
}

func (s *mainState) requestSessionLogExport(win *mygo.Window, sessions *session.Store, id string) bool {
	if win == nil || sessions == nil || id == "" || sessions.SelectedSessionID() != id {
		return false
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	key := sessionLogExportKey{client, id}
	s.mu.Lock()
	if client == nil || s.sessions != sessions || s.quitting || s.sessionLogDownloads[key] != nil {
		s.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	operation := &sessionLogExportOperation{cancel: cancel}
	if s.sessionLogDownloads == nil {
		s.sessionLogDownloads = make(map[sessionLogExportKey]*sessionLogExportOperation)
	}
	s.sessionLogDownloads[key] = operation
	s.mu.Unlock()
	finish := func(message string) {
		cancel()
		s.mu.Lock()
		current := s.sessionLogDownloads[key] == operation
		if current {
			delete(s.sessionLogDownloads, key)
		}
		s.mu.Unlock()
		if current && !win.IsDestroyed() {
			if message != "" {
				s.notifyUser(win, message)
			}
			win.Invalidate()
		}
	}
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: resumeText("下载会话日志", "Download session log"), DefaultPath: sessionLogFilename(id)})
	if err != nil || path == "" || ctx.Err() != nil {
		message := ""
		if err != nil {
			message = resumeText("无法选择日志保存位置：", "Could not choose the log destination: ") + err.Error()
		}
		finish(message)
		return false
	}
	s.notifyUser(win, resumeText("正在准备当前会话、子会话和附件的 ZIP 文件…", "Preparing a ZIP of this session, its sub-sessions, and attachments…"))
	go func() {
		// A failed/cancelled stream never truncates an existing user file.
		file, err := os.CreateTemp(filepath.Dir(path), ".dsh-session-*.zip")
		if err == nil {
			temporary := file.Name()
			defer os.Remove(temporary)
			err = client.DownloadSessionLog(ctx, id, file)
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
			if err == nil {
				err = ctx.Err()
			}
			if err == nil {
				err = os.Rename(temporary, path)
			}
		}
		message := resumeText("会话日志已保存", "Session log saved")
		if err != nil {
			message = resumeText("会话日志导出失败：", "Session log export failed: ") + err.Error()
		}
		mygo.RunOnMain(func() { finish(message) })
	}()
	return true
}
