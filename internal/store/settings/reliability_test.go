package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestSaveProviderSettingsClearsSavingWhenNamespaceDisappears(t *testing.T) {
	saveEntered := make(chan struct{})
	releaseSave := make(chan struct{})
	var includeAlpha atomic.Bool
	includeAlpha.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID  string `json:"rpcId"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			nss := []map[string]any{
				{
					"ns":       "beta",
					"revision": 1,
					"schema":   json.RawMessage(permissionSchema),
					"value":    map[string]any{"defaultPreset": "read-only"},
					"base":     map[string]any{"defaultPreset": "read-only"},
					"user":     map[string]any{},
				},
			}
			if includeAlpha.Load() {
				nss = append([]map[string]any{{
					"ns":       "alpha",
					"revision": 1,
					"schema":   json.RawMessage(permissionSchema),
					"value":    map[string]any{"defaultPreset": "read-only"},
					"base":     map[string]any{"defaultPreset": "read-only"},
					"user":     map[string]any{},
				}}, nss...)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": body.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"hasDocument": true,
						"writable":    true,
						"namespaces":  nss,
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"), strings.HasSuffix(r.URL.Path, "/settings/mutate"):
			select {
			case <-saveEntered:
			default:
				close(saveEntered)
			}
			<-releaseSave
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": body.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"ns":       "alpha",
						"revision": 2,
						"value":    map[string]any{"defaultPreset": "workspace-write"},
						"base":     map[string]any{"defaultPreset": "read-only"},
						"user":     map[string]any{"defaultPreset": "workspace-write"},
					},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":   "server-response",
				"rpcId":  body.RPCID,
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		}
	}))
	defer srv.Close()

	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.Attach(client, nil)
	s.RefreshDescribe(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for !s.Snapshot().Value.Described && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	b := s.BindNamespace("alpha")
	if b == nil {
		t.Fatal("failed to bind alpha namespace")
	}
	s.Select("alpha")
	s.SetDraft("defaultPreset", "workspace-write")

	errCh := make(chan error, 1)
	go func() {
		_, err := b.SaveProviderSettings(context.Background())
		errCh <- err
	}()
	select {
	case <-saveEntered:
	case err := <-errCh:
		t.Fatalf("SaveProviderSettings returned early before update RPC: %v (snap=%+v)", err, s.Snapshot().Value)
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for saveEntered (snap=%+v)", s.Snapshot().Value)
	}

	// Remove alpha while SaveProviderSettings is in flight.
	includeAlpha.Store(false)
	s.RefreshDescribe(context.Background())
	for time.Now().Before(deadline) {
		v := s.Snapshot().Value
		if !v.Loading && len(v.Namespaces) == 1 && v.Namespaces[0].NS == "beta" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	close(releaseSave)

	if err := <-errCh; !errors.Is(err, ErrSettingsNotReady) {
		t.Fatalf("expected ErrSettingsNotReady when namespace disappeared, got %v", err)
	}
	if s.Snapshot().Value.Saving {
		t.Fatal("Saving remained wedged true after disappeared namespace save")
	}
}

func TestRefreshDescribeDropsOutOfOrderStaleResponses(t *testing.T) {
	var callNum atomic.Int32
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/settings/describe") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":   "server-response",
				"rpcId":  body.RPCID,
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
			return
		}
		n := callNum.Add(1)
		if n == 1 {
			close(firstEntered)
			<-releaseFirst
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":   "server-response",
				"rpcId":  body.RPCID,
				"result": map[string]any{"ok": false, "error": map[string]any{"message": "stale describe error"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body.RPCID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"hasDocument": true,
					"writable":    true,
					"namespaces": []map[string]any{{
						"ns":       "general",
						"revision": 7,
						"schema":   map[string]any{"type": "object", "properties": map[string]any{}},
						"value":    map[string]any{},
					}},
				},
			},
		})
	}))
	defer srv.Close()

	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.RefreshDescribe(context.Background())
	<-firstEntered
	s.RefreshDescribe(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Value
		if v.Described && len(v.Namespaces) == 1 && v.Namespaces[0].Revision == 7 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	close(releaseFirst)
	time.Sleep(40 * time.Millisecond)

	v := s.Snapshot().Value
	if !v.Described || len(v.Namespaces) != 1 || v.Namespaces[0].Revision != 7 || v.Err != "" {
		t.Fatalf("stale RefreshDescribe overwrote newer state: described=%v err=%v ns=%+v", v.Described, v.Err, v.Namespaces)
	}
}
