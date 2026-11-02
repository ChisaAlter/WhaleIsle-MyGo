package session

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// CaptureOpenUI pins the user-visible assistant block's queue target and returns
// a revocation check. It never derives an address from renderer event fields.
func (s *Store) CaptureOpenUI(binding, sid, messageID, program string) (*ComposerTarget, func() bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	contains := func() bool {
		if strings.HasPrefix(messageID, "tool:") {
			callID := strings.TrimPrefix(messageID, "tool:")
			latest, ok := latestWhaleUITool(s.messages, callID)
			if !ok {
				return false
			}
			doc, ok := WhaleUIToolDocument(latest)
			return ok && doc.Program == program && whaleUICallNamed(s.messages, callID)
		}
		for _, m := range s.messages {
			if (m.ID == messageID || m.SourceMessageID == messageID) && m.Role == "assistant" && strings.Contains(m.Body, program) {
				return true
			}
		}
		return false
	}
	if s.client == nil || sid == "" || s.selectedID != sid || s.planHistoryBinding != binding || program == "" || !contains() {
		return nil, nil, errors.New("OpenUI block owner is no longer active")
	}
	target := &ComposerTarget{SessionID: sid, store: s, client: s.client, epoch: s.composerHostGeneration, followGeneration: s.followGeneration, mode: PromptModeQueue}
	valid := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return target.currentLocked() && s.followGeneration == target.followGeneration && s.selectedID == sid && s.planHistoryBinding == binding && contains()
	}
	return target, valid, nil
}

// WhaleUIDocument is a settled, successful render_ui result, bound to its real
// tool call. All consumers use this parser; arbitrary output text is never UI.
type WhaleUIDocument struct {
	CallID   string
	Language string
	Program  string
	Seq      float64
}

func WhaleUIToolDocument(msg Message) (WhaleUIDocument, bool) {
	if msg.Tool != nil {
		if msg.Tool.State != "ok" || msg.Tool.Result == nil || !isRenderUIName(msg.Tool.Name) {
			return WhaleUIDocument{}, false
		}
		doc, ok := WhaleUIToolDocument(*msg.Tool.Result)
		return doc, ok && doc.CallID == msg.Tool.CallID
	}
	if msg.Kind != MessageKindTool {
		return WhaleUIDocument{}, false
	}
	var e struct {
		Type string `json:"type"`
		Data struct {
			Name      string          `json:"name"`
			SubCallID string          `json:"subCallId"`
			IsError   bool            `json:"isError"`
			Error     json.RawMessage `json:"error"`
			Message   struct {
				ToolCallID string `json:"toolCallId"`
				IsError    bool   `json:"isError"`
				Source     struct {
					CallID string `json:"callId"`
				} `json:"source"`
			} `json:"message"`
			Meta struct {
				WhaleUI struct {
					Version  int    `json:"version"`
					Language string `json:"language"`
					Program  string `json:"program"`
				} `json:"whaleUi"`
			} `json:"meta"`
		} `json:"data"`
	}
	if json.Unmarshal(msg.Raw, &e) != nil || (e.Type != "tool/result" && e.Type != "tool/ptc-dispatch") || (msg.EventType != "" && msg.EventType != e.Type) || e.Data.IsError || e.Data.Message.IsError || (len(e.Data.Error) > 0 && string(e.Data.Error) != "null") {
		return WhaleUIDocument{}, false
	}
	d := e.Data.Meta.WhaleUI
	if d.Version != 1 || (d.Language != "whale-ui" && d.Language != "dsh-ui" && d.Language != "openui" && d.Language != "openui-lang") || strings.TrimSpace(d.Program) == "" || len(d.Program) > 262144 || !utf8.ValidString(d.Program) {
		return WhaleUIDocument{}, false
	}
	id := e.Data.Message.ToolCallID
	if id == "" {
		id = e.Data.Message.Source.CallID
	}
	if e.Data.Message.ToolCallID != "" && e.Data.Message.Source.CallID != "" && e.Data.Message.ToolCallID != e.Data.Message.Source.CallID {
		return WhaleUIDocument{}, false
	}
	if e.Type == "tool/ptc-dispatch" {
		if !isRenderUIName(e.Data.Name) {
			return WhaleUIDocument{}, false
		}
		id = e.Data.SubCallID
	}
	if strings.TrimSpace(id) == "" {
		return WhaleUIDocument{}, false
	}
	return WhaleUIDocument{CallID: id, Language: d.Language, Program: d.Program, Seq: msg.Seq}, true
}

// CaptureWhaleUIPanel pins a native panel's queue target to the same follow and
// Host epoch; the UI adds a snapshot-generation check supplied only by itself.
func (s *Store) CaptureWhaleUIPanel(binding, sid string, contentCurrent func() bool) (*ComposerTarget, func() bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || sid == "" || s.selectedID != sid || s.planHistoryBinding != binding || contentCurrent == nil || !contentCurrent() {
		return nil, nil, errors.New("Whale UI panel owner is no longer active")
	}
	target := &ComposerTarget{SessionID: sid, store: s, client: s.client, epoch: s.composerHostGeneration, followGeneration: s.followGeneration, mode: PromptModeQueue}
	valid := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return target.currentLocked() && s.followGeneration == target.followGeneration && s.selectedID == sid && s.planHistoryBinding == binding && contentCurrent()
	}
	return target, valid, nil
}
func latestWhaleUITool(messages []Message, id string) (Message, bool) {
	var latest Message
	found := false
	for _, m := range messages {
		p, ok := readToolLifecyclePart(m)
		if ok && p.terminal && p.id == id && (!found || m.Seq >= latest.Seq) {
			latest = m
			found = true
		}
	}
	return latest, found
}

// WhaleUIStorageScope excludes credentials and volatile client pointers. It is
// stable across reconnection/restart for the same Host origin and user profile.
func (s *Store) WhaleUIStorageScope() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return ""
	}
	return s.client.Endpoint.Origin
}
func (t *ComposerTarget) StorageScope() string {
	if t == nil || t.client == nil {
		return ""
	}
	return t.client.Endpoint.Origin
}
func isRenderUIName(name string) bool {
	return name == "render_ui" || strings.HasSuffix(name, ".render_ui") || strings.HasSuffix(name, "__render_ui")
}
func whaleUICallNamed(messages []Message, id string) bool {
	for _, m := range messages {
		p, ok := readToolLifecyclePart(m)
		if ok && p.id == id && isRenderUIName(p.name) && (!p.terminal || m.EventType == "tool/ptc-dispatch") {
			return true
		}
	}
	return false
}
