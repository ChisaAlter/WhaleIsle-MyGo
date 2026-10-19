package phoneremote

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opt-in source directory allows the exact original SPA assets to be exercised
// without a browser, daemon, relay, device, account, or JavaScript execution.
func TestFunctionalOwnedMobileWebServer(t *testing.T) {
	root := os.Getenv("WHALEISLE_REMOTE_MOBILE_ROOT")
	if root == "" {
		t.Skip("set WHALEISLE_REMOTE_MOBILE_ROOT to the read-only reference mobile/web")
	}
	profile := t.TempDir()
	t.Setenv("HOME", profile)
	t.Setenv("USERPROFILE", profile)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := CreateMobileWebServer(MobileWebServerOptions{Root: root, BindAddress: "127.0.0.1"})
	done := make(chan error, 1)
	go func() { done <- server.Serve(l) }()
	defer CloseMobileWebServer(server)
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	base := "http://" + l.Addr().String()
	for _, file := range []string{"index.html", "app.js", "app.css", "tokens.css", "host/backend.js", "host/offer.js", "chisacode/daemon-client.bundle.js"} {
		want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Get(base + "/" + file)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || !bytes.Equal(got, want) {
			t.Fatalf("exact original asset %s failed: status=%d err=%v", file, resp.StatusCode, err)
		}
		if resp.Header.Get("Referrer-Policy") != "no-referrer" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("missing original mobile headers")
		}
	}
	resp, err := client.Get(base + "/pair/owned#offer=synthetic-only")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	want, _ := os.ReadFile(filepath.Join(root, "index.html"))
	if resp.StatusCode != 200 || !bytes.Equal(got, want) || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("SPA fallback differs from original index")
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"HEAD", "/index.html", 200}, {"POST", "/index.html", 405}, {"GET", "/%2e%2e/package.json", 404}, {"GET", "/missing-asset.js", 404}} {
		req, _ := http.NewRequest(tc.method, base+tc.path, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != tc.status || (tc.method == "HEAD" && len(body) != 0) {
			t.Fatalf("%s %s status=%d", tc.method, tc.path, res.StatusCode)
		}
	}
	if err = CloseMobileWebServer(server); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-done:
		if err != http.ErrServerClosed {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("owned mobile service did not stop")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	c, err := (&net.Dialer{}).DialContext(ctx, "tcp", l.Addr().String())
	if err == nil {
		c.Close()
		t.Fatal("mobile listener survived close")
	}
	t.Log("real loopback HTTP: exact original HTML/JS/CSS/module bytes; SPA fallback/HEAD/method/traversal headers; owned listener shutdown; no daemon/relay/device pairing")
}
