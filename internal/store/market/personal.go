package market

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// InstallHistory retains bounded, redacted operation diagnostics. Raw transport
// envelopes and credentials are never persisted.
type InstallHistory struct {
	Attempt    string    `json:"attempt,omitempty"`
	Action     string    `json:"action,omitempty"`
	At         time.Time `json:"at"`
	Plugin     string    `json:"plugin"`
	Result     string    `json:"result"`
	Error      string    `json:"error,omitempty"`
	Log        string    `json:"log,omitempty"`
	Diagnostic string    `json:"diagnostic,omitempty"`
}
type personalData struct {
	Favorites map[string]bool  `json:"favorites"`
	History   []InstallHistory `json:"history"`
}
type personalState struct {
	path    string
	data    personalData
	err     string
	loadErr error
}
type Personal struct {
	Favorites map[string]bool
	History   []InstallHistory
	Err       string
}

func NewWithPersonalPath(path string) *Store {
	s := &Store{discovery: newDiscoveryState()}
	s.personal.path = path
	s.personal.data.Favorites = map[string]bool{}
	raw, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(raw, &s.personal.data)
	}
	if err != nil && !os.IsNotExist(err) {
		s.personal.loadErr = err
		s.personal.err = "Could not read local favorites/history; existing data was not overwritten."
	}
	if s.personal.data.Favorites == nil {
		s.personal.data.Favorites = map[string]bool{}
	}
	// A running receipt left by a previous process is not proof of success.
	changed := false
	for i := range s.personal.data.History {
		row := &s.personal.data.History[i]
		log, diagnostic := RedactOperationLog(row.Log, 16000), RedactOperationLog(row.Diagnostic, 2000)
		changed = changed || log != row.Log || diagnostic != row.Diagnostic
		row.Log, row.Diagnostic = log, diagnostic
		if s.personal.data.History[i].Result == "queued" {
			s.personal.data.History[i].Result = "not-started"
			changed = true
		}
		if s.personal.data.History[i].Result == "running" {
			s.personal.data.History[i].Result = "interrupted"
			s.personal.data.History[i].Error = "result-unconfirmed"
			changed = true
		}
	}
	if changed && s.personal.loadErr == nil {
		if writePersonal(s.personal.path, nil, s.personal.data) != nil { // The store exposes persistence failure in Personal.Err.
			s.personal.err = personalSaveFailed
		}
	}
	return s
}
func (s *Store) Personal() Personal {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Personal{Favorites: map[string]bool{}, History: append([]InstallHistory(nil), s.personal.data.History...), Err: s.personal.err}
	for id, v := range s.personal.data.Favorites {
		out.Favorites[id] = v
	}
	return out
}

const personalSaveFailed = "Could not save local favorites/history; this change will not survive restart."

// writePersonal writes the favorites/history document atomically. It touches
// no store state, so callers run it without mu held.
func writePersonal(path string, loadErr error, data personalData) error {
	if loadErr != nil {
		return loadErr
	}
	if path == "" {
		return errors.New("personal storage path unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".market-personal-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// commitPersonal serializes local favorites/history writes through
// persistMu: edit computes the next document under mu, the file is written
// without mu held so readers never wait on the disk, and the outcome is
// adopted under mu. keepHistory retains the edited history in memory when
// the write fails, for receipts this session must keep regardless.
func (s *Store) commitPersonal(keepHistory bool, edit func(cur personalData) (personalData, error)) error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.Lock()
	next, err := edit(s.personal.data)
	path, loadErr := s.personal.path, s.personal.loadErr
	s.mu.Unlock()
	if err != nil {
		return err
	}
	err = writePersonal(path, loadErr, next)
	s.mu.Lock()
	if err != nil {
		s.personal.err = personalSaveFailed
		if keepHistory {
			s.personal.data.History = next.History
		}
	} else {
		s.personal.data = next
		s.personal.err = ""
	}
	s.publishLocked()
	s.mu.Unlock()
	return err
}

func (s *Store) ToggleFavorite(id string) error {
	err := s.commitPersonal(false, func(cur personalData) (personalData, error) {
		known := cur.Favorites[id]
		for _, item := range s.discovery.catalog.Items {
			if item.ID == id {
				known = true
				break
			}
		}
		if !known {
			return cur, errUnknownFavorite
		}
		data := cur
		data.Favorites = map[string]bool{}
		for k, v := range cur.Favorites {
			data.Favorites[k] = v
		}
		if data.Favorites[id] {
			delete(data.Favorites, id)
		} else {
			data.Favorites[id] = true
		}
		s.discovery.page = 0
		s.discovery.expanded = false
		return data, nil
	})
	if errors.Is(err, errUnknownFavorite) {
		return err
	}
	s.fire()
	return err
}

var errUnknownFavorite = errors.New("catalog item unavailable")

var safePluginID = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func (s *Store) RecordInstall(plugin, result, errorCode string) error {
	switch result {
	case "applied", "restart-required", "failed", "cancelled", "unknown":
	default:
		return nil
	}
	if !safePluginID.MatchString(plugin) {
		plugin = "manual-install"
	}
	// Only our own generic classifications are persisted; server errors may contain secrets.
	switch errorCode {
	case "operation-failed", "result-unconfirmed":
	default:
		errorCode = ""
	}
	entry := InstallHistory{At: time.Now().UTC(), Plugin: plugin, Result: result, Error: errorCode}
	// Retain the observed result for this session even if disk persistence fails.
	err := s.commitPersonal(true, func(data personalData) (personalData, error) {
		data.History = append([]InstallHistory{entry}, data.History...)
		if len(data.History) > 200 {
			data.History = data.History[:200]
		}
		return data, nil
	})
	s.fire()
	return err
}
