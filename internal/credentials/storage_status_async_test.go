package credentials

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInspectStorageAsyncCacheAndInvalidation(t *testing.T) {
	dir := t.TempDir()
	InvalidateStorageStatusCache(dir)

	if _, ok := InspectStorageCached(dir); ok {
		t.Fatal("expected cold cache miss")
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"apiKey":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	updated := make(chan StorageStatus, 1)
	initial := InspectStorageAsync(dir, func(s StorageStatus) {
		select {
		case updated <- s:
		default:
		}
	})
	if initial.Mode != "unknown" {
		t.Fatalf("cold async call should return immediate placeholder, got %+v", initial)
	}
	select {
	case fresh := <-updated:
		if fresh.Mode != "plaintext" {
			t.Fatalf("async refresh got %+v, want plaintext", fresh)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async storage refresh")
	}

	// Subsequent call hits fresh cache immediately with zero callback delay.
	cached, ok := InspectStorageCached(dir)
	if !ok || cached.Mode != "plaintext" {
		t.Fatalf("cached status = %+v ok=%v", cached, ok)
	}
	if got := InspectStorageAsync(dir, func(StorageStatus) {
		t.Error("fresh cache hit should not trigger redundant background read")
	}); got.Mode != "plaintext" {
		t.Fatalf("warm async call got %+v", got)
	}

	// Concurrent callers deduplicate safely under -race.
	InvalidateStorageStatusCache(dir)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done := make(chan struct{}, 1)
			_ = InspectStorageAsync(dir, func(StorageStatus) {
				select {
				case done <- struct{}{}:
				default:
				}
			})
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}()
	}
	wg.Wait()
}

func TestInspectStorageAsyncInvalidationPreventsStaleCacheRepopulation(t *testing.T) {
	dir := t.TempDir()
	InvalidateStorageStatusCache(dir)
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"apiKey":"old-plain"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	var firstRead atomic.Bool
	origRead := readStorageFile
	t.Cleanup(func() { readStorageFile = origRead })
	readStorageFile = func(name string) ([]byte, error) {
		if firstRead.CompareAndSwap(false, true) {
			data, err := origRead(name)
			close(entered)
			<-release
			return data, err
		}
		return origRead(name)
	}

	staleDone := make(chan StorageStatus, 1)
	_ = InspectStorageAsync(dir, func(s StorageStatus) {
		staleDone <- s
	})

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("first async read did not enter read hook")
	}

	// Simulate sealAndWriteRaw replacing the file and invalidating cache while
	// the earlier background read is still in flight.
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"version":"safeStorage-v1","payload":"djEw"}`), 0o600); err != nil {
		close(release)
		t.Fatal(err)
	}
	InvalidateStorageStatusCache(dir)

	// Let the stale read complete; it must NOT repopulate the cache.
	close(release)
	select {
	case <-staleDone:
	case <-time.After(2 * time.Second):
		t.Fatal("stale read callback did not finish")
	}
	if cached, ok := InspectStorageCached(dir); ok {
		t.Fatalf("stale in-flight read repopulated invalidated cache: %+v", cached)
	}

	// Subsequent InspectStorageAsync starts a fresh read and caches encrypted mode.
	freshDone := make(chan StorageStatus, 1)
	_ = InspectStorageAsync(dir, func(s StorageStatus) {
		freshDone <- s
	})
	select {
	case fresh := <-freshDone:
		if fresh.Mode != "encrypted" {
			t.Fatalf("fresh read got %+v, want encrypted", fresh)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fresh async read did not complete")
	}
	if cached, ok := InspectStorageCached(dir); !ok || cached.Mode != "encrypted" {
		t.Fatalf("expected encrypted in cache, got %+v ok=%v", cached, ok)
	}
}
