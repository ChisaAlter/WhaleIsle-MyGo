package config

import (
	"encoding/json"
)

// PluginRecoveryInfo is the pluginRecovery object subset used by the launcher.
type PluginRecoveryInfo struct {
	Present         bool
	SkipUserPlugins bool
	Reason          string
}

// PluginRecovery returns the current pluginRecovery projection.
func (s *Store) PluginRecovery() PluginRecoveryInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := PluginRecoveryInfo{SkipUserPlugins: s.cfg.SkipUserPlugins}
	raw, ok := s.raw["pluginRecovery"]
	if !ok {
		return out
	}
	out.Present = true
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	if v, ok := m["reason"].(string); ok {
		out.Reason = v
	}
	return out
}

// UpdatePluginRecovery mutates the top-level pluginRecovery object and persists
// config.json. Empty objects remove pluginRecovery. SkipUserPlugins on Config
// is re-read from the saved object.
func (s *Store) UpdatePluginRecovery(fn func(map[string]any)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkBaseline(); err != nil {
		return err
	}

	rec := map[string]any{}
	if raw, ok := s.raw["pluginRecovery"]; ok {
		_ = json.Unmarshal(raw, &rec)
	}
	fn(rec)

	merged := make(map[string]json.RawMessage, len(s.raw)+1)
	for k, v := range s.raw {
		merged[k] = v
	}
	if len(rec) == 0 {
		delete(merged, "pluginRecovery")
	} else {
		b, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		merged["pluginRecovery"] = b
	}

	next := cloneConfig(s.cfg)
	next.SkipUserPlugins = false
	if raw, ok := merged["pluginRecovery"]; ok {
		var r struct {
			SkipUserPlugins bool `json:"skipUserPlugins"`
		}
		if json.Unmarshal(raw, &r) == nil {
			next.SkipUserPlugins = r.SkipUserPlugins
		}
	}
	next = normalize(cloneConfig(next))

	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &known); err != nil {
		return err
	}
	merged = mergeKnownConfig(merged, known)
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}

	unlock, err := s.lockCommit()
	if err != nil {
		return err
	}
	defer unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	if err := writeAtomic(s.path, data, s.rename); err != nil {
		return err
	}
	s.raw = merged
	s.setCfgLocked(next)
	s.baseline = fileBaseline{exists: true, data: data}
	return nil
}

// ClearPluginRecovery removes pluginRecovery from disk and clears SkipUserPlugins.
func (s *Store) ClearPluginRecovery() error {
	return s.UpdatePluginRecovery(func(m map[string]any) {
		for k := range m {
			delete(m, k)
		}
	})
}
