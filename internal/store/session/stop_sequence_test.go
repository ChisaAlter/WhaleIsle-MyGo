package session

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestStopSequenceHostBinding(t *testing.T) {
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		switch r.Header.Get("Cookie") {
		case "host=first":
			w.Write([]byte(`<script>globalThis["__DSH_SHORTCUTS_CONFIG__"] = {"stopSequenceMs":800}</script>`))
		case "host=second":
			w.Write([]byte(`<script>globalThis["__DSH_SHORTCUTS_CONFIG__"] = {"stopSequenceMs":1200}</script>`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()
	store := New()
	changed := make(chan struct{}, 4)
	notify := func() { changed <- struct{}{} }
	await := func() {
		t.Helper()
		select {
		case <-changed:
		case <-time.After(2 * time.Second):
			t.Fatal("bootstrap did not publish")
		}
	}
	first := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL, Cookie: "host=first"}}
	store.Attach(first, notify)
	store.LoadStopSequenceConfig()
	await()
	if got := store.Snapshot().StopSequenceMS; got != 800 {
		t.Fatalf("first Host interval: %d", got)
	}
	store.LoadStopSequenceConfig()
	if reads.Load() != 1 {
		t.Fatal("same binding fetched bootstrap again")
	}
	second := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL, Cookie: "host=second"}}
	store.Attach(second, notify)
	if got := store.Snapshot().StopSequenceMS; got != 0 {
		t.Fatal("new binding retained old Host interval")
	}
	store.LoadStopSequenceConfig()
	await()
	if got := store.Snapshot().StopSequenceMS; got != 1200 || reads.Load() != 2 {
		t.Fatal("new binding did not adopt its interval")
	}
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, notify)
	store.LoadStopSequenceConfig()
	await()
	if store.Snapshot().StopSequenceMS != 0 {
		t.Fatal("failed authenticated bootstrap enabled sequence")
	}
}
