package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"whaleisle/internal/dsh"
)

func TestComposerAttachmentHostEpoch(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		n := calls.Add(1)
		rpcOK(w, b["rpcId"], map[string]any{"receiptId": fmt.Sprintf("receipt-%d", n), "file": map[string]any{"attachmentId": "f", "name": "notes.txt", "bytes": 5}})
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	a, err := s.PrepareComposerAttachment(context.Background(), "a", ComposerAttachment{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	b, err := s.PrepareComposerAttachment(context.Background(), "a", a)
	if err != nil || calls.Load() != 2 || b.HostScope == a.HostScope || b.Upload.ReceiptId == a.Upload.ReceiptId {
		t.Fatalf("old receipt reused: %+v %+v %v", a, b, err)
	}
}
