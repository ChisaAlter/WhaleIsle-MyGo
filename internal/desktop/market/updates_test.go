package market

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestLatestVersionUsesOnlyVerifiedRegistryMetadata(t *testing.T) {
	item := CatalogItem{ID: "owner/demo", PackageName: "demo", InstallSpec: "demo", Screenshots: []string{"https://example.invalid/one.png"}}
	calls := 0
	svc := NewServiceWithClient("", &http.Client{Transport: updateTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://registry.npmjs.org/demo/latest" {
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"demo","version":"1.2.3"}`)), Header: make(http.Header)}, nil
	})})
	svc.known[item.ID] = item
	reviewed := item
	reviewed.Screenshots = append([]string(nil), item.Screenshots...)
	v, err := svc.LatestVersion(context.Background(), reviewed)
	if err != nil || v != "1.2.3" || calls != 1 {
		t.Fatalf("%q %v calls=%d", v, err, calls)
	}
	item.Screenshots = []string{"https://example.invalid/replaced.png"}
	if _, err = svc.LatestVersion(context.Background(), item); err == nil || calls != 1 {
		t.Fatal("changed reviewed screenshot metadata fetched")
	}
	item.Screenshots = reviewed.Screenshots
	item.InstallSpec = "github:owner/demo"
	if _, err = svc.LatestVersion(context.Background(), item); err == nil || calls != 1 {
		t.Fatal("unverified target fetched")
	}
}
func TestCompareStableVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b  string
		want  int
		known bool
	}{
		{"1.9.0", "1.10.0", -1, true}, {"2.0.0", "1.0.0", 1, true}, {"1.0.0", "1.0.0", 0, true},
		{"", "1.0.0", 0, false}, {"dev", "1.0.0", 0, false}, {"1.0.0-beta", "1.0.0", -1, true}, {"01.0.0", "1.0.0", 0, false},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1, true}, {"1.0.0-alpha.2", "1.0.0-alpha.11", -1, true},
		{"1.0.0-alpha.11", "1.0.0-beta", -1, true}, {"1.0.0-rc.1", "1.0.0", -1, true},
		{"1.0.0+build.1", "1.0.0+build.2", 0, true}, {"1.0.0-01", "1.0.0", 0, false},
		{"1.0.0-alpha..1", "2.0.0", 0, false},
	} {
		got, known := CompareStableVersions(tc.a, tc.b)
		if got != tc.want || known != tc.known {
			t.Fatalf("%+v: %d %v", tc, got, known)
		}
	}
}
