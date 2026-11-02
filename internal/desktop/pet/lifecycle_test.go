package pet

import (
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

// pinCfgStore returns a store whose pet is disabled: the lifecycle test drives
// SetReady only and never needs a real page.
func pinCfgStore(t *testing.T) *config.Store {
	t.Helper()
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// TestCompanionRecreatesAfterClose is the round154 lifecycle defect: the window
// was created once, closed by a main-window recreation (or quit path), and never
// re-opened because nothing re-entered sync(). SetReady(true) must re-arm the
// manager and ask for a new window, with no duplicates and no window while
// disabled.
func TestCompanionRecreatesAfterClose(t *testing.T) {
	previous, previousDisplays := openCompanionWindow, petDisplays
	t.Cleanup(func() { openCompanionWindow, petDisplays = previous, previousDisplays })
	petDisplays = func() []mygo.Display {
		return []mygo.Display{{Bounds: mygo.Rectangle{X: -1920, Y: -200, Width: 1920, Height: 1080}}, {Bounds: mygo.Rectangle{Width: 2560, Height: 1440}}}
	}
	attempts := 0
	openCompanionWindow = func(opts mygo.WindowOptions) *mygo.Window {
		attempts++
		if opts.X != -1920 || opts.Y != -200 || opts.Width != 1920 || opts.Height != 1080 || !opts.Transparent || !opts.Frameless || !opts.DisableResize || opts.StateKey != "" {
			t.Fatalf("actual companion overlay creation = %+v", opts)
		}
		return nil // the failure path: no window, still a counted attempt
	}

	store := pinCfgStore(t)
	if err := store.SaveDesktopPet(config.DesktopPet{Enabled: true, Behavior: config.DefaultPetBehavior()}); err != nil {
		t.Fatal(err)
	}
	p := New(store)
	p.ready = false

	// Not ready yet: sync must not even try.
	p.sync()
	if attempts != 0 {
		t.Fatalf("created %d windows while not ready", attempts)
	}
	// Ready: one attempt.
	p.SetReady(true)
	if attempts != 1 {
		t.Fatalf("ready created %d windows, want 1", attempts)
	}
	// Close (a main-window recreation or the quit path does this) and re-arm.
	p.Close()
	if attempts != 1 {
		t.Fatalf("Close created another window (%d)", attempts)
	}
	p.SetReady(true)
	if attempts != 2 {
		t.Fatalf("SetReady after Close created %d windows total, want 2", attempts)
	}
	// Disabling the pet must not create another window.
	if err := store.SaveDesktopPet(config.DesktopPet{Enabled: false, Behavior: config.DefaultPetBehavior()}); err != nil {
		t.Fatal(err)
	}
	p.SetReady(true)
	if attempts != 2 {
		t.Fatalf("disabled pet created another window (%d)", attempts)
	}
	if p.win != nil {
		t.Fatal("disabled pet kept a window handle")
	}
}
