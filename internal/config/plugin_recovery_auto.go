package config

import (
	"encoding/json"
	"time"
	"whaleisle/internal/startuprec"
)

// PluginRecoveryEvidence is compatible with the original launcher's marker.
// An empty appVersion is an explicit/manual setting and survives upgrades.
type PluginRecoveryEvidence struct {
	SkipUserPlugins bool     `json:"skipUserPlugins"`
	Reason          string   `json:"reason"`
	At              string   `json:"at"`
	AppVersion      string   `json:"appVersion"`
	LogTail         []string `json:"logTail"`
}

func (s *Store) PluginRecoveryEvidence() PluginRecoveryEvidence {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out PluginRecoveryEvidence
	_ = json.Unmarshal(s.raw["pluginRecovery"], &out)
	out.Reason = startuprec.RedactLine(out.Reason)
	out.LogTail = startuprec.SanitizeTail(out.LogTail)
	return out
}

// RecordAutomaticPluginRecovery is called before a recovery process is started.
// Errors prevent the automatic retry rather than claiming a durable recovery.
func (s *Store) RecordAutomaticPluginRecovery(version, reason string, logs []string) error {
	return s.UpdatePluginRecovery(func(m map[string]any) {
		m["skipUserPlugins"] = true
		m["reason"] = startuprec.RedactLine(reason)
		m["appVersion"] = version
		m["at"] = time.Now().UTC().Format(time.RFC3339)
		m["logTail"] = startuprec.SanitizeTail(logs)
	})
}
func (s *Store) ReconcilePluginRecovery(version string) error {
	rec := s.PluginRecoveryEvidence()
	if !rec.SkipUserPlugins || rec.AppVersion == "" || rec.AppVersion == version {
		return nil
	}
	return s.ClearPluginRecovery()
}
