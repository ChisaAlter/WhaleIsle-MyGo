package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/dsh/events"
)

// AgentPresetOption is one row in the composer agent preset picker.
type AgentPresetOption struct {
	Description, Broken string
	RequiresCodingTools bool
	Label               string
	ID                  string
}

func flattenAgentPresetRoster(roster api.AgentPresetRoster) []AgentPresetOption {
	var out []AgentPresetOption
	seen := map[string]struct{}{}
	for _, p := range roster.Presets {
		if p.Id == "" {
			continue
		}
		if p.Hidden != nil && *p.Hidden {
			continue
		}
		if _, ok := seen[p.Id]; ok {
			continue
		}
		seen[p.Id] = struct{}{}
		label := p.Id
		if p.Name != nil && *p.Name != "" {
			label = *p.Name
		}
		if p.Broken != nil && *p.Broken != "" {
			label = label + " ⚠"
		}
		out = append(out, AgentPresetOption{Description: derefString(p.Description), Broken: derefString(p.Broken), Label: label, ID: p.Id, RequiresCodingTools: p.Name == nil && (p.Id == "ptc" || p.Id == "minimal")})
	}
	return out
}

func labelForAgentPresetID(id string, options []AgentPresetOption) string {
	for _, o := range options {
		if o.ID == id {
			return o.Label
		}
	}
	if id != "" {
		return id
	}
	return ""
}

func agentPresetOptionByLabel(label string, options []AgentPresetOption) (AgentPresetOption, bool) {
	var match AgentPresetOption
	found := false
	for _, o := range options {
		if o.Label == label {
			if found {
				return AgentPresetOption{}, false
			}
			match, found = o, true
		}
	}
	return match, found
}

func defaultAgentPresetLabel(options []AgentPresetOption, roster api.AgentPresetRoster) string {
	for _, p := range roster.Presets {
		if p.IsDefault {
			if lbl := labelForAgentPresetID(p.Id, options); lbl != "" {
				return lbl
			}
		}
	}
	if len(options) > 0 {
		return options[0].Label
	}
	return ""
}

// RefreshAgentPresets loads agentPresets/list for the composer picker.
func (s *Store) RefreshAgentPresets(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var roster api.AgentPresetRoster
		err := client.Call(ctx, "agentPresets/list", map[string]any{}, &roster)
		s.mu.Lock()
		if err != nil {
			s.agentPresetsErr = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.agentPresetsErr = nil
		s.agentPresetOptions = flattenAgentPresetRoster(roster)
		if lbl := defaultAgentPresetLabel(s.agentPresetOptions, roster); lbl != "" && s.selectedAgentPresetLabel == "" {
			s.selectedAgentPresetLabel = lbl
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ReadAgentPresetDocument loads agentPresets/read for the preset id (or label match).
func (s *Store) ReadAgentPresetDocument(ctx context.Context, labelOrID string) (api.AgentPresetDocument, error) {
	labelOrID = strings.TrimSpace(labelOrID)
	s.mu.Lock()
	client := s.client
	id := labelOrID
	if opt, ok := agentPresetOptionByLabel(labelOrID, s.agentPresetOptions); ok {
		id = opt.ID
	}
	s.mu.Unlock()
	if client == nil || id == "" {
		return api.AgentPresetDocument{}, fmt.Errorf("harness not ready")
	}
	var doc api.AgentPresetDocument
	err := client.Call(ctx, "agentPresets/read", map[string]any{"agentPreset": id}, &doc)
	return doc, err
}

// SelectAgentPresetForSession applies agentPresets/select for the active session.
func (s *Store) SelectAgentPresetForSession(ctx context.Context, label string) {
	s.mu.Lock()
	opt, ok := agentPresetOptionByLabel(label, s.agentPresetOptions)
	id := s.selectedID
	s.mu.Unlock()
	if ok {
		s.SelectAgentPresetIDForSession(ctx, id, opt.ID)
	}
}

// SelectAgentPresetIDForSession binds the gesture to a real roster ID and session.
func (s *Store) SelectAgentPresetIDForSession(ctx context.Context, targetID, presetID string) {
	s.mu.Lock()
	opt, ok := agentPresetOptionByID(presetID, s.agentPresetOptions)
	client, sessionID, generation := s.client, s.selectedID, s.followGeneration
	cwd := s.cwd
	p := s.presetStateLocked()
	if !ok || client == nil || sessionID != targetID || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if p.busy && p.session == sessionID && p.generation == generation {
		s.mu.Unlock()
		return
	}
	p.session, p.generation = sessionID, generation
	p.error = ""
	if !s.blankPresetTargetLocked(sessionID) || opt.Broken != "" {
		p.error = opt.Label + ": presets can only be applied to a blank session"
		if opt.Broken != "" {
			p.error = opt.Label + ": " + opt.Broken
		}
		s.mu.Unlock()
		s.fire()
		return
	}
	if sessionID == "" {
		p.staged[filepathClean(s.cwd)] = opt.ID
		s.mu.Unlock()
		s.fire()
		return
	}
	p.busy = true
	s.mu.Unlock()
	s.fire()
	label := opt.Label
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		s.mu.Lock()
		current := s.client == client && s.selectedID == sessionID && s.followGeneration == generation && s.blankPresetTargetLocked(sessionID)
		if !current {
			if s.client == client && s.presetDraft.session == sessionID && s.presetDraft.generation == generation {
				s.presetDraft.busy = false
				s.presetDraft.error = label + ": session changed; select again"
			}
			s.mu.Unlock()
			s.fire()
			return
		}
		s.mu.Unlock()
		var result api.SelectResult
		err := client.Call(ctx, "agentPresets/select", map[string]any{
			"agentId":     sessionID,
			"agentPreset": opt.ID,
		}, &result)
		s.mu.Lock()
		if s.client != client || s.selectedID != sessionID || s.followGeneration != generation {
			s.mu.Unlock()
			return
		}
		p := s.presetStateLocked()
		p.busy = false
		if err != nil {
			p.error = label + ": " + err.Error()
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		if lbl := labelForAgentPresetID(string(result), s.agentPresetOptions); lbl != "" {
			s.selectedAgentPresetLabel = lbl
		} else {
			s.selectedAgentPresetLabel = label
		}
		if s.stagedPresetLocked(cwd) == opt.ID {
			delete(s.presetDraft.staged, filepathClean(cwd))
		}
		s.lastAction = "agent preset selected"
		s.invalidateCommandsLocked()
		s.invalidateSkillOptionsLocked()
		go s.RefreshSkills(context.Background())
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) applyAgentPresetSelectedEvent(raw json.RawMessage) bool {
	var ev events.AgentPresetSelected
	if err := json.Unmarshal(raw, &ev); err != nil || ev.Type != events.TypeAgentPresetSelected {
		return false
	}
	id := ev.Data.AgentPreset
	if id == "" {
		return true
	}
	s.mu.Lock()
	if lbl := labelForAgentPresetID(id, s.agentPresetOptions); lbl != "" {
		s.selectedAgentPresetLabel = lbl
		s.invalidateCommandsLocked()
		s.invalidateSkillOptionsLocked()
		go s.RefreshSkills(context.Background())
	}
	s.mu.Unlock()
	s.fire()
	return true
}

// AgentPresetLabelsForDeveloperTools projects the same roster without altering
// the session's selected preset or the Host's default.
func (s *Store) AgentPresetLabelsForDeveloperTools(enabled bool) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var labels []string
	for _, row := range s.agentPresetOptions {
		if enabled || !row.RequiresCodingTools {
			labels = append(labels, row.Label)
		}
	}
	return labels
}
