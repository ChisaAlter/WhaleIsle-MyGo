package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func whaleTaskText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}
func (s *mainState) assistantWhaleTasksModel(ep harness.Endpoint) assistantui.WhaleTasksModel {
	client, host := s.whaleSettingsOwner(ep)
	if host != "" {
		s.whaleTasks.Bind(host)
	}
	current := func() bool {
		now, key := s.whaleSettingsOwner(ep)
		return client != nil && now == client && host != "" && key == host
	}
	var execute func(string, string)
	execute = func(action, id string) {
		if !current() {
			return
		}
		ticket, draft, ok := s.whaleTasks.Begin(host, action)
		if !ok {
			return
		}
		s.refreshWhaleSettingsUI()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			accepted := false
			notice := ""
			var catalog *pluginrpc.WhaleTasks
			rpc := s.pluginRPCClient(ep)
			err := func() error {
				if !current() || !s.whaleTasks.Current(ticket) {
					return errors.New("task panel or Host changed")
				}
				switch action {
				case "start":
					if _, err := rpc.WhaleStartTask(ctx, draft.Draft()); err != nil {
						return err
					}
					accepted = true
					notice = whaleTaskText("任务已接收，可离开此面板。", "Task accepted. You can leave this panel.")
				case "stop", "resume":
					if _, err := rpc.WhaleControlTask(ctx, id, action); err != nil {
						return err
					}
					accepted = true
				case "owner", "revoke-owner":
					if err := rpc.WhaleTaskOwner(ctx, strings.TrimSpace(draft.BotID), strings.TrimSpace(draft.Actor), action == "revoke-owner"); err != nil {
						return err
					}
					accepted = true
					notice = whaleTaskText("飞书任务授权已更新。", "Feishu task authorization updated.")
				case "refresh":
				default:
					return errors.New("unknown task action")
				}
				value, err := rpc.WhaleTasks(ctx)
				if err != nil {
					return err
				}
				catalog = &value
				return nil
			}()
			if !current() {
				s.whaleTasks.Cancel(ticket)
				s.refreshWhaleSettingsUI()
				return
			}
			if !s.whaleTasks.Finish(ticket, action, catalog, accepted, notice, err) {
				return
			}
			s.refreshWhaleSettingsUI()
			if err == nil && catalog != nil {
				poll := false
				for _, task := range catalog.Tasks {
					poll = poll || task.Active()
					for _, effect := range task.Effects {
						poll = poll || effect.State == "pending" || effect.State == "sending"
					}
				}
				if poll {
					time.AfterFunc(2*time.Second, func() {
						if current() {
							execute("refresh", "")
						}
					})
				}
			}
		}()
	}
	snap := s.whaleTasks.Snapshot()
	if current() && !snap.Loaded && !snap.Busy && snap.Error == "" {
		execute("refresh", "")
		snap = s.whaleTasks.Snapshot()
	}
	return assistantui.WhaleTasksModel{State: snap,
		OnRefresh: func() { execute("refresh", "") }, OnStart: func() { execute("start", "") },
		OnControl:   func(id, action string) { execute(action, id) },
		OnAuthorize: func() { execute("owner", "") }, OnRevoke: func() { execute("revoke-owner", "") },
		OnEdit: func(field, value string) {
			if current() {
				s.whaleTasks.Edit(host, field, value)
				s.refreshWhaleSettingsUI()
			}
		},
		OnPickFiles: func() {
			if !current() {
				return
			}
			s.mu.Lock()
			win := s.win
			s.mu.Unlock()
			files, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: whaleTaskText("选择整理资料", "Choose reading inputs"), Multiple: true, Filters: []mygo.FileFilter{{Name: whaleTaskText("文本资料", "Text files"), Extensions: []string{"txt", "md", "mdx", "csv", "json", "log", "rst"}}}})
			if err != nil {
				s.whaleTasks.SetError(host, err)
				s.refreshWhaleSettingsUI()
				return
			}
			if len(files) == 0 || !current() {
				return
			}
			s.whaleTasks.Edit(host, "files", strings.Join(files, "\n"))
			s.refreshWhaleSettingsUI()
		},
		OnOpenSession: func(id string) {
			if !current() {
				return
			}
			found := false
			for _, task := range s.whaleTasks.Snapshot().Catalog.Tasks {
				found = found || task.Execution != nil && task.Execution.SessionID == id
			}
			if !found {
				return
			}
			s.mu.Lock()
			sessions := s.sessions
			s.assistantOpen = false
			s.assistantConversationGeneration++
			s.mu.Unlock()
			s.whaleTasks.Suspend()
			if sessions != nil {
				sessions.Select(id)
			}
			s.refreshWhaleSettingsUI()
		},
	}
}
