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
