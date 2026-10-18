package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"whaleisle/internal/dsh"
)

func TestPermissionRiskGateBeforeAnyCommand(t *testing.T) {
	for _, value := range []string{PermissionFullAccess, PermissionAutoReview} {
		t.Run(value, func(t *testing.T) {
			var commands atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					RPCID   any `json:"rpcId"`
					Payload struct {
						Args map[string]json.RawMessage `json:"args"`
					} `json:"payload"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.HasSuffix(r.URL.Path, "/commands/execute") {
					commands.Add(1)
					if string(body.Payload.Args["agentId"]) != `"A"` || string(body.Payload.Args["line"]) != `"/permission `+value+`"` {
						t.Errorf("wrong target: %v", body.Payload.Args)
					}
					rpcOK(w, body.RPCID, map[string]any{"result": map[string]any{"kind": "success"}})
				} else if strings.HasSuffix(r.URL.Path, "/session/projections") {
					rpcOK(w, body.RPCID, map[string]any{"asOfSeq": 2, "values": map[string]any{"permissions": map[string]any{"currentValue": value}}})
				} else {
					t.Errorf("unexpected RPC: %s", r.URL.Path)
					http.Error(w, "unexpected", 500)
				}
			}))
			defer srv.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			makeStore := func() *Store {
				s := New()
				s.Attach(client, nil)
				s.selectedID = "A"
				s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}, {Label: "Risk", Value: value}}
				s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 1)
				return s
			}
			s := makeStore()
			s.SelectPermissionPreset(context.Background(), "Risk")
			p := s.PermissionRiskConfirmation()
			if p == nil || p.Value != value || p.Acknowledged {
				t.Fatal("missing explicit risk gate")
			}
			s.ConfirmPermissionRisk(context.Background(), p.Token)
			if commands.Load() != 0 || s.permissionProjection.Pending || s.Snapshot().SelectedPermission != "Read" {
				t.Fatal("unacknowledged risk reached Host")
			}
			s.AcknowledgePermissionRisk(p.Token, true)
			s.CancelPermissionRisk(p.Token)
			s.ConfirmPermissionRisk(context.Background(), p.Token)
			if commands.Load() != 0 || s.PermissionRiskConfirmation() != nil {
				t.Fatal("cancelled token survived")
			}
			s.SelectPermissionPreset(context.Background(), "Risk")
			p = s.PermissionRiskConfirmation()
			s.AcknowledgePermissionRisk(p.Token, true)
			s.mu.Lock()
			s.selectSessionLocked("B")
			s.mu.Unlock()
			s.ConfirmPermissionRisk(context.Background(), p.Token)
			if commands.Load() != 0 || s.PermissionRiskConfirmation() != nil {
				t.Fatal("selection change retained gate")
			}
			s = makeStore()
			s.SelectPermissionPreset(context.Background(), "Risk")
			p = s.PermissionRiskConfirmation()
			s.AcknowledgePermissionRisk(p.Token, true)
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			s.ConfirmPermissionRisk(context.Background(), p.Token)
			if commands.Load() != 0 || s.PermissionRiskConfirmation() != nil {
				t.Fatal("Host change retained gate")
			}
			s = makeStore()
			s.SelectPermissionPreset(context.Background(), "Risk")
			p = s.PermissionRiskConfirmation()
			s.AcknowledgePermissionRisk(p.Token, true)
			s.ConfirmPermissionRisk(context.Background(), p.Token)
			s.ConfirmPermissionRisk(context.Background(), p.Token)
			permissionFixtureWait(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return !s.permissionProjection.Pending })
			if commands.Load() != 1 || s.PermissionRiskConfirmation() != nil || s.Snapshot().SelectedPermission != "Risk" {
				t.Fatalf("confirmed action not once: %d %+v", commands.Load(), s.Snapshot())
			}
		})
	}
}
