package assistant

import (
	"fmt"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

// WhaleSettingsEditor holds only this native editor's in-memory drafts. It
// never reads local MEMORY.md, persists drafts, or starts model/session work.
type WhaleSettingsEditor struct {
	mu                          sync.Mutex
	host                        string
	epoch, serial, revision     uint64
	pending                     uint64
	loaded, seen, busy, confirm bool
	server                      pluginrpc.WhaleSettings
	draft                       pluginrpc.WhaleSettings
	notice, lastErr             string
}
type WhaleSettingsSnapshot struct {
	HostKey                              string
	Loaded, Busy, ConfirmClear           bool
	Name, UserTitle, PersonaText, Memory string
	Server                               pluginrpc.WhaleSettings
	Notice, Error                        string
	Revision                             uint64
}
type WhaleSettingsTicket struct {
	host                    string
	epoch, serial, revision uint64
}
type WhaleSettingsCommand struct {
	Action      string
	Base, Value pluginrpc.WhaleSettings
}

func (s *WhaleSettingsEditor) Bind(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.host == host {
		return
	}
	s.epoch++
	s.host = host
	s.pending = 0
	s.loaded = false
	s.seen = false
	s.busy = false
	s.confirm = false
	s.server = pluginrpc.WhaleSettings{}
	s.draft = pluginrpc.WhaleSettings{}
	s.notice = ""
	s.lastErr = ""
	s.revision++
}
func (s *WhaleSettingsEditor) Suspend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.epoch++
	s.pending = 0
	if s.busy {
		s.loaded = false
		s.notice = "操作已中断，请重新读取核对 / Interrupted; refresh to verify"
	}
	s.busy = false
	s.confirm = false
}
func (s *WhaleSettingsEditor) Snapshot() WhaleSettingsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return WhaleSettingsSnapshot{HostKey: s.host, Loaded: s.loaded, Busy: s.busy, ConfirmClear: s.confirm, Name: s.draft.Name, UserTitle: s.draft.UserTitle, PersonaText: s.draft.PersonaText, Memory: s.draft.Memory, Server: s.server, Notice: s.notice, Error: s.lastErr, Revision: s.revision}
}
func (s *WhaleSettingsEditor) Edit(host, field, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || s.host != host {
		return
	}
	switch field {
	case "memory":
		s.draft.Memory = value
	case "persona":
		s.draft.PersonaText = value
	case "name":
		s.draft.Name = value
	case "title":
		s.draft.UserTitle = value
	default:
		return
	}
	s.revision++
	s.confirm = false
	s.notice = ""
}
func (s *WhaleSettingsEditor) Restore(host, field string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host != s.host || s.busy || !s.loaded {
		return
	}
	switch field {
	case "memory":
		s.draft.Memory = s.server.Memory
	case "persona":
		s.draft.PersonaText = s.server.PersonaText
	case "identity":
		s.draft.Name = s.server.Name
		s.draft.UserTitle = s.server.UserTitle
	default:
		return
	}
	s.revision++
	s.confirm = false
	s.lastErr = ""
	s.notice = "草稿已恢复为当前 Host 内容 / Draft restored from Host"
}
func (s *WhaleSettingsEditor) ticket() WhaleSettingsTicket {
	s.serial++
	s.pending = s.serial
	s.busy = true
	s.confirm = false
	s.lastErr = ""
	s.notice = ""
	return WhaleSettingsTicket{s.host, s.epoch, s.serial, s.revision}
}
func (s *WhaleSettingsEditor) matches(q WhaleSettingsTicket) bool {
	return q.host == s.host && q.epoch == s.epoch && q.serial == s.pending && s.busy
}
func (s *WhaleSettingsEditor) BeginLoad(host string) (WhaleSettingsTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.host || s.busy {
		return WhaleSettingsTicket{}, false
	}
	return s.ticket(), true
}
func (s *WhaleSettingsEditor) mergeRead(v pluginrpc.WhaleSettings) {
	if !s.seen || !s.server.SameIdentity(v) {
		s.draft = v
		s.revision++
		if s.seen {
			s.notice = "Whale 身份已变更，已隔离之前的草稿 / Whale identity changed"
		}
	} else {
		if s.draft.Memory == s.server.Memory {
			s.draft.Memory = v.Memory
		}
		if s.draft.Name == s.server.Name {
			s.draft.Name = v.Name
		}
		if s.draft.UserTitle == s.server.UserTitle {
			s.draft.UserTitle = v.UserTitle
		}
		if s.draft.PersonaText == s.server.PersonaText {
			s.draft.PersonaText = v.PersonaText
		}
	}
	s.server = v
	s.loaded = true
	s.seen = true
}
func (s *WhaleSettingsEditor) FinishLoad(q WhaleSettingsTicket, v pluginrpc.WhaleSettings, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.matches(q) {
		return false
	}
	s.busy = false
	s.pending = 0
	if err != nil {
		s.loaded = false
		s.lastErr = err.Error()
		return true
	}
	s.mergeRead(v)
	if s.notice == "" {
		s.notice = "已读取；未保存的草稿保持不变 / Loaded; unsaved drafts preserved"
	}
	return true
}
func (s *WhaleSettingsEditor) ReviewClear(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == s.host && s.loaded && !s.busy {
		s.confirm = true
		s.notice = ""
	}
}
func (s *WhaleSettingsEditor) CancelClear(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == s.host && !s.busy {
		s.confirm = false
	}
}
func (s *WhaleSettingsEditor) BeginWrite(host, action string) (WhaleSettingsTicket, WhaleSettingsCommand, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.host || s.busy || !s.loaded {
		return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
	}
	cmd := WhaleSettingsCommand{action, s.server, s.draft}
	var err error
	switch action {
	case "clear":
		if !s.confirm {
			return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
		}
	case "memory":
		if pluginrpc.WhaleTextLength(cmd.Value.Memory) > 40000 {
			err = fmt.Errorf("长期记忆超过 40000 字符限制 / Memory exceeds 40000 UTF-16 units")
		}
	case "identity":
		err = pluginrpc.ValidateWhaleSettingsPatch(pluginrpc.WhaleSettingsPatch{Name: &cmd.Value.Name, UserTitle: &cmd.Value.UserTitle})
	case "persona":
		err = pluginrpc.ValidateWhaleSettingsPatch(pluginrpc.WhaleSettingsPatch{PersonaText: &cmd.Value.PersonaText})
	default:
		return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
	}
	if err != nil {
		s.lastErr = err.Error()
		return WhaleSettingsTicket{}, WhaleSettingsCommand{}, false
	}
	return s.ticket(), cmd, true
}

