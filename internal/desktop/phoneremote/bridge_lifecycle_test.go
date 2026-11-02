package phoneremote

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/processutil"
)

// Only the production adapter and a fake constructor run; this never constructs
// the original daemon, binds mobile LAN3180, contacts a relay or reads a profile.
func TestBridgeReadyBudgetAndCancelableStop(t *testing.T) {
	node := nodeBinary(t)
	if node == "" {
		t.Skip("node not available")
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--test", filepath.Join(dir, "bridge_lifecycle.test.cjs"))
	processutil.Background(cmd)
	cmd.Dir = dir
	// No inherited Node preload, credentials or Host configuration. The fixture
	// uses VM fakes for process, timers, require and readline, with no networking.
	cmd.Env = daemonEnv([]string{"SystemRoot=" + os.Getenv("SystemRoot"), "PATH=" + filepath.Dir(node)})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fake bridge lifecycle failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "fail 0") {
		t.Fatalf("bridge lifecycle table reported failures:\n%s", out)
	}
	t.Logf("fake constructor / protocol lifecycle passed:\n%s", out)
}

func TestLivePhoneFixtureConfigurationIsExplicitAndRelayLocal(t *testing.T) {
	for _, key := range []string{"", "WHALEISLE_PHONE_LIVE", "WHALEISLE_PHONE_LIVE_VENDOR", "WHALEISLE_PHONE_LIVE_BIND"} {
		values := map[string]string{}
		if key != "" {
			values[key] = map[string]string{"WHALEISLE_PHONE_LIVE": "1", "WHALEISLE_PHONE_LIVE_VENDOR": "/fixture", "WHALEISLE_PHONE_LIVE_BIND": "0.0.0.0"}[key]
		}
		if got := livePhoneOptIn(func(name string) string { return values[name] }); got != (key != "") {
			t.Fatalf("live opt-in for %q = %v", key, got)
		}
	}
	for _, bind := range []string{"127.0.0.1", "0.0.0.0"} {
		config := livePhoneConfig(bind, 43210, "127.0.0.1:43211")
		if config["remoteBindAddress"] != bind || config["remoteRelayEndpoint"] != "127.0.0.1:43211" || config["remoteRelayUseTls"] != false || config["remotePort"] != 43210 {
			t.Fatalf("fixture config changed the explicit bind or inherited an external relay: %+v", config)
		}
	}
}

func TestBridgeLifecycleFailureCodesAreSafe(t *testing.T) {
	child := &deviceControlChild{}
	for _, tc := range []struct {
		code, want string
	}{
		{"operation", "remote startup or pairing failed; check installed runtime and retry"},
		{"stop", "remote daemon could not stop cleanly; check installed runtime before retrying"},
		{"fictional-private-detail", "remote startup or pairing failed; check installed runtime and retry"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			r := NewRuntime()
			r.child = child
			r.consume(child, []byte(`{"kind":"error","code":"`+tc.code+`","error":"fictional-private-detail"}`))
			if got := r.Snapshot(); got.Status != "offline" || got.Message != tc.want || got.PairingURL != "" || strings.Contains(got.Message, "fictional-private") {
				t.Fatalf("unsafe or incorrect failure state: %+v", got)
			}
		})
	}
}
