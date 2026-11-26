package credentials

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// ErrConflict is returned by Store writes when credentials.json changed on disk
// after the Store loaded or last committed it.
var ErrConflict = errors.New("credentials changed on disk since last load")

type fileBaseline struct {
	exists bool
	size   int64
	sha256 [32]byte
}

func baselineFromBytes(data []byte, err error) (fileBaseline, error) {
	if errors.Is(err, fs.ErrNotExist) {
		return fileBaseline{}, nil
	}
	if err != nil {
		return fileBaseline{}, err
	}
	return fileBaseline{
		exists: true,
		size:   int64(len(data)),
		sha256: sha256.Sum256(data),
	}, nil
}

func readBytesAndBaseline(path string) ([]byte, fileBaseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fileBaseline{}, err
	}
	return data, fileBaseline{
		exists: true,
		size:   int64(len(data)),
		sha256: sha256.Sum256(data),
	}, nil
}

func readFileBaseline(path string) (fileBaseline, error) {
	data, err := os.ReadFile(path)
	return baselineFromBytes(data, err)
}

func (b fileBaseline) matches(other fileBaseline) bool {
	return b.exists == other.exists && b.size == other.size && b.sha256 == other.sha256
}

func guardedRename(path string, expected fileBaseline) func(string, string) error {
	return func(oldpath, newpath string) error {
		disk, err := readFileBaseline(path)
		if err != nil || !expected.matches(disk) {
			return fmt.Errorf("%w (%s)", ErrConflict, path)
		}
		return os.Rename(oldpath, newpath)
	}
}

// Store holds an in-memory snapshot of credentials.json with its disk baseline
// and unmodeled JSON fields preserved across updates.
type Store struct {
	mu       sync.RWMutex
	dir      string
	path     string
	creds    Credentials
	raw      map[string]json.RawMessage
	baseline fileBaseline
	shared   bool
}

// Open loads a private credentials store with participating-writer locking.
// Desktop callers sharing with Electron must use OpenSharedProfile instead.
// Missing files yield an empty Store without creating files or directories.
func Open(dir string) (*Store, error) {
	return openStore(dir, false)
}

// OpenSharedProfile reads the desktop profile without requiring write access.
// Every subsequent Store write must acquire the platform's shared write lease.
func OpenSharedProfile(dir string) (*Store, error) {
	return openStore(dir, true)
}

