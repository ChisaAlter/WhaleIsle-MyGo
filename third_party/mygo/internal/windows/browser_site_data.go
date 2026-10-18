//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"unsafe"
)

// Verified against Microsoft's fixed Microsoft.Web.WebView2 1.0.3537.50 NuGet
// build/native/include/WebView2.h (SHA256
// 387a0524c1a4498db32756a6d3b1f4d43a90ac80cb921527f15d3dbf4a857459).
// Slots include IUnknown and all inherited methods in the SDK C vtables.
// https://www.nuget.org/packages/Microsoft.Web.WebView2/1.0.3537.50
// https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2profile2?view=webview2-1.0.3537.50
var (
	iidBrowserWebView13 = guid("f75f09a8-667e-4983-88d6-c8773f315e84")
	iidBrowserProfile2  = guid("fa740d4b-5eae-4344-a8ad-74be31925397")
)

const (
	browserWebViewGetProfile = 105 // ICoreWebView2_13::get_Profile(ICoreWebView2Profile**)
	browserProfileClearData  = 10  // ICoreWebView2Profile2::ClearBrowsingData(kinds, handler)
)

// browserSiteDataProfile returns one owned Profile2 reference, on the UI thread.
// It is obtained from this exact ready WebView, never the default environment.
func (w *window) browserSiteDataProfile() uintptr {
	if !w.BrowserCookieToolsAvailable() {
		return 0
	}
	v13 := queryInterface(w.webview, &iidBrowserWebView13)
	if v13 == 0 {
		return 0
	}
	defer release(v13)
	var profile uintptr
	hr := comCall(v13, browserWebViewGetProfile, uintptr(unsafe.Pointer(&profile)))
	if profile != 0 {
		defer release(profile)
	}
	if failed(hr) || profile == 0 {
		return 0
	}
	return queryInterface(profile, &iidBrowserProfile2)
}

func (w *window) BrowserSiteDataToolsAvailable() bool {
	profile := w.browserSiteDataProfile()
	if profile == 0 {
		return false
	}
	release(profile)
	return true
}

func (w *window) ClearBrowserSiteData(done func(error)) {
	profile := w.browserSiteDataProfile()
	if profile == 0 {
		done(errors.New("mygo: five-kind cleanup requires WebView2 Profile2 and a ready, verified Browser workspace profile"))
		return
	}
	defer release(profile)
	owner := w.browser
	w.nextCall++
	id := w.nextCall
	// Existing window cleanup completes pending calls even when WebView2 releases
	// the handler without invoking it after controller close. Late callbacks find
	// no call, and cannot update a replacement window or environment generation.
	w.calls[id] = func(_ string, err error) { done(err) }
	complete := func(err error) {
		cb := w.calls[id]
		delete(w.calls, id)
		if cb != nil {
			cb("", err)
		}
	}
	// Invoke(HRESULT) is slot 3 of ICoreWebView2ClearBrowsingDataCompletedHandler
	// (IID e9710a06-1d1d-49b2-8234-226f35846ae5). Reuse the existing shared
	// handler thunk; no per-call native callback or page-script bridge is added.
	hr := withHandler(func(hr, _ uintptr) {
		if w.calls[id] == nil {
			return
		}
		if failed(hr) {
			complete(hresultError("ClearBrowsingData (five site-data kinds)", hr))
			return
		}
		if w.browser != owner || !w.BrowserCookieToolsAvailable() {
			complete(errDestroyed)
			return
		}
		complete(nil)
	}, func(h uintptr) uintptr {
		return comCall(profile, browserProfileClearData, browserSiteDataKinds, h)
	})
	if failed(hr) {
		complete(hresultError("ClearBrowsingData (five site-data kinds)", hr))
	}
	// Unsupported kinds/runtime errors are returned unchanged. Never retry with
	// ALL_SITE/ALL_PROFILE, enumerate origins, or remove a profile directory.
}
