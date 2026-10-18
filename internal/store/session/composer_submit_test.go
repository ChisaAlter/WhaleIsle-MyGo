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

// One fixture exercises the successful byte-upload -> receipt -> prompt chain,
// then an application-level command failure and a changed current session.
func TestComposerSubmissionChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("actual bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args map[string]json.RawMessage `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		args := body.Payload.Args
		switch {
		case strings.HasSuffix(r.URL.Path, "/fileUploads/upload"):
			var req struct {
				Data string `json:"data"`
				Name string `json:"name"`
			}
			_ = json.Unmarshal(args["request"], &req)
			if req.Data != base64.StdEncoding.EncodeToString([]byte("actual bytes")) || req.Name != "notes.txt" {
				t.Errorf("wrong bytes: %+v", req)
			}
			rpcOK(w, body.RPCID, map[string]any{"receiptId": "receipt-1", "file": map[string]any{"attachmentId": "file-1", "name": "notes.txt", "bytes": 12}})
		case strings.HasSuffix(r.URL.Path, "/session/prompt"):
			var req struct {
				SessionID string           `json:"sessionId"`
				Content   []map[string]any `json:"content"`
			}
			_ = json.Unmarshal(args["request"], &req)
			if req.SessionID != "original" || len(req.Content) != 2 || req.Content[1]["receiptId"] != "receipt-1" || req.Content[0]["text"] != "Keep\n\nformatting" {
				t.Errorf("wrong prompt: %+v", req)
			}
			rpcOK(w, body.RPCID, map[string]any{"accepted": true})
		case strings.HasSuffix(r.URL.Path, "/commands/list"):
			rpcOK(w, body.RPCID, []any{map[string]any{"name": "review", "input": map[string]any{"attachments": true}}})
		case strings.HasSuffix(r.URL.Path, "/commands/execute"):
			var attachments []map[string]any
			_ = json.Unmarshal(args["submittedAttachments"], &attachments)
			if len(attachments) != 1 || attachments[0]["receiptId"] != "receipt-1" {
				t.Errorf("wrong command attachments: %v", attachments)
			}
			kind := "success"
			if fail {
				kind = "error"
			}
			rpcOK(w, body.RPCID, map[string]any{"commandId": "cmd-1", "result": map[string]any{"kind": kind, "text": "retryable failure"}})
		default:
			rpcOK(w, body.RPCID, map[string]any{})
		}
	}))
	defer srv.Close()
	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.Select("original")
	a, err := st.PrepareComposerAttachment(context.Background(), "original", ComposerAttachment{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if a.Upload.ReceiptId != "receipt-1" || a.Upload.File.Name != "notes.txt" {
		t.Fatalf("lost receipt: %+v", a)
	}
	st.Select("other")
	if err = st.SubmitComposer(context.Background(), "original", "Keep\n\nformatting", []ComposerAttachment{a}, 0); err != nil {
		t.Fatal(err)
	}
	if len(st.Snapshot().Messages) != 0 {
		t.Fatal("completion contaminated current session")
	}
	if err = st.SubmitComposer(context.Background(), "original", "/review", []ComposerAttachment{a}, 0); err != nil {
		t.Fatal(err)
	}
	fail = true
	if err = st.SubmitComposer(context.Background(), "original", "/review", []ComposerAttachment{a}, 0); err == nil {
		t.Fatal("command failure treated as success")
	}
	if a.Upload.ReceiptId != "receipt-1" {
		t.Fatal("failure lost retry receipt")
	}
}
