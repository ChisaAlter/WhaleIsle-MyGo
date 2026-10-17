package harness

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureWorkspace(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspace/create" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Cookie") != "dsh-auth-x=v1" {
			t.Fatalf("cookie %q", r.Header.Get("Cookie"))
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		rpcID, _ := gotBody["rpcId"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": rpcID,
			"result": map[string]any{
				"ok":    true,
				"value": map[string]any{"workspace": map[string]any{"path": "/tmp/ws"}},
			},
		})
	}))
	defer srv.Close()

	err := EnsureWorkspace(context.Background(), srv.Client(), Endpoint{
		Origin: srv.URL,
		Cookie: "dsh-auth-x=v1",
	}, `C:\work\proj`)
	if err != nil {
		t.Fatal(err)
	}
	args := gotBody["payload"].(map[string]any)["args"].(map[string]any)
	req := args["request"].(map[string]any)
	if req["path"] != `C:\work\proj` {
		t.Fatalf("path = %#v", req["path"])
	}
}

func TestEnsureWorkspaceRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok":    false,
				"error": map[string]any{"message": "nope"},
			},
		})
	}))
	defer srv.Close()
	err := EnsureWorkspace(context.Background(), srv.Client(), Endpoint{Origin: srv.URL}, "/x")
	if err == nil || err.Error() != "nope" {
		t.Fatalf("got %v", err)
	}
}
