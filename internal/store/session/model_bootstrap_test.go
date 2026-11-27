package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
