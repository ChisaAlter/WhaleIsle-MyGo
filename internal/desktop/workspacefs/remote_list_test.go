package workspacefs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestListRemoteDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		if r.URL.Path != "/api/workspaceFiles/list" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": env.RPCID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"path": "",
					"entries": []any{
						map[string]any{"name": "src", "type": "directory"},
						map[string]any{"name": "README.md", "type": "file", "size": 12.0},
					},
				},
			},
		})
	}))
	defer srv.Close()

	ents, err := ListRemoteDir(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 2 || !ents[0].IsDir || ents[0].Name != "src" || ents[1].Name != "README.md" {
		t.Fatalf("%+v", ents)
	}
}
