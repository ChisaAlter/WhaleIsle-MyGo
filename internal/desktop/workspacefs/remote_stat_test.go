package workspacefs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

func TestStatRemoteScopedUsesCapturedScope(t *testing.T) {
	for _, scope := range []string{"session-a", "session-b"} {
		t.Run(scope, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var env struct {
					Type    string `json:"type"`
					RPCID   string `json:"rpcId"`
					Method  string `json:"method"`
					Payload struct {
						Args map[string]any `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
					t.Error(err)
				}
				if r.Method != http.MethodPost || r.URL.Path != "/api/workspaceFiles/stat" || env.Type != "client-request" || env.Method != "workspaceFiles/stat" || env.RPCID == "" {
					t.Errorf("wrong Remote request: %s %s %+v", r.Method, r.URL.Path, env)
				}
				args := env.Payload.Args
				if len(args) != 2 || args["workspaceFileScopeId"] != scope || args["path"] != "same.txt" {
					t.Errorf("missing or wrong captured scope/path: %#v", args)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"absolutePath": "owned/same.txt", "version": scope, "bytes": 12}}})
			}))
			defer server.Close()
			got, err := StatRemoteScoped(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, scope, "  same.txt  ")
			if err != nil || got.Version != scope || got.AbsolutePath != "owned/same.txt" || got.Bytes == nil || *got.Bytes != 12 {
				t.Fatalf("scoped stat = %+v, %v", got, err)
			}
		})
	}
}

func TestStatRemoteScopedRejectsMissingScopeBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	for _, scope := range []string{"", "  "} {
		if _, err := StatRemoteScoped(context.Background(), client, scope, "same.txt"); err == nil {
			t.Fatal("missing scope accepted")
		}
	}
	for _, path := range []string{"", "  "} {
		if _, err := StatRemoteScoped(context.Background(), client, "session-a", path); err == nil {
			t.Fatal("empty path accepted")
		}
	}
	if _, err := StatRemoteScoped(context.Background(), nil, "session-a", "same.txt"); err == nil {
		t.Fatal("nil client accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("sent %d invalid stat requests", calls.Load())
	}
}

func TestStatRemotePreservesLegacyUnscopedContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args map[string]any `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			t.Error(err)
		}
		if len(env.Payload.Args) != 1 || env.Payload.Args["path"] != "legacy.txt" {
			t.Errorf("legacy contract changed: %#v", env.Payload.Args)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"version": "legacy"}}})
	}))
	defer server.Close()
	got, err := StatRemote(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, "legacy.txt")
	if err != nil || got.Version != "legacy" {
		t.Fatalf("legacy stat = %+v, %v", got, err)
	}
}
