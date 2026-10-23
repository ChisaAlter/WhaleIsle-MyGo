//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// ABI values verified against Microsoft's Microsoft.Web.WebView2 1.0.3537.50
// NuGet build/native/include/WebView2.h, not inferred from interface names.
// _22 is required: the legacy resource filter misses opaque-origin iframes.
const (
	previewAddFrameNavigationStarting = 17
	previewAddSourceFilter            = 123
	previewSourceKindsAll             = 0xffffffff
	previewAddBrowserProcessExited    = 12
	previewRemoveBrowserProcessExited = 13
	previewGetUserDataFolder          = 15
	previewPutHostObjectsAllowed      = 16
)

var (
	iidPreviewWebView22    = guid("db75dfc7-a857-4632-a398-6969dde26c0a")
	iidPreviewEnvironment5 = guid("319e423d-e0d7-4b8d-9254-ae9475de9b17")
	iidPreviewEnvironment7 = guid("43c22296-3bbd-43a4-9c00-5c0df6dd29a2")
)

// No user-supplied directory is accepted. Cleanup uses the identity of the
// directory we created and waits for BrowserProcessExited, not merely Close.
type previewEnvironment struct {
	env, events       uintptr
	path              string
	identity          previewDirectoryIdentity
	identityHandle    syscall.Handle
	exitToken         int64
	closed            bool
	creating          bool
	exiting           bool
	controllerPending bool
	browserStarted    bool
	controllerSetup   bool
	exitPending       bool
	frameStarted      bool
	openUIFrames      *previewOpenUIFrames
}

func (w *window) environment() uintptr {
	if w.opts.IsolatedPreview || w.opts.TransientProfile {
		if w.isolated == nil {
			return 0
		}
		return w.isolated.env
	}
	if w.opts.BrowserProfile {
		if w.browser == nil {
			return 0
		}
		return w.browser.env
	}
	return w.b.env
}

func (w *window) startPreviewEnvironment() {
	// WebView2 environment overrides can defeat the explicit UDF or sandbox.
	// Do not modify process-global overrides and do not fall back to shared state.
	for _, name := range []string{"WEBVIEW2_USER_DATA_FOLDER", "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "WEBVIEW2_PIPE_FOR_SCRIPT_DEBUGGER", "WEBVIEW2_WAIT_FOR_SCRIPT_DEBUGGER"} {
		if os.Getenv(name) != "" {
			w.webViewFailed(fmt.Errorf("mygo: isolated preview rejects %s override", name))
			return
		}
	}
	dir, err := os.MkdirTemp("", "mygo-preview-")
	if err != nil {
		w.webViewFailed(err)
		return
	}
	p, err := newPreviewEnvironment(dir)
	if err != nil {
		w.webViewFailed(err)
		return
	}
	w.isolated = p
	p.creating = true
	err = createEnvironment(dir, func(env uintptr, err error) {
		// Environment queries and event registration can pump COM callbacks.
		// Keep initialization ownership until the full callback has settled.
		defer func() {
			p.creating = false
			if p.closed || w.closed || w.destroying {
				p.close()
			}
		}()
		if err != nil {
			p.removeOwnedDirectory()
			w.webViewFailed(err)
			return
		}
		p.env = env
		if w.closed || w.destroying {
			p.close()
			return
		}
		// Verify before subscribing: without a controller there may never be
		// a browser process whose exit can release an event subscription.
		// Registry policy can also override the UDF. Verify the effective folder.
		env7 := queryInterface(env, &iidPreviewEnvironment7)
		if env7 == 0 {
			p.close()
			w.webViewFailed(errors.New("mygo: isolated preview cannot verify its user data folder"))
			return
		}
		var actual uintptr
		hr := comCall(env7, previewGetUserDataFolder, uintptr(unsafe.Pointer(&actual)))
		release(env7)
		actualPath := takeWstr(actual)
		if failed(hr) || !strings.EqualFold(filepath.Clean(actualPath), filepath.Clean(dir)) {
			p.close()
			w.webViewFailed(errors.New("mygo: isolated preview user data folder was overridden"))
			return
		}
		if w.closed || w.destroying {
			p.close()
			return
		}
		p.events = queryInterface(env, &iidPreviewEnvironment5)
		if p.events == 0 {
			p.close()
			w.webViewFailed(errors.New("mygo: isolated preview requires WebView2 Environment5"))
			return
		}
		if p.closed || w.closed || w.destroying {
			p.close()
			return
		}
		hr = withHandler(func(_, _ uintptr) { w.previewBrowserExited(p) }, func(h uintptr) uintptr {
			return comCall(p.events, previewAddBrowserProcessExited, h, uintptr(unsafe.Pointer(&p.exitToken)))
		})
		if failed(hr) {
			release(p.events)
			p.events = 0
			p.close()
			w.webViewFailed(hresultError("isolated preview exit subscription", hr))
			return
		}
		if p.closed || w.closed || w.destroying {
			p.close()
			return
		}
		w.createWebView()
	})
	if err != nil {
		p.creating = false
		p.removeOwnedDirectory()
		w.webViewFailed(err)
	}
}

