package session

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

const agentPresetNamespace = "agent-preset-registry"

// ManagedAgentPreset keeps broken entries visible for diagnosis but not selectable.
type ManagedAgentPreset struct {
	RequiresCodingTools           bool
	BuiltIn                       bool
	ID, Name, Description, Broken string
	Hidden                        bool
}
type AgentPresetManagerState struct {
	Rows                   []ManagedAgentPreset
	DefaultID              string
	Revision               float64
	Writable, Loaded, Busy bool
	Err, Status            string
}

// AgentPresetManager manages the persisted default for future sessions. It never
// selects a composition on the active session or writes preset source documents.
type AgentPresetManager struct {
	mu                        sync.Mutex
	source                    *Store
	client                    *dsh.Client
	state                     AgentPresetManagerState
	codingToolsObserved       bool
	codingToolsDisabled       bool
	codingToolsClient         *dsh.Client
	codingToolsRevision       float64
	codingToolsPresetRevision float64
	codingToolsRefreshPending bool
}

func NewAgentPresetManager(source *Store) *AgentPresetManager {
	return &AgentPresetManager{source: source}
}
func (m *AgentPresetManager) Snapshot() AgentPresetManagerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.state
	s.Rows = append([]ManagedAgentPreset(nil), s.Rows...)
	return s
}
func (m *AgentPresetManager) currentClient() *dsh.Client {
	m.source.mu.Lock()
	defer m.source.mu.Unlock()
	return m.source.client
}
func (m *AgentPresetManager) finish(err error, status string) {
	m.mu.Lock()
	m.state.Busy = false
	m.state.Err = ""
	m.state.Status = status
	if err != nil {
		m.state.Err = err.Error()
		m.state.Status = ""
	}
	pending := m.codingToolsRefreshPending
	m.codingToolsRefreshPending = false
	m.mu.Unlock()
	m.source.fire()
	if pending {
		m.refreshCodingToolsDefault()
	}
}

// Refresh reads both roster and host settings, then reconciles a hidden built-in
// default when Coding Tools has been disabled in the accepted configuration.
func (m *AgentPresetManager) Refresh(ctx context.Context) {
	m.mu.Lock()
	if m.state.Busy {
		m.mu.Unlock()
		return
	}
	m.state.Busy = true
	m.mu.Unlock()
	m.source.fire()
	client := m.currentClient()
	if client == nil {
		m.finish(errors.New(storelocale.T(storelocale.KeyStoreSessionHostNotConnected)), "")
		return
	}
	var roster api.AgentPresetRoster
	if err := client.Call(ctx, "agentPresets/list", map[string]any{}, &roster); err != nil {
		m.finish(err, "")
		return
	}
	var settings api.SettingsDescribeValue
	if err := client.Call(ctx, "settings/describe", map[string]any{}, &settings); err != nil {
		m.finish(err, "")
		return
	}
	next := AgentPresetManagerState{Loaded: true, Busy: true}
	seen := map[string]bool{}
	for _, p := range roster.Presets {
		if p.Id == "" || seen[p.Id] {
			continue
		}
		seen[p.Id] = true
		row := ManagedAgentPreset{ID: p.Id, Name: p.Id, RequiresCodingTools: p.Name == nil && (p.Id == "ptc" || p.Id == "minimal")}
		row.BuiltIn = p.Name == nil && (p.Id == "standard" || p.Id == "ptc" || p.Id == "minimal" || p.Id == "cordis")
		if p.Name != nil && *p.Name != "" {
			row.Name = *p.Name
		}
		if p.Description != nil {
			row.Description = *p.Description
		}
		if p.Broken != nil {
			row.Broken = *p.Broken
		}
		if p.Hidden != nil {
			row.Hidden = *p.Hidden
		}
		next.Rows = append(next.Rows, row)
		if p.IsDefault {
			next.DefaultID = p.Id
		}
	}
	codingToolsDisabled := false
	for _, ns := range settings.Namespaces {
		if ns.Ns == "ui-settings" {
			var preference struct {
				Enabled bool `json:"enabled"`
			}
			codingToolsDisabled = json.Unmarshal(ns.Value, &preference) == nil && !preference.Enabled
		}
		if ns.Ns == agentPresetNamespace {
			next.Writable = settings.Writable
			next.Revision = ns.Revision
			if id := agentDefaultID(ns.Value); id != "" {
				next.DefaultID = id
			}
		}
	}
	if m.currentClient() != client {
		m.finish(errors.New(storelocale.T(storelocale.KeyStoreSessionHostChangedRefresh)), "")
		return
	}
	m.mu.Lock()
	repair := codingToolsDisabled && m.codingToolsObserved && m.codingToolsDisabled && m.codingToolsClient == client
	m.mu.Unlock()
	if repair {
		if err := m.repairCodingToolsDefault(ctx, client, &next); err != nil {
			m.mu.Lock()
			m.client, m.state = client, next
			m.mu.Unlock()
			m.finish(err, "")
			return
		}
	}
	m.mu.Lock()
	m.client = client
	m.state = next
	m.mu.Unlock()
	m.finish(nil, "")
}

// SaveDefault uses the last observed revision and publishes only a confirmed
// response. Refusals preserve the old default and remain visible for retry.
func (m *AgentPresetManager) SaveDefault(ctx context.Context, id string) {
	m.mu.Lock()
	if m.state.Busy {
		m.mu.Unlock()
		return
	}
	client := m.client
	revision := m.state.Revision
	eligible := false
	for _, row := range m.state.Rows {
		if row.ID == id && row.Broken == "" && !row.Hidden && (!m.codingToolsDisabled || !row.RequiresCodingTools) {
			eligible = true
		}
	}
	ready := m.state.Loaded && m.state.Writable && eligible
	m.state.Busy = true
	m.mu.Unlock()
	m.source.fire()
	if !ready {
		m.finish(errors.New(storelocale.T(storelocale.KeyStoreSessionThisPresetCannotBeSetAsDefaultOrThe)), "")
		return
	}
	if client == nil || m.currentClient() != client {
		m.finish(errors.New(storelocale.T(storelocale.KeyStoreSessionHostChangedRefresh)), "")
		return
	}
	var out api.SettingsNamespaceView
	err := client.Call(ctx, "settings/update", map[string]any{"ns": agentPresetNamespace, "patch": map[string]any{"selectedDefault": id}, "expectedRevision": revision}, &out)
	if err != nil {
		m.finish(err, "")
		return
	}
	if agentDefaultID(out.Value) != id {
		m.finish(errors.New(storelocale.T(storelocale.KeyStoreSessionTheHostDidNotConfirmSavingTheDefault)), "")
		return
	}
	if m.currentClient() != client {
		m.finish(errors.New(storelocale.T(storelocale.KeyStoreSessionHostChangedRefreshToConfirmTheSaveRes)), "")
		return
	}
	m.mu.Lock()
	m.state.DefaultID = id
	m.state.Revision = out.Revision
	m.mu.Unlock()
	m.finish(nil, storelocale.T(storelocale.KeyStoreSessionSavedLaterNewTasksUseThisDefaultPrese))
}

func agentDefaultID(raw json.RawMessage) string {
	var v struct {
		SelectedDefault string `json:"selectedDefault"`
	}
	_ = json.Unmarshal(raw, &v)
	return v.SelectedDefault
}
