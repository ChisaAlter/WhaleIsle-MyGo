//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"unsafe"
)

// ABI verified from Microsoft's Microsoft.Web.WebView2 1.0.3537.50
// build/native/include/WebView2.h. Frame7 (1.0.3240.44+) reports actual direct
// descendants; frame names, srcdoc contents and page messages are not identities.
const (
	previewWebView4AddFrameCreated    = 73
	previewWebView4RemoveFrameCreated = 74
	previewFrameCreatedGetFrame       = 3
	previewFrameAddDestroyed          = 8
	previewFrameRemoveDestroyed       = 9
	previewFrame2AddNavigation        = 11
	previewFrame2RemoveNavigation     = 12
	previewFrame7AddFrameCreated      = 30
	previewFrame7RemoveFrameCreated   = 31
)

var (
	iidPreviewIUnknown = guid("00000000-0000-0000-c000-000000000046")
	iidPreviewFrame2   = guid("7a6a5834-d185-4dbf-b63f-4a9bc43107d4")
	iidPreviewFrame7   = guid("3598cfa2-d85d-5a9f-9228-4dde1f59ec64")
)

// All state belongs to the main thread. Retained IUnknown identities prevent
// pointer reuse while a frame is live; destruction revokes admission. The
// unique viewer slot is never recycled, even after that viewer is destroyed.
type previewOpenUIFrames struct {
	w             *window
	webview4      uintptr
	rootToken     int64
	rootListening bool
	viewerSeen    bool
	viewer        *previewOpenUIFrame
	frames        map[uintptr]*previewOpenUIFrame
	closed        bool
}

type previewOpenUIFrame struct {
	identity, frame2, frame7 uintptr
	navigation, destroyed    int64
	children                 int64
	hasNavigation            bool
	hasDestroyed             bool
	hasChildren              bool
	parent                   *previewOpenUIFrame
	ready, started, closed   bool
}

func (w *window) EnableIsolatedOpenUIFrames() error {
	p := w.isolated
	if w.closed || w.destroying || !w.ready || w.webview == 0 || p == nil || p.closed || !w.opts.IsolatedPreview ||
		w.opts.LocalPDFViewerURL != "" || w.opts.LocalPDFURL != "" || w.opts.LocalHTMLURL != "" || w.opts.LocalMediaURL != "" ||
		p.frameStarted || p.openUIFrames != nil || w.URL() != "about:blank" {
		return errors.New("mygo: OpenUI frame policy requires an unused isolated blank preview")
	}
	v4 := queryInterface(w.webview, &iidICoreWebView2_4)
	if v4 == 0 {
		return errors.New("mygo: OpenUI preview requires frame creation tracking")
	}
	policy := &previewOpenUIFrames{w: w, webview4: v4, frames: make(map[uintptr]*previewOpenUIFrame)}
	// Publish the deny-by-default policy before installing any frame callbacks.
	// A failed installation cannot fall back to ordinary first-frame admission.
	p.openUIFrames = policy
	hr := withHandler(func(_, args uintptr) { policy.created(nil, args) }, func(h uintptr) uintptr {
		return comCall(v4, previewWebView4AddFrameCreated, h, uintptr(unsafe.Pointer(&policy.rootToken)))
	})
	if failed(hr) {
		policy.close()
		return hresultError("OpenUI root frame tracking", hr)
	}
	policy.rootListening = true
	return nil
}

