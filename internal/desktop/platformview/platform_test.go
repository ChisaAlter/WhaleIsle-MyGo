package platformview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlatformSessionScope(t *testing.T) {
	good := Session{Origin: "https://account.example", Token: "fixture-only"}
	for _, origin := range []string{"http://account.example", "https://u:p@account.example", "https://account.example/path", "https://account.example?x=1"} {
		s := good
		s.Origin = origin
		if s.Validate() == nil {
			t.Fatal("accepted invalid origin")
		}
	}
	s := good
	s.RequestHeaders = map[string]string{"Host": "other.example"}
	if s.Validate() == nil {
		t.Fatal("accepted authority override")
	}
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dshd-platform/session" || r.Header.Get("Authorization") != "Bearer publisher-fixture" {
			t.Error("wrong scoped request")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"session": good})
	}))
	defer host.Close()
	got, err := Fetch(context.Background(), host.URL, "publisher-fixture")
	if err != nil || got.Token != good.Token {
		t.Fatalf("session fixture: %v", err)
	}
	if _, err := Fetch(context.Background(), "https://external.example", "fixture"); err == nil {
		t.Fatal("accepted non-loopback Host")
	}
}
func TestPlatformGatewayIsolation(t *testing.T) {
	requests := 0
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Cookie") != "deployment=fixture" {
			t.Error("deployment cookie missing or lease leaked")
		}
		if r.URL.Path == "/api" && r.Header.Get("Authorization") != "Bearer fixture-only" {
			t.Error("matched account token missing")
		}
		if r.Header.Get("x-client-locale") != "en_US" {
			t.Error("locale mismatch")
		}
		w.Header().Set("Set-Cookie", "untrusted=1")
		_, _ = w.Write([]byte("fixture"))
	}))
	defer upstream.Close()
	g, err := New(Session{Origin: upstream.URL, Token: "fixture-only", RequestHeaders: map[string]string{"Cookie": "deployment=fixture"}}, ClientMetadata{Locale: "en", Platform: "desktop-win"})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	g.transport.TLSClientConfig = upstream.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	entry, _ := g.Entry("usage")
	response, err := client.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal(response.StatusCode)
	}
	req, _ := http.NewRequest("GET", g.origin+"/api", nil)
	req.Header.Set("Authorization", "Bearer fixture-only")
	response, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.Header.Get("Set-Cookie") != "" {
		t.Fatal("upstream cookie escaped")
	}
	req, _ = http.NewRequest("GET", g.origin+"/api", nil)
	req.Header.Set("Origin", "https://other.example")
	response, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("cross-origin request accepted")
	}
	response, err = http.Get(g.origin + "/api")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("unauthenticated request accepted")
	}
	if requests != 2 {
		t.Fatalf("unexpected upstream requests: %d", requests)
	}
	if g.Allows("https://other.example") || !g.Allows(g.origin+"/usage") {
		t.Fatal("navigation scope")
	}
	if !strings.Contains(g.Preload("en"), "window!==window.top") {
		t.Fatal("missing main-frame gate")
	}
	g.Close()
	if _, err := client.Get(entry); err == nil {
		t.Fatal("closed lease remained accessible")
	}
}