// A private environment must never restart after its owned folder is removed.
// Make an unexpected browser exit terminal before disposing of the directory.
func (w *window) previewBrowserExited(p *previewEnvironment) {
	// A COM call in setup may pump this event. Finish that call stack before
	// tearing down the controller/interfaces it is still using.
	if p.controllerSetup {
		p.exitPending = true
		p.closed = true
		return
	}
	if p.exiting {
		return
	}
	// Publish terminal state before calling COM, which can dispatch another
	// main-thread callback while unregistering or releasing an interface.
	p.exiting = true
	p.closed = true
	p.browserStarted = false
	p.unsubscribeExit()
	if !w.closed && w.b != nil && w.b.windows[w.hwnd] == w {
		procKillTimer.Call(w.hwnd, timerWebView)
	}
	if !w.closed && !w.destroying {
		w.webViewFailed(errors.New("mygo: private preview browser exited"))
		w.destroy()
	}
	p.close()
	p.removeOwnedDirectory()
	p.exiting = false
}

func (p *previewEnvironment) unsubscribeExit() {
	events := p.events
	p.events = 0
	if events != 0 {
		comCall(events, previewRemoveBrowserProcessExited, uintptr(p.exitToken))
		release(events)
	}
}

func (p *previewEnvironment) close() {
	p.closed = true
	if p.creating {
		return
	}
	env := p.env
	p.env = 0
	if env != 0 {
		release(env)
	}
	// An aborted controller request can finish without ever starting a
	// browser. After its completion there is no exit event to wait for.
	if !p.browserStarted && !p.controllerPending && !p.creating && !p.exiting {
		p.unsubscribeExit()
	}
	if p.events == 0 && !p.creating && !p.controllerPending && !p.exiting {
		p.releaseDirectoryIdentity()
	}
	// Started browsers retain the exit subscription and original identity.
	// Never remove a potentially live browser's folder on a timer.
}
func (p *previewEnvironment) removeOwnedDirectory() {
	if p.identityHandle == 0 || p.identityHandle == syscall.InvalidHandle {
		return
	}
	defer p.releaseDirectoryIdentity()
	cleanup, err := lockPreviewDirectory(p.path, p.identity)
	if err != nil {
		return
	}
	defer cleanup.close()
	// On any error retain the folder; never broaden or retry by an old path.
	_ = cleanup.remove()
}

