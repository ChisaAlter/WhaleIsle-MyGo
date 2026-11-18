package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComposerFileIntakeRejectsDirectoryOffThreadBeforeAnyEdit(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "note.txt")
	if err := os.WriteFile(file, []byte("note"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectComposerFiles(context.Background(), []string{file, root}, true); err == nil {
		t.Fatal("drop/paste admitted directory")
	}
	files, err := inspectComposerFiles(context.Background(), []string{root}, false)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planNativeComposerFiles(filepath.Dir(root), files, true)
	if err != nil || len(plan.references) != 1 || len(plan.uploads) != 0 {
		t.Fatalf("explicit folder reference lost: %+v %v", plan, err)
	}
}

func TestComposerFileWorkerBoundsAndCancelsQueuedIntake(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 2)
	done := make(chan error, 2)
	for range 2 {
		go func() {
			done <- withComposerFileWorker(context.Background(), func() error { started <- struct{}{}; <-release; return nil })
		}()
	}
	defer func() {
		close(release)
		for range 2 {
			select {
			case err := <-done:
				if err != nil {
					t.Error(err)
				}
			case <-time.After(time.Second):
				t.Error("owned worker did not exit")
			}
		}
	}()
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers not started")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- withComposerFileWorker(ctx, func() error { t.Error("cancelled queue performed I/O"); return nil })
	}()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued intake did not cancel")
	}
	// Leave both occupied workers running until the deferred release: a third
	// queued request returned on cancellation without releasing somebody else's slot.
}

func TestComposerFilePlanUsesOriginalHostScope(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a b.txt")
	image := filepath.Join(root, "image.png")
	if err := os.WriteFile(file, []byte("note"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(image, []byte("\x89PNG\r\n\x1a\n"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := inspectComposerFiles(context.Background(), []string{file, image}, true)
	if err != nil {
		t.Fatal(err)
	}
	local, err := planNativeComposerFiles(root, files, true)
	if err != nil || len(local.references) != 1 || local.references[0] != "@\"a b.txt\"" || len(local.uploads) != 1 {
		t.Fatalf("local=%+v %v", local, err)
	}
	remote, err := planNativeComposerFiles(root, files, false)
	if err != nil || len(remote.references) != 0 || len(remote.uploads) != 2 {
		t.Fatalf("remote=%+v %v", remote, err)
	}
}
