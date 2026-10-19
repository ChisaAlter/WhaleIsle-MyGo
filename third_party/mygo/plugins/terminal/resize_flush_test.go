package terminal

import (
	"io"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
)

type resizeFlushWire struct{ resized, closed int }

func (*resizeFlushWire) Read([]byte) (int, error)    { return 0, io.EOF }
func (*resizeFlushWire) Write(p []byte) (int, error) { return len(p), nil }
func (w *resizeFlushWire) Close() error              { w.closed++; return nil }
func (w *resizeFlushWire) Resize(int, int) error     { w.resized++; return nil }

// No Load/New, native emulator, local process or Host connection is used.
func TestFlushPTYResizeRetiresPendingTimerWithoutResizingRemoteConnection(t *testing.T) {
	wire := &resizeFlushWire{}
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	term := &Terminal{src: wire, ptyResizeTimer: timer, ptyResizeVersion: 4, size: gridSize{100, 30, 8, 16}}
	term.FlushPTYResize()
	if term.ptyResizeTimer != nil || term.ptyResizeVersion != 5 || timer.Stop() || wire.resized != 0 {
		t.Fatalf("flush timer=%v version=%d remote=%d", term.ptyResizeTimer, term.ptyResizeVersion, wire.resized)
	}
	term.FlushPTYResize()
	if term.ptyResizeVersion != 5 || wire.resized != 0 {
		t.Fatal("second flush reissued geometry")
	}
}

func TestCloseRetiresPendingPTYGeometryBeforeRendererDisposal(t *testing.T) {
	wire := &resizeFlushWire{}
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	// Zero handles make Free a no-op, so this exercises the production close
	// lifecycle without loading Ghostty or allocating a renderer/program.
	term := &Terminal{src: wire, term: &vt.Terminal{}, rs: &vt.RenderState{}, ptyResizeTimer: timer, ptyResizeVersion: 9}
	if err := term.Close(); err != nil {
		t.Fatal(err)
	}
	if !term.closed || term.ptyResizeTimer != nil || term.ptyResizeVersion != 10 || wire.closed != 1 || wire.resized != 0 || timer.Stop() {
		t.Fatalf("closed=%v timer=%v version=%d wire=%+v", term.closed, term.ptyResizeTimer, term.ptyResizeVersion, wire)
	}
	if err := term.Close(); err != nil || wire.closed != 1 {
		t.Fatalf("close repeated: err=%v wire=%+v", err, wire)
	}
}
