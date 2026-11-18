package pluginmanager

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestHostInventorySharesReadAndRetiresChangedGeneration(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var calls atomic.Int32
	source := NewHostInventory(fixtureCall(func(ctx context.Context, method string, _, out any) error {
		n := calls.Add(1)
		started <- struct{}{}
		if n == 1 {
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return reply(out, map[string]any{"generation": n})
	}), nil)
	defer source.Close()
	var wg sync.WaitGroup
	results := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out struct{ Generation int }
			if err := source.Call(context.Background(), "pluginInventory/list", nil, &out); err != nil {
				t.Error(err)
			}
			results <- out.Generation
		}()
	}
	<-started
	source.InvalidateInventory()
	close(release)
	wg.Wait()
	close(results)
	for got := range results {
		if got != 2 {
			t.Fatalf("stale read published %d", got)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("consumers did not share generation: %d", calls.Load())
	}
}

func TestHostInventoryFailureDoesNotBecomeEmptySuccess(t *testing.T) {
	var calls atomic.Int32
	source := NewHostInventory(fixtureCall(func(_ context.Context, _ string, _, out any) error {
		if calls.Add(1) == 1 {
			return errors.New("offline")
		}
		return reply(out, map[string]bool{"ready": true})
	}), nil)
	defer source.Close()
	var out map[string]bool
	if source.Call(context.Background(), "pluginInventory/list", nil, &out) == nil {
		t.Fatal("failed read accepted")
	}
	if _, ok := source.Cached("pluginInventory/list", &out); ok {
		t.Fatal("error became authoritative empty inventory")
	}
	if err := source.Call(context.Background(), "pluginInventory/list", nil, &out); err != nil || !out["ready"] {
		t.Fatalf("explicit retry failed: %v %v", out, err)
	}
}
