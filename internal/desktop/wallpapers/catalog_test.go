package wallpapers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogURLAndSSRFBlocklist(t *testing.T) {
	rejected := []string{
		"",
		"http://example.com/wallpapers.json",
		"ftp://example.com/wallpapers.json",
		"https://user:pass@example.com/wallpapers.json",
		"https://user@example.com/wallpapers.json",
		"https://localhost/wallpapers.json",
		"https://localhost./wallpapers.json",
		"https://sub.localhost/wallpapers.json",
		"https://printer.local/wallpapers.json",
		"https://api.internal/wallpapers.json",
		"https://127.0.0.1/wallpapers.json",
		"https://0.0.0.0/wallpapers.json",
		"https://10.0.0.1/wallpapers.json",
		"https://100.64.0.1/wallpapers.json",
		"https://100.127.255.254/wallpapers.json",
		"https://169.254.169.254/latest/meta-data",
		"https://172.16.0.1/wallpapers.json",
		"https://192.0.0.1/wallpapers.json",
		"https://192.0.2.1/wallpapers.json",
		"https://192.88.99.1/wallpapers.json",
		"https://192.168.1.1/wallpapers.json",
		"https://198.18.0.1/wallpapers.json",
		"https://198.19.255.254/wallpapers.json",
		"https://198.51.100.1/wallpapers.json",
		"https://203.0.113.1/wallpapers.json",
		"https://224.0.1.1/wallpapers.json",
		"https://239.255.255.250/wallpapers.json",
		"https://240.0.0.1/wallpapers.json",
		"https://255.255.255.255/wallpapers.json",
		"https://[::]/wallpapers.json",
		"https://[::1]/wallpapers.json",
		"https://[fc00::1]/wallpapers.json",
		"https://[fd12::1]/wallpapers.json",
		"https://[fe80::1]/wallpapers.json",
		"https://[fec0::1]/wallpapers.json",
		"https://[ff02::1]/wallpapers.json",
		"https://[ff0e::1]/wallpapers.json",
		"https://[100::1]/wallpapers.json",
		"https://[2001:db8::1]/wallpapers.json",
		"https://[2002:7f00:1::]/wallpapers.json",
		"https://[64:ff9b:1::1]/wallpapers.json",
		"https://[::ffff:127.0.0.1]/wallpapers.json",
		"https://[::ffff:10.0.0.1]/wallpapers.json",
		"https://[::10.0.0.1]/wallpapers.json",
		"https://[::127.0.0.1]/wallpapers.json",
		"https://[64:ff9b::10.0.0.1]/wallpapers.json",
		"https://[64:ff9b::100.64.0.1]/wallpapers.json",
		"https://example.com/" + strings.Repeat("a", 2000),
	}
	for _, raw := range rejected {
		if _, err := URL(raw); err == nil {
			t.Fatalf("URL(%q) succeeded, expected rejection", raw)
		}
	}
	if !blocked("fe80::1%eth0") {
		t.Fatal("blocked failed to reject IPv6 zone-id host")
	}

	accepted := []string{
		"https://example.com/wallpapers.json",
		"https://1.1.1.1/wallpapers.json",
		"https://[2606:4700:4700::1111]/wallpapers.json",
		"https://[::ffff:1.1.1.1]/wallpapers.json",
		"https://[64:ff9b::1.1.1.1]/wallpapers.json",
	}
	for _, raw := range accepted {
		if _, err := URL(raw); err != nil {
			t.Fatalf("URL(%q) failed unexpectedly: %v", raw, err)
		}
	}
}

func TestCatalogFetchBlocksLoopbackByDefaultAndParsesInTestHTTP(t *testing.T) {
	if _, err := Fetch(context.Background(), "https://127.0.0.1/catalog.json", false); err == nil {
		t.Fatal("Fetch allowed loopback without DSHD_WALLPAPER_ALLOW_HTTP=1")
	}

	t.Setenv("DSHD_WALLPAPER_ALLOW_HTTP", "1")
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"items":[{"id":"c1","title":"Coast","thumbUrl":"%s/thumb.jpg","imageUrl":"%s/full.jpg"}]}`, srv.URL, srv.URL)
		case "/redirect-loop":
			http.Redirect(w, r, srv.URL+"/redirect-loop", http.StatusFound)
		case "/full.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte{0xff, 0xd8, 0xff, 0xd9})
		case "/not-image":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	res := List(context.Background(), Query{Source: Source{ID: "custom-1", Kind: "catalog", URL: srv.URL + "/catalog.json"}})
	if res.Warning != "" || len(res.Items) != 1 || res.Items[0].ID != "c1" {
		t.Fatalf("custom catalog list = %+v", res)
	}
	if _, err := Fetch(context.Background(), srv.URL+"/redirect-loop", false); err == nil {
		t.Fatal("expected >4 redirects to fail")
	}
	if _, err := Fetch(context.Background(), srv.URL+"/not-image", true); err == nil {
		t.Fatal("expected non-image Content-Type to fail")
	}
	img, err := Fetch(context.Background(), srv.URL+"/full.jpg", true)
	if err != nil || len(img) != 4 {
		t.Fatalf("Fetch image = %d bytes, err=%v", len(img), err)
	}
}
