//go:build darwin

package darwin

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/internal/platform"
)

// WKWebsiteDataStore's public named-store API requires macOS 14.0. No KVC
// directory override or private SPI is used, and old systems never fall back
// to the default or an ephemeral store for a persistent Browser partition.
func browserProfileUUID(key string) string {
	sum := sha256.Sum256([]byte("mygo.browser.partition.v1:" + key))
	sum[6] = sum[6]&0x0f | 0x80 // UUIDv8, custom SHA-256-derived identifier
	sum[8] = sum[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func newBrowserStore(key string) (id, error) {
	if !platform.BrowserPartitionValid(key) {
		return 0, errors.New("mygo: invalid Browser partition key")
	}
	cls := class("WKWebsiteDataStore")
	if !respondsTo(cls, "dataStoreForIdentifier:") {
		return 0, errors.New("mygo: persistent Browser partitions require macOS 14 or later")
	}
	uuid := send(send(class("NSUUID"), "alloc"), "initWithUUIDString:", uintptr(nsString(browserProfileUUID(key))))
	defer release(uuid)
	if uuid == 0 {
		return 0, errors.New("mygo: cannot construct Browser partition identifier")
	}
	store := send(cls, "dataStoreForIdentifier:", uintptr(uuid))
	if store == 0 || send(store, "isPersistent") == 0 || send(send(store, "identifier"), "isEqual:", uintptr(uuid)) == 0 {
		return 0, errors.New("mygo: cannot verify persistent Browser partition")
	}
	return retain(store), nil
}

func (w *window) browserReady() bool {
	return w != nil && !w.closed && w.web != 0 && w.opts != nil && w.opts.BrowserProfile && w.browserStore != 0 &&
		send(send(w.web, "configuration"), "websiteDataStore") == w.browserStore
}
func (w *window) BrowserToolsAvailable() bool       { return w.browserReady() }
func (w *window) BrowserCookieToolsAvailable() bool { return w.browserReady() }
func (w *window) BrowserColorSchemeAvailable() bool { return w.browserReady() }

// The public WKWebView inspectable property (macOS 13.3+) enables Safari's
// Develop menu. It cannot open an in-app inspector. Do not claim our explicit
// OpenBrowserDevTools contract is supported or invoke private _inspector SPI.
func (w *window) BrowserDevToolsAvailable() bool { return false }
func (w *window) OpenBrowserDevTools() error {
	return errors.New("mygo: WKWebView has no public API to open an in-app Browser inspector")
}

func (w *window) SetBrowserColorScheme(scheme string, done func(error)) {
	if !w.browserReady() {
		done(errors.New("mygo: Browser profile is not ready"))
		return
	}
	withPool(func() {
		var appearance id
		switch scheme {
		case "system":
		case "light":
			appearance = send(class("NSAppearance"), "appearanceNamed:", uintptr(appKitString("NSAppearanceNameAqua")))
		case "dark":
			appearance = send(class("NSAppearance"), "appearanceNamed:", uintptr(appKitString("NSAppearanceNameDarkAqua")))
		default:
			done(errors.New("mygo: invalid Browser color scheme"))
			return
		}
		send(w.web, "setAppearance:", uintptr(appearance))
		done(nil)
	})
}

var browserCacheTypes = []string{"WKWebsiteDataTypeMemoryCache", "WKWebsiteDataTypeDiskCache"}
var browserCookieTypes = []string{"WKWebsiteDataTypeCookies"}
var browserSiteTypes = []string{
	"WKWebsiteDataTypeCookies", "WKWebsiteDataTypeLocalStorage", "WKWebsiteDataTypeIndexedDBDatabases",
	"WKWebsiteDataTypeWebSQLDatabases", "WKWebsiteDataTypeServiceWorkerRegistrations",
}

// Resolve actual exported NSString constants, never guess their string values.
func browserDataType(name string) id {
	p, err := purego.Dlsym(libWebKit, name)
	if err != nil || p == 0 {
		return 0
	}
	return **(**id)(unsafe.Pointer(&p))
}
func (w *window) BrowserSiteDataToolsAvailable() bool {
	if !w.browserReady() {
		return false
	}
	for _, name := range browserSiteTypes {
		if browserDataType(name) == 0 {
			return false
		}
	}
	return true
}
func (w *window) clearBrowserData(names []string, done func(error)) {
	if !w.browserReady() {
		done(errors.New("mygo: Browser profile is not ready"))
		return
	}
	withPool(func() {
		types := alloc("NSMutableSet")
		defer release(types)
		for _, name := range names {
			typ := browserDataType(name)
			if typ == 0 {
				done(fmt.Errorf("mygo: WebKit data type %s is unavailable", name))
				return
			}
			send(types, "addObject:", uintptr(typ))
		}
		store := retain(w.browserStore)
		blk := newBlock(func(_ objc.Block) {
			w.b.runOnMain(func() { release(store); done(nil) })
		})
		send(store, "removeDataOfTypes:modifiedSince:completionHandler:", uintptr(types), uintptr(send(class("NSDate"), "distantPast")), uintptr(blk))
		blk.Release()
	})
}
func (w *window) ClearBrowserHTTPCache(done func(error)) { w.clearBrowserData(browserCacheTypes, done) }
func (w *window) ClearBrowserLoginCookies(done func(error)) {
	w.clearBrowserData(browserCookieTypes, done)
}
func (w *window) ClearBrowserSiteData(done func(error)) { w.clearBrowserData(browserSiteTypes, done) }
