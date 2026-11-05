package workspacefs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestScopedDirectoryRootUsesSessionRelativeDot(t *testing.T) {
	for _, tc := range []struct {
		name, session, path, want string
	}{
		{"scoped root", "owned-session", "", "."},
		{"scoped child", "owned-session", "nested", "nested"},
		{"legacy root", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var env struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args map[string]any `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
					t.Error(err)
					return
				}
				if r.URL.Path != "/api/workspaceFiles/list" || env.Payload.Args["path"] != tc.want {
					t.Errorf("wrong directory request: %s %#v", r.URL.Path, env.Payload.Args)
				}
				value, present := env.Payload.Args["workspaceFileScopeId"]
				if tc.session == "" && present || tc.session != "" && value != tc.session {
					t.Errorf("changed Session scope contract: %#v", env.Payload.Args)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"path": "", "entries": []any{}}}})
			}))
			defer server.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
			var err error
			if tc.session == "" {
				_, err = ListRemoteDir(context.Background(), client, tc.path)
			} else {
				_, err = ListRemoteDirScoped(context.Background(), client, tc.session, tc.path)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
