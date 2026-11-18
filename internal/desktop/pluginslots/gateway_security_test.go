package pluginslots

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/coder/websocket"
)

func TestPluginSlotGatewayHardeningAndRuntimeCompatibility(t *testing.T) {
	var mu sync.Mutex
	var hits []string
	var lastCookie, lastAuth, lastForwarded, lastXFF string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits = append(hits, r.URL.Path)
		lastCookie = r.Header.Get("Cookie")
		lastAuth = r.Header.Get("Authorization")
		lastForwarded = r.Header.Get("Forwarded")
		lastXFF = r.Header.Get("X-Forwarded-For")
		mu.Unlock()

		if r.URL.Path == "/api/remote.mux" && strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				return
			}
			defer conn.CloseNow()
			kind, data, err := conn.Read(context.Background())
			if err == nil {
				_ = conn.Write(context.Background(), kind, data)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	g, err := NewGatewayWithResponseFilter(upstream.URL, "sid=host-cookie", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	entry, err := g.EntryURL(upstream.URL + "/?desktopNativePlugins=1")
	if err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	noRedirect := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	authResp, err := noRedirect.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	_ = authResp.Body.Close()
	if authResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("entry auth status = %d", authResp.StatusCode)
	}
	if authResp.Header.Get("X-Content-Type-Options") != "nosniff" || authResp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("auth response missing security headers: %v", authResp.Header)
	}

	client := &http.Client{Jar: jar}
	assertBlocked := func(method, path string, headers map[string]string) {
		t.Helper()
		req, err := http.NewRequest(method, g.browserOrigin+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s %s (headers=%v) status = %d, want 403", method, path, headers, resp.StatusCode)
		}
		if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("X-Frame-Options") != "DENY" {
			t.Fatalf("403 response for %s missing security headers: %v", path, resp.Header)
		}
	}

	for _, path := range []string{
		"/dshd-platform",
		"/dshd-platform/",
		"/dshd-platform/session",
		"/DSHD-PLATFORM/session",
		"/plugins/../dshd-platform/session",
		"/dshd-platform%2fsession",
		"/connect",
		"/connect/",
		"/connect/ws",
		"/plugins/../connect",
		"/CONNECT",
	} {
		assertBlocked(http.MethodGet, path, nil)
	}

	assertBlocked(http.MethodConnect, "/api/rpc", nil)
	assertBlocked(http.MethodGet, "/api/rpc", map[string]string{"Upgrade": "h2c"})
	assertBlocked(http.MethodGet, "/api/rpc", map[string]string{"Upgrade": "websocket, h2c"})
	assertBlocked(http.MethodPost, "/api/remote.mux", map[string]string{"Upgrade": "websocket"})
	assertBlocked(http.MethodGet, "/__whale_slot_auth?key="+url.QueryEscape(g.token)+"&next=%2F", map[string]string{"Upgrade": "websocket"})
	assertBlocked(http.MethodGet, "/api/rpc", map[string]string{"Sec-Fetch-Site": "cross-site"})
	assertBlocked(http.MethodGet, "/api/rpc", map[string]string{"Sec-Fetch-Site": "same-site"})
	assertBlocked(http.MethodGet, "/api/rpc", map[string]string{"Sec-Fetch-Site": "Same-Site"})

	mu.Lock()
	blockedHits := len(hits)
	mu.Unlock()
	if blockedHits != 0 {
		t.Fatalf("blocked requests leaked to upstream: %v", hits)
	}

	for _, tc := range []struct {
		path      string
		fetchSite string
	}{
		{"/api/rpc", ""},
		{"/$events", "same-origin"},
		{"/api/connect-health", "none"},
	} {
		req, err := http.NewRequest(http.MethodGet, g.browserOrigin+tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", g.browserOrigin)
		if tc.fetchSite != "" {
			req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
		}
		req.Header.Set("Authorization", "Bearer rogue")
		req.Header.Set("Forwarded", "for=1.2.3.4")
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("allowed path %s status = %d", tc.path, resp.StatusCode)
		}
		if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("X-Frame-Options") != "DENY" {
			t.Fatalf("proxied response for %s missing security headers: %v", tc.path, resp.Header)
		}
	}

	mu.Lock()
	if lastCookie != "sid=host-cookie" || lastAuth != "" || lastForwarded != "" || lastXFF != "" {
		t.Fatalf("unexpected upstream headers: cookie=%q auth=%q forwarded=%q xff=%q", lastCookie, lastAuth, lastForwarded, lastXFF)
	}
	mu.Unlock()

	baseURL, _ := url.Parse(g.browserOrigin)
	wsURL, _ := url.Parse(g.browserOrigin + "/api/remote.mux")
	wsURL.Scheme = "ws"
	wsHeader := http.Header{}
	wsHeader.Set("Origin", g.browserOrigin)
	wsHeader.Set("Sec-Fetch-Site", "same-origin")
	for _, c := range jar.Cookies(baseURL) {
		wsHeader.Add("Cookie", c.Name+"="+c.Value)
	}
	conn, _, err := websocket.Dial(context.Background(), wsURL.String(), &websocket.DialOptions{HTTPHeader: wsHeader})
	if err != nil {
		t.Fatalf("remote.mux websocket dial failed: %v", err)
	}
	defer conn.CloseNow()
	if err := conn.Write(context.Background(), websocket.MessageText, []byte("mux-ping")); err != nil {
		t.Fatal(err)
	}
	_, msg, err := conn.Read(context.Background())
	if err != nil || string(msg) != "mux-ping" {
		t.Fatalf("remote.mux websocket echo = %q err=%v", string(msg), err)
	}
}
