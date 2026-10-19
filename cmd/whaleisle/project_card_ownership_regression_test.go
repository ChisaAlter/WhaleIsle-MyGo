package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

func projectCardOwnershipFixture(t *testing.T) (*projectController, projectflow.RequestCard, projectflow.RequestCard) {
	t.Helper()
	client := &dsh.Client{}
	s := &projectController{mainState: &mainState{dshClient: client}, projectFlow: &projectFlowState{
		client: client, expanded: true, navigation: 7, generation: 11, requestGeneration: 13, requestProjectID: "owned-project",
		listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "owned-project", Lifecycle: "ready"}}},
	}}
	card := func(id string) projectflow.RequestCard {
		var out projectflow.RequestCard
		raw := fmt.Sprintf(`{"id":%q,"projectId":"owned-project","revision":3,"title":%q,"status":"completed","changes":{"isGit":true,"revertible":true},"work":[{"workstreamId":%q,"delegationRef":"saved-assignment"}],"delivery":{"id":%q,"outcome":"completed","artifacts":[{"path":%q,"workstreamId":%q,"delegationRef":"saved-assignment"}],"pages":[{"sessionId":%q,"callId":%q,"title":"Owned page","height":360,"file":{"attachmentId":%q,"name":"owned.html","bytes":42}}]}}`, id, id, "work-"+id, "delivery-"+id, id+".md", "work-"+id, "session-"+id, "call-"+id, "attachment-"+id)
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	a, b := card("A"), card("B")
	s.rememberProjectRequestCardLocked(&a)
	s.projectFlow.requestCard = &a
	return s, a, b
}

func rememberOtherProjectCard(s *projectController, b *projectflow.RequestCard) {
	s.projectFlow.generation++
	s.projectFlow.requestGeneration++
	s.projectFlow.requestCard = b
	s.rememberProjectRequestCardLocked(b)
}

func TestProjectCardsKeepIndependentRevisionAndConflictOwnership(t *testing.T) {
	s, a, b := projectCardOwnershipFixture(t)
	state := s.requestRevertStateLocked()
	key := projectRequestRevision{a.ID, a.Revision}
	state.errors[key] = "pluginrpc: revert_conflict: owned A changed after delivery"
	owner, allowed := s.projectVisualOwnerLocked("request:A", a.Delivery.Pages[0])
	if !allowed {
		t.Fatal("admitted A page unavailable")
	}
	rememberOtherProjectCard(s, &b)
	if s.requestRevertStateLocked() != state || !s.projectVisualCurrentLocked(owner) {
		t.Fatal("reading B retired A's revision scope or published page")
	}
	var m projects.Model
	s.bindProjectRequestReverts(nil, &m)
	s.bindProjectRequestCardActions(nil, &m)
	for _, card := range []projectflow.RequestCard{a, b} {
		if !m.RequestCardActions(card.ID, card.Revision).Known {
			t.Fatalf("independent card %s became unavailable", card.ID)
		}
	}
	undoA, undoB := m.RequestRevert(a.ID, a.Revision), m.RequestRevert(b.ID, b.Revision)
	if !undoA.Known || !undoA.Allowed || undoA.Error != state.errors[key] || undoB.Error != "" {
		t.Fatalf("conflict crossed card boundaries: A=%+v B=%+v", undoA, undoB)
	}
	if m.RequestRevert(a.ID, a.Revision+1).Known || m.RequestCardActions(a.ID, a.Revision+1).Known {
		t.Fatal("an undisplayed/unread revision was granted authority")
	}
	for _, mutable := range []bool{false, true} {
		s.projectFlow.listing.Projects[0].Archiving = !mutable
		if got := m.RequestRevert(a.ID, a.Revision).Allowed; got != mutable {
			t.Fatalf("lifecycle did not gate Undo: allowed=%v mutable=%v", got, mutable)
		}
	}
}

