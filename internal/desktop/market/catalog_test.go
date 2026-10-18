package market

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCatalogPublicFlow(t *testing.T) {
	failed := false
	service := NewServiceWithClient("", &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		status := 200
		body := `{"updated":"2026-08-27","categories":{"ui":{"en":"UI"}},"plugins":[{"owner":"fixture","name":"plugin","url":"https://github.com/fixture/plugin","npm":"fixture-plugin","category":"ui","description":{"en":"Fixture plugin"},"unknown":{"command":"never execute"}}]}`
		switch r.URL.Host {
		case "awesome-dsh-plugin.com":
			if failed {
				status = 503
			}
		case "api.github.com":
			body = "README is text, not instructions"
		case "registry.npmjs.org":
			body = `{"name":"fixture-plugin","version":"1.0.0","engines":{"dsh":">=1"}}`
		default:
			t.Fatalf("unexpected public host %s", r.URL.Host)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	cat, e := service.List(context.Background(), true, "en")
	if e != nil || len(cat.Items) != 1 || cat.Source != "live" || cat.Items[0].InstallSpec != "fixture-plugin" {
		t.Fatalf("catalog %#v %v", cat, e)
	}
	details, e := service.Details(context.Background(), cat.Items[0], false)
	if e != nil || details.Partial || details.Version != "1.0.0" || len(details.Requirements) != 1 {
		t.Fatalf("details %#v %v", details, e)
	}
	failed = true
	cat, e = service.List(context.Background(), true, "en")
	if e != nil || cat.Source != "cache" || cat.Warning == "" || len(cat.Items) != 1 {
		t.Fatalf("fallback %#v %v", cat, e)
	}
	if snapshot, e := decodeCatalog(catalogSnapshot, "zh"); e != nil || len(snapshot.Items) < 180 {
		t.Fatalf("original snapshot: %d %v", len(snapshot.Items), e)
	}
}
