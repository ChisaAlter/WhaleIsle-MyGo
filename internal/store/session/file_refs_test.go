package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestListFileReferences(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/fileReferences/list"):
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			if args["agentId"] != "s1" || args["query"] != "main" {
				t.Errorf("args = %#v", args)
			}
			rpcOK(w, body["rpcId"], []map[string]any{
				{"path": "cmd/main.go", "kind": "file"},
				{"path": "cmd/main.go", "kind": "file"},
				{"path": "internal/main.go", "kind": "file"},
			})
		case strings.HasSuffix(r.URL.Path, "/sessionReferenceResolver/candidates"):
			rpcOK(w, body["rpcId"], []map[string]any{
				{"mention": "@session/other", "sessionId": "s2", "label": "other", "sameWorkspace": true, "createdAt": 1.0},
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.mu.Lock()
	st.selectedID = "s1"
	st.mu.Unlock()

	got := st.ListFileReferences(context.Background(), "main")
	if len(got) != 3 || got[0] != "cmd/main.go" || got[1] != "internal/main.go" || got[2] != "@session/other" {
		t.Fatalf("%v", got)
	}
}
