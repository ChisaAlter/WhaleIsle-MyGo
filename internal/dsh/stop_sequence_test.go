package dsh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStopSequenceConfiguration(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int64
		bad   bool
	}{{`{}`, 500, false}, {`{"stopSequenceMs":800}`, 800, false}, {`{"stopSequenceMs":2147483646}`, 2147483646, false}, {`{"stopSequenceMs":0}`, 0, true}, {`{"stopSequenceMs":-1}`, 0, true}, {`{"stopSequenceMs":1.5}`, 0, true}, {`{"stopSequenceMs":2147483647}`, 0, true}, {`{"stopSequenceMs":null}`, 0, true}, {`{"stopSequenceMs":500}; evil()`, 0, true}} {
		got, err := parseStopSequenceInterval(`<html><head><script>globalThis["__DSH_SHORTCUTS_CONFIG__"] = ` + tc.value + `</script></head></html>`)
		if (err != nil) != tc.bad || !tc.bad && got != tc.want {
			t.Fatalf("%s: %d %v", tc.value, got, err)
		}
	}
	if got, err := parseStopSequenceInterval(`<html><script>unrelated()</script></html>`); err != nil || got != 500 {
		t.Fatal("missing injection default")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/" || r.Header.Get("Cookie") != "fixture=ok" {
			t.Errorf("bootstrap request lost binding")
		}
		w.Write([]byte(`<script>globalThis["__DSH_SHORTCUTS_CONFIG__"] = {"stopSequenceMs":800}</script>`))
	}))
	defer server.Close()
	client := &Client{Endpoint: Endpoint{Origin: server.URL, Cookie: "fixture=ok"}}
	if got, err := client.ReadStopSequenceInterval(context.Background()); err != nil || got != 800 {
		t.Fatalf("authenticated bootstrap: %d %v", got, err)
	}
	foreignHits := 0
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { foreignHits++ }))
	defer foreign.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, foreign.URL, http.StatusFound) }))
	defer redirect.Close()
	client.Endpoint.Origin = redirect.URL
	if _, err := client.ReadStopSequenceInterval(context.Background()); err == nil || foreignHits != 0 {
		t.Fatal("cross-origin bootstrap not rejected before request")
	}
}
