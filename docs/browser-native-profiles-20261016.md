# Browser native profiles on macOS and Linux

## Delivered scope

BrowserProfile/BrowserPartition now reach the existing WKWebView and WebKitGTK
backends. No engine replacement, private profile-directory override, data
migration, automatic cleanup, profile-directory deletion, or default-profile
fallback is introduced. Ordinary non-Browser pages retain their existing setup.

- macOS 14+: `WKWebsiteDataStore dataStoreForIdentifier:` supplies persistent
  named stores. A versioned SHA-256 mapping turns the complete partition key
  into a nonzero UUIDv8. The store's persistent flag and identifier are checked
  before constructing the view. The retained store is assigned to its
  configuration before initialization and released with the window.
- Linux WebKitGTK 2.16+, API 4.1/4.0: each active partition has its own
  WebsiteDataManager and WebContext. Concurrent tabs with the same key share
  that context; different keys do not. Storage is beneath XDG_DATA_HOME (or
  ~/.local/share), in mygo/<SHA256(application name)>/browser-workspaces/<key>.
  Cookie persistence explicitly uses a partition-local SQLite file. The context
  receives its own custom-scheme and download handlers. References are released
  when the final tab closes; persistent files remain.
- Both: partition-scoped HTTP memory/disk cache and Cookie clearing use native
  asynchronous completion. Five-kind site-data clearing selects only Cookies,
  localStorage, IndexedDB, WebSQL and Service Worker registrations, retaining
  HTTP cache and CacheStorage. Linux requires 2.30+ for the Service Worker type.
  macOS resolves the real exported data-type constants and refuses unavailable
  types. Existing caller confirmations and cancellation semantics are unchanged.
- macOS: per-view NSAppearance supplies light/dark/system appearance without
  CSS injection or changing the global application theme.
- Host initialization now has a narrow error adapter for MyGo's backend-creation
  failure panic. Unsupported runtime initialization can be shown as an error.

## Capability boundaries

Older macOS explicitly refuses persistent Browser profiles; it never substitutes
shared or in-memory storage. Missing Linux APIs likewise return an error.

BrowserToolsAvailable remains the cache capability. The new
BrowserColorSchemeAvailable is separate, wired through Host and the appearance
menu. GTK3 has no implemented public per-page color-scheme override here; Linux
appearance remains system-controlled, while cache/Cookie cleanup stays usable.
Changing GTK's global application-prefer-dark-theme would affect unrelated tabs
and is intentionally not used.

The explicit Browser inspector remains unavailable on macOS: WKWebView's public
inspectable property enables Safari inspection, but cannot open an in-app
inspector. The existing private `_inspector` path is not called by Browser APIs.
Linux has a public WebInspector API, but the Browser-only explicit-inspector
adapter is not part of this increment. Its developer-extras/automatic-entrypoint
contract still needs implementation. This is unfinished work, not a claim that
Linux WebKit cannot provide an inspector.

## Verification and remaining runtime checks

Passed (CGO_ENABLED=0):

- Linux Browser mock tests: unavailable/ordinary capabilities, same-key context
  sharing, different-key isolation, mismatched-context rejection, reference
  ownership, invalid keys, missing/old runtime APIs, failed creation, exact
  cleanup masks and manager selection, asynchronous completion, and the 2.30
  Service Worker gate. These tests use temporary fixture directories and mock
  native calls; no user browser data is read or cleared.
- Native backend test-binary cross-compilation: macOS arm64 and Linux arm64.
- Windows amd64 compilation of the MyGo package.
- Focused Browser Host/UI tests, including the initialization-error adapter.
- Host/UI package builds for macOS arm64 and Linux amd64.

No macOS or WebKitGTK GUI session was exercised. Native profile persistence,
Cookie behavior across restarts, real WebKit cleanup, OS appearance propagation,
and native child-window interaction remain runtime acceptance checks. A compiled
binary or mock result does not establish those outcomes.

An additional SDK-wide test attempt failed and is not a release pass. Observed
failures include MyGo TestPaths (read-only HOME), TestSingleInstance, SQLite and
terminal tests (native dependency/cache setup), and ui.TestDiffFrameAllocs
(allocation assertion; cause not diagnosed). Output was truncated, so this is not
an exhaustive failure inventory. No unrelated repair was attempted.

## Official API references

- Apple named stores and persistent state:
  https://developer.apple.com/documentation/webkit/wkwebsitedatastore/init(foridentifier:)?language=objc
- WebKit public header declaring macOS 14 availability:
  https://github.com/WebKit/WebKit/blob/main/Source/WebKit/UIProcess/API/Cocoa/WKWebsiteDataStore.h
- WebKitGTK context construction:
  https://webkitgtk.org/reference/webkit2gtk/stable/ctor.WebContext.new_with_website_data_manager.html
- WebKitGTK exact data-type masks:
  https://webkitgtk.org/reference/webkit2gtk/2.40.3/flags.WebsiteDataTypes.html
- WebKitGTK async clearing and completion:
  https://webkitgtk.org/reference/webkit2gtk/2.42.3/method.WebsiteDataManager.clear.html
- WebKitGTK Cookie persistence:
  https://webkitgtk.org/reference/webkit2gtk/2.34.4/WebKitCookieManager.html
- WebKitGTK inspector prerequisites:
  https://www.webkitgtk.org/reference/webkit2gtk/stable/class.WebInspector.html
