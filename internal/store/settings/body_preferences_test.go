package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	for len(st.Snapshot().Value.PreferencePending) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	snap := st.Snapshot().Value
	if calls.Load() != 1 || snap.Err != "" || snap.Saving || len(snap.PreferencePending) != 0 || snap.SelectedNS != "general" || !snap.Dirty || snap.Drafts["draft"] != "unsaved" || snap.Namespaces[1].Revision != 5 {
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
	if snap := st.Snapshot().Value; snap.Err == "" || snap.Saving {
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
	if st.Snapshot().Value.Saving {
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
	if snap := st.Snapshot().Value; snap.Err == "" || snap.Saving || calls.Load() != 0 {
		t.Fatal("part of an invalid coupled edit reached the Host")
	}
	st.PatchDescribedFields(context.Background(), "ui-conversation", map[string]string{"composerBeamStyle": `{"period":3}`, "composerBeamPresets": `{"Work":{"period":3}}`})
	deadline := time.Now().Add(3 * time.Second)
	for len(st.Snapshot().Value.PreferencePending) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if snap := st.Snapshot().Value; calls.Load() != 1 || snap.Err != "" || snap.Saving || len(snap.PreferencePending) != 0 || snap.Namespaces[0].Revision != 8 {
		t.Fatalf("atomic preference write failed: %+v", snap)
	}
}

// Reproducing the real Host config-editor contract: toggling a preference off
// writes {"viewTabs":false} to user, then toggling it back on (matching base:
// {"viewTabs":true}) prunes the override so user becomes {} while base & value
// both hold true. Both transitions must be accepted without false refusal.
func TestBodyPreferenceToggleBackToInheritedDefaultAcceptedWhenUserPruned(t *testing.T) {
	var mu sync.Mutex
	rev := 0
	viewTabs := true
	sessionCost := true
	schema := json.RawMessage(`{"uid":2,"refs":{"1":{"type":"boolean"},"2":{"type":"object","dict":{"viewTabs":1,"sessionCost":1}}}}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		mu.Lock()
		defer mu.Unlock()
		if args["expectedRevision"] != float64(rev) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": body["rpcId"],
				"result": map[string]any{"ok": false, "error": map[string]any{"code": "settings/conflict", "message": "SETTINGS_CONFLICT"}},
			})
			return
		}
		for _, raw := range args["ops"].([]any) {
			op := raw.(map[string]any)
			key := op["path"].([]any)[0].(string)
			val := op["value"].(bool)
			if key == "viewTabs" {
				viewTabs = val
			} else if key == "sessionCost" {
				sessionCost = val
			}
		}
		rev++
		user := map[string]any{}
		if !viewTabs {
			user["viewTabs"] = false
		}
		if !sessionCost {
			user["sessionCost"] = false
		}
		rpcOK(w, body["rpcId"], map[string]any{
			"ns":       "ui-conversation",
			"revision": rev,
			"base":     map[string]any{"viewTabs": true, "sessionCost": true},
			"value":    map[string]any{"viewTabs": viewTabs, "sessionCost": sessionCost},
			"user":     user,
		})
	}))
	defer srv.Close()

	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient = client
	st.writable = true
	st.namespaces = []Namespace{{
		NS:       "ui-conversation",
		Revision: 0,
		Schema:   schema,
		Fields:   []Field{{Key: "viewTabs", Kind: FieldBool}, {Key: "sessionCost", Kind: FieldBool}},
		Base:     json.RawMessage(`{"viewTabs":true,"sessionCost":true}`),
		Value:    json.RawMessage(`{"viewTabs":true,"sessionCost":true}`),
		User:     json.RawMessage(`{}`),
	}}
	st.mu.Unlock()

	waitPending := func() ViewState {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			snap := st.Snapshot().Value
			if len(snap.PreferencePending) == 0 {
				return snap
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatalf("preference write did not settle: %+v", st.Snapshot().Value)
		return ViewState{}
	}

	// Off -> On -> Off -> On for both keys; toggling back to default leaves user={}.
	for _, step := range []struct {
		key, val string
		wantRev  float64
	}{
		{"viewTabs", "false", 1},
		{"viewTabs", "true", 2},
		{"sessionCost", "false", 3},
		{"sessionCost", "true", 4},
	} {
		st.PatchDescribedField(context.Background(), "ui-conversation", step.key, step.val)
		snap := waitPending()
		if snap.Err != "" || len(snap.PreferenceErrors) != 0 {
			t.Fatalf("step %s=%s failed: err=%q prefErrs=%v", step.key, step.val, snap.Err, snap.PreferenceErrors)
		}
		if snap.Namespaces[0].Revision != step.wantRev {
			t.Fatalf("step %s=%s rev=%v want %v", step.key, step.val, snap.Namespaces[0].Revision, step.wantRev)
		}
	}
}

// Verifies that: (1) background settings/document-updated refreshes never flip
// Loading=true to drop consecutive toggles or regress a namespace to an older
// revision, and (2) a stale expectedRevision conflict retries once cleanly.
func TestBodyPreferenceConsecutiveWritesSurviveStaleDescribeAndConflictRetry(t *testing.T) {
	var mu sync.Mutex
	rev := 1 // Host is already at rev 1 while store starts stale at rev 0.
	transcript := "compact"
	collapse := "never"
	describeGate := make(chan struct{})
	describeEntered := make(chan struct{}, 2)
	var describeCalls atomic.Int32
	schema := json.RawMessage(`{"uid":3,"refs":{"1":{"type":"union","list":[{"type":"const","value":"compact"},{"type":"const","value":"detailed"}]},"2":{"type":"union","list":[{"type":"const","value":"never"},{"type":"const","value":"always"}]},"3":{"type":"object","dict":{"transcriptView":1,"collapseToolCalls":2}}}}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			n := describeCalls.Add(1)
			if n == 1 {
				// First describe (background event) snapshots an OLD revision (0)
				// and returns AFTER a newer mutate has already landed revision 2.
				describeEntered <- struct{}{}
				<-describeGate
				rpcOK(w, body["rpcId"], map[string]any{
					"hasDocument": true,
					"writable":    true,
					"namespaces": []any{map[string]any{
						"ns": "ui-chat", "revision": 0, "schema": schema,
						"base":  map[string]any{"transcriptView": "compact", "collapseToolCalls": "never"},
						"value": map[string]any{"transcriptView": "compact", "collapseToolCalls": "never"},
						"user":  map[string]any{},
					}},
				})
				return
			}
			mu.Lock()
			curRev, curT, curC := rev, transcript, collapse
			mu.Unlock()
			rpcOK(w, body["rpcId"], map[string]any{
				"hasDocument": true,
				"writable":    true,
				"namespaces": []any{map[string]any{
					"ns": "ui-chat", "revision": curRev, "schema": schema,
					"base":  map[string]any{"transcriptView": "compact", "collapseToolCalls": "never"},
					"value": map[string]any{"transcriptView": curT, "collapseToolCalls": curC},
					"user":  map[string]any{"transcriptView": curT, "collapseToolCalls": curC},
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/mutate"):
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			mu.Lock()
			if args["expectedRevision"] != float64(rev) {
				mu.Unlock()
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type": "server-response", "rpcId": body["rpcId"],
					"result": map[string]any{"ok": false, "error": map[string]any{"code": "settings/conflict", "message": "SETTINGS_CONFLICT"}},
				})
				return
			}
			for _, raw := range args["ops"].([]any) {
				op := raw.(map[string]any)
				key := op["path"].([]any)[0].(string)
				if key == "transcriptView" {
					transcript = op["value"].(string)
				} else if key == "collapseToolCalls" {
					collapse = op["value"].(string)
				}
			}
			rev++
			out := map[string]any{
				"ns": "ui-chat", "revision": rev,
				"base":  map[string]any{"transcriptView": "compact", "collapseToolCalls": "never"},
				"value": map[string]any{"transcriptView": transcript, "collapseToolCalls": collapse},
				"user":  map[string]any{"transcriptView": transcript, "collapseToolCalls": collapse},
			}
			mu.Unlock()
			rpcOK(w, body["rpcId"], out)
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient = client
	st.writable = true
	st.namespaces = []Namespace{{
		NS: "ui-chat", Revision: 0, Schema: schema,
		Fields: []Field{
			{Key: "transcriptView", Kind: FieldEnum, EnumOptions: []string{"compact", "detailed"}},
			{Key: "collapseToolCalls", Kind: FieldEnum, EnumOptions: []string{"never", "always"}},
		},
		Base:  json.RawMessage(`{"transcriptView":"compact","collapseToolCalls":"never"}`),
		Value: json.RawMessage(`{"transcriptView":"compact","collapseToolCalls":"never"}`),
		User:  json.RawMessage(`{}`),
	}}
	st.mu.Unlock()

	// Trigger a background settings/document-updated refresh; Loading must stay false.
	st.OnEmitMaybeRefresh(context.Background(), dsh.EventsEmit{Event: "settings/document-updated"})
	<-describeEntered
	if st.Snapshot().Value.Loading {
		t.Fatal("background settings/document-updated set Loading=true and would drop clicks")
	}

	// While the stale describe is still in flight, perform two consecutive writes.
	// Write 1 hits expectedRevision=0 vs Host rev=1, retries with fresh rev=1 -> rev=2.
	st.PatchDescribedField(context.Background(), "ui-chat", "transcriptView", "detailed")
	st.PatchDescribedField(context.Background(), "ui-chat", "collapseToolCalls", "always")

	deadline := time.Now().Add(3 * time.Second)
	for len(st.Snapshot().Value.PreferencePending) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(describeGate)
	time.Sleep(40 * time.Millisecond)

	snap := st.Snapshot().Value
	if snap.Err != "" || len(snap.PreferenceErrors) != 0 {
		t.Fatalf("consecutive writes failed: err=%q prefErrs=%v", snap.Err, snap.PreferenceErrors)
	}
	if snap.Namespaces[0].Revision != 3 {
		t.Fatalf("stale describe regressed revision: got %v, want 3", snap.Namespaces[0].Revision)
	}
	if !strings.Contains(string(snap.Namespaces[0].Value), `"detailed"`) || !strings.Contains(string(snap.Namespaces[0].Value), `"always"`) {
		t.Fatalf("final values lost: %s", snap.Namespaces[0].Value)
	}
}

func TestBodyPreferenceFieldErrorIsolatedFromGlobalErrAndClearedOnLeave(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/settings/mutate") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": body["rpcId"],
				"result": map[string]any{"ok": false, "error": map[string]any{"code": "failed", "message": "denied"}},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{
			"hasDocument": true, "writable": true,
			"namespaces": []any{map[string]any{
				"ns": "ui-chat", "revision": 1,
				"schema": json.RawMessage(`{"uid":2,"refs":{"1":{"type":"boolean"},"2":{"type":"object","dict":{"flag":1}}}}`),
				"value":  map[string]any{"flag": false},
			}},
		})
	}))
	defer srv.Close()

	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient = client
	st.writable = true
	st.namespaces = []Namespace{{
		NS: "ui-chat", Revision: 1,
		Fields: []Field{{Key: "flag", Kind: FieldBool}},
		Value:  json.RawMessage(`{"flag":false}`),
	}}
	st.mu.Unlock()

	st.PatchDescribedField(context.Background(), "ui-chat", "flag", "true")
	deadline := time.Now().Add(3 * time.Second)
	for len(st.Snapshot().Value.PreferencePending) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	snap := st.Snapshot().Value
	if snap.PreferenceErrors["ui-chat.flag"] == "" {
		t.Fatalf("expected field-level preference error, got %+v", snap)
	}
	if snap.Err != "" {
		t.Fatalf("field preference failure leaked into global ViewState.Err: %q", snap.Err)
	}

	// Simulate a page-level validation error as well, then leave the section.
	st.PatchDescribedField(context.Background(), "ui-chat", "flag", "not-a-bool")
	if st.Snapshot().Value.Err == "" {
		t.Fatal("expected synchronous validation error on Err")
	}
	st.ClearTransientErrors()
	cleared := st.Snapshot().Value
	if cleared.Err != "" || len(cleared.PreferenceErrors) != 0 {
		t.Fatalf("ClearTransientErrors left stale errors: err=%q prefErrs=%v", cleared.Err, cleared.PreferenceErrors)
	}
}
