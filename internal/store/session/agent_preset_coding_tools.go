package session

import (
	"context"
	"errors"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func (m *AgentPresetManager) ForSource(source *Store) bool { return m.source == source }

// ObserveCodingTools waits for accepted settings and any pending preference
// write. A connection/configuration change requests one fresh roster + revision
// read; a failed CAS is shown, never automatically replayed.
func (m *AgentPresetManager) ObserveCodingTools(ready, disabled bool, revision, presetRevision float64) {
	client := m.currentClient()
	m.mu.Lock()
	if client == nil {
		m.codingToolsObserved = false
		m.mu.Unlock()
		return
	}
	if !ready {
		m.mu.Unlock()
		return
	}
	changed := !m.codingToolsObserved || m.codingToolsClient != client || m.codingToolsDisabled != disabled || m.codingToolsRevision != revision || m.codingToolsPresetRevision != presetRevision
	m.codingToolsObserved, m.codingToolsDisabled = true, disabled
	m.codingToolsClient, m.codingToolsRevision = client, revision
	m.codingToolsPresetRevision = presetRevision
	if !changed || !disabled {
		m.mu.Unlock()
		return
	}
	if m.state.Busy {
		m.codingToolsRefreshPending = true
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	m.refreshCodingToolsDefault()
}

func (m *AgentPresetManager) refreshCodingToolsDefault() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		m.Refresh(ctx)
	}()
}

func (m *AgentPresetManager) repairCodingToolsDefault(ctx context.Context, client *dsh.Client, next *AgentPresetManagerState) error {
	hiddenDefault, standard := false, false
	for _, row := range next.Rows {
		if row.ID == next.DefaultID && row.BuiltIn && row.RequiresCodingTools {
			hiddenDefault = true
		}
		if row.ID == "standard" && row.BuiltIn && row.Broken == "" && !row.Hidden {
			standard = true
		}
	}
	if !hiddenDefault {
		return nil
	}
	if !next.Writable || !standard {
		return errors.New("Coding Tools is disabled, but the default preset cannot be changed to Standard; refresh after restoring a writable Standard preset")
	}
	m.mu.Lock()
	current := m.codingToolsObserved && m.codingToolsDisabled && m.codingToolsClient == client
	m.mu.Unlock()
	if !current || m.currentClient() != client {
		return nil
	}
	var out api.SettingsNamespaceView
	if err := client.Call(ctx, "settings/update", map[string]any{"ns": agentPresetNamespace, "patch": map[string]any{"selectedDefault": "standard"}, "expectedRevision": next.Revision}, &out); err != nil {
		return err
	}
	if agentDefaultID(out.Value) != "standard" || m.currentClient() != client {
		return errors.New("Refresh presets to confirm the default preset change")
	}
	next.DefaultID, next.Revision = "standard", out.Revision
	return nil
}
