// Package deeplink implements the WHALEISLE custom URL scheme (M11).
//
// Scheme: whaleisle (URLs like whaleisle://settings?section=desktop),
// registered via mygo.json urlSchemes and mygo.App.RegisterURLScheme.
//
// Parse maps URLs to Target actions (open / launcher / settings / session /
// usage). cmd/whaleisle applies them in applyDeeplink.
package deeplink

// Scheme is the registered custom URL scheme (lowercase; WHALEISLE branding).
const Scheme = "whaleisle"
