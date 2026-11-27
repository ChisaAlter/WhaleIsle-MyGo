package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestModelCatalogPendingTracksInitializeDefaultModelLifecycle(t *testing.T) {
	initEntered := make(chan struct{}, 1)
	releaseInit := make(chan struct{})
	catalogEntered := make(chan struct{}, 1)
	releaseCatalog := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/initializeDefaultModel"):
			select {
			case initEntered <- struct{}{}:
			default:
			}
			<-releaseInit
			rpcOK(w, body["rpcId"], map[string]any{})
		case strings.HasSuffix(r.URL.Path, "/session/modelCatalog"):
			select {
			case catalogEntered <- struct{}{}:
			default:
			}
			<-releaseCatalog
			rpcOK(w, body["rpcId"], map[string]any{
				"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-chat"},
				"groups": []map[string]any{{
					"id":   "deepseek-official",
					"name": "DeepSeek",
					"models": []map[string]any{{
						"id":   "deepseek-chat",
						"name": "DeepSeek Chat",
					}},
				}},
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, nil)
	defer s.Detach()

	if s.ModelCatalogPending() {
		t.Fatal("ModelCatalogPending should be false before InitializeDefaultModel starts")
	}

	s.InitializeDefaultModel(context.Background())
	if !s.ModelCatalogPending() {
		t.Fatal("ModelCatalogPending should be true immediately after InitializeDefaultModel")
	}

	select {
	case <-initEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("initializeDefaultModel RPC was not called")
	}
	if !s.ModelCatalogPending() {
		t.Fatal("ModelCatalogPending dropped while initializeDefaultModel was in flight")
	}

	close(releaseInit)
	select {
	case <-catalogEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("modelCatalog RPC was not called after initializeDefaultModel")
	}
	if !s.ModelCatalogPending() {
		t.Fatal("ModelCatalogPending dropped while modelCatalog was in flight")
	}

	close(releaseCatalog)
	waitFor(t, "model catalog settled", func() bool { return !s.ModelCatalogPending() })
	if _, ready := s.CachedModelCatalog(client); !ready {
		t.Fatal("expected CachedModelCatalog ready once ModelCatalogPending settled")
	}
}

func TestRefreshModelCatalogPublishesLoadingStateAcrossEmptyReloads(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/modelCatalog") {
			entered <- struct{}{}
			<-release
			rpcOK(w, body["rpcId"], map[string]any{
				"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-flash"},
				"groups":  []any{},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	var notifications atomic.Int32
	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, func() { notifications.Add(1) })
	defer s.Detach()

	// First reload on an empty catalog publishes ModelCatalogLoading=true immediately and clears it on settle.
	s.RefreshModelCatalog(context.Background())
	if !s.Snapshot().ModelCatalogLoading {
		t.Fatal("ViewState.ModelCatalogLoading should be true immediately when RefreshModelCatalog starts")
	}
	if notifications.Load() == 0 {
		t.Fatal("RefreshModelCatalog did not fire notify on loading start")
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first modelCatalog RPC was not called")
	}
	release <- struct{}{}
	waitFor(t, "first empty catalog reload settled", func() bool { return !s.Snapshot().ModelCatalogLoading })

	// Repeated reload on an already-empty catalog must still toggle ModelCatalogLoading true -> false.
	before := notifications.Load()
	s.RefreshModelCatalog(context.Background())
	if !s.Snapshot().ModelCatalogLoading {
		t.Fatal("second RefreshModelCatalog did not publish ModelCatalogLoading=true")
	}
	if notifications.Load() <= before {
		t.Fatal("second RefreshModelCatalog did not notify UI on start")
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("second modelCatalog RPC was not called")
	}
	release <- struct{}{}
	waitFor(t, "second empty catalog reload settled", func() bool { return !s.Snapshot().ModelCatalogLoading })
	if len(s.Snapshot().ModelLabels) != 0 {
		t.Fatalf("expected empty catalog, got %v", s.Snapshot().ModelLabels)
	}
}