func openStore(dir string, shared bool) (*Store, error) {
	s := &Store{
		dir:    dir,
		path:   filepath.Join(dir, FileName),
		raw:    map[string]json.RawMessage{},
		shared: shared,
	}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// WriteAccessError reports a known platform limitation without touching the
// filesystem. A nil result still requires the real lease at commit time.
func (s *Store) WriteAccessError() error {
	if s.shared {
		return sharedProfileWriteSupport()
	}
	return nil
}

// Reload refreshes the Store snapshot, preserved unmodeled JSON fields, and
// disk baseline under the cross-process credentials lock.
func (s *Store) Reload() error {
	return withDirLock(s.dir, false, func() error {
		var (
			creds    Credentials
			raw      map[string]json.RawMessage
			baseline fileBaseline
		)
		data, b, readErr := readBytesAndBaseline(s.path)
		baseline = b
		if errors.Is(readErr, fs.ErrNotExist) {
			raw = map[string]json.RawMessage{}
		} else if readErr != nil {
			return readErr
		} else {
			var decErr error
			raw, creds, decErr = decodeDocumentBytes(s.dir, s.path, data)
			if decErr != nil {
				return decErr
			}
		}
		// Publish into memory while still holding dirLock so concurrent
		// Reload and UpdateChecked preserve disk-to-memory ordering.
		s.mu.Lock()
		clearRawMap(s.raw)
		s.creds = cloneCredentials(creds)
		s.raw = cloneRawMap(raw)
		s.baseline = baseline
		s.mu.Unlock()
		clearRawMap(raw)
		return nil
	})
}

// Get returns a defensive copy of the committed Credentials.
func (s *Store) Get() Credentials {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneCredentials(s.creds)
}

// Save replaces the modeled credential fields if the file on disk still matches
// the Store's baseline, preserving unknown top-level and nested JSON fields.
func (s *Store) Save(creds Credentials) error {
	return s.UpdateChecked(func(c *Credentials) error {
		*c = cloneCredentials(creds)
		return nil
	})
}

// Update applies a mutation and persists it if the disk baseline still matches.
func (s *Store) Update(fn func(*Credentials)) error {
	return s.UpdateChecked(func(c *Credentials) error {
		fn(c)
		return nil
	})
}

// UpdateChecked applies fn to a copy of the committed Credentials and persists
// atomically if fn succeeds and credentials.json still matches the Store's baseline.
func (s *Store) UpdateChecked(fn func(*Credentials) error) error {
	s.mu.RLock()
	draft := cloneCredentials(s.creds)
	baseRaw := cloneRawMap(s.raw)
	expected := s.baseline
	s.mu.RUnlock()
	defer clearRawMap(baseRaw)

	if err := fn(&draft); err != nil {
		return err
	}

	var (
		committedBaseline fileBaseline
		committedRaw      map[string]json.RawMessage
	)
	defer func() {
		clearRawMap(committedRaw)
	}()
	err := withProfileDirLock(s.dir, true, s.shared, func() error {
		diskRaw, create, diskBaseline, readErr := readExistingRawForSave(s.dir)
		if readErr != nil {
			return readErr
		}
		defer clearRawMap(diskRaw)

		if !expected.matches(diskBaseline) {
			return fmt.Errorf("%w (%s)", ErrConflict, s.path)
		}
		mergeBase := baseRaw
		if len(diskRaw) > 0 {
			mergeBase = diskRaw
		}
		merged, mergeErr := mergeCredentialsIntoRaw(mergeBase, draft)
		if mergeErr != nil {
			return mergeErr
		}
		defer clearRawMap(merged)

		if err := sealAndWriteRaw(s.dir, merged, create, guardedRename(s.path, diskBaseline)); err != nil {
			return err
		}
		committedBaseline, readErr = readFileBaseline(s.path)
		if readErr != nil {
			return readErr
		}
		committedRaw = cloneRawMap(merged)
		// Publish committed state while still holding the profile dir lock so
		// concurrent Reload/UpdateChecked cannot publish out of disk order.
		s.mu.Lock()
		clearRawMap(s.raw)
		s.creds = cloneCredentials(draft)
		s.raw = cloneRawMap(committedRaw)
		s.baseline = committedBaseline
		s.mu.Unlock()
		return nil
	})
	return err
}

// LoadNamespace decodes a single encrypted namespace key from credentials.json
// without creating directories or files when absent.
func LoadNamespace(dir, key string, out any) (bool, error) {
	if key == "" {
		return false, errors.New("credentials: namespace key required")
	}
	var (
		found bool
		entry json.RawMessage
	)
	err := withDirLock(dir, false, func() error {
		raw, _, loadErr := loadRawUnlocked(dir)
		if loadErr != nil {
			return loadErr
		}
		if raw == nil {
			return nil
		}
		defer clearRawMap(raw)
		v, ok := raw[key]
		trimmed := bytes.TrimSpace(v)
		if !ok || len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			return nil
		}
		entry = bytes.Clone(trimmed)
		found = true
		return nil
	})
	if err != nil || !found {
		clear(entry)
		return false, err
	}
	defer clear(entry)
	if err := json.Unmarshal(entry, out); err != nil {
		return false, fmt.Errorf("credentials: decode namespace %q: %w", key, err)
	}
	return true, nil
}

// UpdateNamespace atomically updates a single namespace key inside encrypted
// credentials.json while preserving all sibling credential and unknown fields.
// Returning nil or empty bytes deletes the namespace key. Shared desktop
// profiles must use UpdateSharedProfileNamespace instead.
func UpdateNamespace(dir, key string, fn func(existing json.RawMessage) (json.RawMessage, error)) error {
	return updateNamespace(dir, key, false, fn)
}

// UpdateSharedProfileNamespace preserves the encrypted namespace contract and
// additionally requires write ownership of the desktop's shared profile.
func UpdateSharedProfileNamespace(dir, key string, fn func(existing json.RawMessage) (json.RawMessage, error)) error {
	return updateNamespace(dir, key, true, fn)
}

func updateNamespace(dir, key string, shared bool, fn func(existing json.RawMessage) (json.RawMessage, error)) error {
	if key == "" {
		return errors.New("credentials: namespace key required")
	}
	return withProfileDirLock(dir, true, shared, func() error {
		raw, create, baseline, err := readExistingRawForSave(dir)
		if err != nil {
			return err
		}
		defer clearRawMap(raw)

		existing := bytes.Clone(raw[key])
		defer clear(existing)
		next, err := fn(existing)
		defer clear(next)
		if err != nil {
			return err
		}
		trimmed := bytes.TrimSpace(next)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			if _, had := raw[key]; !had {
				return nil
			}
			delete(raw, key)
		} else {
			if !json.Valid(trimmed) {
				return fmt.Errorf("credentials: namespace %q produced invalid JSON", key)
			}
			raw[key] = bytes.Clone(trimmed)
		}
		return sealAndWriteRaw(dir, raw, create, guardedRename(filepath.Join(dir, FileName), baseline))
	})
}

func cloneCredentials(c Credentials) Credentials {
	if c.RemoteDevices == nil {
		return c
	}
	cloned := make([]RemoteDevice, len(c.RemoteDevices))
	for i, d := range c.RemoteDevices {
		cloned[i] = cloneAnyMap(d)
	}
	c.RemoteDevices = cloned
	return c
}

func cloneAnyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneAnyValue(v)
	}
	return out
}

func cloneAnyValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneAnyValue(item)
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	case []byte:
		return bytes.Clone(typed)
	default:
		return typed
	}
}
