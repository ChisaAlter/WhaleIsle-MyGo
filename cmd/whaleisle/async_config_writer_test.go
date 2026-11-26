package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func TestAsyncConfigWriterConcurrentEnqueueAndCloseDropsNoAcceptedOp(t *testing.T) {
	for round := 0; round < 8; round++ {
		dir := t.TempDir()
		store, err := config.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		w := newAsyncConfigWriter(store, 2*time.Millisecond, nil)

		var acceptedMu sync.Mutex
		accepted := map[string]bool{}
		start := make(chan struct{})
		var wg sync.WaitGroup

		const workers = 6
		const opsPerWorker = 10
		useMerge := round%2 == 1
		for g := 0; g < workers; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-start
				for i := 0; i < opsPerWorker; i++ {
					tag := fmt.Sprintf("r%d-w%d-i%d", round, id, i)
					ok := w.tryEnqueue(asyncConfigOp{
						merge: useMerge,
						fn: func(c *config.Config) {
							c.DisabledPlugins = append(c.DisabledPlugins, tag)
						},
					})
					if ok {
						acceptedMu.Lock()
						accepted[tag] = true
						acceptedMu.Unlock()
					}
				}
			}(g)
		}

		closeErrCh := make(chan error, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			time.Sleep(time.Duration(round%4) * time.Millisecond)
			closeErrCh <- w.Close()
		}()

		close(start)
		wg.Wait()
		if err := <-closeErrCh; err != nil {
			t.Fatalf("round %d: Close() error: %v", round, err)
		}

		// After Close, no further ops may be accepted.
		if w.tryEnqueue(asyncConfigOp{fn: func(c *config.Config) { c.GoPort = 9999 }}) {
			t.Fatalf("round %d: writer accepted op after Close()", round)
		}

		reopened, err := config.Open(dir)
		if err != nil {
			t.Fatalf("round %d: reopen store: %v", round, err)
		}
		persisted := map[string]bool{}
		for _, tag := range reopened.Get().DisabledPlugins {
			persisted[tag] = true
		}
		for tag := range accepted {
			if !persisted[tag] {
				t.Fatalf("round %d: accepted op %q was silently dropped on Close()", round, tag)
			}
		}
	}
}

func TestConfigWriterForFlushesReplacedWriter(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	storeA, err := config.Open(dirA)
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := config.Open(dirB)
	if err != nil {
		t.Fatal(err)
	}

	s := &mainState{}
	// Enqueue an edit on storeA with the normal debounce window (120ms); it has
	// not flushed to disk yet.
	s.enqueueConfigUpdate(storeA, func(c *config.Config) {
		c.GoPort = 4567
	})

	// Switching to storeB must synchronously flush and close the old writer for storeA.
	wB := s.configWriterFor(storeB)
	if wB == nil || wB.store != storeB {
		t.Fatal("configWriterFor did not return writer for storeB")
	}
	defer func() { _ = wB.Close() }()

	reopenedA, err := config.Open(dirA)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopenedA.Get().GoPort; got != 4567 {
		t.Fatalf("replaced writer was not flushed: GoPort = %d, want 4567", got)
	}
}

func TestSaveDesktopConfigAsyncGenerationSupersedesStaleSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WHALEISLE_ISOLATED", "1")
	t.Setenv(paths.EnvUserData, dir)
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	s := &mainState{
		cfgStore:               store,
		hostAppearanceAccepted: true, // headless test has no mygo event loop for Theme.SetSource
	}
	s.seedDesktopDraft()

	// Enqueue a slow op on the config writer so save #1 blocks on writer.Flush()
	// while save #2 (invalid port) supersedes it.
	inFlush := make(chan struct{}, 1)
	releaseFlush := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseFlush) })
	w := s.configWriterFor(store)
	w.EnqueueUpdate(func(c *config.Config) {
		select {
		case inFlush <- struct{}{}:
		default:
		}
		<-releaseFlush
	})
	// Trigger drain so writer.Flush() inside save #1 waits behind it.
	go func() { _ = w.Flush() }()
	select {
	case <-inFlush:
	case <-time.After(2 * time.Second):
		t.Fatal("background writer did not start blocking op")
	}

	// Save #1 is valid (port 4100), starts async goroutine blocked on writer.Flush().
	s.mu.Lock()
	s.desktopDraft.port = "4100"
	s.desktopDraft.closeToTray = true
	s.mu.Unlock()
	s.saveDesktopConfigAsync(nil)

	// Save #2 immediately supersedes with an invalid port and sets desktopSaveErr.
	s.mu.Lock()
	s.desktopDraft.port = "not-a-port"
	s.mu.Unlock()
	s.saveDesktopConfigAsync(nil)

	s.mu.Lock()
	errAfterSecond := s.desktopSaveErr
	genAfterSecond := s.desktopSaveGen
	s.mu.Unlock()
	if errAfterSecond != i18n.T(i18n.KeyCmdDesktopInvalidPort) || genAfterSecond < 2 {
		t.Fatalf("second save did not record validation error: err=%q gen=%d", errAfterSecond, genAfterSecond)
	}

	// Unblock save #1 and wait briefly for its goroutine to exit.
	releaseOnce.Do(func() { close(releaseFlush) })
	time.Sleep(80 * time.Millisecond)

	s.mu.Lock()
	finalErr := s.desktopSaveErr
	finalAction := s.desktopLastAction
	s.mu.Unlock()
	if finalErr != i18n.T(i18n.KeyCmdDesktopInvalidPort) || finalAction != "" {
		t.Fatalf("stale async save overwrote newer error state: err=%q action=%q", finalErr, finalAction)
	}
	if got := store.Get().GoPort; got == 4100 {
		t.Fatalf("superseded async save committed stale port 4100 to store")
	}
}

func TestShortcutEditorSaveAndResetRespectLockDiscipline(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WHALEISLE_ISOLATED", "1")
	t.Setenv(paths.EnvUserData, dir)
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: cfg}
	editor := s.pageShortcutEditor()

	*editor.Draft("session.search") = "Mod+Alt+F10"
	if !editor.Save() {
		t.Fatalf("shortcut Save failed: %q", editor.Err)
	}
	editor.RequestReset()
	if !editor.ConfirmReset() {
		t.Fatalf("shortcut ConfirmReset failed: %q", editor.Err)
	}
	s.mu.Lock()
	rev := s.shortcutRevision
	s.mu.Unlock()
	if rev == "" {
		t.Fatal("expected shortcutRevision to be populated after Save/Reset")
	}
}
