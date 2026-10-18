package session

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

func permissionFixtureWait(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("permission fixture did not settle")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestPermissionComposerUsesSessionCommandAndProjection(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "refusal"}[fail], func(t *testing.T) {
			var commands atomic.Int32
			var reads atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					RPCID   any `json:"rpcId"`
					Payload struct {
						Args map[string]json.RawMessage `json:"args"`
					} `json:"payload"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				switch {
				case strings.HasSuffix(r.URL.Path, "/commands/execute"):
					commands.Add(1)
					if string(body.Payload.Args["agentId"]) != `"A"` || string(body.Payload.Args["line"]) != `"/permission workspace-write"` || string(body.Payload.Args["submittedAttachments"]) != `[]` {
						t.Errorf("wrong permission command: %s", body.Payload.Args)
					}
					kind := "success"
					if fail {
						kind = "error"
					}
					rpcOK(w, body.RPCID, map[string]any{"result": map[string]any{"kind": kind, "text": "fixture refusal"}})
				case strings.HasSuffix(r.URL.Path, "/session/projections"):
					reads.Add(1)
					if !strings.Contains(string(body.Payload.Args["request"]), `"sessionId":"A"`) {
						t.Errorf("wrong projection target: %s", body.Payload.Args)
					}
					rpcOK(w, body.RPCID, map[string]any{"asOfSeq": 12, "values": map[string]any{"permissions": map[string]any{"currentValue": "workspace-write"}}})
				default:
					t.Errorf("composer must not write a default: %s", r.URL.Path)
					http.Error(w, "unexpected endpoint", 500)
				}
			}))
			defer srv.Close()
			s := New()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			s.selectedID = "A"
			s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}, {Label: "Write", Value: "workspace-write"}}
			s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 10)
			s.SelectPermissionPreset(context.Background(), "Write")
			permissionFixtureWait(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return !s.permissionProjection.Pending })
			got := s.Snapshot()
			if commands.Load() != 1 {
				t.Fatalf("commands = %d", commands.Load())
			}
			if fail {
				if got.SelectedPermission != "Read" || got.PermissionsErr == "" || reads.Load() != 0 {
					t.Fatalf("refusal replaced current permission: %+v", got)
				}
			} else if got.SelectedPermission != "Write" || got.PermissionsErr != "" || reads.Load() != 1 {
				t.Fatalf("projection not reflected: %+v", got)
			}
		})
	}
}

func TestPermissionProjectionIsSessionScopedAndMonotonic(t *testing.T) {
	s := New()
	s.selectedID = "A"
	s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}, {Label: "Write", Value: "workspace-write"}}
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"workspace-write"}}`), 10)
	s.applyPermissionControlLocked(json.RawMessage(`{"type":"projection","sessionId":"B","key":"permissions","seq":20,"value":{"currentValue":"read-only"}}`))
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 9)
	if got := s.Snapshot().SelectedPermission; got != "Write" {
		t.Fatalf("stale/other session projection won: %q", got)
	}
	s.selectSessionLocked("B")
	if got := s.Snapshot(); got.SelectedPermission != "" || len(got.PermissionLabels) != 0 {
		t.Fatalf("A leaked into B: %+v", got)
	}
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"custom"}}`), 2)
	if got := s.Snapshot().SelectedPermission; got != "custom" {
		t.Fatalf("custom current value lost: %q", got)
	}
	s.applyPermissionValuesLocked(json.RawMessage(`{}`), 3)
	if got := s.Snapshot(); got.SelectedPermission != "" || len(got.PermissionLabels) != 0 {
		t.Fatalf("absent capability became a default: %+v", got)
	}
}

func TestPermissionCommandCannotSettleIntoReplacementSession(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/commands/execute") {
			close(started)
			<-release
			rpcOK(w, body["rpcId"], map[string]any{"result": map[string]any{"kind": "success"}})
		} else {
			rpcOK(w, body["rpcId"], map[string]any{"asOfSeq": 20, "values": map[string]any{"permissions": map[string]any{"currentValue": "workspace-write"}}})
		}
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.selectedID = "A"
	s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}, {Label: "Write", Value: "workspace-write"}}
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runPermissionPreset(context.Background(), s.client, "A", s.followGeneration, PermissionOption{Label: "Write", Value: "workspace-write"})
	}()
	wait(t, started)
	s.mu.Lock()
	s.selectSessionLocked("B")
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 2)
	s.lastAction = "B status"
	s.mu.Unlock()
	close(release)
	wait(t, done)
	if got := s.Snapshot(); got.SelectedPermission != "Read" || got.LastAction != "B status" {
		t.Fatalf("old session result changed B: %+v", got)
	}
}
