package credentials

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// StorageStatus contains format metadata only. It never contains credential
// values and does not decrypt the envelope or test the OS key's availability.
type StorageStatus struct {
	Mode              string
	Backend           string
	SupportedPlatform bool
	Error             string
}

const storageStatusCacheTTL = 5 * time.Second

type storageStatusCacheEntry struct {
	status    StorageStatus
	updatedAt time.Time
}

var storageStatusCache struct {
	mu       sync.Mutex
	entries  map[string]storageStatusCacheEntry
	inflight map[string][]func(StorageStatus)
}

func storageStatusCacheKey(dir string) string {
	return filepath.Clean(dir)
}

func defaultStorageStatus() StorageStatus {
	return StorageStatus{
		Mode:              "unknown",
		Backend:           credentialOS.backend,
		SupportedPlatform: credentialOS.supported,
	}
}

func storeStorageStatusCache(dir string, s StorageStatus) {
	key := storageStatusCacheKey(dir)
	storageStatusCache.mu.Lock()
	if storageStatusCache.entries == nil {
		storageStatusCache.entries = make(map[string]storageStatusCacheEntry)
	}
	storageStatusCache.entries[key] = storageStatusCacheEntry{
		status:    s,
		updatedAt: time.Now(),
	}
	storageStatusCache.mu.Unlock()
}

// InspectStorage reads credential storage metadata synchronously from disk and
// refreshes the in-memory cache.
func InspectStorage(dir string) StorageStatus {
	s := defaultStorageStatus()
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		s.Mode = "missing"
		storeStorageStatusCache(dir, s)
		return s
	}
	if err != nil {
		s.Error = "Credential storage metadata could not be read."
		storeStorageStatusCache(dir, s)
		return s
	}
	kind, err := classifyDocument(data)
	if err != nil {
		s.Error = "Credential storage format is invalid or unsupported."
		storeStorageStatusCache(dir, s)
		return s
	}
	s.Mode = "plaintext"
	if kind == docEnvelope {
		s.Mode = "encrypted"
		var env envelope
		_ = json.Unmarshal(data, &env) // Already structurally validated above.
		if env.Version != credentialOS.version {
			s.SupportedPlatform = false
			if env.Version == KeychainEnvelopeVersion {
				s.Backend = "macOS Keychain + AES-256-GCM"
			} else {
				s.Backend = "Windows DPAPI + AES-256-GCM (safeStorage-v1)"
				if runtime.GOOS == "darwin" {
					s.Backend = "macOS Keychain + Electron safeStorage (AES-128-CBC)"
					s.SupportedPlatform = credentialOS.supported
				}
			}
		}
	}
	storeStorageStatusCache(dir, s)
	return s
}

// InspectStorageCached returns the cached StorageStatus for dir without
// performing any filesystem I/O.
func InspectStorageCached(dir string) (StorageStatus, bool) {
	key := storageStatusCacheKey(dir)
	storageStatusCache.mu.Lock()
	entry, ok := storageStatusCache.entries[key]
	storageStatusCache.mu.Unlock()
	if !ok {
		return StorageStatus{}, false
	}
	return entry.status, true
}

// InspectStorageAsync returns immediately without blocking the caller on disk
// I/O. When a fresh cached entry exists (within storageStatusCacheTTL), it is
// returned directly. Otherwise the stale cached value (or a default unknown
// status) is returned immediately and a deduplicated background refresh runs
// InspectStorage(dir), invoking onUpdate (if non-nil) once fresh metadata is
// available.
func InspectStorageAsync(dir string, onUpdate func(StorageStatus)) StorageStatus {
	key := storageStatusCacheKey(dir)
	now := time.Now()
	storageStatusCache.mu.Lock()
	entry, ok := storageStatusCache.entries[key]
	if ok && now.Sub(entry.updatedAt) < storageStatusCacheTTL {
		storageStatusCache.mu.Unlock()
		return entry.status
	}
	current := defaultStorageStatus()
	if ok {
		current = entry.status
	}
	if storageStatusCache.inflight == nil {
		storageStatusCache.inflight = make(map[string][]func(StorageStatus))
	}
	waiters, running := storageStatusCache.inflight[key]
	if onUpdate != nil {
		waiters = append(waiters, onUpdate)
	}
	storageStatusCache.inflight[key] = waiters
	storageStatusCache.mu.Unlock()

	if !running {
		go func() {
			fresh := InspectStorage(dir)
			storageStatusCache.mu.Lock()
			callbacks := storageStatusCache.inflight[key]
			delete(storageStatusCache.inflight, key)
			storageStatusCache.mu.Unlock()
			for _, cb := range callbacks {
				if cb != nil {
					cb(fresh)
				}
			}
		}()
	}
	return current
}

// PrefetchStorageStatus warms the storage status cache on a background
// goroutine so the first About-page render hits cache with zero I/O.
func PrefetchStorageStatus(dir string) {
	_ = InspectStorageAsync(dir, nil)
}

// InvalidateStorageStatusCache clears cached metadata for dir (or all dirs when
// dir is empty).
func InvalidateStorageStatusCache(dir string) {
	storageStatusCache.mu.Lock()
	if dir == "" {
		storageStatusCache.entries = nil
	} else {
		delete(storageStatusCache.entries, storageStatusCacheKey(dir))
	}
	storageStatusCache.mu.Unlock()
}