// Check compares only the edited field, plus the canonical Whale identity.
// The upstream API has no CAS; this preflight detects known changes, not races.
func (cmd WhaleSettingsCommand) Check(current pluginrpc.WhaleSettings) error {
	if !cmd.Base.SameIdentity(current) {
		return fmt.Errorf("Whale 身份已改变，请重新读取 / Whale identity changed; refresh")
	}
	same := false
	switch cmd.Action {
	case "memory", "clear":
		same = cmd.Base.Memory == current.Memory
	case "identity":
		same = cmd.Base.Name == current.Name && cmd.Base.UserTitle == current.UserTitle
	case "persona":
		same = cmd.Base.PersonaText == current.PersonaText
	}
	if !same {
		return fmt.Errorf("Host 内容已变化，未提交；请重新读取并核对草稿 / Host content changed; refresh and compare your draft")
	}
	return nil
}
func (s *WhaleSettingsEditor) FinishWrite(q WhaleSettingsTicket, cmd WhaleSettingsCommand, actual *pluginrpc.WhaleSettings, accepted bool, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.matches(q) {
		return false
	}
	s.busy = false
	s.pending = 0
	s.confirm = false
	if err != nil || actual == nil || !accepted {
		s.loaded = false
		if accepted {
			s.notice = "Host 已确认操作；重新读取后才能继续 / Host confirmed the operation; refresh to verify"
		} else {
			s.notice = "未确认完成；草稿已保留，先重新读取再决定是否重试 / Not confirmed; draft retained. Refresh before retrying"
		}
		if err != nil {
			s.lastErr = err.Error()
		}
		return true
	}
	if !cmd.Base.SameIdentity(*actual) {
		s.loaded = false
		s.lastErr = "操作后 Whale 身份已变化，请重新读取核对 / Whale identity changed after submission"
		return true
	}
	matches := true
	switch cmd.Action {
	case "memory":
		matches = actual.Memory == cmd.Value.Memory
	case "persona":
		matches = actual.PersonaText == cmd.Value.PersonaText
	case "identity":
		name := strings.TrimSpace(cmd.Value.Name)
		if name == "" {
			name = "鲸鱼娘"
		}
		matches = actual.Name == name && actual.UserTitle == strings.TrimSpace(cmd.Value.UserTitle)
	}
	if !matches {
		s.mergeWriteRead(*actual, cmd.Action, true)
		s.lastErr = "Host 已接受写入，但读回内容不一致；草稿已保留，请核对 / Host accepted the write but read-back differs; compare the preserved draft"
		return true
	}
	if q.revision == s.revision {
		switch cmd.Action {
		case "memory", "clear":
			s.draft.Memory = actual.Memory
		case "identity":
			s.draft.Name = actual.Name
			s.draft.UserTitle = actual.UserTitle
		case "persona":
			s.draft.PersonaText = actual.PersonaText
		}
	}
	s.mergeWriteRead(*actual, cmd.Action, q.revision != s.revision)
	s.lastErr = ""
	s.notice = "已保存并从 Host 重新读取 / Saved and read back from Host"
	if cmd.Action == "clear" {
		s.notice = "清空已由 Host 确认；已读取当前内容 / Clear confirmed by Host; current contents loaded"
	}
	return true
}

// Current invalidates a captured write when the editor closes or owner changes.
func (s *WhaleSettingsEditor) Current(q WhaleSettingsTicket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.matches(q)
}

// A newer edit can intentionally equal the previous server value. Do not let
// the usual clean-field refresh heuristic consume that edit after a write.
func (s *WhaleSettingsEditor) mergeWriteRead(actual pluginrpc.WhaleSettings, action string, preserve bool) {
	draft := s.draft
	s.mergeRead(actual)
	if !preserve {
		return
	}
	switch action {
	case "memory", "clear":
		s.draft.Memory = draft.Memory
	case "persona":
		s.draft.PersonaText = draft.PersonaText
	case "identity":
		s.draft.Name = draft.Name
		s.draft.UserTitle = draft.UserTitle
	}
}
