package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestFetchAttachment(t *testing.T) {
	dir := t.TempDir()
	SetAttachmentCacheDir(dir)
	t.Cleanup(func() { SetAttachmentCacheDir("") })
	raw := []byte{0x89, 0x50, 0x4e, 0x47}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/attachment") {
			rpcOK(w, body["rpcId"], map[string]any{
				"attachment": map[string]any{
					"attachmentId": "a1",
					"mediaType":    "image/png",
					"bytes":        float64(len(raw)),
					"width":        1.0,
					"height":       1.0,
				},
				"data": base64.StdEncoding.EncodeToString(raw),
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.Select("s1")
	read := st.CaptureAttachmentRead(st.PlanHistoryBinding(), "s1")
	path, err := st.FetchAttachment(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("bytes=%v", got)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("ext=%s", path)
	}
	if read.Cached("a1") != path {
		t.Fatal("cache miss")
	}
	if CachedAttachmentPath("a1") != "" {
		t.Fatal("session attachment escaped its owner namespace")
	}
}
