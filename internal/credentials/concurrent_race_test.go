package credentials

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentReloadAndUpdateCheckedMatchesDisk(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = s.Reload()
			}
		}
	}()

	for i := 0; i < 15; i++ {
		val := fmt.Sprintf("key-%02d", i)
		for tries := 0; tries < 10; tries++ {
			err := s.UpdateChecked(func(c *Credentials) error {
				c.APIKey = val
				return nil
			})
			if err == nil {
				break
			}
			_ = s.Reload()
		}
	}
	close(stop)
	wg.Wait()

	// Final in-memory state must match what a fresh Open reads from disk.
	mem := s.Get()
	fresh, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	disk := fresh.Get()
	if mem.APIKey != disk.APIKey {
		t.Fatalf("stale in-memory credentials after concurrent Reload+UpdateChecked: mem=%q disk=%q", mem.APIKey, disk.APIKey)
	}
}
