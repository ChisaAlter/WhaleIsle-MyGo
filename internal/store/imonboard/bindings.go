package imonboard

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

// Preset contains only the existing connection.status public catalog fields.
type Preset struct {
	ID, Label string
}

var bindingBotID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var bindingPresetID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (s *Store) EditWorkspace(scope uint64, id, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if scope != s.view.Scope || !s.view.Open || s.view.Busy || s.view.Review != "" {
		return
	}
	for _, bot := range s.view.Bots {
		if bot.BotID == id {
			if s.view.WorkspaceDrafts == nil {
				s.view.WorkspaceDrafts = map[string]string{}
			}
			s.view.WorkspaceDrafts[id] = value
			return
		}
	}
}

func (s *Store) SaveWorkspace(scope uint64, id, value string) {
	s.runScoped(scope, "bot.workspace.set", map[string]any{"botId": id, "workspace": strings.TrimSpace(value)}, false)
}

func (s *Store) SaveAgentPreset(scope uint64, id, value string) {
	var preset any
	if value = strings.TrimSpace(value); value != "" {
		preset = value
	}
	s.runScoped(scope, "bot.preset.set", map[string]any{"botId": id, "agentPreset": preset}, false)
}

func (s *Store) validateBindingLocked(method string, payload map[string]any) string {
	id, _ := payload["botId"].(string)
	found := false
	for _, bot := range s.view.Bots {
		if bot.BotID == id {
			found = true
			break
		}
	}
	if !found || !bindingBotID.MatchString(id) {
		return "机器人已不可用，请刷新后重试 / Bot unavailable; refresh to retry"
	}
	if method == "bot.workspace.set" {
		workspace, _ := payload["workspace"].(string)
		absolute := filepath.IsAbs(workspace) || strings.HasPrefix(workspace, "/") || strings.HasPrefix(workspace, `\\`) || len(workspace) >= 3 && ((workspace[0] >= 'A' && workspace[0] <= 'Z') || (workspace[0] >= 'a' && workspace[0] <= 'z')) && workspace[1] == ':' && (workspace[2] == '\\' || workspace[2] == '/')
		if !absolute || len(workspace) > 4096 {
			return "请输入工作区绝对路径 / Enter an absolute workspace path"
		}
		return ""
	}
	if payload["agentPreset"] == nil {
		return "" // The original bot.preset.set uses null to inherit Host default.
	}
	id, _ = payload["agentPreset"].(string)
	if bindingPresetID.MatchString(id) {
		for _, preset := range s.view.Presets {
			if preset.ID == id {
				return ""
			}
		}
	}
	return "Agent Preset 不在当前 Host 公开目录中 / Preset unavailable on this Host"
}

func decodeConnectionStatus(raw json.RawMessage) ([]Bot, []Preset, bool) {
	type snapshot struct {
		Bots    []Bot `json:"bots"`
		Catalog struct {
			Items []json.RawMessage `json:"items"`
		} `json:"agentPresetCatalog"`
	}
	var result struct {
		snapshot
		Snapshot *snapshot `json:"snapshot"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return nil, nil, false
	}
	data := result.snapshot
	if result.Snapshot != nil {
		data = *result.Snapshot
	}
	if data.Bots == nil {
		return nil, nil, false
	}
	var presets []Preset
	seen := map[string]bool{}
	for _, raw := range data.Catalog.Items {
		var id string
		var item struct{ ID, Label, Name string }
		if json.Unmarshal(raw, &id) == nil {
			item.ID = id
		} else if json.Unmarshal(raw, &item) != nil {
			continue
		}
		id = strings.TrimSpace(item.ID)
		if !bindingPresetID.MatchString(id) || seen[id] {
			continue
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = strings.TrimSpace(item.Name)
		}
		if label == "" {
			label = id
		}
		seen[id] = true
		presets = append(presets, Preset{ID: id, Label: label})
	}
	return data.Bots, presets, true
}

func (s *Store) applyConnectionStatusLocked(bots []Bot, presets []Preset) {
	s.view.Bots, s.view.Presets = bots, presets
	// Authoritative state may clear an already-saved edit, never replace an
	// unfinished draft with a periodic snapshot from the Host.
	for _, bot := range bots {
		if draft, ok := s.view.WorkspaceDrafts[bot.BotID]; ok && strings.TrimSpace(draft) == bot.Workspace {
			delete(s.view.WorkspaceDrafts, bot.BotID)
		}
	}
}
