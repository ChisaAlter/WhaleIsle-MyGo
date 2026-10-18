//go:build linux && (amd64 || arm64)

package previewprocess

import (
	"context"
	"strings"
	"testing"
)

func TestLinuxNamespaceAttestation(t *testing.T) {
	a, b := []string{}, []string{}
	for _, kind := range namespaceKinds {
		a = append(a, kind+":[1]")
		b = append(b, kind+":[2]")
	}
	parent, current := strings.Join(a, ","), strings.Join(b, ",")
	if !isolatedNamespace(parent, current) {
		t.Fatal("isolated identities rejected")
	}
	for i := range a {
		altered := append([]string(nil), b...)
		altered[i] = a[i]
		if isolatedNamespace(parent, strings.Join(altered, ",")) {
			t.Fatalf("unchanged %s namespace accepted", namespaceKinds[i])
		}
	}
	if isolatedNamespace("", current) || isolatedNamespace(parent, "bogus") {
		t.Fatal("malformed identity accepted")
	}
}
func TestLinuxSandboxUsesOnlyPackagedRuntime(t *testing.T) {
	args := linuxSandboxArgs("/trusted/helper", "/trusted/runtime", "nonce", "parent")
	joined := strings.Join(args, "\n")
	for _, required := range []string{"--unshare-all", "--die-with-parent", "--cap-drop\nALL", "--clearenv", "--ro-bind\n/trusted/runtime\n/", "--ro-bind\n/trusted/helper\n/whaleisle-preview", "--tmpfs\n/tmp", "--tmpfs\n/home", "/usr/bin/dbus-run-session"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing policy: %s", required)
		}
	}
	for _, forbidden := range []string{"--share-net", "--share-pid", "--bind\n", "WAYLAND_DISPLAY", "DBUS_SESSION_BUS_ADDRESS", "--ro-bind\n/usr\n"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("host exposure: %s", forbidden)
		}
	}
}
func TestLinuxRejectsInlineCaptureBeforeLaunch(t *testing.T) {
	_, err := openPlatform(context.Background(), Options{HelperPath: "/missing"}, Document{Inline: true, capture: true})
	if err == nil || !strings.Contains(err.Error(), "inline capture") {
		t.Fatalf("unexpected result: %v", err)
	}
}
