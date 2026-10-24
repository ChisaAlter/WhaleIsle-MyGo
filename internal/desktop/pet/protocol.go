package pet

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
)

// The whale-pet scheme must exist before any WebView2 environment or window is
// created: mygo builds each window's custom-scheme filter list from
// Protocol.schemes() when the window is created (third_party/mygo/window.go:510,
// internal/windows/webview.go:312), and the platform environment is created for
// the first window. Registering from New() -- which runs inside the main
// window's WhenReady callback -- is too late: the pet page then navigates to
// http://whale-pet.localhost/ without a filter and WebView2 aborts the
// navigation (COREWEBVIEW2_WEB_ERROR_STATUS 14, round154 §7).
//
// The package therefore registers a stable dispatcher at init time (before
// main, before mygo.App.Run) and New() only publishes the companion it created.
// The dispatcher forwards to whatever companion is current, so replacing or
// closing the pet never needs another registration, and requests without a
// companion are a plain 404 instead of a broken scheme.
const PetScheme = "whale-pet"

var (
	petDispatcher atomic.Pointer[Companion]
	petSchemeOnce sync.Once
	petSchemeErr  error
)

// RegisterProtocol installs the whale-pet dispatcher. It is idempotent and runs
// from init, so an explicit call from a program that links this package is
// never required; tests may call it to observe the error.
func RegisterProtocol() error {
	petSchemeOnce.Do(func() {
		petSchemeErr = mygo.Protocol.Handle(PetScheme, petProtocol{})
	})
	return petSchemeErr
}

// Current reports the companion serving the scheme, or nil.
func Current() *Companion { return petDispatcher.Load() }

// setCurrent publishes the companion that requests are dispatched to.
func setCurrent(p *Companion) { petDispatcher.Store(p) }

func init() { _ = RegisterProtocol() }

// petProtocol is the stable handler registered with mygo. It never changes, so
// the scheme survives companion replacement and needs no Unhandle.
type petProtocol struct{}

func (petProtocol) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p := petDispatcher.Load(); p != nil {
		p.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}
