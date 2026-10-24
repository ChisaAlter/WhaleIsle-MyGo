package webview

import (
	"net/url"
	"strings"

	"github.com/egoist/mygo"
)

// IsLoopbackHarnessOrigin reports whether origin is the local Harness service's
// loopback transport. Port of the native isHarnessOrigin
// (reference/electron/src/main/media-permissions.js:10-18): http(s) on
// 127.0.0.1 or localhost, nothing else.
func IsLoopbackHarnessOrigin(origin string) bool {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "127.0.0.1", "localhost":
		return true
	default:
		return false
	}
}

// PermissionPolicy decides what one hosted page may use. It is the narrowed port
// of the native installMediaPermissions handlers
// (reference/electron/src/main/media-permissions.js:24-60): the microphone only
// from the loopback Harness session, camera and geolocation always denied, and
// notifications only while the shell's system-notification gate is open.
//
// Residuals: mygo's PermissionRequest carries only Permissions and Origin, so
// the native checks for the primary harness view, its main frame and
// mediaType == audio cannot be reproduced here. Any page served from a loopback
// origin by a session host may ask for the microphone, and a request that also
// claims the camera is denied outright because the camera itself is refused.
// Other permission kinds keep the engine default (allow), as the native handler
// leaves every non-media, non-notification permission to Electron.
type PermissionPolicy struct {
	// SessionHost marks a host that carries the loopback Harness session: only
	// there may a page ask for the microphone.
	SessionHost bool
	// NotificationsAllowed reports the shell's system-notification gate; nil
	// denies, so a host that does not say keeps notifications off.
	NotificationsAllowed func() bool
}

// Decide answers one permission request. It denies the whole request when any
// permission in it is refused, so a page cannot pair a denied camera with an
// allowed microphone.
func (p PermissionPolicy) Decide(req mygo.PermissionRequest) bool {
	for _, permission := range req.Permissions {
		switch permission {
		case mygo.PermissionNotifications:
			if p.NotificationsAllowed == nil || !p.NotificationsAllowed() {
				return false
			}
		case mygo.PermissionMicrophone:
			if !p.SessionHost || !IsLoopbackHarnessOrigin(req.Origin) {
				return false
			}
		case mygo.PermissionCamera, mygo.PermissionGeolocation:
			// The camera is refused because the native media rule only grants
			// audio. Geolocation is a deliberate narrowing: the native check
			// handler leaves it to Electron's defaults, but WebView2 here has no
			// OS location service configured and the shell never offers one.
			return false
		default:
			// The native handler leaves other permissions to the engine.
		}
	}
	return true
}

// DenyAllPermissions is the isolated, transient profile's answer: nothing at all,
// which is what those single-capability windows have always had.
func DenyAllPermissions(mygo.PermissionRequest) bool { return false }
