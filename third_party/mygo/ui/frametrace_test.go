package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The trace row ends with the present phase's three sub-laps, after every
// column the report already reads.
func TestFrameTracePresentSubLaps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frames.csv")
	tr := newFrameTrace(path)
	if tr == nil {
		t.Fatal("no trace writer")
	}
	defer tr.close()
	tt := coreNewTester(func(c *context) { coreText(c, "x") }, 100, 50)

	f := newFrameStats(0)
	f.begin(tt.rt)
	f.presentUpload = time.Millisecond
	f.presentDraw = 1500 * time.Microsecond
	f.presentSwap = 500 * time.Microsecond
	tr.mark("step-one")
	tr.frame(tt.rt, f, 4*time.Millisecond)
	_ = tr.w.Flush()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 4 {
		t.Fatalf("trace wrote %d lines, want header+cal+mark+frame: %q", len(lines), raw)
	}
	header := strings.Split(lines[0], ",")
	if got := len(header); got != 23 {
		t.Fatalf("header has %d columns, want 23: %q", got, lines[0])
	}
	if got := strings.Join(header[20:], ","); got != "present_upload_ns,present_draw_ns,present_swap_ns" {
		t.Fatalf("header tail %q", got)
	}
	for i, line := range lines[1:] {
		if got := len(strings.Split(line, ",")); got != 23 {
			t.Fatalf("row %d has %d columns, want 23: %q", i+1, got, line)
		}
	}
	cells := strings.Split(lines[3], ",")
	if cells[20] != "1000000" || cells[21] != "1500000" || cells[22] != "500000" {
		t.Fatalf("present sub-lap cells %q", strings.Join(cells[20:], ","))
	}
}

// A host that reports no sub-laps (the memory host, every non-D3D path)
// writes zeros.
func TestFrameTracePresentSubLapsZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frames.csv")
	tr := newFrameTrace(path)
	if tr == nil {
		t.Fatal("no trace writer")
	}
	defer tr.close()
	tt := coreNewTester(func(c *context) { coreText(c, "x") }, 100, 50)
	f := newFrameStats(0)
	f.begin(tt.rt)
	tr.frame(tt.rt, f, time.Millisecond)
	_ = tr.w.Flush()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	cells := strings.Split(lines[len(lines)-1], ",")
	if cells[20] != "0" || cells[21] != "0" || cells[22] != "0" {
		t.Fatalf("unreported sub-laps must write zeros: %q", strings.Join(cells[20:], ","))
	}
}
