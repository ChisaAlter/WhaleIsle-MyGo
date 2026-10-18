package config

import "encoding/json"

const unifiedShortcutsKey = "goUnifiedShortcuts"

// ShortcutDocument is a single MyGo authority, not the independent Electron
// keybindings file. Legacy native preferences remain an untouched migration source.
func (s *Store) ShortcutDocument() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBaseline(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.raw[unifiedShortcutsKey]...), nil
}
func (s *Store) InitializeShortcutDocument(data []byte) error {
	return s.writeShortcutDocument(data, true)
}
func (s *Store) SaveShortcutDocument(data []byte) error { return s.writeShortcutDocument(data, false) }
func (s *Store) writeShortcutDocument(data []byte, initialize bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkBaseline(); err != nil {
		return err
	}
	if initialize && len(s.raw[unifiedShortcutsKey]) > 0 {
		return nil
	}
	if !json.Valid(data) {
		return &json.SyntaxError{}
	}
	merged := map[string]json.RawMessage{}
	for k, v := range s.raw {
		merged[k] = v
	}
	merged[unifiedShortcutsKey] = append([]byte(nil), data...)
	encoded, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	commitMu.Lock()
	defer commitMu.Unlock()
	if err = s.checkBaseline(); err != nil {
		return err
	}
	if err = writeAtomic(s.path, encoded, s.rename); err != nil {
		return err
	}
	s.raw = merged
	s.baseline = fileBaseline{exists: true, data: encoded}
	return nil
}
