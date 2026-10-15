package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"whaleisle/internal/dsh"
)

func TestQueueImageOwnerAndIdentity(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		req := b["payload"].(map[string]any)["args"].(map[string]any)["request"].(map[string]any)
		seen = req["sessionId"].(string)
		rpcOK(w, b["rpcId"], map[string]any{"attachment": map[string]any{"attachmentId": "img", "mediaType": "image/png"}, "data": base64.StdEncoding.EncodeToString([]byte("image fixture"))})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "a"
	SetAttachmentCacheDir(t.TempDir())
	defer SetAttachmentCacheDir("")
	loader := s.QueueImageLoader("a", s.Snapshot().PlanBoard.QueueActions.Scope)
	p, err := loader(context.Background(), "img")
	if err != nil || seen != "a" {
		t.Fatalf("%s %v", seen, err)
	}
	if data, err := os.ReadFile(p); err != nil || string(data) != "image fixture" {
		t.Fatalf("%q %v", data, err)
	}
	if _, err = loader(context.Background(), "wrong"); err == nil {
		t.Fatal("mismatched image identity accepted")
	}
	s.followGeneration++
	if _, err = loader(context.Background(), "img"); err == nil {
		t.Fatal("old generation image loader accepted")
	}
}
