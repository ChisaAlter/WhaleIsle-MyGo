//go:build linux && (amd64 || arm64)

package linux

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/internal/platform"
)

// All objects and references below belong to the GTK main thread. A partition
// shares one context while its tabs are alive; no operation uses the default
// context or deletes profile directories.
type browserProfile struct {
	key              string
	context, manager ptr
	refs             int
	schemes          map[string]bool
}

var (
	browserManagerNew                        func(name *byte, value *byte, name2 *byte, value2 *byte, end ptr) ptr
	browserContextNew                        func(manager ptr) ptr
	browserWebViewType                       func() ptr
	browserObjectNew                         func(typ ptr, name *byte, value ptr, name2 *byte, value2 ptr, end ptr) ptr
	browserGetContext                        func(web ptr) ptr
	browserGetCookieManager                  func(manager ptr) ptr
	browserCookieStorage                     func(manager ptr, path *byte, format int32)
	browserMajorVersion, browserMinorVersion func() uint32
)

func loadBrowserProfileAPI(lib ptr) {
	bind(lib, &browserManagerNew, "webkit_website_data_manager_new")
	bind(lib, &browserContextNew, "webkit_web_context_new_with_website_data_manager")
	bind(lib, &browserWebViewType, "webkit_web_view_get_type")
	bind(libGObject, &browserObjectNew, "g_object_new")
	bind(lib, &browserGetContext, "webkit_web_view_get_context")
	bind(lib, &browserGetCookieManager, "webkit_website_data_manager_get_cookie_manager")
	bind(lib, &browserCookieStorage, "webkit_cookie_manager_set_persistent_storage")
	bind(lib, &browserMajorVersion, "webkit_get_major_version")
	bind(lib, &browserMinorVersion, "webkit_get_minor_version")
}

func browserVersionAtLeast(major, minor uint32) bool {
	return browserMajorVersion != nil && browserMinorVersion != nil &&
		(browserMajorVersion() > major || browserMajorVersion() == major && browserMinorVersion() >= minor)
}

func (b *Backend) acquireBrowserProfile(key string) (*browserProfile, error) {
	if !platform.BrowserPartitionValid(key) {
		return nil, errors.New("mygo: invalid Browser partition key")
	}
	if browserManagerNew == nil || browserContextNew == nil || browserObjectNew == nil || browserWebViewType == nil || browserGetContext == nil || browserGetCookieManager == nil || browserCookieStorage == nil || !browserVersionAtLeast(2, 16) {
		return nil, errors.New("mygo: Browser profiles require WebKitGTK 2.16+ data-manager APIs")
	}
	if p := b.browserProfiles[key]; p != nil {
		p.refs++
		return p, nil
	}
	// Hash the application name, so it cannot add path components. Preserve the
	// full partition key. Neither paths nor user data are migrated from default.
	appID := fmt.Sprintf("%x", sha256.Sum256([]byte(b.name)))
	base := xdgDir("XDG_DATA_HOME", ".local/share")
	if !filepath.IsAbs(base) {
		return nil, errors.New("mygo: Browser profiles require an absolute user data directory")
	}
	root := filepath.Join(base, "mygo", appID, "browser-workspaces", key)
	dataDir, cacheDir := filepath.Join(root, "data"), filepath.Join(root, "cache")
	for _, path := range []string{dataDir, cacheDir} {
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, fmt.Errorf("mygo: create Browser partition: %w", err)
		}
	}
	manager := browserManagerNew(cs("base-data-directory"), cs(dataDir), cs("base-cache-directory"), cs(cacheDir), 0)
	if manager == 0 {
		return nil, errors.New("mygo: cannot create Browser website data manager")
	}
	cookies := browserGetCookieManager(manager)
	if cookies == 0 {
		gObjectUnref(manager)
		return nil, errors.New("mygo: cannot create Browser cookie store")
	}
	// WebKitGTK cookies are otherwise session-only even with a persistent manager.
	browserCookieStorage(cookies, cs(filepath.Join(dataDir, "cookies.sqlite")), 1) // SQLITE
	ctx := browserContextNew(manager)
	if ctx == 0 {
		gObjectUnref(manager)
		return nil, errors.New("mygo: cannot create Browser web context")
	}
	p := &browserProfile{key: key, context: ctx, manager: manager, refs: 1, schemes: map[string]bool{}}
	if b.browserProfiles == nil {
		b.browserProfiles = map[string]*browserProfile{}
	}
	b.browserProfiles[key] = p
	connect(ctx, "download-started", cbDownloadStarted, 0)
	return p, nil
}

func (w *window) releaseBrowserProfile() {
	p := w.browser
	w.browser = nil
	if p == nil {
		return
	}
	p.refs--
	if p.refs == 0 {
		delete(w.b.browserProfiles, p.key)
		gObjectUnref(p.context)
		gObjectUnref(p.manager)
	}
}

func (w *window) browserReady() bool {
	return w != nil && !w.closed && w.web != 0 && w.opts != nil && w.opts.BrowserProfile &&
		w.browser != nil && w.b != nil && w.b.browserProfiles[w.opts.BrowserPartition] == w.browser &&
		w.browser.key == w.opts.BrowserPartition && browserGetContext != nil && browserGetContext(w.web) == w.browser.context
}

func (w *window) BrowserToolsAvailable() bool       { return w.browserReady() }
func (w *window) BrowserCookieToolsAvailable() bool { return w.browserReady() }
func (w *window) BrowserSiteDataToolsAvailable() bool {
	return w.browserReady() && browserVersionAtLeast(2, 30)
}
func (w *window) BrowserColorSchemeAvailable() bool { return false }

func (w *window) SetBrowserColorScheme(_ string, done func(error)) {
	// GTK3's application-prefer-dark-theme is process-global, not a per-page
	// override. Do not change unrelated tabs/windows or inject CSS as a substitute.
	done(errors.New("mygo: per-page color scheme is unavailable in the GTK3 WebKit backend"))
}

const (
	browserHTTPCacheTypes uint32 = 1<<0 | 1<<1 // memory cache, disk cache
	browserCookieTypes    uint32 = 1 << 8
	browserSiteDataTypes  uint32 = 1<<4 | 1<<5 | 1<<6 | 1<<8 | 1<<12
)

func (w *window) clearBrowserData(types uint32, done func(error)) {
	if !w.browserReady() {
		done(errors.New("mygo: Browser profile is not ready"))
		return
	}
	manager := w.browser.manager
	// Keep the manager alive even if the last tab is destroyed before completion.
	gObjectRef(manager)
	request := pending.add(func(source, result ptr) {
		defer gObjectUnref(manager)
		var nativeError ptr
		ok := webkitWebsiteDataManagerClearFinish(source, result, &nativeError)
		err := gErr(nativeError)
		if !ok && err == nil {
			err = errors.New("mygo: WebKitGTK failed to clear Browser data")
		}
		done(err)
	})
	webkitWebsiteDataManagerClear(manager, types, 0, 0, cbAsyncReady, request)
}
func (w *window) ClearBrowserHTTPCache(done func(error)) {
	w.clearBrowserData(browserHTTPCacheTypes, done)
}
func (w *window) ClearBrowserLoginCookies(done func(error)) {
	w.clearBrowserData(browserCookieTypes, done)
}
func (w *window) ClearBrowserSiteData(done func(error)) {
	if !w.BrowserSiteDataToolsAvailable() {
		done(errors.New("mygo: Browser site-data clearing requires WebKitGTK 2.30+"))
		return
	}
	w.clearBrowserData(browserSiteDataTypes, done)
}
