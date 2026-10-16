package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/harness"
	panel "whaleisle/internal/store/custompanel"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
	panelui "whaleisle/internal/ui/custompanel"
)

type customPanelState struct {
	Store     *panel.Store
	OpenError string
	Editors   map[panel.Scope]*panelui.Editor
}

func (s *mainState) openCustomPanels(userData string) {
	store, err := panel.Open(userData)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customPanels.Store = store
	if err != nil {
		s.customPanels.OpenError = err.Error()
	}
}
func customPanelBotScope(hostScope, origin, sid string, roster dshbotstore.Snapshot) (panel.Scope, string) {
	if hostScope == "" || origin == "" || sid == "" || roster.Origin != origin {
		return panel.Scope{}, ""
	}
	for _, row := range roster.Rows {
		if row.Kind != "room" && !row.Hidden && row.ID != "" && row.SessionID == sid {
			return panel.Scope{Host: hostScope, Bot: row.ID}, row.DisplayLabel
		}
	}
	return panel.Scope{}, ""
}
func (s *mainState) currentCustomPanelScope(sid string) (panel.Scope, string) {
	s.mu.Lock()
	roster, cfg, sessions, ep, ready := s.dshbotRoster, s.cfgStore, s.sessions, s.bootStatus.Endpoint, s.bootStatus.State == harness.Ready
	s.mu.Unlock()
	if !ready || roster == nil || cfg == nil || !cfg.Get().DshbotEnabled || sessions == nil || sessions.Snapshot().SelectedID != sid {
		return panel.Scope{}, ""
	}
	return customPanelBotScope(sessions.WorkspaceLayoutScope(), ep.Origin, sid, roster.Snapshot())
}
func latestCustomPanelReply(messages []session.Message) (panel.Definition, error) {
	for n := len(messages) - 1; n >= 0; n-- {
		m := messages[n]
		if m.Role == "assistant" && m.Kind == session.MessageKindText && !m.Streaming && strings.TrimSpace(m.Body) != "" {
			return panel.Parse(m.Body)
		}
	}
	return panel.Definition{}, fmt.Errorf("尚无已完成的 Bot JSON 回复 / No completed Bot JSON reply")
}

