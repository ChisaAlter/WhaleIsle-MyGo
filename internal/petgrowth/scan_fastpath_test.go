package petgrowth

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanFastPathEmptyAndUnchangedSessionsSkipNode(t *testing.T) {
	resetScanCache()
	defer resetScanCache()

	missingHome := t.TempDir()
	if _, err := Scan(context.Background(), "nonexistent-node-binary", missingHome); err == nil {
		t.Fatal("missing sessions directory should fail closed without spawning node")
	}

	emptyHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(emptyHome, "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	emptySamples, err := Scan(context.Background(), "nonexistent-node-binary", emptyHome)
	if err != nil || len(emptySamples) != 0 {
		t.Fatalf("empty sessions directory should succeed without spawning node: samples=%v err=%v", emptySamples, err)
	}

	node := scanTestNode(t)
	resetScanCache()
	home := t.TempDir()
	sessionDir := filepath.Join(home, "sessions", "ws", "session-a")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(sessionDir, "session.v4.jsonl.zstd")
	writeLogFrame(t, node, logFile, `{"type":"session/header","seq":0}`, usageEvent(1, 1, 1, 15))

	first, err := Scan(context.Background(), node, home)
	if err != nil || len(first) != 1 {
		t.Fatalf("initial scan failed: samples=%v err=%v", first, err)
	}

	// Identical logs must hit the pure Go cache without invoking node.exe.
	cached, err := Scan(context.Background(), "nonexistent-node-binary", home)
	if err != nil || !reflect.DeepEqual(cached, first) {
		t.Fatalf("unchanged scan did not reuse Go fast-path cache: got=%v want=%v err=%v", cached, first, err)
	}

	// Appending a new frame invalidates the signature and requires Node decode.
	writeLogFrame(t, node, logFile, usageEvent(2, 2, 1, 9))
	if _, err := Scan(context.Background(), "nonexistent-node-binary", home); err == nil {
		t.Fatal("modified log unexpectedly stayed on fast-path cache")
	}
	updated, err := Scan(context.Background(), node, home)
	if err != nil || len(updated) != 2 {
		t.Fatalf("re-scan after append failed: samples=%v err=%v", updated, err)
	}
}
