package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchShortcutDocumentChangesDetectsExternalWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"locale":"zh"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var hits atomic.Int32
	done := make(chan struct{})
	go func() {
		watchShortcutDocumentChanges(ctx, path, func() {
			hits.Add(1)
		})
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	if hits.Load() != 0 {
		t.Fatalf("unexpected change notification before write: %d", hits.Load())
	}
	if err := os.WriteFile(path, []byte(`{"locale":"en"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for hits.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if runtime.GOOS == "windows" && hits.Load() == 0 {
		t.Fatal("Windows directory change watcher did not fire after config write")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watchShortcutDocumentChanges did not exit promptly after ctx cancel")
	}
}
