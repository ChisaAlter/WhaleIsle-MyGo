package imgcache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsHTTPURLRejectsUserinfo(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://example.com/a.png", true},
		{"http://example.com/a.png", true},
		{"https://user@example.com/a.png", false},
		{"http://user:pass@example.com/a.png", false},
		{"file:///C:/temp/a.png", false},
	}
	for _, tc := range cases {
		if got := IsHTTPURL(tc.raw); got != tc.want {
			t.Errorf("IsHTTPURL(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestCheckRedirectSafety(t *testing.T) {
	mustURL := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	publicOrigin := &http.Request{URL: mustURL("https://example.com/start.png")}
	loopbackOrigin := &http.Request{URL: mustURL("http://127.0.0.1:8080/start.png")}

	if err := checkRedirectSafety(&http.Request{URL: mustURL("https://cdn.example.com/next.png")}, []*http.Request{publicOrigin}); err != nil {
		t.Fatalf("public -> public redirect rejected: %v", err)
	}
	if err := checkRedirectSafety(&http.Request{URL: mustURL("http://127.0.0.1:8080/next.png")}, []*http.Request{loopbackOrigin}); err != nil {
		t.Fatalf("loopback -> loopback redirect rejected: %v", err)
	}

	blocked := []struct {
		name string
		req  *http.Request
		via  []*http.Request
	}{
		{"too many redirects", &http.Request{URL: mustURL("https://example.com/6.png")}, make([]*http.Request, 5)},
		{"userinfo in redirect", &http.Request{URL: mustURL("https://user:pass@example.com/x.png")}, []*http.Request{publicOrigin}},
		{"non-http scheme", &http.Request{URL: mustURL("file:///etc/passwd")}, []*http.Request{publicOrigin}},
		{"link-local metadata", &http.Request{URL: mustURL("http://169.254.169.254/latest")}, []*http.Request{loopbackOrigin}},
		{"public to loopback", &http.Request{URL: mustURL("http://127.0.0.1/admin")}, []*http.Request{publicOrigin}},
		{"public to private", &http.Request{URL: mustURL("http://10.0.0.1/secret")}, []*http.Request{publicOrigin}},
	}
	for _, tc := range blocked {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkRedirectSafety(tc.req, tc.via); err == nil {
				t.Fatalf("checkRedirectSafety(%s) = nil, want block", tc.name)
			}
		})
	}
}

func TestFetchBlocksUnsafeRedirectWithoutTempLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := New(dir)
	c.client = srv.Client()

	if _, err := c.Fetch(context.Background(), srv.URL+"/avatar.png"); err == nil {
		t.Fatal("Fetch succeeded across unsafe metadata redirect, want error")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") || filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatalf("unexpected temp artifact left behind: %s", entry.Name())
		}
	}
}
