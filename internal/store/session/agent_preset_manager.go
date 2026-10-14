package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

const agentPresetNamespace = "agent-preset-registry"

// ManagedAgentPreset keeps broken entries visible for diagnosis but not selectable.
type ManagedAgentPreset struct {
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
	mu     sync.Mutex
	source *Store
	client *dsh.Client
	state  AgentPresetManagerState
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
	m.mu.Unlock()
	m.source.fire()
}

// Refresh reads both roster and host settings; no mutation occurs on refresh.
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
		m.finish(fmt.Errorf("Host 未连接"), "")
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
		row := ManagedAgentPreset{ID: p.Id, Name: p.Id}
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
	for _, ns := range settings.Namespaces {
		if ns.Ns == agentPresetNamespace {
			next.Writable = settings.Writable
			next.Revision = ns.Revision
			if id := agentDefaultID(ns.Value); id != "" {
				next.DefaultID = id
			}
			break
		}
	}
	if m.currentClient() != client {
		m.finish(fmt.Errorf("Host 已切换，请刷新"), "")
		return
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
		if row.ID == id && row.Broken == "" && !row.Hidden {
			eligible = true
		}
	}
	ready := m.state.Loaded && m.state.Writable && eligible
	m.state.Busy = true
	m.mu.Unlock()
	m.source.fire()
	if !ready {
		m.finish(fmt.Errorf("此预设不可设为默认，或配置只读；请刷新后重试"), "")
		return
	}
	if client == nil || m.currentClient() != client {
		m.finish(fmt.Errorf("Host 已切换，请刷新"), "")
		return
	}
	var out api.SettingsNamespaceView
	err := client.Call(ctx, "settings/update", map[string]any{"ns": agentPresetNamespace, "patch": map[string]any{"selectedDefault": id}, "expectedRevision": revision}, &out)
	if err != nil {
		m.finish(err, "")
		return
	}
	if agentDefaultID(out.Value) != id {
		m.finish(fmt.Errorf("Host 未确认保存所选默认预设，请刷新"), "")
		return
	}
	if m.currentClient() != client {
		m.finish(fmt.Errorf("Host 已切换，请刷新确认保存结果"), "")
		return
	}
	m.mu.Lock()
	m.state.DefaultID = id
	m.state.Revision = out.Revision
	m.mu.Unlock()
	m.finish(nil, "已保存；后续新任务使用此默认预设")
}

func agentDefaultID(raw json.RawMessage) string {
	var v struct {
		SelectedDefault string `json:"selectedDefault"`
	}
	_ = json.Unmarshal(raw, &v)
	return v.SelectedDefault
}