func TestLoadNativeCommandModelClearsInFlightRefreshModelCatalogLoading(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	for _, tc := range []struct {
		name         string
		failSecond   bool
		expectErr    bool
		expectLabels int
	}{
		{name: "native-command-succeeds", failSecond: false, expectErr: false, expectLabels: 1},
		{name: "native-command-fails", failSecond: true, expectErr: true, expectLabels: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var callCount atomic.Int32
			firstEntered := make(chan struct{}, 1)
			releaseFirst := make(chan struct{})

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.HasSuffix(r.URL.Path, "/session/modelCatalog") {
					n := callCount.Add(1)
					if n == 1 {
						select {
						case firstEntered <- struct{}{}:
						default:
						}
						<-releaseFirst
						rpcOK(w, body["rpcId"], map[string]any{
							"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-flash"},
							"groups":  []any{},
						})
						return
					}
					if tc.failSecond {
						http.Error(w, "catalog unavailable", http.StatusBadGateway)
						return
					}
					rpcOK(w, body["rpcId"], map[string]any{
						"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-chat"},
						"groups": []map[string]any{{
							"id":   "deepseek-official",
							"name": "DeepSeek",
							"models": []map[string]any{{
								"id":   "deepseek-chat",
								"name": "DeepSeek Chat",
							}},
						}},
					})
					return
				}
				rpcOK(w, body["rpcId"], map[string]any{})
			}))
			defer srv.Close()

			client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
			defer client.Close()
			s := nativeSelectionStore(client, graph)
			var notifyCount atomic.Int32
			s.mu.Lock()
			s.notify = func() { notifyCount.Add(1) }
			s.mu.Unlock()

			// Start RefreshModelCatalog so CatalogLoading=true and CatalogRequest=R1.
			s.RefreshModelCatalog(context.Background())
			if !s.Snapshot().ModelCatalogLoading {
				t.Fatal("ModelCatalogLoading should be true while RefreshModelCatalog is in flight")
			}
			select {
			case <-firstEntered:
			case <-time.After(3 * time.Second):
				t.Fatal("RefreshModelCatalog RPC did not start")
			}

			// Interleave /model LoadNativeCommandOptions, which advances CatalogRequest to R2.
			beforeNotify := notifyCount.Load()
			rows, err := s.LoadNativeCommandOptions(context.Background(), nativeSelectionOwner(s, "model"), "model")
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error from failing LoadNativeCommandOptions")
				}
			} else {
				if err != nil {
					t.Fatalf("LoadNativeCommandOptions failed: %v", err)
				}
				if len(rows) != 1 {
					t.Fatalf("got %d native model rows, want 1", len(rows))
				}
			}

			// Even before the stale R1 RefreshModelCatalog finishes, ModelCatalogLoading
			// must have been cleared and the UI notified.
			if s.Snapshot().ModelCatalogLoading {
				t.Fatal("ModelCatalogLoading remained stuck true after interleaved LoadNativeCommandOptions")
			}
			if notifyCount.Load() <= beforeNotify {
				t.Fatal("LoadNativeCommandOptions did not fire UI notification when clearing ModelCatalogLoading")
			}

			close(releaseFirst)
			// Wait briefly for the superseded RefreshModelCatalog goroutine to exit cleanly.
			time.Sleep(30 * time.Millisecond)
			if s.Snapshot().ModelCatalogLoading {
				t.Fatal("stale RefreshModelCatalog re-stuck ModelCatalogLoading after returning")
			}
			if got := len(s.Snapshot().ModelLabels); got != tc.expectLabels {
				t.Fatalf("ModelLabels len = %d, want %d", got, tc.expectLabels)
			}
		})
	}
}

func TestBeginEventsReconnectClearsInFlightBootstrapAndCatalogLoading(t *testing.T) {
	initEntered := make(chan struct{}, 1)
	releaseInit := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/initializeDefaultModel"):
			select {
			case initEntered <- struct{}{}:
			default:
			}
			<-releaseInit
			rpcOK(w, body["rpcId"], map[string]any{})
		case strings.HasSuffix(r.URL.Path, "/session/modelCatalog"):
			rpcOK(w, body["rpcId"], map[string]any{
				"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-chat"},
				"groups": []map[string]any{{
					"id":   "deepseek-official",
					"name": "DeepSeek",
					"models": []map[string]any{{
						"id":   "deepseek-chat",
						"name": "DeepSeek Chat",
					}},
				}},
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	s := New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	s.Attach(client, nil)
	defer s.Detach()

	s.InitializeDefaultModel(context.Background())
	select {
	case <-initEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("initializeDefaultModel RPC did not start")
	}
	if !s.Snapshot().ModelCatalogLoading || !s.ModelCatalogPending() {
		t.Fatal("expected ModelCatalogLoading and ModelCatalogPending true while bootstrap is in flight")
	}

	// Simulate $events reconnect bumping eventsGeneration while bootstrap is in flight.
	if owner := s.BeginEvents(context.Background(), client); owner == nil {
		t.Fatal("BeginEvents returned nil")
	}
	if s.Snapshot().ModelCatalogLoading || s.ModelCatalogPending() {
		t.Fatal("BeginEvents reconnect did not clear stale BootstrapPending/CatalogLoading")
	}

	// Release stale pre-reconnect RPC; it must not re-stick loading.
	close(releaseInit)
	time.Sleep(30 * time.Millisecond)
	if s.Snapshot().ModelCatalogLoading || s.ModelCatalogPending() {
		t.Fatal("stale pre-reconnect InitializeDefaultModel left loading stuck true")
	}

	// Subsequent RefreshModelCatalog on the new generation must settle cleanly.
	s.RefreshModelCatalog(context.Background())
	waitFor(t, "post-reconnect RefreshModelCatalog settled", func() bool {
		_, ready := s.CachedModelCatalog(client)
		return ready && !s.Snapshot().ModelCatalogLoading && !s.ModelCatalogPending()
	})
}
