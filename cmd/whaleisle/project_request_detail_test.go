package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"whaleisle/internal/projectflow"
)

type requestDetailFixture func(*http.Request) (*http.Response, error)

func (f requestDetailFixture) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestWorkDetailLoadsTeamOnFirstTaskEntryWithoutMutations(t *testing.T) {
	calls := []string{}
	transport := requestDetailFixture(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, r.URL.Path)
		var value any
		switch r.URL.Path {
		case "/dsh-project/requests/get":
			value = map[string]any{"card": map[string]any{"id": "request", "projectId": "project", "revision": 2, "work": []any{map[string]any{"workstreamId": "work"}}}}
		case "/dsh-project/detail":
			value = map[string]any{"project": map[string]any{"id": "project", "coordinatorSessionId": "lead"}}
		case "/api/session/projections":
			value = map[string]any{"values": map[string]any{"agentTeam": map[string]any{"members": []any{}, "tasks": []any{}}}}
		case "/api/session/list":
			// Team status enrichment is read-only and precedes the work reader.
			value = map[string]any{"items": []any{}}
		case "/dsh-project/workstreams/get":
			value = map[string]any{"workstream": map[string]any{"id": "work", "projectId": "project", "request": map[string]any{"id": "request", "revision": 2}}}
		default:
			t.Errorf("inspector attempted an unexpected operation: %s", r.URL.Path)
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})
	rpc := projectflow.Client{Origin: "http://127.0.0.1:12345", HTTP: &http.Client{Transport: transport}}
	team, detail, err := readProjectRequestWork(context.Background(), rpc, "project", "request", 2, "work")
	if err != nil || team.Project.ID != "project" || team.Team == nil || !team.Team.StatusKnown || detail.Workstream.ID != "work" || len(calls) != 5 {
		t.Fatalf("first-entry work inspector did not load its read context: %v %#v", err, calls)
	}
	for _, tc := range []struct {
		revision int
		work     string
	}{{1, "work"}, {2, "foreign"}} {
		calls = nil
		if _, _, err := readProjectRequestWork(context.Background(), rpc, "project", "request", tc.revision, tc.work); err == nil || len(calls) != 1 {
			t.Fatalf("stale/foreign card reached the inspector: %v %#v", err, calls)
		}
	}
}

func TestSidebarProjectDirectoryNeedsKnownProjectWithoutDeliveryCard(t *testing.T) {
	f := projectFlowState{listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "project"}}}}
	if _, err := projectAssetReference(&f, "project", "directory", projectflow.Artifact{}, "", false); err != nil {
		t.Fatalf("sidebar directory incorrectly required a delivery: %v", err)
	}
	if _, err := projectAssetReference(&f, "other", "directory", projectflow.Artifact{}, "", false); err == nil {
		t.Fatal("unknown Project directory was admitted")
	}
	if _, err := projectAssetReference(&f, "project", "diff", projectflow.Artifact{}, "work", false); err == nil {
		t.Fatal("directory fix removed the delivery work guard")
	}
	f.requestProjectID = "project"
	f.requestCard = &projectflow.RequestCard{ID: "request", Work: []projectflow.RequestWork{{WorkstreamID: "work"}}}
	if _, err := projectAssetReference(&f, "project", "diff", projectflow.Artifact{}, "foreign", false); err == nil {
		t.Fatal("foreign delivery work admitted")
	}
	if id, err := projectAssetReference(&f, "project", "diff", projectflow.Artifact{}, "work", false); err != nil || id != "request" {
		t.Fatal("known delivery work rejected")
	}
}
