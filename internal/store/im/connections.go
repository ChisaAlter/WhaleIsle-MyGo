package im

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

// Preset is the host's public, non-secret agent preset descriptor.
type Preset struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type presetCatalog struct {
	Items []json.RawMessage `json:"items"`
}

var botIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var presetIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func (c presetCatalog) presets() []Preset {
	var out []Preset
	seen := map[string]bool{}
	for _, raw := range c.Items {
		var p Preset
		var text string
		if json.Unmarshal(raw, &text) == nil {
			p = Preset{ID: text, Label: text}
		} else {
			var item struct {
				ID    string `json:"id"`
				Label string `json:"label"`
				Name  string `json:"name"`
			}
			if json.Unmarshal(raw, &item) != nil {
				continue
			}
			p = Preset{ID: item.ID, Label: item.Label}
			if p.Label == "" {
				p.Label = item.Name
			}
		}
		if !presetIDPattern.MatchString(p.ID) || seen[p.ID] {
			continue
		}
		if p.Label == "" {
			p.Label = p.ID
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out
}
func (s *Store) workspaceDraftLocked(id string) string {
	if draft, ok := s.workspaceDrafts[id]; ok {
		return draft
	}
	for _, row := range s.rows {
		if row.ID == id {
			return row.Workspace
		}
	}
	return ""
}

// SetWorkspaceDraft keeps unfinished edits separate from remote state. It never persists credentials.
func (s *Store) SetWorkspaceDraft(id, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.workspaceDrafts == nil {
		s.workspaceDrafts = map[string]string{}
	}
	s.workspaceDrafts[id] = value
}

// ReconnectBot reconnects without sending a test message.
func (s *Store) ReconnectBot(ctx context.Context, id string) error {
	return s.changeConnection(ctx, id, "bot.reconnect", map[string]any{"sendTest": false})
}
func (s *Store) SaveWorkspace(ctx context.Context, id, workspace string) error {
	workspace = strings.TrimSpace(workspace)
	if len(workspace) > 4096 || !absoluteWorkspace(workspace) {
		return s.connectionFailure(storelocale.T(storelocale.KeyStoreImEnterAnAbsoluteWorkspacePath))
	}
	return s.changeConnection(ctx, id, "bot.workspace.set", map[string]any{"workspace": workspace})
}
func absoluteWorkspace(value string) bool {
	// The host may be Windows even when this client runs on another platform.
	return filepath.IsAbs(value) || strings.HasPrefix(value, "/") || (len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '\\' || value[2] == '/')) || strings.HasPrefix(value, `\\`)
}
func (s *Store) SaveAgentPreset(ctx context.Context, id, preset string) error {
	preset = strings.TrimSpace(preset)
	if preset != "" && !presetIDPattern.MatchString(preset) {
		return s.connectionFailure(storelocale.T(storelocale.KeyStoreImInvalidAgentPreset))
	}
	var value any
	if preset != "" {
		value = preset
	}
	return s.changeConnection(ctx, id, "bot.preset.set", map[string]any{"agentPreset": value})
}
func (s *Store) connectionFailure(message string) error {
	s.mu.Lock()
	s.connectionNote = message
	s.mu.Unlock()
	s.fire()
	return errors.New(message)
}
func (s *Store) changeConnection(ctx context.Context, id, method string, payload map[string]any) error {
	s.mu.Lock()
	var target Entry
	for _, row := range s.rows {
		if row.ID == id {
			target = row
			break
		}
	}
	supported := false
	for _, channel := range api.IMCordisChannels() {
		if target.Channel == channel {
			supported = true
			break
		}
	}
	if !supported || !botIDPattern.MatchString(target.BotID) || s.plugin == nil || s.connectionBusy || s.loading {
		s.mu.Unlock()
		return s.connectionFailure(storelocale.T(storelocale.KeyStoreImTheBotIsUnavailableOrAnOperationIsRu))
	}
	if method == "bot.preset.set" && payload["agentPreset"] != nil {
		found := false
		for _, preset := range target.Presets {
			if preset.ID == payload["agentPreset"] {
				found = true
			}
		}
		if !found {
			s.mu.Unlock()
			return s.connectionFailure(storelocale.T(storelocale.KeyStoreImThatAgentPresetIsNotInTheHostCatalog))
		}
	}
	plugin, epoch := s.plugin, s.connectionEpoch
	s.connectionBusy = true
	s.connectionNote = storelocale.T(storelocale.KeyStoreImUpdatingTheBot)
	payload["botId"] = target.BotID
	s.mu.Unlock()
	s.fire()
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		var result json.RawMessage
		err := plugin.Call(callCtx, target.Channel, method, payload, &result)
		note := storelocale.T(storelocale.KeyStoreImOperationFailedCheckTheConnectionAndRe)
		var rows []Entry
		refreshed := false
		if err == nil {
			note = storelocale.T(storelocale.KeyStoreImTheOperationFinishedButTheStatusRefres)
			raw, refreshErr := plugin.IMChannelStatus(callCtx, target.Channel)
			if refreshErr == nil {
				var status struct {
					Bots     []json.RawMessage `json:"bots"`
					Snapshot *struct {
						Bots []json.RawMessage `json:"bots"`
					} `json:"snapshot"`
				}
				if json.Unmarshal(raw, &status) == nil && (status.Bots != nil || (status.Snapshot != nil && status.Snapshot.Bots != nil)) {
					rows = parseCordisBots(raw, target.Channel)
					refreshed = true
					note = storelocale.T(storelocale.KeyStoreImBotStatusUpdated)
				}
			}
		}
		s.mu.Lock()
		if epoch != s.connectionEpoch || plugin != s.plugin {
			s.mu.Unlock()
			return
		}
		s.connectionBusy = false
		s.connectionNote = note
		if refreshed {
			kept := make([]Entry, 0, len(s.rows)+len(rows))
			for _, row := range s.rows {
				if row.Channel != target.Channel {
					kept = append(kept, row)
				}
			}
			s.rows = append(kept, rows...)
		}
		if refreshed && method == "bot.workspace.set" && s.workspaceDrafts[id] == payload["workspace"] {
			for _, row := range rows {
				if row.ID == id && row.Workspace == payload["workspace"] {
					delete(s.workspaceDrafts, id)
					break
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
	return nil
}
