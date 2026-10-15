// Package recovery persists a bounded desktop resume snapshot. It never writes
// workspace files, session history, credentials, or starts terminal commands.
package recovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const FileName = "go-resume.json"
const MaxBytes = 32 << 20

var ErrConflict = errors.New("resume data changed in another instance; keep this window open and resolve the other instance before retrying")
var ErrBusy = errors.New("resume data is being saved by another instance; retry after it finishes (if all other windows have closed, remove go-resume.json.lock)")

// Layout stores preferences, never viewport-constrained measurements.
type Layout struct {
	WorkspacePanes  json.RawMessage `json:"workspacePanes,omitempty"`
	Sidebar         float32         `json:"sidebar"`
	SidebarExpanded float32         `json:"sidebarExpanded,omitempty"`
	Right           float32         `json:"right"`
	Bottom          float32         `json:"bottom"`
	RightTree       float32         `json:"rightTree"`
	RightOpen       bool            `json:"rightOpen"`
	RightTab        string          `json:"rightTab"`
	ChatTab         string          `json:"chatTab"`
	PlanOpen        bool            `json:"planOpen"`
	TodoCollapsed   bool            `json:"todoCollapsed"`
}

type State struct {
	Version         int    `json:"version"`
	Workspace       string `json:"workspace"`
	SelectedSession string `json:"selectedSession"`
	Layout          Layout `json:"layout"`
	// Drafts are scoped by session id; the empty key means the new-session composer.
	Drafts map[string]json.RawMessage `json:"drafts,omitempty"`
	// Files belongs to the files editor, including unsaved buffers. It is opaque
	// here to avoid coupling persistence to the UI or the file-writing machinery.
	Files json.RawMessage `json:"files,omitempty"`
}

type Store struct {
	mu       sync.Mutex
	path     string
	data     State
	baseline []byte
	exists   bool
}

func read(path string) ([]byte, bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if len(b) > MaxBytes {
		return nil, true, fmt.Errorf("resume file exceeds %d MiB; original preserved at %s", MaxBytes>>20, path)
	}
	return b, true, err
}

// Open is read-only. Invalid/newer files are left exactly where they were;
// callers must show the error rather than silently replace them with defaults.
func Open(dir string) (*Store, error) {
	s := &Store{path: filepath.Join(dir, FileName), data: State{Version: 1}}
	b, exists, err := read(s.path)
	if err != nil {
		return nil, fmt.Errorf("resume: %w", err)
	}
	if exists {
		if err = json.Unmarshal(b, &s.data); err != nil {
			return nil, fmt.Errorf("cannot read resume file %s; original preserved: %w", s.path, err)
		}
		if s.data.Version != 1 {
			return nil, fmt.Errorf("unsupported resume version %d at %s; original preserved", s.data.Version, s.path)
		}
	}
	s.baseline, s.exists = b, exists
	return s, nil
}

func clone(s State) State {
	if s.Drafts != nil {
		drafts := make(map[string]json.RawMessage, len(s.Drafts))
		for k, v := range s.Drafts {
			drafts[k] = append(json.RawMessage(nil), v...)
		}
		s.Drafts = drafts
	}
	s.Files = append(json.RawMessage(nil), s.Files...)
	return s
}
func (s *Store) Snapshot() State { s.mu.Lock(); defer s.mu.Unlock(); return clone(s.data) }
func (s *Store) Path() string    { return s.path }

// Update commits all-or-nothing. Cross-process lock plus exact baseline prevents
// a second stale instance from overwriting unsaved work. No automatic lock
// stealing or automatic conflict resolution can discard the other writer.
func (s *Store) Update(fn func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.data)
	fn(&next)
	next.Version = 1
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > MaxBytes {
		return fmt.Errorf("resume snapshot exceeds %d MiB; nothing was discarded", MaxBytes>>20)
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	lock := s.path + ".lock"
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return ErrBusy
	}
	if err != nil {
		return err
	}
	f.Close()
	defer os.Remove(lock)
	current, exists, err := read(s.path)
	if err != nil {
		return err
	}
	if exists != s.exists || !bytes.Equal(current, s.baseline) {
		return ErrConflict
	}
	if bytes.Equal(b, s.baseline) {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".go-resume-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp.Name(), s.path); err != nil {
		return err
	}
	s.data, s.baseline, s.exists = clone(next), b, true
	return nil
}