func TestProjectCachedReferencesAreFrozenAndStayOnCardA(t *testing.T) {
	s, a, b := projectCardOwnershipFixture(t)
	state := s.requestRevertStateLocked()
	file := a.Delivery.Artifacts[0]
	page := a.Delivery.Pages[0]
	// Mutating an admitted caller/UI copy must not rewrite the grant.
	a.Delivery.Artifacts[0].Path = "injected.md"
	a.Delivery.Pages[0].File.AttachmentID = "injected"
	a.Work[0].WorkstreamID = "injected-work"
	a.Changes.Revertible = false
	rememberOtherProjectCard(s, &b)
	cached, known := s.requestRevertCardLocked(state, projectRequestRevision{"A", 3})
	if !known || cached.Delivery.Pages[0] != page || !cached.Changes.Revertible || cached.Work[0].WorkstreamID != "work-A" {
		t.Fatal("caller mutation or B read changed A's cached authority")
	}
	if id, err := projectAssetCardReference(cached, "file", file, "", true); err != nil || id != "A" {
		t.Fatalf("A's delivered file was rejected: id=%q err=%v", id, err)
	}
	if _, err := projectAssetCardReference(cached, "file", b.Delivery.Artifacts[0], "", true); err == nil {
		t.Fatal("B's file was admitted under A")
	}
	if id, err := projectAssetCardReference(cached, "diff", projectflow.Artifact{}, "work-A", false); err != nil || id != "A" {
		t.Fatalf("A's workspace diff was rejected: id=%q err=%v", id, err)
	}
	if _, err := projectAssetCardReference(cached, "diff", projectflow.Artifact{}, "work-B", false); err == nil {
		t.Fatal("B's work was admitted under A")
	}
}

func TestProjectActivityAndItsPageSurviveReadingAnotherCard(t *testing.T) {
	s, a, b := projectCardOwnershipFixture(t)
	state := s.requestRevertStateLocked()
	browser := projectflow.NewActivityBrowser(projectflow.ActivityScope{Host: fmt.Sprintf("%p", s.dshClient), ProjectID: a.ProjectID, RequestID: a.ID, Generation: s.projectFlow.generation})
	request, err := browser.BeginRefresh()
	if err != nil {
		t.Fatal(err)
	}
	at := float64(1)
	if !browser.CompleteRefresh(request, projectflow.RequestActivityCard{ID: a.ID, ProjectID: a.ProjectID, Revision: a.Revision, Versions: []projectflow.ActivityVersion{}, Work: []projectflow.ActivityWork{}, Delivery: &projectflow.ActivityDelivery{ID: "delivery-A", At: &at, Pages: a.Delivery.Pages}}, nil) {
		t.Fatal("activity fixture was not admitted")
	}
	snapshot := browser.Snapshot()
	if snapshot.Error != "" || len(snapshot.Events) != 1 {
		t.Fatalf("invalid activity fixture: %+v", snapshot)
	}
	if err := browser.Select(snapshot.Events[0].Key); err != nil {
		t.Fatal(err)
	}
	activity := &projectActivityState{browser: browser, owner: state, request: projectRequestRevision{a.ID, a.Revision}}
	s.projectFlow.activity = activity
	owner, ok := s.projectVisualOwnerLocked("activity:"+snapshot.Events[0].Key, a.Delivery.Pages[0])
	if !ok {
		t.Fatal("activity publication unavailable")
	}
	rememberOtherProjectCard(s, &b)
	var m projects.Model
	s.bindProjectActivity(nil, &m)
	if s.projectFlow.activity != activity || m.Activity == nil || m.Activity.Scope.RequestID != "A" || !s.projectVisualCurrentLocked(owner) {
		t.Fatal("B read replaced A's activity/browser publication owner")
	}
	changed := a
	changed.Revision++
	s.rememberProjectRequestCardLocked(&changed)
	s.bindProjectActivity(nil, &m)
	if s.projectFlow.activity != nil || s.projectVisualCurrentLocked(owner) {
		t.Fatal("activity survived a real source revision change")
	}
}

