package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestCustomInstructionsContract(t *testing.T) {
	var hits atomic.Int32
	responses := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		if args["ns"] != "ui-conversation" {
			t.Errorf("namespace = %v", args["ns"])
		}
		patch := args["patch"].(map[string]any)
		value := patch["customInstructions"].(string)
		hits.Add(1)
		rpcOK(w, body["rpcId"], map[string]any{"ns": "ui-conversation", "revision": 2, "value": patch})
		responses <- value
	}))
	defer server.Close()
	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	st.namespaces = []Namespace{{NS: "ui-conversation", Revision: 1, Fields: []Field{{Key: "customInstructions", Kind: FieldString}}, Value: json.RawMessage(`{"customInstructions":"old"}`)}}
	st.writable = true
	st.Select("ui-conversation")
	st.SetDraft("customInstructions", strings.Repeat("😀", 751))
	st.SaveUpdate(context.Background())
	deadline := time.Now().Add(time.Second)
	for st.Snapshot().Saving && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if hits.Load() != 0 || st.Snapshot().Err == "" {
		t.Fatal("over-limit instructions must fail before RPC and retain the draft")
	}
	if st.Snapshot().Drafts["customInstructions"] != strings.Repeat("😀", 751) {
		t.Fatal("invalid draft lost")
	}
	for _, value := range []string{"literal {{tone}}\n中文😀", ""} {
		st.SetDraft("customInstructions", value)
		st.SaveUpdate(context.Background())
		select {
		case got := <-responses:
			if got != value {
				t.Fatalf("text changed: %q", got)
			}
		case <-time.After(time.Second):
			t.Fatal("save timed out")
		}
		deadline = time.Now().Add(time.Second)
		for st.Snapshot().Saving && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
	}
}