func (w *window) installPreviewPolicy() error {
	if w.webview == 0 || w.settings == 0 || w.environment() == 0 {
		return errors.New("mygo: missing isolated preview context")
	}
	v22 := queryInterface(w.webview, &iidPreviewWebView22)
	if v22 == 0 {
		return errors.New("mygo: isolated preview requires all-source resource filtering (WebView2 1.0.2365.46+)")
	}
	defer release(v22)
	v4 := queryInterface(w.webview, &iidICoreWebView2_4)
	if v4 == 0 {
		return errors.New("mygo: isolated preview requires download cancellation")
	}
	defer release(v4)
	var token int64
	add := func(obj uintptr, index int, fn func(uintptr, uintptr)) error {
		hr := withHandler(fn, func(h uintptr) uintptr { return comCall(obj, index, h, uintptr(unsafe.Pointer(&token))) })
		if failed(hr) {
			return hresultError("isolated preview event registration", hr)
		}
		return nil
	}
	if hr := comCall(w.settings, setPutIsScriptEnabled, boolArg(w.opts.LocalPDFURL == "" && w.opts.LocalHTMLURL == "" && w.opts.LocalMediaURL == "")); failed(hr) {
		return hresultError("isolated preview script policy", hr)
	}
	for _, setting := range []int{setPutIsWebMessageEnabled, setPutAreDefaultScriptDialogsEnabled, setPutAreDevToolsEnabled, previewPutHostObjectsAllowed, setPutAreDefaultContextMenusEnabled} {
		if hr := comCall(w.settings, setting, 0); failed(hr) {
			return hresultError("isolated preview settings", hr)
		}
	}
	if err := add(w.webview, wvAddWebResourceRequested, func(_, args uintptr) { w.denyPreviewResource(args) }); err != nil {
		return err
	}
	if hr := comCall(v22, previewAddSourceFilter, uintptr(unsafe.Pointer(u16("*"))), 0, previewSourceKindsAll); failed(hr) {
		return hresultError("isolated preview all-source resource filter", hr)
	}
	if w.opts.LocalPDFViewerURL != "" {
		// ICoreWebView2::add_SourceChanged is slot 11. Same-document hash/
		// history changes do not emit ContentLoading. No page-to-host RPC.
		if err := add(w.webview, 11, func(_, args uintptr) {
			var newDocument int32
			if failed(comCall(args, 3, uintptr(unsafe.Pointer(&newDocument)))) {
				w.destroy()
				return
			}
			if newDocument == 0 {
				w.h.NavigationCommitted(w.URL())
				w.h.LoadFinished()
			}
		}); err != nil {
			return err
		}
	}
	if err := add(w.webview, wvAddNavigationStarting, w.navigationStarting); err != nil {
		return err
	}
	if err := add(w.webview, previewAddFrameNavigationStarting, func(_, args uintptr) {
		if w.isolated.openUIFrames != nil {
			// Deny first, including unknown/deeper frames. WebView2 invokes this
			// shared event before the specific frame's NavigationStarting event;
			// only our tracked viewer/direct child may clear that cancellation.
			w.cancelPreviewNavigation(args)
			return
		}
		var uri uintptr
		hr := comCall(args, navStartingGetURI, uintptr(unsafe.Pointer(&uri)))
		target := takeWstr(uri)
		if !failed(hr) && platform.LocalPDFAllowed(w.opts.LocalPDFURL, target) {
			return
		}
		if !failed(hr) && w.opts.LocalPDFViewerURL == "" && w.opts.LocalPDFURL == "" && w.opts.LocalMediaURL == "" && target == "about:srcdoc" && !w.isolated.frameStarted {
			w.isolated.frameStarted = true
			return
		}
		w.cancelPreviewNavigation(args)
	}); err != nil {
		return err
	}
	if err := add(w.webview, wvAddNewWindowRequested, w.newWindowRequested); err != nil {
		return err
	}
	if err := add(w.webview, wvAddPermissionRequested, w.permissionRequested); err != nil {
		return err
	}
	return add(v4, wv4AddDownloadStarting, w.downloadStarting)
}

func (w *window) previewMust(hr uintptr) {
	if failed(hr) {
		w.webViewFailed(hresultError("isolated preview policy", hr))
		w.destroy()
	}
}
func (w *window) cancelPreviewNavigation(args uintptr) {
	w.previewMust(comCall(args, navStartingPutCancel, 1))
}
func (w *window) denyPreviewResource(args uintptr) {
	if w.closed || w.environment() == 0 {
		return
	}
	if w.opts.LocalPDFViewerURL != "" || w.opts.LocalPDFURL != "" || w.opts.LocalHTMLURL != "" || w.opts.LocalMediaURL != "" {
		var req uintptr
		if !failed(comCall(args, resReqGetRequest, uintptr(unsafe.Pointer(&req)))) && req != 0 {
			defer release(req)
			var uri, method uintptr
			uriHR := comCall(req, reqGetURI, uintptr(unsafe.Pointer(&uri)))
			methodHR := comCall(req, reqGetMethod, uintptr(unsafe.Pointer(&method)))
			target, verb := takeWstr(uri), takeWstr(method)
			pdfAllowed := platform.LocalPDFAllowed(w.opts.LocalPDFURL, target)
			// Resource requests have no fragment allowance: HTML grants only one
			// GET/HEAD resource, never arbitrary same-origin paths or redirects.
			htmlAllowed := target == w.opts.LocalHTMLURL && platform.LocalHTMLAllowed(w.opts.LocalHTMLURL, target)
			if !failed(uriHR) && !failed(methodHR) && (verb == "GET" || verb == "HEAD") && (platform.LocalPDFViewerResourceAllowed(w.opts.LocalPDFViewerURL, target) || pdfAllowed || htmlAllowed || platform.LocalMediaResourceAllowed(w.opts.LocalMediaURL, target)) {
				return
			}
		}
	}
	// data: images are embedded by the Host and normally do not issue resource
	// requests. Any event that does arrive is denied, including custom schemes,
	// file:, loopback, redirects, subframes, service and shared workers.
	var response uintptr
	hr := comCall(w.environment(), envCreateWebResourceResponse, 0, http.StatusForbidden, uintptr(unsafe.Pointer(u16("Forbidden"))), uintptr(unsafe.Pointer(u16("Content-Length: 0\r\nCache-Control: no-store\r\n"))), uintptr(unsafe.Pointer(&response)))
	if failed(hr) || response == 0 {
		w.webViewFailed(errors.New("mygo: cannot deny isolated preview resource"))
		w.destroy()
		return
	}
	defer release(response)
	w.previewMust(comCall(args, resReqPutResponse, response))
}
