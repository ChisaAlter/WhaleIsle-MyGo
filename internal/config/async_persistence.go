package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultAsyncDebounce = 150 * time.Millisecond

type asyncPersistState struct {
	writeMu    sync.Mutex
	debounce   time.Duration
	timer      *time.Timer
	dirty      bool
	seq        uint64
	lastErr    error
	errHandler func(error)
	hasPending atomic.Bool
}

func (s *Store) asyncDebounceLocked() time.Duration {
	if s.async.debounce > 0 {
		return s.async.debounce
	}
	return defaultAsyncDebounce
}

// SetAsyncDebounce configures the coalescing window for asynchronous config
// writes. Zero restores the default (150ms).
func (s *Store) SetAsyncDebounce(d time.Duration) {
	s.mu.Lock()
	s.async.debounce = d
	s.mu.Unlock()
}

// SetAsyncErrorHandler registers an optional callback invoked outside Store.mu
// whenever a background debounced save fails.
func (s *Store) SetAsyncErrorHandler(fn func(error)) {
	s.mu.Lock()
	s.async.errHandler = fn
	s.mu.Unlock()
}

// LastAsyncError returns the error from the most recent background save, or nil
// if the last flush succeeded.
func (s *Store) LastAsyncError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.async.lastErr
}

func (s *Store) hasPendingAsync() bool {
	return s.async.hasPending.Load()
}

func (s *Store) scheduleAsyncPersistLocked() {
	s.async.dirty = true
	s.async.seq++
	s.async.hasPending.Store(true)
	delay := s.asyncDebounceLocked()
	if s.async.timer != nil {
		s.async.timer.Reset(delay)
		return
	}
	s.async.timer = time.AfterFunc(delay, func() {
		_ = s.flushAsync(false)
	})
}

// UpdateAsync optimistically applies fn to the in-memory configuration and
// schedules a single-writer debounced atomic disk write (temp file + Sync +
// guarded Rename) off the caller thread. Call Flush before process exit or
// restart to guarantee durability.
func (s *Store) UpdateAsync(fn func(*Config)) error {
	if fn == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	candidate := cloneConfig(s.cfg)
	fn(&candidate)
	if candidate.SkipUserPlugins != s.cfg.SkipUserPlugins {
		candidate.SkipUserPlugins = s.cfg.SkipUserPlugins
	}
	next := normalize(cloneConfig(candidate))
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &known); err != nil {
		return err
	}
	merged := mergeKnownConfig(s.raw, known)
	s.raw = merged
	s.setCfgLocked(next)
	s.scheduleAsyncPersistLocked()
	return nil
}

// MergeAsync optimistically updates in-memory configuration like UpdateAsync
// and schedules a single-writer debounced atomic write.
func (s *Store) MergeAsync(fn func(*Config)) error {
	return s.UpdateAsync(fn)
}

// SaveWorkspaceNavigationPreferencesAsync optimistically updates workspace
// navigation preferences in memory and schedules a debounced atomic disk write
// without blocking the caller on fsync or holding Store.mu across disk I/O.
func (s *Store) SaveWorkspaceNavigationPreferencesAsync(hostKey string, prefs WorkspaceNavigationPreferences) error {
	if strings.TrimSpace(hostKey) == "" {
		return errors.New("workspace preference Host is missing")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	merged, err := s.prepareWorkspacePreferencesRawLocked(hostKey, prefs)
	if err != nil {
		return err
	}
	s.raw = merged
	s.scheduleAsyncPersistLocked()
	return nil
}

func (s *Store) prepareWorkspacePreferencesRawLocked(hostKey string, prefs WorkspaceNavigationPreferences) (map[string]json.RawMessage, error) {
	root := map[string]json.RawMessage{}
	if raw := s.raw[workspaceNavigationKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			return nil, err
		}
	}
	if root == nil {
		root = map[string]json.RawMessage{}
	}
	host := map[string]json.RawMessage{}
	if raw := root[hostKey]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &host); err != nil {
			return nil, err
		}
	}
	if host == nil {
		host = map[string]json.RawMessage{}
	}
	encoded, err := json.Marshal(prefs)
	if err != nil {
		return nil, err
	}
	var known map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &known); err != nil {
		return nil, err
	}
	for k, v := range known {
		host[k] = v
	}
	hostRaw, err := json.Marshal(host)
	if err != nil {
		return nil, err
	}
	root[hostKey] = hostRaw
	rootRaw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	merged := cloneRawMap(s.raw)
	merged[workspaceNavigationKey] = rootRaw
	return merged, nil
}

