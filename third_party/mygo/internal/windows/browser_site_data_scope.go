package windows

// Microsoft.Web.WebView2 1.0.3537.50, build/native/include/WebView2.h:
// COREWEBVIEW2_BROWSING_DATA_KINDS. These are individual kinds, never the
// inclusive ALL_DOM_STORAGE, ALL_SITE or ALL_PROFILE selectors.
const (
	browserDataIndexedDB      = 0x0002
	browserDataLocalStorage   = 0x0004
	browserDataWebSQL         = 0x0008
	browserDataCookies        = 0x0040
	browserDataServiceWorkers = 0x8000
	browserSiteDataKinds      = browserDataCookies | browserDataLocalStorage | browserDataIndexedDB | browserDataWebSQL | browserDataServiceWorkers
)
