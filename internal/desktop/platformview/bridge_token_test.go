package platformview

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPlatformPreloadUsesBridgeTokenOnly(t *testing.T) {
	const accountToken = "super-secret-account-session-token"
	var seenAuth []string
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	g, err := New(Session{Origin: upstream.URL, Token: accountToken}, ClientMetadata{Locale: "en", Platform: "desktop-win"})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	g.transport.TLSClientConfig = upstream.Client().Transport.(*http.Transport).TLSClientConfig.Clone()

	if len(g.bridgeToken) != 64 || g.bridgeToken == accountToken || g.bridgeToken == g.key {
		t.Fatalf("unexpected bridge token: %q", g.bridgeToken)
	}

	preload := g.Preload("en")
	if strings.Contains(preload, accountToken) {
		t.Fatal("preload leaked upstream account session token")
	}
	if !strings.Contains(preload, g.bridgeToken) {
		t.Fatal("preload missing dedicated bridge token")
	}

	bare := &Gateway{origin: g.origin, session: Session{Origin: upstream.URL, Token: accountToken}}
	if strings.Contains(bare.Preload("en"), accountToken) {
		t.Fatal("bare gateway preload leaked upstream account session token")
	}

	client := &http.Client{}
	send := func(auth string) int {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, g.origin+"/api/usage", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: g.cookieName, Value: g.key})
		req.Header.Set("Origin", g.origin)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	if code := send("Bearer " + g.bridgeToken); code != http.StatusOK {
		t.Fatalf("bridge token request status = %d", code)
	}
	if code := send("Bearer " + accountToken); code != http.StatusOK {
		t.Fatalf("compat account token request status = %d", code)
	}
	if code := send("Bearer wrong-token"); code != http.StatusOK {
		t.Fatalf("wrong token request status = %d", code)
	}
	if code := send(""); code != http.StatusOK {
		t.Fatalf("unauthenticated request status = %d", code)
	}

	wantAuth := []string{"Bearer " + accountToken, "Bearer " + accountToken, "", ""}
	if len(seenAuth) != len(wantAuth) {
		t.Fatalf("upstream saw %v, want %v", seenAuth, wantAuth)
	}
	for i, want := range wantAuth {
		if seenAuth[i] != want {
			t.Fatalf("request %d upstream auth = %q, want %q", i, seenAuth[i], want)
		}
	}

	for _, leaked := range []string{
		"https://pay.example/?token=" + g.bridgeToken,
		"https://pay.example/?token=" + url.QueryEscape(g.bridgeToken),
		"https://pay.example/?token=" + accountToken,
		"https://pay.example/?token=" + g.key,
	} {
		if _, ok := g.ExternalURL(leaked); ok {
			t.Fatalf("ExternalURL allowed leaked secret URL: %s", leaked)
		}
	}
	if got, ok := g.ExternalURL("https://pay.example/?ok=1"); !ok || got != "https://pay.example/?ok=1" {
		t.Fatalf("ExternalURL rejected clean URL: %q %v", got, ok)
	}
}
