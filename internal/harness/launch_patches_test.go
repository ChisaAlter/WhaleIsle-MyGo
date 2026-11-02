package harness

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestLaunchPatchesRefreshAcrossOwnedAttempts(t *testing.T) {
	enabled := false
	calls := 0
	base := LaunchOptions{PatchFiles: []string{"base"}}
	read := func(context.Context) ([]string, error) {
		calls++
		if enabled {
			return []string{"project"}, nil
		}
		return nil, nil
	}
	first, err := readLaunchPatches(context.Background(), base, read)
	if err != nil || !reflect.DeepEqual(first.PatchFiles, []string{"base"}) {
		t.Fatal(first, err)
	}
	enabled = true
	second, err := readLaunchPatches(context.Background(), base, read)
	if err != nil || !reflect.DeepEqual(second.PatchFiles, []string{"base", "project"}) {
		t.Fatal(second, err)
	}
	enabled = false
	third, err := readLaunchPatches(context.Background(), base, read)
	if err != nil || !reflect.DeepEqual(third.PatchFiles, []string{"base"}) || calls != 3 {
		t.Fatal(third, err, calls)
	}
	second.PatchFiles[0] = "changed"
	if base.PatchFiles[0] != "base" || first.PatchFiles[0] != "base" {
		t.Fatal("attempt shares mutable patches")
	}
	base.SkipUserPlugins = true
	if _, err = readLaunchPatches(context.Background(), base, read); err != nil || calls != 3 {
		t.Fatal("recovery mounted Project")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	base.SkipUserPlugins = false
	if _, err = readLaunchPatches(ctx, base, read); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestProjectRestartWaitsForExactRuntimeGeneration(t *testing.T) {
	s := New(Options{})
	s.mu.Lock()
	s.gen = 4
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	if ticket, issued := s.startOwned(); issued || ticket != 0 {
		t.Fatal("idempotent Start adopted another owner's generation")
	}
	done := make(chan error, 1)
	go func() { done <- s.waitReady(context.Background(), 4) }()
	s.set(4, Status{State: Ready})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.gen = 5
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	go func() { done <- s.waitReady(context.Background(), 5) }()
	s.mu.Lock()
	s.gen = 6
	s.setLocked(Status{State: Ready})
	s.mu.Unlock()
	if err := <-done; err == nil {
		t.Fatal("replacement runtime satisfied old Project switch")
	}
}
