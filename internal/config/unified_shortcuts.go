package config

import (
	"bytes"
	"encoding/json"
	"path/filepath"
)

const unifiedShortcutsKey = "goUnifiedShortcuts"

type shortcutDocumentSnapshot struct {
	raw      map[string]json.RawMessage
	baseline fileBaseline
}

// ReloadShortcutDocument is only for the explicit discard-and-reload gesture.
// It accepts a shortcut-owned persisted snapshot without changing the general
// config baseline. An unrelated stale desktop form must still fail its CAS.
func (s *Store) ReloadShortcutDocument() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	commitMu.Lock()
	defer commitMu.Unlock()
	fresh, err := Open(filepath.Dir(s.path))
	if err != nil {
		return nil, err
	}
	s.shortcutReload = &shortcutDocumentSnapshot{raw: fresh.raw, baseline: fresh.baseline}
	s.shortcutGen.Add(1)
	return append([]byte(nil), fresh.raw[unifiedShortcutsKey]...), nil
}

func sameConfigBaseline(a, b fileBaseline) bool {
	return a.exists == b.exists && bytes.Equal(a.data, b.data)
}

func (s *Store) shortcutSnapshotLocked() (map[string]json.RawMessage, fileBaseline, error) {
	if s.shortcutReload == nil {
		if err := s.checkBaseline(); err != nil {
			return nil, fileBaseline{}, err
		}
		return s.raw, s.baseline, nil
	}
	view := s.shortcutReload
	current, err := view.baseline.matches(s.path)
	if err != nil {
		return nil, fileBaseline{}, err
	}
	if !current {
		// A normal accepted write in this same Store may advance the disk.
		// Only that exact accepted baseline can refresh this shortcut view.
		current, err = s.baseline.matches(s.path)
		if err != nil {
			return nil, fileBaseline{}, err
		}
		if !current {
			return nil, fileBaseline{}, ErrConflict
		}
		view.raw, view.baseline = s.raw, s.baseline
	}
	return view.raw, view.baseline, nil
}

// ShortcutDocument is a single MyGo authority, not the independent Electron
// keybindings file. Legacy native preferences remain an untouched migration source.
func (s *Store) ShortcutDocument() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, _, err := s.shortcutSnapshotLocked()
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), raw[unifiedShortcutsKey]...), nil
}

// ShortcutDocumentGeneration advances on every accepted change this store makes
// to the shortcut document (and on an explicit reload). Consumers that cache
// values derived from the document compare this instead of rereading the file
// on every frame; tmp/stalls showed that read on the UI thread.
func (s *Store) ShortcutDocumentGeneration() uint64 {
	return s.shortcutGen.Load()
}
func (s *Store) InitializeShortcutDocument(data []byte) error {
	return s.writeShortcutDocument(data, true)
}
func (s *Store) SaveShortcutDocument(data []byte) error { return s.writeShortcutDocument(data, false) }
func (s *Store) writeShortcutDocument(data []byte, initialize bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, baseline, err := s.shortcutSnapshotLocked()
	if err != nil {
		return err
	}
	if initialize && len(raw[unifiedShortcutsKey]) > 0 {
		return nil
	}
	if !json.Valid(data) {
		return &json.SyntaxError{}
	}
	merged := map[string]json.RawMessage{}
	for k, v := range raw {
		merged[k] = v
	}
	merged[unifiedShortcutsKey] = append([]byte(nil), data...)
	encoded, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	commitMu.Lock()
	defer commitMu.Unlock()
	current, err := baseline.matches(s.path)
	if err != nil {
		return err
	}
	if !current {
		return ErrConflict
	}
	if err = writeAtomic(s.path, encoded, s.rename); err != nil {
		return err
	}
	next := fileBaseline{exists: true, data: encoded}
	if sameConfigBaseline(s.baseline, baseline) {
		s.raw, s.baseline = merged, next
	}
	if s.shortcutReload != nil {
		s.shortcutReload.raw, s.shortcutReload.baseline = merged, next
	}
	s.shortcutGen.Add(1)
	return nil
}
