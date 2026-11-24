package credentials

import (
	"os"
	"path/filepath"
	"sync"
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
