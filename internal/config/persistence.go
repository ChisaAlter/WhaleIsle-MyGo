package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrConflict reports that config.json changed on disk since this Store last
// read or wrote it. Update refuses to overwrite in that case: the caller must
// reopen the store and deliberately redo the change.
var ErrConflict = errors.New("config: configuration changed on disk since it was read")

// commitMu serializes the final compare-and-replace step across every Store in
// this process. lockCommit also takes the shared advisory .config.lock before
// each writer's final baseline check and rename. Non-cooperating writers remain
// outside that lock; the baseline check detects their edits made before it.
var commitMu sync.Mutex

// lockCommit protects every config.json writer, including namespace-only saves.
// The caller holds s.mu and must retain the returned lock through replacement.
func (s *Store) lockCommit() (func(), error) {
	commitMu.Lock()
	lock := s.commitLock
	if lock == nil {
		lock = acquireCommitLock
	}
	release, err := lock(s.path)
	if err != nil {
		commitMu.Unlock()
		return nil, err
	}
	return func() {
		release()
		commitMu.Unlock()
	}, nil
}

// fileBaseline is the exact on-disk state a Store last observed. It is
// compared byte-for-byte so permissive reparsing or timestamps cannot mask an
// external edit.
type fileBaseline struct {
	exists bool
	data   []byte
}

func readBaseline(path string) (fileBaseline, error) {
	var data []byte
	var err error
	for attempt := 0; attempt < 8; attempt++ {
		data, err = os.ReadFile(path)
		if err == nil {
			return fileBaseline{exists: true, data: data}, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			return fileBaseline{}, nil
		}
		// Windows can briefly share-lock the file during a concurrent rename.
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return fileBaseline{}, err
}

// matches reports whether the file on disk is still identical to the baseline.
// A read error other than "not exist" is returned so callers stop instead of
// guessing.
func (b fileBaseline) matches(path string) (bool, error) {
	cur, err := readBaseline(path)
	if err != nil {
		return false, err
	}
	return cur.exists == b.exists && bytes.Equal(cur.data, b.data), nil
}

// guardedRenameFor returns a rename wrapper that verifies expected matches
// config.json immediately before replacing it, aborting with ErrConflict if an
// external edit landed in between.
func (s *Store) guardedRenameFor(expected fileBaseline) func(oldpath, newpath string) error {
	rename := s.rename
	if rename == nil {
		rename = os.Rename
	}
	return func(oldpath, newpath string) error {
		ok, err := expected.matches(s.path)
		if err != nil {
			return err
		}
		if !ok {
			return ErrConflict
		}
		return rename(oldpath, newpath)
	}
}

// commitRawLocked commits a merged raw config map under lockCommit with
// pre-rename baseline verification. Callers must hold s.mu.
func (s *Store) commitRawLocked(expected fileBaseline, merged map[string]json.RawMessage) (fileBaseline, error) {
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fileBaseline{}, err
	}
	unlock, err := s.lockCommit()
	if err != nil {
		return fileBaseline{}, err
	}
	defer unlock()
	ok, err := expected.matches(s.path)
	if err != nil {
		return fileBaseline{}, err
	}
	if !ok {
		return fileBaseline{}, ErrConflict
	}
	if err := writeAtomic(s.path, data, s.guardedRenameFor(expected)); err != nil {
		return fileBaseline{}, err
	}
	return fileBaseline{exists: true, data: data}, nil
}

// writeAtomic writes data to a temporary file next to path and renames it into
// place. rename is injectable so tests can force a deterministic failure.
func writeAtomic(path string, data []byte, rename func(oldpath, newpath string) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if rename == nil {
		rename = os.Rename
	}
	var errRename error
	for attempt := 0; attempt < 8; attempt++ {
		errRename = rename(tmp.Name(), path)
		if errRename == nil {
			return nil
		}
		if errors.Is(errRename, ErrConflict) {
			return errRename
		}
		time.Sleep(jitteredDuration(time.Duration(attempt+1) * 5 * time.Millisecond))
	}
	return errRename
}
