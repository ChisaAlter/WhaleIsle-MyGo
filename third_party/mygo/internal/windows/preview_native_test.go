//go:build windows && (amd64 || arm64)

package windows

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// This is an opt-in real Windows/WebView2 fixture. It uses an empty temporary
// profile tree and a fixture-only loopback HTML server, never an app profile.
func TestPreviewWebViewNativeExitCleanup(t *testing.T) {
	if os.Getenv("MYGO_PREVIEW_NATIVE_TEST") != "1" {
		t.Skip("set MYGO_PREVIEW_NATIVE_TEST=1 in a Windows desktop session")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previousThread, previousBackend := mainThreadID, theBackend
	mainThreadID = currentThreadID()
	defer func() { mainThreadID = previousThread; theBackend = previousBackend }()
	root := t.TempDir()
	t.Setenv("TEMP", root)
	t.Setenv("TMP", root)
	capability := "/" + strings.Repeat("1", 64) + "/document.html"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != capability {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<!doctype html><title>Preview lifetime fixture</title><p>temporary native fixture</p>")
	}))
	defer server.Close()
	b := New()
	if err := b.Init(previewNativeApp{}, platform.AppOptions{Name: "MyGoPreviewLifecycleFixture"}); err != nil {
		t.Fatal(err)
	}
	defer procDestroyWindow.Call(b.appHwnd)
	wait := func(label string, ready func() bool) {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		for !ready() {
			if time.Now().After(deadline) {
				t.Fatalf("%s did not settle", label)
			}
			b.Step()
		}
	}
	for _, mode := range []string{"normal-exit", "close-during-creation", "first-controller-fails"} {
		// Keep native calls on the locked COM thread; t.Run would move them.
		func() {
			t.Logf("native case: %s", mode)
			h := &previewNativeWindow{}
			url := server.URL + capability
			opts := &platform.WindowOptions{IsolatedPreview: true, LocalHTMLURL: url, Title: "MyGo private preview lifetime fixture", Width: 320, Height: 180, Closable: true, Movable: true}
			if mode == "first-controller-fails" {
				TestFailWebViews(1)
			}
			defer TestFailWebViews(0)
			created, err := b.NewWindow(opts, h)
			if err != nil {
				t.Fatal(err)
			}
			w := created.(*window)
			defer func() {
				w.destroy()
				deadline := time.Now().Add(10 * time.Second)
				for w.isolated != nil && (w.isolated.creating || w.isolated.events != 0) && time.Now().Before(deadline) {
					b.Step()
				}
			}()
			p := w.isolated
			if p == nil {
				t.Fatalf("private environment missing: %v", w.webViewErr)
			}
			originalHandle := p.identityHandle
			path := p.path
			if filepath.Dir(path) != root {
				t.Fatalf("fixture profile escaped temporary root: %q", path)
			}
			if mode == "close-during-creation" {
				environmentPending := p.creating
				controllerPending := p.controllerPending
				if !environmentPending && !controllerPending {
					t.Fatal("no pending creation phase exercised")
				}
				w.destroy()
				if (p.creating || p.controllerPending) && p.identityHandle == 0 {
					t.Fatal("early close released still-pending original")
				}
				wait("late environment completion", func() bool {
					return !p.creating && !p.controllerPending && p.env == 0 && p.events == 0 && p.identityHandle == 0
				})
				if p.identity.valid {
					t.Fatal("late completion retained stale identity")
				}
				t.Logf("immediate close settled; environmentPending=%t controllerPending=%t identity released", environmentPending, controllerPending)
			} else {
				wait("private controller creation", func() bool { return w.ready || w.closed || w.webViewErr != nil })
				if w.ready {
					if p.events == 0 || p.identityHandle == 0 {
						t.Fatal("ready browser lost cleanup owner")
					}
					w.LoadURL(url)
					wait("local HTML load", func() bool { return h.loaded || h.loadErr != "" || w.closed })
					if !h.loaded {
						t.Fatalf("fixture HTML did not load: %s %v", h.loadErr, w.webViewErr)
					}
					if mode == "first-controller-fails" && w.webViewFails == 0 {
						t.Fatal("controller failure not exercised")
					}
					w.destroy()
				} else if mode != "first-controller-fails" {
					t.Fatalf("private browser failed: %v", w.webViewErr)
				}
				wait("BrowserProcessExited cleanup", func() bool { return p.events == 0 && p.env == 0 && p.identityHandle == 0 })
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("exited original profile retained: %v", err)
				}
				// Drain a queued retry; a terminal private environment cannot run it.
				w.createWebView()
				w.creationFailed(eFail)
				t.Logf("browser exit cleaned original; controller failures=%d terminal=%t", w.webViewFails, p.closed)
			}
			var info syscall.ByHandleFileInformation
			if err := syscall.GetFileInformationByHandle(originalHandle, &info); err == nil {
				t.Fatal("original directory handle leaked")
			}
			if !w.closed {
				t.Fatal("native fixture window still live")
			}
		}()
	}
}

type previewNativeApp struct{ platform.AppHandler }

func (previewNativeApp) Ready()            {}
func (previewNativeApp) Dispatch()         {}
func (previewNativeApp) ThemeChanged()     {}
func (previewNativeApp) DisplaysChanged()  {}
func (previewNativeApp) PowerEvent(string) {}
func (previewNativeApp) DidBecomeActive()  {}
func (previewNativeApp) DidResignActive()  {}

type previewNativeWindow struct {
	platform.WindowHandler
	loaded  bool
	loadErr string
}

func (*previewNativeWindow) ShouldClose() bool                     { return true }
func (*previewNativeWindow) Closed()                               {}
func (*previewNativeWindow) Focused()                              {}
func (*previewNativeWindow) Blurred()                              {}
func (*previewNativeWindow) Resized()                              {}
func (*previewNativeWindow) Moved()                                {}
func (*previewNativeWindow) Minimized()                            {}
func (*previewNativeWindow) Restored()                             {}
func (*previewNativeWindow) Maximized()                            {}
func (*previewNativeWindow) Unmaximized()                          {}
func (*previewNativeWindow) TitleBarChanged()                      {}
func (*previewNativeWindow) TitleChanged(string)                   {}
func (*previewNativeWindow) NavigationCommitted(string)            {}
func (*previewNativeWindow) NavigationStarted(string)              {}
func (*previewNativeWindow) WillNavigate(platform.Navigation) bool { return true }
func (h *previewNativeWindow) LoadFinished()                       { h.loaded = true }
func (h *previewNativeWindow) LoadFailed(_ string, _ int, description string) {
	h.loadErr = description
}
func (h *previewNativeWindow) RenderProcessGone(reason string) { h.loadErr = reason }
