package mygo

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

type petDragWindow struct {
	platform.Window
	drags int
}

func (w *petDragWindow) StartDrag() { w.drags++ }

func TestWindowControlsOnlyDragIsolation(t *testing.T) {
	native := &petDragWindow{}
	w := &Window{secret: "test:", windowControlsOnly: true, native: native, trusted: true, trustedOrigins: []string{"*"}}
	w.handleMessage(`test:{"t":"drag"}`)
	if native.drags != 1 {
		t.Fatal("native drag was not delivered")
	}
	// Any accidentally accepted message below reaches a nil embedded native
	// method, or a native call goroutine. Neither may run for this page.
	w.handleMessage(`test:{"t":"call","id":1,"m":"mygo:window.Close","a":[]}`)
	w.handleMessage(`test:{"t":"dblclick"}`)
	w.handleMessage(`test:{"t":"drop","x":1,"y":2}`)
	if w.isTrusted("mygo://app") {
		t.Fatal("restricted page trusted")
	}
	w.enqueue(message{head: []byte("private app event")}, true)
	if len(w.held) != 0 || len(w.outbox) != 0 {
		t.Fatal("application data exposed")
	}
	normal := &Window{}
	if !normal.isTrusted("about:blank") {
		t.Fatal("normal trust changed")
	}
}
