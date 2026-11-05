package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestFetchAttachmentPreservesOpaqueWhitespaceIDs(t *testing.T) {
	SetAttachmentCacheDir(t.TempDir())
	t.Cleanup(func() { SetAttachmentCacheDir("") })
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		payload, _ := body["payload"].(map[string]any)
		args, _ := payload["args"].(map[string]any)
		request, _ := args["request"].(map[string]any)
		id, _ := request["attachmentId"].(string)
		if r.URL.Path == "/api/session/attachment" {
			requested = append(requested, id)
		} else {
			rpcOK(w, body["rpcId"], map[string]any{})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"attachment": map[string]any{"attachmentId": id, "mediaType": "image/png", "bytes": 1, "width": 1, "height": 1}, "data": base64.StdEncoding.EncodeToString([]byte(id))})
	}))
	t.Cleanup(srv.Close)
	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.Select("opaque-id-session")
	read := st.CaptureAttachmentRead(st.PlanHistoryBinding(), "opaque-id-session")
	paths := map[string]string{}
	for _, id := range []string{"opaque", " opaque ", " "} {
		read.Forget(id)
		local, err := st.FetchAttachment(context.Background(), id)
		if err != nil || local == "" {
			t.Fatalf("id%q local%q err%v", id, local, err)
		}
		if read.Cached(id) != local {
			t.Fatalf("cache lost exact id %q", id)
		}
		paths[id] = local
	}
	if len(requested) != 3 || requested[0] != "opaque" || requested[1] != " opaque " || requested[2] != " " || paths["opaque"] == paths[" opaque "] {
		t.Fatalf("IDs merged or rewritten: %q %v", requested, paths)
	}
	read.Forget(" opaque ")
	if read.Cached(" opaque ") != "" || read.Cached("opaque") != paths["opaque"] {
		t.Fatal("retry cleared a different opaque identity")
	}
	t.Cleanup(func() {
		for id := range paths {
			read.Forget(id)
		}
	})
}
