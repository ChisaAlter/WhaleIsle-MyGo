package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type selectedOwnerAuditTransport func(*http.Request) (*http.Response, error)

func (f selectedOwnerAuditTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func selectedOwnerAuditFixture(transport selectedOwnerAuditTransport, notify func()) (*Store, *dsh.Client, *workspaceSelectedOwner) {
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://selected-owner-fixture.invalid"}, HTTP: &http.Client{Transport: transport}}
	s := New()
	s.Attach(client, notify)
	s.mu.Lock()
	s.selectedID = "selected"
	s.cwd = "/fixture"
	owner := &workspaceSelectedOwner{client: client, hostGeneration: s.eventsGeneration, composerGeneration: s.composerHostGeneration, followGeneration: s.followGeneration, sessionID: s.selectedID, cwd: s.cwd}
	s.mu.Unlock()
	return s, client, owner
}

func selectedOwnerAuditWait(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Fatal("fixture did not reach synchronization point")
	}
}

// A cwd change invalidates this read without changing its catalog version. The
// retiring request must release its own Loading lease before it drops the data.
func TestSelectedOwnerAuditCatalogLoadingRetiredOnCwdChange(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "error"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered, release := make(chan struct{}), make(chan struct{})
				transport := selectedOwnerAuditTransport(func(r *http.Request) (*http.Response, error) {
					var request struct {
						RPCID string `json:"rpcId"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						return nil, err
					}
					close(entered)
					<-release
					if fail {
						return nil, context.Canceled
					}
					raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"skills": []any{map[string]any{"name": "late"}}}}})
					if err != nil {
						return nil, err
					}
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
				})
				s, _, owner := selectedOwnerAuditFixture(transport, nil)
				s.refreshSkillCatalogForOwner(context.Background(), owner)
				<-entered
				s.SetWorkspaceCwd("/other")
				close(release)
				synctest.Wait()
				s.SetWorkspaceCwd("/fixture")
				s.mu.Lock()
				loading := s.skillCatalogLoading
				s.mu.Unlock()
				if loading {
					t.Fatal("obsolete request retained the catalog loading lease after returning to its cwd")
				}
			})
		})
	}
}

// With the same client, a previously accepted catalog row is not evidence of
// membership in a new events generation after that generation rejects a read.
func TestSelectedOwnerAuditCatalogOldRowsNotReauthorizedByCanceledRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		transport := selectedOwnerAuditTransport(func(r *http.Request) (*http.Response, error) {
			var request struct {
				RPCID string `json:"rpcId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, err
			}
			var value any
			switch r.URL.Path {
			case "/api/skillInventory/list":
				close(entered)
				<-release
				value = map[string]any{"skills": []any{}}
			case "/api/skillInventory/get":
				value = map[string]any{"name": "old", "source": "project", "path": "/fake/old/SKILL.md", "writable": true, "content": "fixture only", "description": "fixture"}
			case "/api/skillInventory/update":
				value = map[string]any{}
			default:
				t.Errorf("unexpected RPC %s", r.URL.Path)
				return nil, context.Canceled
			}
			raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		})
		s, client, owner := selectedOwnerAuditFixture(transport, nil)
		path := "/fake/old/SKILL.md"
		s.mu.Lock()
		s.skillCatalog = SkillCatalog{Skills: []api.SkillInventorySnapshotSkillsItem{{Name: "old", Source: "project", Path: &path, Writable: true}}}
		s.skillCatalogClient, s.skillCatalogSession, s.skillCatalogCwd = client, s.selectedID, s.cwd
		s.mu.Unlock()
		s.refreshSkillCatalogForOwner(context.Background(), owner)
		<-entered
		s.BeginEvents(context.Background(), client)
		close(release)
		synctest.Wait()
		if target, err := s.CaptureCatalogSkillEdit("old"); err == nil {
			loaded, loadErr := s.LoadCatalogSkillEdit(context.Background(), target)
			if loadErr == nil && s.SkillEditorCurrent(loaded) {
				fields := loaded.Fields()
				fields.Description = "changed"
				fields.Content = "changed"
				err = s.SaveCatalogSkillEditor(context.Background(), loaded, fields)
				t.Fatalf("old Host row became a writable current edit target; fixture update result: %v", err)
			}
		}
	})
}

func TestSelectedOwnerAuditGoalReadRevocation(t *testing.T) {
	for _, revoke := range []string{"same-client-events", "context-canceled"} {
		t.Run(revoke, func(t *testing.T) {
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			transport := selectedOwnerAuditTransport(func(r *http.Request) (*http.Response, error) {
				var request struct {
					RPCID string `json:"rpcId"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, err
				}
				close(entered)
				<-release
				raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"id": "old", "objective": "obsolete", "revision": 1, "phase": "active"}}})
				if err != nil {
					return nil, err
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
			})
			s, client, owner := selectedOwnerAuditFixture(transport, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				defer close(done)
				s.runRefreshGoalForOwner(ctx, client, owner.sessionID, owner.followGeneration, owner)
			}()
			selectedOwnerAuditWait(t, entered)
			if revoke == "same-client-events" {
				s.BeginEvents(context.Background(), client)
			} else {
				cancel()
			}
			s.mu.Lock()
			s.planBoard.Goal = &PlanGoal{ID: "current", Objective: "current state"}
			s.mu.Unlock()
			close(release)
			selectedOwnerAuditWait(t, done)
			s.mu.Lock()
			goal := s.planBoard.Goal
			s.mu.Unlock()
			if goal == nil || goal.ID != "current" {
				t.Fatalf("revoked goal read changed current goal: %+v", goal)
			}
		})
	}
}

func TestSelectedOwnerAuditGoalReadAdmission(t *testing.T) {
	var calls atomic.Int32
	transport := selectedOwnerAuditTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, context.Canceled
	})
	s, client, owner := selectedOwnerAuditFixture(transport, nil)
	s.BeginEvents(context.Background(), client)
	s.runRefreshGoalForOwner(context.Background(), client, owner.sessionID, owner.followGeneration, owner)
	if calls.Load() != 0 {
		t.Fatal("revoked owner started a goal RPC")
	}
}

// The notify hook observes Loading before the owner is revoked. Once the
// re-entrant admission check retires that flag, observers must hear about it.
func TestSelectedOwnerAuditFeedbackRetirementNotifies(t *testing.T) {
	for _, revoke := range []string{"same-client-events", "context-canceled"} {
		t.Run(revoke, func(t *testing.T) {
			var calls atomic.Int32
			transport := selectedOwnerAuditTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, context.Canceled
			})
			s, client, owner := selectedOwnerAuditFixture(transport, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var states []bool
			s.Attach(client, func() {
				states = append(states, s.Feedback().Loading)
				if len(states) == 1 {
					if revoke == "same-client-events" {
						s.BeginEvents(context.Background(), client)
					} else {
						cancel()
					}
				}
			})
			s.refreshFeedbackForOwner(ctx, owner)
			if calls.Load() != 0 {
				t.Fatal("revoked feedback request started")
			}
			if s.Feedback().Loading {
				t.Fatal("revoked feedback request retained Loading")
			}
			if len(states) < 2 || states[len(states)-1] {
				t.Fatalf("observers did not receive retired Loading state: %v", states)
			}
		})
	}
}
