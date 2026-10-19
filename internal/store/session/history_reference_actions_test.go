package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

func referenceFixtureResponse(r *http.Request, raw string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(raw)), Request: r}
}

func TestRecordedFileApplicationListAndExplicitHandoffKeepCoordinates(t *testing.T) {
	s := New()
	var calls atomic.Int32
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid", Cookie: "fixture-only"}, HTTP: &http.Client{Transport: turnDeliveryTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		q := r.URL.Query()
		if r.URL.Path != "/api/present.open" || q.Get("sessionId") != "root" || q.Get("seq") != "2" || q.Get("index") != "0" || q.Has("path") || r.Header.Get("Cookie") != "fixture-only" {
			t.Errorf("unowned app route: %s", r.URL)
		}
		if r.Method == http.MethodGet {
			if q.Has("application") {
				t.Error("read executed an application")
			}
			return referenceFixtureResponse(r, `[{"id":"viewer:notes&x","name":"Viewer","default":true,"icon":null}]`), nil
		}
		if r.Method != http.MethodPost || q.Get("application") != "viewer:notes&x" || q.Has("action") {
			t.Errorf("bad application handoff: %s", r.URL)
		}
		return referenceFixtureResponse(r, ``), nil
	})}}
	s.selectedID = "root"
	s.followGeneration = 1
	s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
	s.messages = []Message{{Raw: json.RawMessage(`{"type":"deliverables/presented","seq":2,"data":{"turn":1,"callId":"p","files":[{"path":"/recorded/a.txt"}]}}`)}, {Raw: json.RawMessage(`{"type":"turn/end","seq":3,"data":{"turn":1}}`)}}
	d := TurnDeliveries(s.messages)[1]
	scope := s.Snapshot().TurnFiles.Scope
	request := TurnFileRequest{Target: d.Files[0].Target, Action: "applications"}
	s.RunTurnFileAction(context.Background(), scope, request)
	apps := s.Snapshot().TurnFiles.Results[request]
	if !apps.Ready || len(apps.Applications) != 1 || !apps.Applications[0].Default {
		t.Fatalf("applications not parsed: %+v", apps)
	}
	request.Action = TurnFileApplicationAction(apps.Applications[0].ID)
	s.RunTurnFileAction(context.Background(), scope, request)
	if !s.Snapshot().TurnFiles.Results[request.StateKey()].Ready || calls.Load() != 2 {
		t.Fatal("explicit application action failed")
	}
	s.followGeneration++
	s.RunTurnFileAction(context.Background(), scope, request)
	if calls.Load() != 2 {
		t.Fatal("stale history application action escaped")
	}
}

