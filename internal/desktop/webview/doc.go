// Package webview hosts third-party plugin web UI for M10 (dshbot / dsh-whale
// pages and marketplace plugin slots).
//
// # Plugin slot URLs
//
// Whale Isle opens the harness web origin with a query consumed by its
// desktop plugin-views adapter. The adapter discovers live registered views
// and opens them through their original owners; the Host shell stays intact:
//
//	SlotURL(base, pluginID, extra) →
//	  {base}/?desktopPlugin={package}[&extra…]
//
// base is the runtime harness origin (harness.Endpoint.Origin), never a
// build-time constant. Known aliases (dsh-whale, dshbot, dsh-im,
// dsh-usage-panel, market, and their Cordis insert ids) normalize to the
// npm package name; unknown ids pass through for marketplace plugins.
//
// # Host surface
//
// Whale Isle's plan (docs/plan.md) is a borderless WebView2 child window
// aligned to a native panel rectangle inside the main Content window.
//
//	type Host struct { /* opaque */ }
//	func NewHost(parent *mygo.Window, opts HostOptions) (*Host, error)
//	func (h *Host) Navigate(url string) error
//	func (h *Host) SetVisible(visible bool)
//	func (h *Host) SetBounds(rect mygo.Rectangle) // DIPs, screen coordinates
//	func (h *Host) SyncBounds()
//	func (h *Host) Close() error
//
// # MyGo today (v0.3.5)
//
// WebView2 is wired for URL windows (mygo.NewWindow with WindowOptions.URL).
// Native ui.Content windows do not expose a child webview; Host creates a
// frameless Parent child URL window and SyncBounds covers the panel area.
// OpenExternal remains the system-browser fallback.
package webview
