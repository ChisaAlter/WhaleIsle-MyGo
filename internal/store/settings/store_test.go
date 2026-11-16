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

func rpcOK(w http.ResponseWriter, rpcID any, value any) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":  "server-response",
		"rpcId": rpcID,
		"result": map[string]any{
			"ok":    true,
			"value": value,
		},
	})
}

func TestRefreshDescribeParsesNamespaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/settings/describe") {
			rpcOK(w, body["rpcId"], map[string]any{})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{
			"writable":    true,
			"hasDocument": true,
			"namespaces": []map[string]any{{
				"ns":           "permission",
				"schema":       json.RawMessage(permissionSchema),
				"value":        map[string]any{"defaultPreset": "read-only"},
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     2.0,
			}},
		})
	}))
	defer srv.Close()

	st := New()
	done := make(chan struct{})
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		snap := st.Snapshot()
		if len(snap.Namespaces) == 1 && !snap.Loading {
			close(done)
		}
	})
	st.RefreshDescribe(context.Background())

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for describe")
	}
	snap := st.Snapshot()
	if snap.SelectedNS != "permission" || len(snap.Namespaces[0].Fields) != 1 {
		t.Fatalf("snap = %+v", snap)
	}
}

func TestSaveUpdateSendsPatch(t *testing.T) {
	var updateHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "permission",
					"schema":       json.RawMessage(permissionSchema),
					"value":        map[string]any{"defaultPreset": "read-only"},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			updateHits.Add(1)
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			if args["ns"] != "permission" {
				t.Errorf("ns = %v", args["ns"])
			}
			patch := args["patch"].(map[string]any)
			if patch["defaultPreset"] != "workspace-write" {
				t.Errorf("patch = %v", patch)
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"ns":           "permission",
				"schema":       json.RawMessage(permissionSchema),
				"value":        map[string]any{"defaultPreset": "workspace-write"},
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     2.0,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	st.SetDraft("defaultPreset", "workspace-write")
	st.SaveUpdate(context.Background())
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if updateHits.Load() > 0 && st.Snapshot().LastAction == "settings saved" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("update hits = %d last = %q", updateHits.Load(), st.Snapshot().LastAction)
}

func TestSaveReplaceFromDraftsSendsSection(t *testing.T) {
	var replaceHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "permission",
					"schema":       json.RawMessage(permissionSchema),
					"value":        map[string]any{"defaultPreset": "read-only"},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/replace"):
			replaceHits.Add(1)
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			if args["ns"] != "permission" {
				t.Errorf("ns = %v", args["ns"])
			}
			section := args["section"].(map[string]any)
			if section["defaultPreset"] != "workspace-write" {
				t.Errorf("section = %v", section)
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"ns":           "permission",
				"schema":       json.RawMessage(permissionSchema),
				"value":        map[string]any{"defaultPreset": "workspace-write"},
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     3.0,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	st.SetDraft("defaultPreset", "workspace-write")
	st.SaveReplaceFromDrafts(context.Background())
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if replaceHits.Load() > 0 && st.Snapshot().LastAction == "settings replaced" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("replace hits = %d last = %q err = %q", replaceHits.Load(), st.Snapshot().LastAction, st.Snapshot().Err)
}

func TestSaveReplaceEmptyFailsClosed(t *testing.T) {
	var replaceHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/settings/describe") {
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "permission",
					"schema":       json.RawMessage(permissionSchema),
					"value":        map[string]any{"defaultPreset": "read-only"},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/settings/replace") {
			replaceHits.Add(1)
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	st.SaveReplace(context.Background(), nil)
	snap := st.Snapshot()
	if replaceHits.Load() != 0 {
		t.Fatalf("RPC should not fire, hits=%d", replaceHits.Load())
	}
	if snap.Err == "" || !strings.Contains(snap.LastAction, "empty section") {
		t.Fatalf("want fail-closed messaging, got err=%q last=%q", snap.Err, snap.LastAction)
	}
}

func TestSaveReplaceFromDraftsInvalidFailsClosed(t *testing.T) {
	var replaceHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns": "obj",
					"schema": json.RawMessage(`{
						"uid": 2,
						"refs": {
							"1": { "type": "object", "dict": {} },
							"2": { "type": "object", "dict": { "blob": 1 } }
						}
					}`),
					"value":        map[string]any{"blob": map[string]any{}},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/replace"):
			replaceHits.Add(1)
			rpcOK(w, body["rpcId"], map[string]any{})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	st.SetDraft("blob", "{not-json")
	st.SaveReplaceFromDrafts(context.Background())
	snap := st.Snapshot()
	if replaceHits.Load() != 0 {
		t.Fatalf("RPC should not fire, hits=%d", replaceHits.Load())
	}
	if snap.Err == "" || !strings.Contains(snap.LastAction, "invalid JSON") {
		t.Fatalf("want invalid JSON fail-closed, got err=%q last=%q", snap.Err, snap.LastAction)
	}
}

func TestSyncViewTabsPatchesConversation(t *testing.T) {
	var updateHits atomic.Int32
	var patched map[string]any
	schema := json.RawMessage(`{
	  "uid": 2,
	  "refs": {
	    "1": { "type": "boolean" },
	    "2": { "type": "object", "dict": { "viewTabs": 1 } }
	  }
	}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "conversation",
					"schema":       schema,
					"value":        map[string]any{"viewTabs": true},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			updateHits.Add(1)
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			patched = args["patch"].(map[string]any)
			rpcOK(w, body["rpcId"], map[string]any{
				"ns":           "conversation",
				"schema":       schema,
				"value":        map[string]any{"viewTabs": false},
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     2.0,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	st.SyncViewTabs(context.Background(), false)
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if updateHits.Load() > 0 {
			if patched["viewTabs"] != false {
				t.Fatalf("patch = %#v", patched)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("update hits = %d last=%q", updateHits.Load(), st.Snapshot().LastAction)
}

func TestPatchBoolDiscardsStaleResponseAcrossDetachAndReattach(t *testing.T) {
	schema := json.RawMessage(`{
	  "uid": 2,
	  "refs": {
	    "1": { "type": "boolean" },
	    "2": { "type": "object", "dict": { "viewTabs": 1 } }
	  }
	}`)
	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once

	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "conversation",
					"schema":       schema,
					"value":        map[string]any{"viewTabs": true},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			enteredOnce.Do(func() { close(entered) })
			<-release
			rpcOK(w, body["rpcId"], map[string]any{
				"ns":           "conversation",
				"schema":       schema,
				"value":        map[string]any{"viewTabs": false},
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     9.0,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srvA.Close()

	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/settings/describe") {
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "conversation",
					"schema":       schema,
					"value":        map[string]any{"viewTabs": true},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srvB.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srvA.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	st.SyncViewTabs(context.Background(), false)
	<-entered

	// Switch to client B on another host while A's update is still in flight.
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srvB.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	close(release)
	time.Sleep(50 * time.Millisecond)

	snap := st.Snapshot()
	if snap.LastAction == "viewTabs synced" {
		t.Fatalf("stale A update overwrote LastAction: %+v", snap)
	}
	if len(snap.Namespaces) != 1 || snap.Namespaces[0].Revision != 1.0 {
		t.Fatalf("stale A update corrupted B namespace: %+v", snap.Namespaces)
	}
	var got struct {
		ViewTabs bool `json:"viewTabs"`
	}
	if err := json.Unmarshal(snap.Namespaces[0].Value, &got); err != nil {
		t.Fatal(err)
	}
	if !got.ViewTabs {
		t.Fatalf("stale A response overwrote B viewTabs: %s", snap.Namespaces[0].Value)
	}
}

func TestPatchBoolSerializesSequentialMutationsAndInvalidatesSnapCache(t *testing.T) {
	schema := json.RawMessage(`{
	  "uid": 2,
	  "refs": {
	    "1": { "type": "boolean" },
	    "2": { "type": "object", "dict": { "viewTabs": 1 } }
	  }
	}`)
	firstEntered := make(chan struct{})
	firstRelease := make(chan struct{})
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	type updateReq struct {
		expectedRev float64
		viewTabs    bool
	}
	var mu sync.Mutex
	var recorded []updateReq

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{{
					"ns":           "conversation",
					"schema":       schema,
					"value":        map[string]any{"viewTabs": true},
					"autoGenerate": true,
					"applies":      "live",
					"secrets":      []any{},
					"revision":     1.0,
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			cur := inFlight.Add(1)
			for {
				prev := maxInFlight.Load()
				if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
					break
				}
			}
			defer inFlight.Add(-1)

			args := body["payload"].(map[string]any)["args"].(map[string]any)
			expRev, _ := args["expectedRevision"].(float64)
			patch := args["patch"].(map[string]any)
			vt, _ := patch["viewTabs"].(bool)

			mu.Lock()
			idx := len(recorded)
			recorded = append(recorded, updateReq{expectedRev: expRev, viewTabs: vt})
			mu.Unlock()

			if idx == 0 {
				close(firstEntered)
				<-firstRelease
			}
			nextRev := float64(idx + 2)
			rpcOK(w, body["rpcId"], map[string]any{
				"ns":           "conversation",
				"schema":       schema,
				"value":        map[string]any{"viewTabs": vt},
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     nextRev,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().Namespaces) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Prime snapCache before mutating via SyncViewTabs.
	_ = st.Snapshot()

	st.SyncViewTabs(context.Background(), false)
	<-firstEntered
	st.SyncViewTabs(context.Background(), true)
	close(firstRelease)

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(recorded)
		mu.Unlock()
		if n == 2 && st.Snapshot().Namespaces[0].Revision == 3.0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if maxInFlight.Load() != 1 {
		t.Fatalf("maxInFlight = %d, want 1 (serialized)", maxInFlight.Load())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(recorded) != 2 {
		t.Fatalf("recorded = %+v, want 2 serialized updates", recorded)
	}
	if recorded[0] != (updateReq{expectedRev: 1.0, viewTabs: false}) {
		t.Fatalf("first update = %+v", recorded[0])
	}
	if recorded[1] != (updateReq{expectedRev: 2.0, viewTabs: true}) {
		t.Fatalf("second update = %+v", recorded[1])
	}
	snap := st.Snapshot()
	if snap.LastAction != "viewTabs synced" {
		t.Fatalf("snap.LastAction = %q (snapCache stale?)", snap.LastAction)
	}
	var got struct {
		ViewTabs bool `json:"viewTabs"`
	}
	if err := json.Unmarshal(snap.Namespaces[0].Value, &got); err != nil {
		t.Fatal(err)
	}
	if !got.ViewTabs {
		t.Fatalf("final viewTabs = false, want true (%s)", snap.Namespaces[0].Value)
	}
}
