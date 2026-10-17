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

func TestPickAndListDirectories(t *testing.T) {
	var listPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/directoryPicker/list"):
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			listPath, _ = args["path"].(string)
			rpcOK(w, body["rpcId"], map[string]any{
				"path": "/home",
				"home": "/home/u",
				"crumbs": []map[string]any{
					{"name": "home", "path": "/home", "hidden": false},
				},
				"entries": []map[string]any{
					{"name": "proj", "path": "/home/proj", "hidden": false},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/directoryPicker/pick"):
			rpcOK(w, body["rpcId"], "/home/proj")
		case strings.HasSuffix(r.URL.Path, "/directoryPicker/createDirectory"):
			rpcOK(w, body["rpcId"], "/home/proj/new")
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)

	listing, err := st.ListDirectories(context.Background(), "/home")
	if err != nil {
		t.Fatal(err)
	}
	if listPath != "/home" || listing.Path != "/home" || len(listing.Entries) != 1 {
		t.Fatalf("list path=%q listing=%#v", listPath, listing)
	}
	picked, err := st.PickDirectory(context.Background())
	if err != nil || picked != "/home/proj" {
		t.Fatalf("pick=%q err=%v", picked, err)
	}
	created, err := st.CreateDirectory(context.Background(), "/home/proj", "new")
	if err != nil || created != "/home/proj/new" {
		t.Fatalf("create=%q err=%v", created, err)
	}
}