// Flush synchronously persists any pending debounced async writes using the
// store's atomic temp-file + Sync + guarded-Rename path. Safe to call
// concurrently and idempotent when nothing is pending.
func (s *Store) Flush() error {
	return s.flushAsync(true)
}

func (s *Store) flushAsync(syncCaller bool) error {
	s.async.writeMu.Lock()
	defer s.async.writeMu.Unlock()
	return s.flushAsyncLocked(syncCaller)
}

// flushAsyncLocked drains all pending async edits while the caller already
// holds s.async.writeMu. It always snapshots the latest s.raw under s.mu so
// interleaved synchronous writes can never be overwritten by stale raw maps.
func (s *Store) flushAsyncLocked(syncCaller bool) error {
	for {
		s.mu.Lock()
		if s.async.timer != nil {
			s.async.timer.Stop()
			s.async.timer = nil
		}
		if !s.async.dirty {
			lastErr := s.async.lastErr
			s.async.hasPending.Store(false)
			s.mu.Unlock()
			if syncCaller {
				return lastErr
			}
			return nil
		}
		seq := s.async.seq
		merged := cloneRawMap(s.raw)
		expected := s.baseline
		path := s.path
		commitLock := s.commitLock
		rename := s.rename
		s.mu.Unlock()

		// Perform serialization, advisory lock, baseline verification, fsync and
		// atomic rename completely outside s.mu so readers and UI threads never
		// stall behind disk I/O.
		baseline, err := commitRawWithSeams(path, expected, merged, commitLock, rename)

		var errHandler func(error)
		s.mu.Lock()
		if err != nil {
			s.async.lastErr = err
			if s.async.seq == seq {
				s.async.dirty = false
				s.async.hasPending.Store(false)
			}
			errHandler = s.async.errHandler
			s.mu.Unlock()
			if errHandler != nil {
				errHandler(err)
			}
			return err
		}
		s.baseline = baseline
		s.async.lastErr = nil
		if s.async.seq == seq {
			s.async.dirty = false
			s.async.hasPending.Store(false)
			s.mu.Unlock()
			return nil
		}
		// Another optimistic write arrived while we were flushing; loop to commit
		// the latest coalesced state under the same single writer.
		s.mu.Unlock()
	}
}

// beginSyncWrite enforces lock hierarchy s.async.writeMu -> s.mu across all
// synchronous config writers, flushing any pending debounced async edits while
// holding writeMu so there is zero TOCTOU window and zero lost writes.
func (s *Store) beginSyncWrite() (func(), error) {
	s.async.writeMu.Lock()
	if s.hasPendingAsync() {
		if err := s.flushAsyncLocked(true); err != nil {
			s.async.writeMu.Unlock()
			return nil, err
		}
	}
	s.mu.Lock()
	return func() {
		s.mu.Unlock()
		s.async.writeMu.Unlock()
	}, nil
}

func (s *Store) flushPendingBeforeSync() error {
	if !s.hasPendingAsync() {
		return nil
	}
	return s.Flush()
}

// commitRawWithSeams commits a merged raw config map without holding s.mu.
func commitRawWithSeams(
	path string,
	expected fileBaseline,
	merged map[string]json.RawMessage,
	commitLock func(string) (func(), error),
	rename func(oldpath, newpath string) error,
) (fileBaseline, error) {
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fileBaseline{}, err
	}
	commitMu.Lock()
	lock := commitLock
	if lock == nil {
		lock = acquireCommitLock
	}
	release, err := lock(path)
	if err != nil {
		commitMu.Unlock()
		return fileBaseline{}, err
	}
	defer func() {
		release()
		commitMu.Unlock()
	}()
	ok, err := expected.matches(path)
	if err != nil {
		return fileBaseline{}, err
	}
	if !ok {
		return fileBaseline{}, ErrConflict
	}
	r := rename
	if r == nil {
		r = os.Rename
	}
	guarded := func(oldpath, newpath string) error {
		ok, err := expected.matches(path)
		if err != nil {
			return err
		}
		if !ok {
			return ErrConflict
		}
		return r(oldpath, newpath)
	}
	if err := writeAtomic(path, data, guarded); err != nil {
		return fileBaseline{}, err
	}
	return fileBaseline{exists: true, data: data}, nil
}
