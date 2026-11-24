package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func hostLogFiles(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "host-*.log"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestHostLogSinkWritesAndFlushesOnStop(t *testing.T) {
	dir := t.TempDir()
	sink := newHostLogSink(dir, 4242)
	if sink == nil {
		t.Fatal("no sink")
	}
	if base := filepath.Base(sink.path); !strings.HasPrefix(base, "host-") || !strings.HasSuffix(base, "-4242.log") {
		t.Fatalf("name %q", base)
	}
	sink.add("first line")
	sink.add("second line")
	sink.close()
	raw, err := os.ReadFile(sink.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "first line\nsecond line\n" {
		t.Fatalf("contents %q", raw)
	}
}

func TestHostLogSinkPrunesToNewest(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < hostLogKeep+3; i++ {
		name := fmt.Sprintf("host-20260101-%02d0000-%d.log", i, i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sink := newHostLogSink(dir, 9)
	if sink == nil {
		t.Fatal("no sink")
	}
	sink.close()
	files := hostLogFiles(t, dir)
	if len(files) != hostLogKeep {
		t.Fatalf("kept %d files, want %d", len(files), hostLogKeep)
	}
	for _, f := range files {
		if filepath.Base(f) == "host-20260101-000000-0.log" {
			t.Fatal("oldest file was not pruned")
		}
	}
	if filepath.Base(sink.path) != "host-"+time.Now().Format("20060102-150405")+"-9.log" {
		t.Fatalf("new file %q", sink.path)
	}
}

func TestHostLogSinkCapWritesMarkerOnce(t *testing.T) {
	dir := t.TempDir()
	const cap = 200
	sink := openHostLogSink(dir, 1, cap)
	if sink == nil {
		t.Fatal("no sink")
	}
	line := strings.Repeat("x", 60)
	for i := 0; i < 10; i++ {
		sink.add(line)
	}
	sink.close()
	raw, err := os.ReadFile(sink.path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "log truncated at 20 MiB\n") {
		t.Fatalf("no truncation marker in %q", text)
	}
	if strings.Count(text, "log truncated") != 1 {
		t.Fatalf("marker repeated in %q", text)
	}
	// Output stops at the cap; the marker itself may pass it by its own size.
	if int64(len(raw)) > cap+int64(len("log truncated at 20 MiB\n")) {
		t.Fatalf("file grew past cap: %d > %d", len(raw), cap)
	}
}

func TestHostLogSinkDropAccounting(t *testing.T) {
	dir := t.TempDir()
	sink := openHostLogSink(dir, 1, hostLogCapBytes)
	if sink == nil {
		t.Fatal("no sink")
	}
	// Queue-full drops only bump the counter (racing the drain makes real
	// drops nondeterministic); seeding it exercises the marker path directly.
	sink.dropped.Add(3)
	sink.close()
	raw, err := os.ReadFile(sink.path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "3 lines dropped") {
		t.Fatalf("no drop marker in %q", raw)
	}
}

func TestHostLogSinkNilOnBadDir(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if sink := newHostLogSink(filepath.Join(blocked, "logs"), 1); sink != nil {
		sink.close()
		t.Fatal("sink opened under a non-directory")
	}
}