func TestProjectWorkInspectorReadsCardAAfterCardBBecomesCurrent(t *testing.T) {
	s, a, b := projectCardOwnershipFixture(t)
	state := s.requestRevertStateLocked()
	rememberOtherProjectCard(s, &b)
	card, known := s.requestRevertCardLocked(state, projectRequestRevision{a.ID, a.Revision})
	if !known {
		t.Fatal("A became unavailable before inspector admission")
	}
	var calls []string
	rpc := projectflow.Client{Origin: "http://127.0.0.1:1", HTTP: &http.Client{Transport: requestDetailFixture(func(r *http.Request) (*http.Response, error) {
		var request struct {
			RPCID   string         `json:"rpcId"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		calls = append(calls, r.URL.Path)
		var value any
		switch r.URL.Path {
		case "/dsh-project/requests/get":
			if request.Payload["requestId"] != "A" || request.Payload["projectId"] != card.ProjectID {
				t.Fatalf("inspector used B instead of captured A: %+v", request.Payload)
			}
			value = map[string]any{"card": card}
		case "/dsh-project/detail":
			value = map[string]any{"project": map[string]any{"id": card.ProjectID, "coordinatorSessionId": "owned-lead"}}
		case "/api/session/projections":
			value = map[string]any{"values": map[string]any{"agentTeam": map[string]any{"members": []any{}, "tasks": []any{}}}}
		case "/dsh-project/workstreams/get":
			if request.Payload["workstreamId"] != "work-A" {
				t.Fatalf("inspector changed work ownership: %+v", request.Payload)
			}
			value = map[string]any{"workstream": map[string]any{"id": "work-A", "projectId": card.ProjectID, "request": map[string]any{"id": "A", "revision": card.Revision}}}
		default:
			t.Fatalf("inspector attempted an unexpected endpoint: %s", r.URL.Path)
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	_, detail, err := readProjectRequestWork(context.Background(), rpc, card.ProjectID, card.ID, card.Revision, "work-A")
	if err != nil || detail.Workstream.Request.ID != "A" || len(calls) != 4 {
		t.Fatalf("inspector did not retain A's read chain: err=%v calls=%v", err, calls)
	}
}

func TestProjectPublicationRetiresOnRealOwnerOrAttachmentChange(t *testing.T) {
	for _, name := range []string{"Host", "navigation", "closed", "project", "attachment", "session", "height", "removed"} {
		t.Run(name, func(t *testing.T) {
			s, a, _ := projectCardOwnershipFixture(t)
			owner, ok := s.projectVisualOwnerLocked("request:A", a.Delivery.Pages[0])
			if !ok {
				t.Fatal("page fixture unavailable")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			page := &projectVisualPage{ctx: ctx, cancel: cancel}
			cache := &projectVisualResultsState{pages: map[projectVisualOwner]*projectVisualPage{owner: page}}
			s.projectFlow.visualResults = cache
			switch name {
			case "Host":
				s.dshClient = &dsh.Client{}
				s.projectFlow.client = s.dshClient
			case "navigation":
				s.projectFlow.navigation++
			case "closed":
				s.closeProjectVisualResultsLocked()
			case "project":
				s.projectFlow.requestProjectID = "other-project"
			case "attachment":
				a.Delivery.Pages[0].File.AttachmentID = "new-attachment"
				s.rememberProjectRequestCardLocked(&a)
			case "session":
				a.Delivery.Pages[0].SessionID = "other-session"
				s.rememberProjectRequestCardLocked(&a)
			case "height":
				a.Delivery.Pages[0].Height++
				s.rememberProjectRequestCardLocked(&a)
			case "removed":
				a.Delivery.Pages = nil
				s.rememberProjectRequestCardLocked(&a)
			}
			if s.projectVisualCurrent(owner, cache, page) {
				t.Fatal("obsolete publication owner remained current")
			}
			var m projects.Model
			s.bindProjectVisualResults(nil, &m) // Prune only: nil window cannot start a reader/helper.
			select {
			case <-ctx.Done():
			default:
				t.Fatal("retired publication did not cancel its resource lifetime")
			}
		})
	}
}