func (p *previewOpenUIFrames) created(parent *previewOpenUIFrame, args uintptr) {
	if p.closed || p.w.closed || p.w.destroying {
		return
	}
	if parent == nil {
		if p.viewerSeen {
			return // All additional root frames stay cancelled by the global guard.
		}
		p.viewerSeen = true
	} else if parent.closed || !parent.ready || !parent.hasChildren {
		return
	}
	var frame uintptr
	hr := comCall(args, previewFrameCreatedGetFrame, uintptr(unsafe.Pointer(&frame)))
	if failed(hr) || frame == 0 {
		p.fail(errors.New("mygo: cannot identify OpenUI preview frame"))
		return
	}
	defer release(frame)
	f := &previewOpenUIFrame{identity: queryInterface(frame, &iidPreviewIUnknown), frame2: queryInterface(frame, &iidPreviewFrame2), frame7: queryInterface(frame, &iidPreviewFrame7), parent: parent}
	if f.identity == 0 || f.frame2 == 0 || (parent != nil && f.frame7 == 0) {
		f.close()
		p.fail(errors.New("mygo: OpenUI preview requires frame identity and navigation tracking"))
		return
	}
	if _, exists := p.frames[f.identity]; exists {
		f.close()
		p.fail(errors.New("mygo: duplicate OpenUI frame identity"))
		return
	}
	p.frames[f.identity] = f
	if parent == nil {
		p.viewer = f
	}
	add := func(obj uintptr, index int, token *int64, fn func(uintptr, uintptr)) bool {
		hr := withHandler(fn, func(h uintptr) uintptr { return comCall(obj, index, h, uintptr(unsafe.Pointer(token))) })
		if failed(hr) {
			p.fail(hresultError("OpenUI frame event registration", hr))
			return false
		}
		return true
	}
	f.hasDestroyed = add(f.frame2, previewFrameAddDestroyed, &f.destroyed, func(_, _ uintptr) {
		delete(p.frames, f.identity)
		f.close()
	})
	if !f.hasDestroyed {
		return
	}
	if f.frame7 != 0 {
		// An older runtime (or unavailable child subscription) keeps the
		// viewer-only policy. The Host reports HtmlView as unavailable.
		hr := withHandler(func(sender, args uintptr) {
			if f.matches(sender) {
				// Descendants are observed too, but only the viewer's immediate
				// children can pass navigationStarting. All deeper handlers deny.
				p.created(f, args)
			}
		}, func(h uintptr) uintptr {
			return comCall(f.frame7, previewFrame7AddFrameCreated, h, uintptr(unsafe.Pointer(&f.children)))
		})
		f.hasChildren = !failed(hr)
		if failed(hr) && parent != nil {
			p.fail(hresultError("HtmlView descendant guard", hr))
			return
		}
	}
	f.hasNavigation = add(f.frame2, previewFrame2AddNavigation, &f.navigation, func(sender, args uintptr) {
		p.navigationStarting(f, sender, args)
	})
	f.ready = f.hasNavigation
}

// IsolatedOpenUIFrameSupport is Host-only feature observation. Before the
// viewer is tracked it reports unknown; without Frame7 the basic viewer is
// still admitted, while every descendant remains cancelled by the global guard.
func (w *window) IsolatedOpenUIFrameSupport() (known, htmlView bool, err error) {
	if w.closed || w.destroying || w.isolated == nil || w.isolated.openUIFrames == nil || w.isolated.openUIFrames.closed {
		return false, false, errors.New("mygo: isolated OpenUI frame policy is unavailable")
	}
	p := w.isolated.openUIFrames
	if p.viewer == nil || !p.viewer.ready {
		return false, false, nil
	}
	if p.viewer.closed {
		return false, false, errors.New("mygo: OpenUI viewer frame was destroyed")
	}
	return true, p.viewer.hasChildren, nil
}

func (f *previewOpenUIFrame) matches(sender uintptr) bool {
	if f.closed || f.identity == 0 {
		return false
	}
	identity := queryInterface(sender, &iidPreviewIUnknown)
	defer release(identity)
	return identity != 0 && identity == f.identity
}

func (p *previewOpenUIFrames) navigationStarting(f *previewOpenUIFrame, sender, args uintptr) {
	// WebView2 documents that the global FrameNavigationStarting handlers run
	// first and share these args. Only this exact tracked frame can clear Cancel.
	if p.closed || p.w.closed || p.w.destroying || p.viewer == nil || !f.ready || f.started || !f.matches(sender) ||
		(f != p.viewer && (f.parent != p.viewer || p.viewer.closed || !p.viewer.started || !f.hasChildren)) {
		p.w.cancelPreviewNavigation(args)
		return
	}
	var uri uintptr
	hr := comCall(args, navStartingGetURI, uintptr(unsafe.Pointer(&uri)))
	target := takeWstr(uri)
	if failed(hr) || target != "about:srcdoc" {
		p.w.cancelPreviewNavigation(args)
		return
	}
	f.started = true
	p.w.previewMust(comCall(args, navStartingPutCancel, 0))
}

func (p *previewOpenUIFrames) fail(err error) {
	p.close()
	p.w.webViewFailed(err)
	p.w.destroy()
}

func (f *previewOpenUIFrame) close() {
	if f.closed {
		return
	}
	f.closed = true
	if f.hasNavigation {
		comCall(f.frame2, previewFrame2RemoveNavigation, uintptr(f.navigation))
	}
	if f.hasChildren {
		comCall(f.frame7, previewFrame7RemoveFrameCreated, uintptr(f.children))
	}
	if f.hasDestroyed {
		comCall(f.frame2, previewFrameRemoveDestroyed, uintptr(f.destroyed))
	}
	release(f.frame7)
	release(f.frame2)
	release(f.identity)
	f.frame7, f.frame2, f.identity = 0, 0, 0
}

func (p *previewOpenUIFrames) close() {
	if p.closed {
		return
	}
	p.closed = true
	if p.rootListening {
		comCall(p.webview4, previewWebView4RemoveFrameCreated, uintptr(p.rootToken))
	}
	for id, f := range p.frames {
		delete(p.frames, id)
		f.close()
	}
	release(p.webview4)
	p.webview4 = 0
}
