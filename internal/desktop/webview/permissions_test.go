package webview

import (
	"testing"

	"github.com/egoist/mygo"
)

// decide is the policy's answer for one request, in the shape the window uses.
func decide(p PermissionPolicy, origin string, kinds ...mygo.Permission) bool {
	return p.Decide(mygo.PermissionRequest{Origin: origin, Permissions: kinds})
}

// TestPermissionPolicyTable is the G42 matrix: the loopback Harness session may
// ask for the microphone, remote origins may not, the camera is always refused,
// and notifications follow the shell gate
// (reference/electron/src/main/media-permissions.js:32-47).
func TestPermissionPolicyTable(t *testing.T) {
	on := func() bool { return true }
	off := func() bool { return false }
	session := PermissionPolicy{SessionHost: true, NotificationsAllowed: on}
	remoteSession := PermissionPolicy{SessionHost: true, NotificationsAllowed: on}
	plain := PermissionPolicy{NotificationsAllowed: on}
	cases := []struct {
		name   string
		policy PermissionPolicy
		origin string
		kind   mygo.Permission
		want   bool
	}{
		{"loopback microphone", session, "http://127.0.0.1:32383", mygo.PermissionMicrophone, true},
		{"localhost microphone", session, "http://localhost:32383", mygo.PermissionMicrophone, true},
		{"https loopback microphone", session, "https://127.0.0.1:1", mygo.PermissionMicrophone, true},
		{"remote microphone", remoteSession, "https://meet.example", mygo.PermissionMicrophone, false},
		{"lan microphone", remoteSession, "http://192.168.1.9:32383", mygo.PermissionMicrophone, false},
		{"loopback lookalike", remoteSession, "http://127.0.0.1.evil.example", mygo.PermissionMicrophone, false},
		{"file origin microphone", remoteSession, "file:///C:/tmp/x.html", mygo.PermissionMicrophone, false},
		{"non-session loopback microphone", plain, "http://127.0.0.1:32383", mygo.PermissionMicrophone, false},
		{"camera on loopback", session, "http://127.0.0.1:32383", mygo.PermissionCamera, false},
		{"geolocation on loopback", session, "http://127.0.0.1:32383", mygo.PermissionGeolocation, false},
		{"notifications allowed", session, "http://127.0.0.1:32383", mygo.PermissionNotifications, true},
		{"notifications denied", PermissionPolicy{SessionHost: true, NotificationsAllowed: off}, "http://127.0.0.1:32383", mygo.PermissionNotifications, false},
		{"notifications without a gate", PermissionPolicy{SessionHost: true}, "http://127.0.0.1:32383", mygo.PermissionNotifications, false},
	}
	for _, tc := range cases {
		if got := decide(tc.policy, tc.origin, tc.kind); got != tc.want {
			t.Errorf("%s: Decide(%s from %s) = %v, want %v", tc.name, tc.kind, tc.origin, got, tc.want)
		}
	}
	// A pair that includes the camera is refused even with a granted microphone,
	// because the camera itself is never offered.
	if decide(session, "http://127.0.0.1:32383", mygo.PermissionMicrophone, mygo.PermissionCamera) {
		t.Error("a microphone+camera request must be refused for the camera")
	}
	// Unknown kinds keep the engine default (allow), as the native handler does.
	if !decide(session, "https://example.com", mygo.Permission("clipboard-read")) {
		t.Error("unknown permissions must keep the engine default")
	}
	if decide(session, "https://example.com", mygo.Permission("clipboard-read"), mygo.PermissionCamera) {
		t.Error("one refused permission must refuse the whole request")
	}
}

// TestIsLoopbackHarnessOrigin pins the native origin test
// (media-permissions.js:10-18): http(s), 127.0.0.1 or localhost, nothing else.
func TestIsLoopbackHarnessOrigin(t *testing.T) {
	for origin, want := range map[string]bool{
		"http://127.0.0.1:32383":  true,
		"http://localhost:32383/": true,
		"https://localhost":       true,
		"http://[::1]:32383":      false, // the native test does not name ::1
		"http://127.0.0.1:1@evil": false,
		"https://127.0.0.1.evil":  false,
		"ftp://127.0.0.1":         false,
		"http://127.0.0.2:1":      false,
		"localhost:32383":         false,
		"":                        false,
	} {
		if got := IsLoopbackHarnessOrigin(origin); got != want {
			t.Errorf("IsLoopbackHarnessOrigin(%q) = %v, want %v", origin, got, want)
		}
	}
}

// TestDenyAllPermissions keeps the isolated capability profiles closed.
func TestDenyAllPermissions(t *testing.T) {
	for _, kind := range []mygo.Permission{
		mygo.PermissionCamera, mygo.PermissionMicrophone, mygo.PermissionGeolocation, mygo.PermissionNotifications,
	} {
		if DenyAllPermissions(mygo.PermissionRequest{Origin: "http://127.0.0.1:1", Permissions: []mygo.Permission{kind}}) {
			t.Errorf("transient profile granted %s", kind)
		}
	}
}