// The content hook resolves ownership from the currently visible canonical session.
// Captured callbacks recheck it before touching disk or the composer.
func (s *mainState) customPanelContent(sid, themeID string) func(*ui.Context) {
	return func(c *ui.Context) {
		scope, label := s.currentCustomPanelScope(sid)
		s.mu.Lock()
		store := s.customPanels.Store
		openErr := s.customPanels.OpenError
		if s.customPanels.Editors == nil {
			s.customPanels.Editors = map[panel.Scope]*panelui.Editor{}
		}
		editor := s.customPanels.Editors[scope]
		if editor == nil {
			editor = &panelui.Editor{JSON: panel.Example}
			s.customPanels.Editors[scope] = editor
		}
		sessions := s.sessions
		endpoint := s.bootStatus.Endpoint
		s.mu.Unlock()
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		current := func() bool {
			s.dshMu.Lock()
			sameClient := client != nil && s.dshClient == client
			s.dshMu.Unlock()
			s.mu.Lock()
			sameEndpoint := s.bootStatus.Endpoint == endpoint && s.sessions == sessions
			s.mu.Unlock()
			now, _ := s.currentCustomPanelScope(sid)
			return sameClient && sameEndpoint && scope.Bot != "" && now == scope
		}
		report := func(err error, notice string) {
			editor.Error = ""
			editor.Notice = ""
			if err != nil {
				editor.Error = err.Error()
			} else {
				editor.Notice = notice
			}
		}
		if openErr != "" {
			editor.Error = "Panel library: " + openErr
		}
		available := store != nil && scope.Bot != ""
		snapshot := panel.Snapshot{}
		if available {
			snapshot = store.Snapshot(scope)
		}
		prepare := func(text string) {
			if !current() {
				return
			}
			s.mu.Lock()
			draft := s.composerDraftLocked(sid)
			if draft.Text != "" || len(draft.Attachments) != 0 || draft.Sending || draft.Preparing > 0 || draft.EditSeq != 0 {
				s.mu.Unlock()
				report(fmt.Errorf("请先处理主聊天已有草稿 / Resolve the existing chat draft first"), "")
				return
			}
			draft.Text = text
			s.mu.Unlock()
			draft.Client.Sync()
			report(nil, "已填入主聊天草稿；请审阅后发送 / Draft prepared; review and send")
		}
		edit := func(d panel.Definition, template bool) {
			if !current() {
				return
			}
			raw, _ := json.MarshalIndent(d, "", "  ")
			editor.JSON = string(raw)
			editor.EditingID = d.ID
			editor.Template = template
			editor.Preview = nil
			editor.Error = ""
			editor.Notice = ""
		}
		model := panelui.Model{ThemeID: themeID, SystemDark: mygo.Theme.IsDark(), BotLabel: label, Available: available, State: snapshot, Editor: editor, OnPrompt: prepare}
		model.OnCreate = func() { prepare(panel.CreationPrompt(editor.Requirements)) }
		model.OnNew = func() {
			if !current() {
				return
			}
			editor.JSON = panel.Example
			editor.EditingID = ""
			editor.Template = false
			editor.Preview = nil
			editor.Error = ""
			editor.Notice = ""
		}
		model.OnEdit = edit
		model.OnImport = func() {
			if !current() || sessions == nil {
				return
			}
			d, err := latestCustomPanelReply(sessions.Snapshot().Messages)
			if err != nil {
				report(err, "")
				return
			}
			d.ID = editor.EditingID
			raw, _ := json.MarshalIndent(d, "", "  ")
			editor.JSON = string(raw)
			editor.Preview = &d
			report(nil, "已导入预览；保存后生效 / Imported for preview; save to apply")
		}
		model.OnPreview = func() {
			if !current() {
				return
			}
			d, err := panel.Parse(editor.JSON)
			editor.Preview = nil
			if err == nil {
				d.ID = editor.EditingID
				editor.Preview = &d
			}
			report(err, "")
		}
		model.OnSave = func() {
			if !current() || store == nil || editor.Preview == nil {
				return
			}
			d, err := panel.Parse(editor.JSON)
			if err != nil {
				report(err, "")
				return
			}
			d.ID = editor.EditingID
			if d.ID == "" {
				_, err = store.Create(scope, d)
			} else if editor.Template {
				err = store.SaveTemplate(d)
			} else {
				found := false
				for _, i := range store.Snapshot(scope).Instances {
					if i.Definition.ID == d.ID {
						i.Definition = d
						err = store.Update(scope, i)
						found = true
						break
					}
				}
				if !found {
					err = fmt.Errorf("Panel instance no longer exists")
				}
			}
			report(err, "已保存；其他 Bot 的副本保持独立 / Saved; other Bot copies remain independent")
			if err == nil {
				editor.Preview = nil
				editor.EditingID = ""
				editor.Template = false
			}
		}
		model.OnAdd = func(id string) {
			if current() {
				report(store.Add(scope, id), "已添加独立副本 / Independent copy added")
			}
		}
		model.OnUpdate = func(i panel.Instance) {
			if current() {
				report(store.Update(scope, i), "")
			}
		}
		model.OnTemplate = func(d panel.Definition) {
			if current() {
				report(store.SaveTemplate(d), "已更新库模板；已有 Bot 副本不变 / Template updated for future copies")
			}
		}
		model.OnMove = func(id string, delta int) {
			if current() {
				report(store.Move(scope, id, delta), "")
			}
		}
		model.OnRemove = func(id string) {
			if current() {
				report(store.Remove(scope, id), "已从本 Bot 移除；可从库重新添加 / Removed from this Bot; template retained")
			}
		}
		panelui.View(c.Key("custom-panels-"+scope.Host+"-"+scope.Bot), model)
	}
}
