package phoneremote

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"
)

// liveVendor resolves the directory that contains the dshd-remote component.
// The test never guesses a checkout: an explicit WHALEISLE_PHONE_LIVE_VENDOR
// wins, otherwise the repository runtime/vendor is used when it is present.
func liveVendor(t *testing.T) string {
	t.Helper()
	if vendor := strings.TrimSpace(os.Getenv("WHALEISLE_PHONE_LIVE_VENDOR")); vendor != "" {
		return vendor
	}
	if _, file, _, ok := goruntime.Caller(0); ok {
		// internal/desktop/phoneremote -> repository root
		root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
		candidate := filepath.Join(root, "runtime", "vendor")
		if info, err := os.Stat(filepath.Join(candidate, "dshd-remote")); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port
}

// TestLiveLanServeSPAHealthAndPairing is the real end-to-end acceptance run for
// the phone remote server: it starts the product runtime (the original
// dshd-remote component under the bundled Node), waits for a listening LAN
// gateway, then performs the same requests a phone does — the SPA landing page
// on the fixed mobile port, the daemon health probe, and an authenticated
// shell route with the pairing token carried by the offer fragment.
//
// The full path needs an explicit WHALEISLE_PHONE_LIVE=1 opt-in, or the existing
// explicit LIVE_VENDOR/LIVE_BIND settings. Ordinary package tests must not bind
// the fixed mobile port 3180 merely because packaged resources are present.
// The default bind and the relay fixture are loopback; an explicit
// WHALEISLE_PHONE_LIVE_BIND=0.0.0.0 still opts into the LAN-adapter contract.
// It skips while another gateway owns 3180 and never terminates that instance.
func TestLiveLanServeSPAHealthAndPairing(t *testing.T) {
	if !livePhoneOptIn(os.Getenv) {
		t.Skip("set WHALEISLE_PHONE_LIVE=1 (or explicit LIVE_VENDOR/LIVE_BIND) for the owned live fixture")
	}
	vendor := liveVendor(t)
	if vendor == "" {
		t.Skip("set WHALEISLE_PHONE_LIVE_VENDOR to the directory containing dshd-remote")
	}
	if _, err := ResolveResources(t.TempDir(), vendor); err != nil {
		t.Skipf("packaged dshd-remote component unavailable: %v", err)
	}
	if probe, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", MobileWebPort)); err != nil {
		t.Skipf("mobile port %d is busy: %v", MobileWebPort, err)
	} else {
		_ = probe.Close()
	}

	userData, err := os.MkdirTemp("", "mygo-phone-live-*")
	if err != nil {
		t.Fatal(err)
	}
	daemonPort := freePort(t)
	// The bind address decides the landing origin the QR points at: loopback is
	// only reachable from the same machine, 0.0.0.0 advertises the preferred LAN
	// address. WHALEISLE_PHONE_LIVE_BIND=0.0.0.0 exercises the phone-reachable
	// case on a machine with a LAN adapter.
	bind := strings.TrimSpace(os.Getenv("WHALEISLE_PHONE_LIVE_BIND"))
	if bind == "" {
		bind = "127.0.0.1"
	}
	m := NewRuntime()
	m.stopTimeout = 20 * time.Second
	stopped := false
	defer func() {
		if !stopped {
			m.Stop()
		}
		// The daemon keeps its own files; best-effort removal only.
		_ = os.RemoveAll(userData)
	}()
	// Hold the relay destination for this fixture's lifetime so config never
	// falls back to a public default or reuses another process's ephemeral port.
	// LAN readiness does not require a successful relay handshake.
	relay, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	go func() {
		for {
			conn, err := relay.Accept()
			if err != nil {
				return
			}
			_ = conn.Close() // This fixture never handles application relay traffic.
		}
	}()
	m.Start(userData, vendor, livePhoneConfig(bind, daemonPort, relay.Addr().String()))

	deadline := time.Now().Add(90 * time.Second)
	var snapshot RuntimeState
	for time.Now().Before(deadline) {
		snapshot = m.Snapshot()
		if snapshot.Status == "ready" {
			break
		}
		if snapshot.Status == "offline" || snapshot.Status == "missing" {
			t.Fatalf("runtime stopped before ready: status=%q message=%q", snapshot.Status, snapshot.Message)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if snapshot.Status != "ready" {
		t.Fatalf("LAN gateway did not become ready: status=%q message=%q", snapshot.Status, snapshot.Message)
	}
	if !snapshot.Listening {
		t.Fatal("ready state without a listening gateway")
	}

	// 1. The offer the desktop shows is a v2 offer the clients can decode.
	offer, expires := decodeLiveOffer(t, snapshot.PairingURL)
	if !expires.After(time.Now()) {
		t.Fatalf("offer already expired: %s", expires)
	}
	if strings.TrimSpace(offer.Relay.Endpoint) == "" {
		t.Fatal("offer carries no relay endpoint; the Android client requires relay.endpoint")
	}
	t.Logf("offer v%d server=%s relay=%s tls=%v expires=%s tokenLen=%d",
		offer.V, offer.ServerID, offer.Relay.Endpoint, offer.Relay.UseTLS,
		expires.Format(time.RFC3339), len(offer.Auth.Token))
	landing := snapshot.PairingURL
	if index := strings.Index(landing, "#offer="); index >= 0 {
		landing = landing[:index]
	}
	t.Logf("pairing landing origin=%s (fragment redacted)", landing)
	if bind == "0.0.0.0" {
		// A wildcard bind must advertise a reachable LAN address, never the
		// loopback literal, or the phone cannot open the landing page.
		if strings.Contains(landing, "127.0.0.1") || strings.Contains(landing, "localhost") {
			t.Fatalf("wildcard bind advertised a loopback landing origin: %s", landing)
		}
	}
	if snapshot.PairingImage == nil {
		// The bridge renders the QR after the first snapshot that carries a new
		// offer, so the bitmap arrives one control tick later.
		for end := time.Now().Add(20 * time.Second); time.Now().Before(end); {
			time.Sleep(100 * time.Millisecond)
			if current := m.Snapshot(); current.PairingImage != nil {
				snapshot = current
				break
			}
		}
	}
	if snapshot.PairingImage == nil {
		t.Fatal("no QR image; the desktop panel would fall back to the link-only state")
	}
	bounds := snapshot.PairingImage.Bounds()
	if bounds.Dx() != bounds.Dy() || bounds.Dx() < 64 {
		t.Fatalf("QR image is not a usable square: %dx%d", bounds.Dx(), bounds.Dy())
	}
	t.Logf("QR bitmap %dx%d", bounds.Dx(), bounds.Dy())

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	mobile := fmt.Sprintf("http://127.0.0.1:%d", MobileWebPort)

	// 2. The mobile SPA landing page AND its entry script.
	indexStatus, indexType, indexBody, indexHeaders := liveGetWithHeaders(t, client, mobile+"/", "", http.MethodGet)
	if indexStatus != 200 || !strings.Contains(indexType, "text/html") {
		t.Fatalf("SPA index: status=%d type=%q", indexStatus, indexType)
	}
	if !strings.Contains(indexBody, "id=\"phone\"") || !strings.Contains(indexBody, "app.js") {
		t.Fatalf("SPA index does not look like the pairing page: %q", truncateForLog(indexBody))
	}
	// The landing page is unauthenticated LAN input: the reference sets only the
	// universally safe headers and keeps the offer in the fragment.
	for header, want := range map[string]string{
		"x-content-type-options": "nosniff",
		"referrer-policy":        "no-referrer",
		"x-frame-options":        "DENY",
		"cache-control":          "no-store",
	} {
		if got := indexHeaders.Get(header); got != want {
			t.Fatalf("SPA index %s = %q, want %q", header, got, want)
		}
	}
	if strings.Contains(indexHeaders.Get("content-security-policy"), "connect-src") {
		t.Fatal("SPA index sets a connect-src CSP; the phone pairs over a different relay origin")
	}
	t.Logf("GET / -> %d %s (%d bytes) nosniff/referrer/DENY/no-store", indexStatus, indexType, len(indexBody))
	headStatus, headType, _, _ := liveGetWithHeaders(t, client, mobile+"/", "", http.MethodHead)
	if headStatus != 200 || !strings.Contains(headType, "text/html") {
		t.Fatalf("SPA index HEAD: status=%d type=%q", headStatus, headType)
	}
	postStatus, _, _, _ := liveGetWithHeaders(t, client, mobile+"/", "", http.MethodPost)
	if postStatus != 405 {
		t.Fatalf("SPA index POST: status=%d, want 405", postStatus)
	}
	scriptStatus, scriptType, scriptBody := liveGet(t, client, mobile+"/app.js", "")
	if scriptStatus != 200 || !strings.Contains(scriptType, "javascript") || len(scriptBody) < 1000 {
		t.Fatalf("SPA entry script: status=%d type=%q bytes=%d", scriptStatus, scriptType, len(scriptBody))
	}
	t.Logf("GET /app.js -> %d %s (%d bytes)", scriptStatus, scriptType, len(scriptBody))
	missingStatus, _, _ := liveGet(t, client, mobile+"/definitely-absent.js", "")
	if missingStatus != 404 {
		t.Fatalf("unknown asset status=%d, want 404", missingStatus)
	}
	// The mobile port serves the SPA only; the authenticated daemon API lives on
	// its own loopback port. Reference mobile-web.js resolveSpaAsset answers an
	// extension-less path with index.html, so /api/health here is the SPA page,
	// never the JSON health document that the daemon port returns.
	apiStatus, apiType, apiBody := liveGet(t, client, mobile+"/api/health", "")
	if !strings.Contains(apiType, "text/html") || !strings.Contains(apiBody, "id=\"phone\"") {
		t.Fatalf("SPA port served %q (%d bytes) for /api/health; want the index fallback", apiType, len(apiBody))
	}
	t.Logf("GET :%d/api/health -> %d %s (index.html fallback, not the daemon API)", MobileWebPort, apiStatus, apiType)

	// 3. The daemon health probe the phone uses before pairing.
	healthStatus, healthType, healthBody := liveGet(t, client, fmt.Sprintf("http://127.0.0.1:%d/api/health", daemonPort), "")
	var health map[string]any
	if err := json.Unmarshal([]byte(healthBody), &health); err != nil || healthStatus != 200 || health["status"] != "ok" {
		t.Fatalf("daemon health: status=%d type=%q body=%q err=%v", healthStatus, healthType, truncateForLog(healthBody), err)
	}
	t.Logf("GET /api/health -> %d %s", healthStatus, healthType)

	// 4. The pairing route: the shell channel answers only with the token the
	// offer fragment carries, never without it.
	shellURL := mobile + "/__remote__/shell/getConfig"
	unauthStatus, _ := livePost(t, client, shellURL, "", `{}`)
	if unauthStatus != 401 {
		t.Fatalf("shell route without a token: status=%d, want 401", unauthStatus)
	}
	var shellStatus int
	var shellBody string
	shellDeadline := time.Now().Add(15 * time.Second)
	for {
		shellStatus, shellBody = livePost(t, client, shellURL, offer.Auth.Token, `{}`)
		if shellStatus != 401 || time.Now().After(shellDeadline) {
			break
		}
		// The shell routes attach to the running SPA server after it listens.
		time.Sleep(250 * time.Millisecond)
	}
	var shell struct {
		OK     bool `json:"ok"`
		Result struct {
			RemoteEnabled bool   `json:"remoteEnabled"`
			RemoteMode    string `json:"remoteMode"`
			RemotePort    int    `json:"remotePort"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(shellBody), &shell); err != nil || shellStatus != 200 || !shell.OK {
		t.Fatalf("shell getConfig: status=%d body=%q err=%v", shellStatus, truncateForLog(shellBody), err)
	}
	if !shell.Result.RemoteEnabled || shell.Result.RemoteMode != "lan" || shell.Result.RemotePort != daemonPort {
		t.Fatalf("shell getConfig reported %+v, want the live LAN config on port %d", shell.Result, daemonPort)
	}
	t.Logf("POST /__remote__/shell/getConfig -> %d %s", shellStatus, truncateForLog(shellBody))

	// Optional hold so an operator can hit the same live service with curl. The
	// ephemeral offer goes to a private temp file, never to the test log.
	if hold := os.Getenv("WHALEISLE_PHONE_LIVE_HOLD_SECONDS"); hold != "" {
		seconds, err := time.ParseDuration(hold + "s")
		if err != nil {
			t.Fatalf("invalid hold duration: %v", err)
		}
		offerFile := filepath.Join(os.TempDir(), "mygo-phone-live-offer.txt")
		if err = os.WriteFile(offerFile, []byte(snapshot.PairingURL), 0600); err != nil {
			t.Fatalf("offer file: %v", err)
		}
		t.Logf("holding the live gateway for %s: SPA %s, daemon http://127.0.0.1:%d, offer in %s",
			seconds, mobile, daemonPort, offerFile)
		time.Sleep(seconds)
		_ = os.Remove(offerFile)
	}

	// 5. Stopping the runtime releases both listeners.
	m.Stop()
	stopped = true
	if s := m.Snapshot(); s.Listening || s.PairingURL != "" {
		t.Fatalf("stopped runtime kept pairing: %+v", s)
	}
	for _, address := range []string{
		fmt.Sprintf("127.0.0.1:%d", MobileWebPort),
		fmt.Sprintf("127.0.0.1:%d", daemonPort),
	} {
		closed := false
		for end := time.Now().Add(10 * time.Second); time.Now().Before(end); {
			conn, dialErr := net.DialTimeout("tcp", address, 200*time.Millisecond)
			if dialErr != nil {
				closed = true
				break
			}
			_ = conn.Close()
			time.Sleep(100 * time.Millisecond)
		}
		if !closed {
			t.Fatalf("listener %s survived stop", address)
		}
	}
}

func livePhoneOptIn(getenv func(string) string) bool {
	return getenv("WHALEISLE_PHONE_LIVE") == "1" ||
		strings.TrimSpace(getenv("WHALEISLE_PHONE_LIVE_VENDOR")) != "" ||
		strings.TrimSpace(getenv("WHALEISLE_PHONE_LIVE_BIND")) != ""
}

func livePhoneConfig(bind string, daemonPort int, relayEndpoint string) map[string]any {
	return map[string]any{
		"remoteEnabled":       true,
		"remoteMode":          "lan",
		"remoteBindAddress":   bind,
		"remotePort":          daemonPort,
		"remoteRelayEndpoint": relayEndpoint,
		"remoteRelayUseTls":   false,
	}
}

type liveOffer struct {
	V        int    `json:"v"`
	ServerID string `json:"serverId"`
	Key      string `json:"daemonPublicKeyB64"`
	Relay    struct {
		Endpoint string `json:"endpoint"`
		UseTLS   bool   `json:"useTls"`
	} `json:"relay"`
	Auth struct {
		Version int    `json:"version"`
		Token   string `json:"pairingToken"`
		Expires int64  `json:"expiresAtMs"`
	} `json:"authBootstrap"`
}

func decodeLiveOffer(t *testing.T, raw string) (liveOffer, time.Time) {
	t.Helper()
	if raw == "" {
		t.Fatal("no pairing URL")
	}
	if !strings.Contains(raw, "#offer=") {
		t.Fatalf("pairing URL carries no #offer= fragment: %q", raw)
	}
	index := strings.Index(raw, "#offer=")
	fragment := raw[index+len("#offer="):]
	if cut := strings.IndexAny(fragment, "&#"); cut >= 0 {
		fragment = fragment[:cut]
	}
	decoded, err := base64.RawURLEncoding.DecodeString(fragment)
	if err != nil {
		t.Fatalf("offer fragment is not base64url: %v", err)
	}
	var offer liveOffer
	if err = json.Unmarshal(decoded, &offer); err != nil {
		t.Fatalf("offer is not JSON: %v", err)
	}
	if offer.V != 2 || offer.ServerID == "" || offer.Key == "" {
		t.Fatalf("offer is not v2: %+v", offer)
	}
	if offer.Auth.Version != 1 || offer.Auth.Token == "" || offer.Auth.Expires <= 0 {
		t.Fatalf("offer authBootstrap is not v1/token/expiry: %+v", offer.Auth)
	}
	return offer, time.UnixMilli(offer.Auth.Expires)
}

func liveGet(t *testing.T, client *http.Client, url, token string) (int, string, string) {
	t.Helper()
	status, contentType, body, _ := liveGetWithHeaders(t, client, url, token, http.MethodGet)
	return status, contentType, body
}

func liveGetWithHeaders(t *testing.T, client *http.Client, url, token, method string) (int, string, string, http.Header) {
	t.Helper()
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("authorization", "Bearer "+token)
	}
	status, _, payload, headers := liveDoWithHeaders(t, client, request)
	return status, headers.Get("content-type"), payload, headers
}

func livePost(t *testing.T, client *http.Client, url, token, body string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("content-type", "application/json")
	if token != "" {
		request.Header.Set("authorization", "Bearer "+token)
	}
	status, _, payload := liveDo(t, client, request)
	return status, payload
}

func liveDo(t *testing.T, client *http.Client, request *http.Request) (int, string, string) {
	t.Helper()
	status, contentType, payload, _ := liveDoWithHeaders(t, client, request)
	return status, contentType, payload
}

func liveDoWithHeaders(t *testing.T, client *http.Client, request *http.Request) (int, string, string, http.Header) {
	t.Helper()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", request.Method, request.URL, err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatalf("%s %s body: %v", request.Method, request.URL, err)
	}
	return response.StatusCode, response.Header.Get("content-type"), string(payload), response.Header
}

func truncateForLog(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 200 {
		return value[:200] + "…"
	}
	return value
}
