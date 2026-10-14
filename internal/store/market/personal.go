package market

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// InstallHistory contains only a safe identifier and bounded outcome, never raw
// install specs, URLs, RPC diagnostics, build output or authentication data.
type InstallHistory struct {
	At     time.Time `json:"at"`
	Plugin string    `json:"plugin"`
	Result string    `json:"result"`
	Error  string    `json:"error,omitempty"`
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
func (s *Store) savePersonalLocked(data personalData) error {
	p := &s.personal
	fail := func(err error) error {
		p.err = "Could not save local favorites/history; this change will not survive restart."
		return err
	}
	if p.loadErr != nil {
		return fail(p.loadErr)
	}
	if p.path == "" {
		return fail(errors.New("personal storage path unavailable"))
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0700); err != nil {
		return fail(err)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return fail(err)
	}
	f, err := os.CreateTemp(filepath.Dir(p.path), ".market-personal-*")
	if err != nil {
		return fail(err)
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
		return fail(err)
	}
	if err = os.Rename(f.Name(), p.path); err != nil {
		return fail(err)
	}
	p.data = data
	p.err = ""
	return nil
}
func (s *Store) ToggleFavorite(id string) error {
	s.mu.Lock()
	known := s.personal.data.Favorites[id]
	for _, item := range s.discovery.catalog.Items {
		if item.ID == id {
			known = true
			break
		}
	}
	if !known {
		s.mu.Unlock()
		return errors.New("catalog item unavailable")
	}
	data := s.personal.data
	data.Favorites = map[string]bool{}
	for k, v := range s.personal.data.Favorites {
		data.Favorites[k] = v
	}
	if data.Favorites[id] {
		delete(data.Favorites, id)
	} else {
		data.Favorites[id] = true
	}
	err := s.savePersonalLocked(data)
	s.discovery.page = 0
	s.mu.Unlock()
	s.fire()
	return err
}

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
	s.mu.Lock()
	data := s.personal.data
	data.History = append([]InstallHistory{entry}, data.History...)
	if len(data.History) > 200 {
		data.History = data.History[:200]
	}
	err := s.savePersonalLocked(data)
	// Retain the observed result for this session even if disk persistence fails.
	if err != nil {
		s.personal.data.History = data.History
	}
	s.mu.Unlock()
	s.fire()
	return err
}
