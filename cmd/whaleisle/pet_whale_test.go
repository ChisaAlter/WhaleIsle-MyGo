package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"whaleisle/internal/desktop/whale"
	"whaleisle/internal/harness"
)

func TestWhaleCatalogCacheFollowsTheHost(t *testing.T) {
	ep := harness.Endpoint{Origin: "http://127.0.0.1:8787", Cookie: "dsh-auth=one"}
	s := &mainState{}
	s.whaleCatalog.key = whaleHostKey(ep)
	s.whaleCatalog.snap = &whale.Catalog{Name: "鲸鱼娘"}
	if snap, _ := s.whaleCatalogSnapshot(ep); snap == nil || snap.Name != "鲸鱼娘" {
		t.Fatalf("same Host lost its snapshot: %+v", snap)
	}
	for _, other := range []harness.Endpoint{
		{Origin: "http://127.0.0.1:8788", Cookie: "dsh-auth=one"},
		{Origin: "http://127.0.0.1:8787", Cookie: "dsh-auth=two"},
	} {
		if snap, _ := s.whaleCatalogSnapshot(other); snap != nil {
			t.Fatalf("snapshot leaked across Hosts: %+v for %+v", snap, other)
		}
	}
}

func TestWhalePetAccessIsNilUntilTheHostIsReady(t *testing.T) {
	s := &mainState{}
	ep := harness.Endpoint{Origin: "http://127.0.0.1:8787"}
	for _, tc := range []struct {
		name  string
		ep    harness.Endpoint
		ready bool
	}{
		{"boot", ep, false},
		{"no origin", harness.Endpoint{}, true},
		{"bad origin", harness.Endpoint{Origin: "ftp://127.0.0.1:1"}, true},
	} {
		snapshot, load, update, replace, clear, toggle := s.whalePetAccess(tc.ep, tc.ready)
		if snapshot != nil || load != nil || update != nil || replace != nil || clear != nil || toggle != nil {
			t.Fatalf("%s: access not nil when the Host cannot answer", tc.name)
		}
	}
}

func TestWhalePetAccessLoadsTheCatalogInTheBackground(t *testing.T) {
	var payload map[string]any
	_ = json.Unmarshal([]byte(`{"name":"鲸鱼娘","homeDir":"/home/whale","skills":[{"name":"diary","enabled":true}]}`), &payload)
	value, _ := json.Marshal(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/dsh-whale/catalog" {
			http.NotFound(w, req)
			return
		}
		var request struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Error(err)
			http.Error(w, "invalid fixture request", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": json.RawMessage(value)}})
	}))
	defer srv.Close()

	s := &mainState{}
	ep := harness.Endpoint{Origin: srv.URL, Cookie: "dsh-auth=fixture"}
	s.bootStatus.Endpoint = ep
	snapshot, load, update, replace, clear, toggle := s.whalePetAccess(ep, true)
	if snapshot != nil {
		t.Fatal("first frame already had a snapshot")
	}
	if load == nil || update == nil || replace == nil || clear == nil || toggle == nil {
		t.Fatal("ready Host did not publish its Whale callbacks")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		catalog, err := s.whaleCatalogSnapshot(ep)
		if catalog != nil {
			if catalog.Name != "鲸鱼娘" || catalog.HomeDir != "/home/whale" || len(catalog.Skills) != 1 {
				t.Fatalf("catalog = %+v", catalog)
			}
			if err != "" {
				t.Fatalf("catalog error alongside a snapshot: %q", err)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("background catalog never arrived (err=%q)", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
