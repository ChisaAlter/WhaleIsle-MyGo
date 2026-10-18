package market

import (
	"errors"
	"time"
)

// BeginInstall commits a receipt before any Host or local installer mutation.
// It intentionally stores no raw spec, URL, diagnostic or credential.
func (s *Store) BeginInstall(attempt, plugin, action string) error {
	if !safePluginID.MatchString(plugin) {
		plugin = "manual-install"
	}
	switch action {
	case "install", "update", "remove", "toggle", "exemption":
	default:
		action = "install"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.personal.data
	data.History = append([]InstallHistory(nil), data.History...)
	for i := range data.History {
		if data.History[i].Attempt == attempt {
			data.History[i].Result = "running"
			return s.savePersonalLocked(data)
		}
	}
	entry := InstallHistory{Attempt: attempt, Action: action, At: time.Now().UTC(), Plugin: plugin, Result: "running"}
	data.History = append([]InstallHistory{entry}, data.History...)
	if len(data.History) > 200 {
		data.History = data.History[:200]
	}
	return s.savePersonalLocked(data)
}

func (s *Store) FinishInstall(attempt, result, errorCode string) error {
	switch result {
	case "applied", "restart-required", "failed", "cancelled", "unknown", "interrupted", "awaiting-approval", "overridden", "not-started":
	default:
		result = "unknown"
	}
	switch errorCode {
	case "operation-failed", "result-unconfirmed":
	default:
		errorCode = ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.personal.data
	data.History = append([]InstallHistory(nil), data.History...)
	for i := range data.History {
		if data.History[i].Attempt != attempt {
			continue
		}
		data.History[i].Result = result
		data.History[i].Error = errorCode
		err := s.savePersonalLocked(data)
		if err != nil {
			s.personal.data.History = data.History
		}
		return err
	}
	return errors.New("operation receipt unavailable")
}

func (s *Store) BundleHistoryID(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.rows {
		if row.Label != name {
			continue
		}
		for _, item := range s.discovery.catalog.Items {
			if InstalledName(item, []Entry{row}) != "" {
				return item.ID
			}
		}
	}
	return "manual-install"
}

// QueueUpdates commits the complete confirmed queue in one write before execution.
func (s *Store) QueueUpdates(targets []InstallTarget, receipts map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.personal.data
	history := make([]InstallHistory, 0, len(targets)+len(data.History))
	for _, target := range targets {
		plugin := target.ID
		if !safePluginID.MatchString(plugin) {
			plugin = "manual-install"
		}
		history = append(history, InstallHistory{Attempt: receipts[target.ID], Action: "update", At: time.Now().UTC(), Plugin: plugin, Result: "queued"})
	}
	history = append(history, data.History...)
	// Do not truncate the active batch; keep at least its full queue.
	limit := 200
	if len(targets) > limit {
		limit = len(targets)
	}
	if len(history) > limit {
		history = history[:limit]
	}
	data.History = history
	return s.savePersonalLocked(data)
}

func (s *Store) CancelQueuedUpdates(receipts map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.personal.data
	data.History = append([]InstallHistory(nil), data.History...)
	for i := range data.History {
		h := &data.History[i]
		if h.Result != "queued" {
			continue
		}
		for _, receipt := range receipts {
			if h.Attempt == receipt {
				h.Result = "not-started"
				break
			}
		}
	}
	err := s.savePersonalLocked(data)
	if err != nil {
		s.personal.data.History = data.History
	}
	return err
}
