package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

type skillHubTestTransport func(*http.Request) (*http.Response, error)

func (f skillHubTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type skillHubTestCall struct {
	method  string
	request map[string]any
	ctx     context.Context
}

type skillHubTestRPC struct {
	mu         sync.Mutex
	calls      []skillHubTestCall
	holdMethod string
	release    chan struct{}
	resultName string
}

func (f *skillHubTestRPC) roundTrip(r *http.Request) (*http.Response, error) {
	var envelope struct {
		RPCID   string `json:"rpcId"`
		Method  string `json:"method"`
		Payload struct {
			Args struct {
				Request map[string]any `json:"request"`
			} `json:"args"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.calls = append(f.calls, skillHubTestCall{method: envelope.Method, request: envelope.Payload.Args.Request, ctx: r.Context()})
	name := f.resultName
	held, release := envelope.Method == f.holdMethod, f.release
	f.mu.Unlock()
	if held {
		// Deliberately deliver a late response even after context cancellation.
		// No listener or real network participates in this transport.
		<-release
	}
	var value any
	switch envelope.Method {
	case "skillInventory/searchHub":
		value = api.SearchHubResult{{SkillName: name, Repo: "owned/repository", Path: "skills/" + name}}
	case "skillInventory/installHub":
		value = api.SkillHubInstallResult{Status: "installed", SkillName: name, Directory: "owned-result"}
	case "skillInventory/list":
		value = api.SkillInventorySnapshot{}
	default:
		value = map[string]any{}
	}
	raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
}

func (f *skillHubTestRPC) unblock() {
	if f.release != nil {
		select {
		case <-f.release:
		default:
			close(f.release)
		}
	}
}

func newSkillHubTestState(t *testing.T, f *skillHubTestRPC) *mainState {
	t.Helper()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned-fixture.invalid"}, HTTP: &http.Client{Transport: skillHubTestTransport(f.roundTrip)}}
	source := session.New()
	source.Attach(client, nil)
	source.SetWorkspaceCwd(t.TempDir())
	s := &mainState{sessions: source, dshClient: client}
	s.openSettingsSkillHub()
	return s
}

func (f *skillHubTestRPC) count(method string) int {
	n := 0
	for _, call := range f.calls {
		if call.method == method {
			n++
		}
	}
	return n
}

func TestSettingsSkillHubReviewBusyAndWireScope(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &skillHubTestRPC{resultName: "owned-skill", holdMethod: "skillInventory/searchHub", release: make(chan struct{})}
		defer f.unblock()
		s := newSkillHubTestState(t, f)
		s.selectSettingsSkillHubRoot(session.SkillRootProject)
		s.searchSettingsSkillHub(" synthetic ")
		synctest.Wait()
		if !s.settingsSkillHubAccess().Searching || len(f.calls) != 1 {
			t.Fatalf("search did not enter its busy state: %+v calls=%d", s.settingsSkillHubAccess(), len(f.calls))
		}
		s.closeSettingsSkillHub()
		s.selectSettingsSkillHubRoot(session.SkillRootUser)
		s.searchSettingsSkillHub("duplicate")
		synctest.Wait()
		if !s.settingsSkillHubAccess().Open || s.settingsSkillRoot != session.SkillRootProject || len(f.calls) != 1 {
			t.Fatal("pending search admitted close, scope switch or duplicate request")
		}
		search := f.calls[0].request
		if search["root"] != session.SkillRootProject || search["cwd"] == "" || search["query"] != "synthetic" || search["limit"] != float64(15) {
			t.Fatalf("search lost its selected scope/limit: %v", search)
		}
		close(f.release)
		synctest.Wait()
		s.selectSettingsSkillHubRoot(session.SkillRootUser)
		if len(s.settingsSkillHubHits) != 0 || s.settingsSkillHubAccess().Confirming != nil {
			t.Fatal("changing roots retained actionable results")
		}
		f.holdMethod = ""
		s.searchSettingsSkillHub("synthetic")
		synctest.Wait()
		if len(s.settingsSkillHubHits) != 1 || f.calls[1].request["root"] != session.SkillRootUser {
			t.Fatal("user-root search was not independently reachable")
		}
		hit := s.settingsSkillHubHits[0]
		s.requestSettingsSkillHubInstall(hit.Repo, hit.Path, hit.SkillName)
		if s.settingsSkillHubAccess().Confirming == nil || f.count("skillInventory/installHub") != 0 {
			t.Fatal("selecting Install wrote before review confirmation")
		}
		s.cancelSettingsSkillHubInstall()
		if s.settingsSkillHubAccess().Confirming != nil || f.count("skillInventory/installHub") != 0 {
			t.Fatal("cancelled review wrote a skill")
		}
		s.requestSettingsSkillHubInstall(hit.Repo, hit.Path, hit.SkillName)
		f.holdMethod, f.release = "skillInventory/installHub", make(chan struct{})
		s.confirmSettingsSkillHubInstall()
		synctest.Wait()
		s.confirmSettingsSkillHubInstall()
		s.requestSettingsSkillHubInstall(hit.Repo, hit.Path, hit.SkillName)
		s.closeSettingsSkillHub()
		s.selectSettingsSkillHubRoot(session.SkillRootProject)
		s.searchSettingsSkillHub("duplicate")
		synctest.Wait()
		if f.count("skillInventory/installHub") != 1 || !s.settingsSkillHubAccess().Installing || !s.settingsSkillHubAccess().Open || s.settingsSkillRoot != session.SkillRootUser {
			t.Fatal("pending installation admitted a duplicate, closure or scope change")
		}
		last := f.calls[len(f.calls)-1].request
		if last["root"] != session.SkillRootUser || last["repo"] != hit.Repo || last["path"] != hit.Path || last["skillName"] != hit.SkillName {
			t.Fatalf("confirmation changed the reviewed source/scope: %v", last)
		}
		close(f.release)
		synctest.Wait()
		if s.settingsSkillHubAccess().Installing || s.settingsSkillHubAccess().Confirming != nil || s.settingsSkillHubHits[0].Status != "installed" || f.count("skillInventory/installHub") != 1 || f.count("skillInventory/list") == 0 {
			t.Fatal("confirmed install did not settle, mark the result and refresh inventory once admitted")
		}
	})
}

func TestSettingsSkillHubRetiresLateCompletionsOnScopeOrHostChange(t *testing.T) {
	for _, operation := range []string{"searchHub", "installHub"} {
		for _, change := range []string{"workspace", "host"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					f := &skillHubTestRPC{resultName: "stale"}
					defer f.unblock()
					s := newSkillHubTestState(t, f)
					s.searchSettingsSkillHub("synthetic")
					synctest.Wait()
					f.holdMethod, f.release = "skillInventory/"+operation, make(chan struct{})
					if operation == "installHub" {
						hit := s.settingsSkillHubHits[0]
						s.requestSettingsSkillHubInstall(hit.Repo, hit.Path, hit.SkillName)
						s.confirmSettingsSkillHubInstall()
					} else {
						s.searchSettingsSkillHub("held")
					}
					synctest.Wait()
					old := f.calls[len(f.calls)-1]
					if change == "workspace" {
						s.sessions.SetWorkspaceCwd(t.TempDir())
					} else {
						client := &dsh.Client{Endpoint: s.dshClient.Endpoint, HTTP: s.dshClient.HTTP}
						s.sessions.Attach(client, nil)
						s.dshMu.Lock()
						s.dshClient = client
						s.dshMu.Unlock()
					}
					if access := s.settingsSkillHubAccess(); access.Open || access.Confirming != nil || access.Searching || access.Installing || len(s.settingsSkillHubHits) != 0 || old.ctx.Err() == nil {
						t.Fatalf("scope change did not retire the pending dialog/context: %+v context=%v", access, old.ctx.Err())
					}
					f.holdMethod, f.resultName = "", "current"
					s.openSettingsSkillHub()
					s.searchSettingsSkillHub("new scope")
					synctest.Wait()
					s.settingsSkillsLastAction = "current scope"
					close(f.release)
					synctest.Wait()
					if len(s.settingsSkillHubHits) != 1 || s.settingsSkillHubHits[0].SkillName != "current" || s.settingsSkillHubHits[0].Status != "" || s.settingsSkillsLastAction != "current scope" || s.settingsSkillHubAccess().Error != "" {
						t.Fatalf("late %s completion changed the new scope: hits=%+v action=%q", operation, s.settingsSkillHubHits, s.settingsSkillsLastAction)
					}
				})
			})
		}
	}
}