func TestHistorySkillReferenceUsesHotAndColdExactCatalogAndRootOwner(t *testing.T) {
	s := New()
	var calls atomic.Int32
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}, HTTP: &http.Client{Transport: turnDeliveryTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		var request struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Method string `json:"method"`
				Args   struct {
					Request struct {
						SessionID string `json:"sessionId"`
					} `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request.Payload.Args.Request.SessionID != "root" {
			t.Error("catalog borrowed another session")
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"skills": []any{map[string]any{"name": "review", "path": "/catalog/SKILL.md", "description": "", "modelInvocable": true}}}}})
		return referenceFixtureResponse(r, string(raw)), nil
	})}}
	s.selectedID = "root"
	s.followGeneration = 1
	s.cwd = "/workspace"
	s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
	scope := s.Snapshot().TurnFiles.Scope
	ref, ok := s.CaptureHistorySkillReference(scope, "review")
	if !ok {
		t.Fatal("cold reference rejected")
	}
	resolved, ok := s.ResolveHistorySkillReference(context.Background(), ref)
	if !ok {
		t.Fatal("cold catalog did not resolve")
	}
	if path, ok := s.HistorySkillReferencePath(resolved); !ok || path != "/catalog/SKILL.md" {
		t.Fatal("wrong scoped skill path")
	}
	hot, _ := s.CaptureHistorySkillReference(scope, "review")
	if _, ok = s.ResolveHistorySkillReference(context.Background(), hot); !ok || calls.Load() != 1 {
		t.Fatal("hot catalog refetched")
	}
	unknown, _ := s.CaptureHistorySkillReference(scope, "unknown")
	if _, ok = s.ResolveHistorySkillReference(context.Background(), unknown); ok {
		t.Fatal("unknown skill guessed path")
	}
	s.subagentIdentities = map[string]sessionSubagentIdentityState{"root": {value: &SessionSubagentIdentity{Mode: "continuable"}}}
	if _, ok = s.CaptureHistorySkillReference(scope, "review"); ok {
		t.Fatal("child borrowed root skill opener")
	}
	s.subagentIdentities = nil
	s.invalidateSkillOptionsLocked()
	if _, ok = s.HistorySkillReferencePath(resolved); ok {
		t.Fatal("old catalog survived preset invalidation")
	}
	s.client.Endpoint.Origin = "https://other.invalid"
	if _, ok = s.CaptureHistorySkillReference(scope, "review"); ok {
		t.Fatal("old Host reference survived endpoint mutation")
	}
}

func TestHistorySkillColdResponseCannotOpenAfterSessionChanges(t *testing.T) {
	s := New()
	started, release := make(chan struct{}), make(chan struct{})
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}, HTTP: &http.Client{Transport: turnDeliveryTransport(func(r *http.Request) (*http.Response, error) {
		var request struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		close(started)
		<-release
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"skills": []any{map[string]any{"name": "review", "path": "/old/SKILL.md", "description": "", "modelInvocable": true}}}}})
		return referenceFixtureResponse(r, string(raw)), nil
	})}}
	s.selectedID = "root"
	s.followGeneration = 1
	s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
	ref, ok := s.CaptureHistorySkillReference(s.Snapshot().TurnFiles.Scope, "review")
	if !ok {
		t.Fatal("capture failed")
	}
	done := make(chan bool, 1)
	go func() { _, valid := s.ResolveHistorySkillReference(context.Background(), ref); done <- valid }()
	<-started
	s.mu.Lock()
	s.selectedID = "other"
	s.followGeneration++
	s.invalidateSkillOptionsLocked()
	s.mu.Unlock()
	close(release)
	if <-done {
		t.Fatal("old cold catalog response acquired new-session preview authority")
	}
}

func TestWaitableHistoryFileHandoffReturnsItsOwnErrorAndFencesOwner(t *testing.T) {
	for _, mode := range []string{"refused", "accepted", "owner-retired"} {
		t.Run(mode, func(t *testing.T) {
			s := New()
			s.selectedID = "root"
			s.cwd = `C:\workspace`
			s.followGeneration = 1
			s.lastAction = "unrelated action"
			var calls int
			s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid"}, HTTP: &http.Client{Transport: turnDeliveryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var request struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args struct {
							Request struct {
								Path string `json:"path"`
							} `json:"request"`
						} `json:"args"`
					} `json:"payload"`
				}
				_ = json.NewDecoder(r.Body).Decode(&request)
				var value any = true
				if strings.HasSuffix(r.URL.Path, "canOpenWorkspacePath") {
					if mode == "owner-retired" {
						s.mu.Lock()
						s.followGeneration++
						s.mu.Unlock()
					}
				} else {
					if r.URL.Path != "/api/session/openWorkspacePath" || request.Payload.Args.Request.Path != `C:\workspace\dir/file.txt` {
						t.Errorf("wrong file handoff: %s %+v", r.URL, request.Payload.Args.Request)
					}
					value = map[string]any{"opened": mode == "accepted"}
				}
				raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
				return referenceFixtureResponse(r, string(raw)), nil
			})}}
			s.turnFilesHistoryBinding = turnFileHistoryBinding(s.client, s.selectedID)
			err := s.OpenHistoryReferenceOnHost(context.Background(), s.Snapshot().TurnFiles.Scope, "dir/file.txt")
			if (err == nil) != (mode == "accepted") {
				t.Fatalf("wrong precise completion: %v", err)
			}
			if mode == "owner-retired" && calls != 1 {
				t.Fatal("retired handoff reached open RPC")
			}
			if s.lastAction != "unrelated action" {
				t.Fatal("reference failure was routed through global status")
			}
		})
	}
}
