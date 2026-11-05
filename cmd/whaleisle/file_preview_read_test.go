package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestWorkspacePreviewStatBindsFrameScopeAndFormatsNote(t *testing.T) {
	for _, tc := range []struct {
		name, note string
		stat       map[string]any
	}{
		{"version-and-bytes", "v-current · 12 B", map[string]any{"version": "v-current", "bytes": 12}},
		{"version", "v-current", map[string]any{"version": "v-current"}},
		{"bytes", "0 B", map[string]any{"bytes": 0}},
		{"path", "owned/same.txt", map[string]any{"absolutePath": "owned/same.txt"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var env struct {
					Type, RPCID, Method string
					Payload             struct{ Args map[string]any }
				}
				if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
					t.Error(err)
				}
				if r.URL.Path != "/api/workspaceFiles/stat" || env.Type != "client-request" || env.Method != "workspaceFiles/stat" || env.RPCID == "" || len(env.Payload.Args) != 2 || env.Payload.Args["workspaceFileScopeId"] != "captured-session" || env.Payload.Args["path"] != "same.txt" {
					t.Errorf("wrong scoped stat envelope: %+v", env)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env.RPCID, "result": map[string]any{"ok": true, "value": tc.stat}})
			}))
			defer server.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
			if got := workspacePreviewStat(context.Background(), client, "captured-session", "same.txt"); got != tc.note {
				t.Fatalf("stat note = %q, want %q", got, tc.note)
			}
		})
	}
}

func TestWorkspacePreviewStatFailureKeepsEmptySupplementalNote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	if got := workspacePreviewStat(context.Background(), client, "captured-session", "missing.txt"); got != "" {
		t.Fatalf("supplemental stat failure = %q", got)
	}
}
