package chrome

import (
	"testing"
	"time"

	"whaleisle/internal/desktop/wallpapers"
)

func latestTestConfig() gradientConfig {
	p := wallpapers.Defaults()
	p.Preset = "aurora"
	return gradientConfiguration(p, 1440, 864, false, false)
}

func waitRenders(t *testing.T, cache *gradientFrameCache, want uint64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		got := cache.renders
		cache.mu.Unlock()
		if got >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("worker render %d did not finish", want)
}

// The first frame of a configuration is synchronous, so a painter never has
// nothing to show; a phase advance returns the previous frame at once and the
// finished one is picked up by a later call.
func TestAmbientGradientLatestDoesNotRenderPhaseAdvanceInline(t *testing.T) {
	cache := gradientFrameCache{}
	c := latestTestConfig()
	now := time.Unix(100, 0)
	first := cache.latest(c, now)
	if first == nil || cache.renders != 1 {
		t.Fatalf("first frame must render synchronously: frame=%v renders=%d", first, cache.renders)
	}
	if again := cache.latest(c, now.Add(time.Millisecond)); again != first || cache.renders != 1 {
		t.Fatal("an unchanged phase rendered again")
	}
	later := now.Add(3 * gradientFrameInterval)
	if got := cache.latest(c, later); got != first {
		t.Fatal("a phase advance must return the previous frame without waiting")
	}
	waitRenders(t, &cache, 2)
	next := cache.latest(c, later)
	if next == nil || next == first || next.tick != gradientTick(later, cache.epoch, false) {
		t.Fatalf("finished frame was not picked up: %+v", next)
	}
	if cache.renders != 2 || cache.rendering {
		t.Fatalf("renders=%d rendering=%v", cache.renders, cache.rendering)
	}
}

// A reset while a worker is in flight revokes its result and must not leave the
// cache believing a render is still running.
func TestAmbientGradientLatestResetRevokesWorker(t *testing.T) {
	cache := gradientFrameCache{}
	c := latestTestConfig()
	now := time.Unix(100, 0)
	cache.latest(c, now)
	cache.latest(c, now.Add(3*gradientFrameInterval))
	cache.reset()
	if cache.frame != nil || cache.rendering {
		t.Fatalf("reset left state: frame=%v rendering=%v", cache.frame, cache.rendering)
	}
	time.Sleep(50 * time.Millisecond) // let the revoked worker finish
	cache.mu.Lock()
	published := cache.frame != nil
	cache.mu.Unlock()
	if published {
		t.Fatal("a revoked worker published after reset")
	}
	if got := cache.latest(c, now); got == nil || cache.rendering {
		t.Fatal("cache did not recover after reset")
	}
}

// A configuration change takes the synchronous path and drops the old worker.
func TestAmbientGradientLatestConfigurationChange(t *testing.T) {
	cache := gradientFrameCache{}
	c := latestTestConfig()
	now := time.Unix(100, 0)
	old := cache.latest(c, now)
	cache.latest(c, now.Add(3*gradientFrameInterval))
	c.colors[2] = gradientColor{1, 0, 0, 1}
	changed := cache.latest(c, now.Add(3*gradientFrameInterval))
	if changed == nil || changed == old || changed.config != c {
		t.Fatal("a changed configuration must render synchronously, not reuse the old frame")
	}
	if cache.rendering {
		t.Fatal("rendering flag leaked across configurations")
	}
}
