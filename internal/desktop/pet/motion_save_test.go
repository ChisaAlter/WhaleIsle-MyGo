package pet

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
)

func TestSaveMotionAnchorIsAsyncDebouncedAndFlushesOnClose(t *testing.T) {
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveDesktopPet(config.DesktopPet{Enabled: true, Behavior: config.DefaultPetBehavior()}); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	block := make(chan struct{})
	started := make(chan struct{}, 1)
	var mu sync.Mutex
	var saved []config.PetPosition

	p := &Companion{Store: store, Enabled: true}
	p.motionSave.debounce = 25 * time.Millisecond
	p.motionSave.write = func(cfg config.DesktopPet) error {
		calls.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-block
		mu.Lock()
		if cfg.Position != nil {
			saved = append(saved, *cfg.Position)
		}
		mu.Unlock()
		return store.SaveDesktopPet(cfg)
	}

	t0 := time.Now()
	// Rapid drag-drop samples must return immediately without blocking on disk I/O.
	p.saveMotionAnchor(motionPoint{x: 100, y: 120})
	p.saveMotionAnchor(motionPoint{x: 140, y: 160})
	p.saveMotionAnchor(motionPoint{x: 220, y: 260})
	if elapsed := time.Since(t0); elapsed > 50*time.Millisecond {
		t.Fatalf("saveMotionAnchor blocked caller thread for %v", elapsed)
	}

	// Wait for the single debounced write to enter the background writer.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("background motion anchor writer did not start")
	}
	close(block)
	p.Close()

	if got := calls.Load(); got != 1 {
		t.Fatalf("debounced write count = %d, want 1", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(saved) != 1 || saved[0].X != 220 || saved[0].Y != 260 {
		t.Fatalf("saved anchors = %+v, want single final anchor (220, 260)", saved)
	}
	final := store.DesktopPet()
	if final.Position == nil || final.Position.X != 220 || final.Position.Y != 260 {
		t.Fatalf("persisted position = %+v, want (220, 260)", final.Position)
	}
}
