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

func TestBodyPreferenceWritesCrossNamespaceWithoutAcknowledgingDrafts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		if r.URL.Path != "/api/rpc/settings/mutate" && r.URL.Path != "/rpc/settings/mutate" {
			// The route prefix is transport-owned; the method suffix is authoritative.
			if !strings.HasSuffix(r.URL.Path, "/settings/mutate") {
				t.Errorf("unexpected method %s", r.URL.Path)
			}
		}
		if args["ns"] != "ui-chat" || args["expectedRevision"] != float64(4) {
			t.Errorf("wrong owner/revision: %#v", args)
		}
		ops := args["ops"].([]any)
		op := ops[0].(map[string]any)
		if len(ops) != 1 || op["op"] != "set" || op["path"].([]any)[0] != "transcriptView" || op["value"] != "detailed" {
			t.Errorf("wrong field mutation: %#v", ops)
		}
		calls.Add(1)
		rpcOK(w, body["rpcId"], map[string]any{"ns": "ui-chat", "revision": 5, "value": map[string]any{"transcriptView": "detailed", "other": true}})
	}))
	defer srv.Close()
	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient = client
	st.writable = true
	st.selectedNS = "general"
	st.namespaces = []Namespace{{NS: "general", Fields: []Field{{Key: "draft", Kind: FieldString}}, Value: json.RawMessage(`{"draft":"saved"}`)},
		{NS: "ui-chat", Revision: 4, Fields: []Field{{Key: "transcriptView", Kind: FieldEnum, EnumOptions: []string{"compact", "detailed"}}, {Key: "other", Kind: FieldBool}}, Value: json.RawMessage(`{"transcriptView":"compact","other":true}`)}}
	st.mu.Unlock()
	st.SetDraft("draft", "unsaved")
	st.PatchDescribedField(context.Background(), "ui-chat", "transcriptView", "detailed")
	deadline := time.Now().Add(3 * time.Second)
	for st.Snapshot().Saving && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	snap := st.Snapshot()
	if calls.Load() != 1 || snap.Err != "" || snap.Saving || snap.SelectedNS != "general" || !snap.Dirty || snap.Drafts["draft"] != "unsaved" || snap.Namespaces[1].Revision != 5 {
		t.Fatalf("cross-namespace save changed editor ownership: %+v calls=%d", snap, calls.Load())
	}
}

func TestBodyPreferenceRejectsUnadvertisedEnumAndStaleHost(t *testing.T) {
	st := New()
	a := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://invalid.fixture"}}
	st.Attach(a, nil)
	st.mu.Lock()
	st.writable = true
	st.describedClient = a
	st.namespaces = []Namespace{{NS: "ui-chat", Fields: []Field{{Key: "transcriptView", Kind: FieldEnum, EnumOptions: []string{"compact", "detailed"}}}}}
	st.mu.Unlock()
	st.PatchDescribedField(context.Background(), "ui-chat", "transcriptView", "imaginary")
	if snap := st.Snapshot(); snap.Err == "" || snap.Saving {
		t.Fatalf("unadvertised enum was not rejected: %+v", snap)
	}
	st.mu.Lock()
	generation := st.generation
	st.mu.Unlock()
	_, _, oldWriter := st.BindDescribedPreferences()
	b := &dsh.Client{}
	st.Attach(b, nil)
	st.mu.Lock()
	st.writable = true
	st.describedClient = b
	st.namespaces = []Namespace{{NS: "ui-chat", Fields: []Field{{Key: "transcriptView", Kind: FieldEnum, EnumOptions: []string{"compact", "detailed"}}}}}
	st.mu.Unlock()
	oldWriter(context.Background(), "ui-chat", map[string]string{"transcriptView": "compact"})
	st.saveMutateOwned(context.Background(), "ui-chat", []json.RawMessage{json.RawMessage(`{"op":"set","path":["transcriptView"],"value":"compact"}`)}, a, generation)
	if st.Snapshot().Saving {
		t.Fatal("a stale row retargeted a replacement Host")
	}
}

func TestBodyPreferenceCoupledEffectsAreOneValidatedMutation(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		ops := args["ops"].([]any)
		if args["ns"] != "ui-conversation" || args["expectedRevision"] != float64(7) || len(ops) != 2 {
			t.Errorf("coupled update was split or lost revision: %#v", args)
		}
		fields := map[string]any{}
		for _, raw := range ops {
			op := raw.(map[string]any)
			fields[op["path"].([]any)[0].(string)] = op["value"]
		}
		if fields["composerBeamStyle"].(map[string]any)["period"] != float64(3) || fields["composerBeamPresets"].(map[string]any)["Work"] == nil {
			t.Errorf("typed values lost: %#v", fields)
		}
		calls.Add(1)
		rpcOK(w, body["rpcId"], map[string]any{"revision": 8, "value": fields})
	}))
	defer srv.Close()
	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.writable = true
	st.describedClient = client
	st.namespaces = []Namespace{{NS: "ui-conversation", Revision: 7, Fields: []Field{{Key: "composerBeamStyle", Kind: FieldJSON}, {Key: "composerBeamPresets", Kind: FieldJSON}}}}
	st.mu.Unlock()
	st.PatchDescribedFields(context.Background(), "ui-conversation", map[string]string{"composerBeamStyle": `{"period":3}`, "composerBeamPresets": `invalid`})
	if snap := st.Snapshot(); snap.Err == "" || snap.Saving || calls.Load() != 0 {
		t.Fatal("part of an invalid coupled edit reached the Host")
	}
	st.PatchDescribedFields(context.Background(), "ui-conversation", map[string]string{"composerBeamStyle": `{"period":3}`, "composerBeamPresets": `{"Work":{"period":3}}`})
	deadline := time.Now().Add(3 * time.Second)
	for st.Snapshot().Saving && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if snap := st.Snapshot(); calls.Load() != 1 || snap.Err != "" || snap.Saving || snap.Namespaces[0].Revision != 8 {
		t.Fatalf("atomic preference write failed: %+v", snap)
	}
}
