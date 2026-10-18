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

func TestUploadEncodedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(path, []byte("PNGDATA"), 0o644); err != nil {
		t.Fatal(err)
	}
	var gotAgent, gotData, gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/fileUploads/upload") {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			gotAgent, _ = args["agentId"].(string)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				gotData, _ = req["data"].(string)
				gotName, _ = req["name"].(string)
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"receiptId": "r1",
				"file":      map[string]any{"attachmentId": "att-9", "name": "pic.png", "bytes": 7},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.Select("s1")
	id, err := st.UploadEncodedFile(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if id != "att-9" {
		t.Fatalf("id=%q", id)
	}
	if gotAgent != "s1" || gotName != "pic.png" {
		t.Fatalf("agent/name=%q/%q", gotAgent, gotName)
	}
	want := base64.StdEncoding.EncodeToString([]byte("PNGDATA"))
	if gotData != want {
		t.Fatalf("data=%q want %q", gotData, want)
	}
}

func TestPromptContentBlocksAttachment(t *testing.T) {
	blocks := PromptContentBlocks("see ![](attachment:att-1)")
	if len(blocks) != 2 {
		t.Fatalf("len=%d %v", len(blocks), blocks)
	}
	var img map[string]string
	if err := json.Unmarshal(blocks[1], &img); err != nil {
		t.Fatal(err)
	}
	if img["type"] != "image" || img["attachmentId"] != "att-1" {
		t.Fatalf("%v", img)
	}
}
