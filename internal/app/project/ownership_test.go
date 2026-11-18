package projectctl

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

type requestDetailFixture func(*http.Request) (*http.Response, error)

func (f requestDetailFixture) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type dummyRecoveryTarget struct {
	snap   LeadDraftSnapshot
	staged string
}

func (d *dummyRecoveryTarget) Snapshot() LeadDraftSnapshot { return d.snap }
func (d *dummyRecoveryTarget) StageRecoveryText(text string) {
	d.snap.Text = text
	d.staged = text
}

func cardOwnershipFixture(t *testing.T) (*Manager, *Controller, *dsh.Client, projectflow.RequestCard, projectflow.RequestCard) {
	t.Helper()
	client := &dsh.Client{}
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} },
	})
	c := mgr.Primary()
	mgr.mu.Lock()
	c.flow = flowState{
		client:            client,
		expanded:          true,
		navigation:        7,
		generation:        11,
		requestGeneration: 13,
		requestProjectID:  "owned-project",
		listing:           projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "owned-project", Lifecycle: "ready"}}},
	}
	mgr.mu.Unlock()

	card := func(id string) projectflow.RequestCard {
		var out projectflow.RequestCard
		raw := fmt.Sprintf(`{"id":%q,"projectId":"owned-project","revision":3,"title":%q,"status":"completed","changes":{"isGit":true,"revertible":true},"work":[{"workstreamId":%q,"delegationRef":"saved-assignment"}],"delivery":{"id":%q,"outcome":"completed","artifacts":[{"path":%q,"workstreamId":%q,"delegationRef":"saved-assignment"}],"pages":[{"sessionId":%q,"callId":%q,"title":"Owned page","height":360,"file":{"attachmentId":%q,"name":"owned.html","bytes":42}}]}}`, id, id, "work-"+id, "delivery-"+id, id+".md", "work-"+id, "session-"+id, "call-"+id, "attachment-"+id)
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	a, b := card("A"), card("B")
	mgr.mu.Lock()
	c.rememberRequestCardLocked(&a)
	c.flow.requestCard = &a
	mgr.mu.Unlock()
	return mgr, c, client, a, b
}

func rememberOtherCard(mgr *Manager, c *Controller, b *projectflow.RequestCard) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	c.flow.generation++
	c.flow.requestGeneration++
	c.flow.requestCard = b
	c.rememberRequestCardLocked(b)
}

func TestCardsKeepIndependentRevisionAndConflictOwnership(t *testing.T) {
	mgr, c, client, a, b := cardOwnershipFixture(t)
	mgr.mu.Lock()
	state := c.revertStateLocked()
	key := requestRevision{a.ID, a.Revision}
	state.errors[key] = "pluginrpc: revert_conflict: owned A changed after delivery"
	owner, allowed := c.visualOwnerLocked("request:A", a.Delivery.Pages[0], client)
	mgr.mu.Unlock()
	if !allowed {
		t.Fatal("admitted A page unavailable")
	}

	rememberOtherCard(mgr, c, &b)

	mgr.mu.Lock()
	sameState := c.revertStateLocked() == state
	stillCurrent := c.visualOwnerCurrentLocked(owner, client)
	mgr.mu.Unlock()
	if !sameState || !stillCurrent {
		t.Fatal("reading B retired A's revision scope or published page")
	}

	var m projects.Model
	c.bindRequestReverts(nil, &m)
	c.bindRequestCardActions(nil, &m)
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
		mgr.mu.Lock()
		c.flow.listing.Projects[0].Archiving = !mutable
		mgr.mu.Unlock()
		if got := m.RequestRevert(a.ID, a.Revision).Allowed; got != mutable {
			t.Fatalf("lifecycle did not gate Undo: allowed=%v mutable=%v", got, mutable)
		}
	}
}

func TestCachedReferencesAreFrozenAndStayOnCardA(t *testing.T) {
	mgr, c, client, a, b := cardOwnershipFixture(t)
	mgr.mu.Lock()
	state := c.revertStateLocked()
	mgr.mu.Unlock()
	file := a.Delivery.Artifacts[0]
	page := a.Delivery.Pages[0]
	// Mutating an admitted caller/UI copy must not rewrite the cached grant.
	a.Delivery.Artifacts[0].Path = "injected.md"
	a.Delivery.Pages[0].File.AttachmentID = "injected"
	a.Work[0].WorkstreamID = "injected-work"
	a.Changes.Revertible = false

	rememberOtherCard(mgr, c, &b)

	mgr.mu.Lock()
	cached, known := c.revertCardLocked(state, requestRevision{"A", 3}, client)
	mgr.mu.Unlock()
	if !known || cached.Delivery.Pages[0] != page || !cached.Changes.Revertible || cached.Work[0].WorkstreamID != "work-A" {
		t.Fatal("caller mutation or B read changed A's cached authority")
	}
	if id, err := AssetCardReference(cached, "file", file, "", true); err != nil || id != "A" {
		t.Fatalf("A's delivered file was rejected: id=%q err=%v", id, err)
	}
	if _, err := AssetCardReference(cached, "file", b.Delivery.Artifacts[0], "", true); err == nil {
		t.Fatal("B's file was admitted under A")
	}
	if id, err := AssetCardReference(cached, "diff", projectflow.Artifact{}, "work-A", false); err != nil || id != "A" {
		t.Fatalf("A's workspace diff was rejected: id=%q err=%v", id, err)
	}
	if _, err := AssetCardReference(cached, "diff", projectflow.Artifact{}, "work-B", false); err == nil {
		t.Fatal("B's work was admitted under A")
	}
}

func TestActivityAndPageSurviveReadingAnotherCard(t *testing.T) {
	mgr, c, client, a, b := cardOwnershipFixture(t)
	mgr.mu.Lock()
	state := c.revertStateLocked()
	gen := c.flow.generation
	mgr.mu.Unlock()

	browser := projectflow.NewActivityBrowser(projectflow.ActivityScope{Host: fmt.Sprintf("%p", client), ProjectID: a.ProjectID, RequestID: a.ID, Generation: gen})
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
	activity := &activityState{browser: browser, owner: state, request: requestRevision{a.ID, a.Revision}}
	mgr.mu.Lock()
	c.flow.activity = activity
	activityView := browser.View()
	owner, ok := c.visualOwnerLocked("activity:"+snapshot.Events[0].Key, a.Delivery.Pages[0], client)
	mgr.mu.Unlock()
	if !ok {
		t.Fatal("activity publication unavailable")
	}
	historicalSourceValid := func() bool {
		mgr.mu.Lock()
		defer mgr.mu.Unlock()
		return c.activityHistoricalSourceValidLocked(&c.flow, activity, activityView, client)
	}
	if !historicalSourceValid() {
		t.Fatal("admitted A activity did not provide a historical source owner")
	}

	rememberOtherCard(mgr, c, &b)
	var m projects.Model
	c.bindActivity(nil, &m)

	mgr.mu.Lock()
	activityKept := c.flow.activity == activity && c.visualOwnerCurrentLocked(owner, client)
	mgr.mu.Unlock()
	if !activityKept || m.Activity == nil || m.Activity.Scope.RequestID != "A" {
		t.Fatal("B read replaced A's activity/browser publication owner")
	}
	if !historicalSourceValid() {
		t.Fatal("reading B retired the cached A historical source revision")
	}

	mgr.mu.Lock()
	c.flow.activity = &activityState{browser: browser, owner: state, request: activity.request}
	mgr.mu.Unlock()
	if historicalSourceValid() {
		t.Fatal("replacement activity retained the old historical source callback")
	}

	mgr.mu.Lock()
	c.flow.activity = activity
	mgr.mu.Unlock()
	if !historicalSourceValid() {
		t.Fatal("restoring the actual activity did not restore its unchanged owner")
	}

	changed := a
	changed.Revision++
	mgr.mu.Lock()
	c.rememberRequestCardLocked(&changed)
	mgr.mu.Unlock()
	if historicalSourceValid() {
		t.Fatal("historical source survived A's actual cached revision change")
	}

	c.bindActivity(nil, &m)
	mgr.mu.Lock()
	cleared := c.flow.activity == nil && !c.visualOwnerCurrentLocked(owner, client)
	mgr.mu.Unlock()
	if !cleared {
		t.Fatal("activity survived a real source revision change")
	}
}

func TestWorkInspectorReadsCardAAfterCardBBecomesCurrent(t *testing.T) {
	mgr, c, client, a, b := cardOwnershipFixture(t)
	mgr.mu.Lock()
	state := c.revertStateLocked()
	mgr.mu.Unlock()
	rememberOtherCard(mgr, c, &b)

	mgr.mu.Lock()
	card, known := c.revertCardLocked(state, requestRevision{a.ID, a.Revision}, client)
	mgr.mu.Unlock()
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
		case "/api/session/list":
			value = map[string]any{"items": []any{}}
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
	_, detail, err := ReadRequestWork(context.Background(), rpc, card.ProjectID, card.ID, card.Revision, "work-A")
	if err != nil || detail.Workstream.Request.ID != "A" || len(calls) != 5 {
		t.Fatalf("inspector did not retain A's read chain: err=%v calls=%v", err, calls)
	}
}

func TestPublicationRetiresOnRealOwnerOrAttachmentChange(t *testing.T) {
	for _, name := range []string{"Host", "navigation", "closed", "project", "attachment", "session", "height", "removed"} {
		t.Run(name, func(t *testing.T) {
			mgr, c, client, a, _ := cardOwnershipFixture(t)
			activeClient := client
			mgr.deps.HostSnapshot = func() HostSnapshot { return HostSnapshot{Client: activeClient} }

			mgr.mu.Lock()
			owner, ok := c.visualOwnerLocked("request:A", a.Delivery.Pages[0], client)
			mgr.mu.Unlock()
			if !ok {
				t.Fatal("page fixture unavailable")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			page := &visualPage{ctx: ctx, cancel: cancel}
			cache := &visualResultsState{pages: map[visualOwner]*visualPage{owner: page}}
			mgr.mu.Lock()
			c.flow.visualResults = cache
			switch name {
			case "Host":
				activeClient = &dsh.Client{}
				c.flow.client = activeClient
			case "navigation":
				c.flow.navigation++
			case "closed":
				c.closeVisualResultsLocked()
			case "project":
				c.flow.requestProjectID = "other-project"
			case "attachment":
				a.Delivery.Pages[0].File.AttachmentID = "new-attachment"
				c.rememberRequestCardLocked(&a)
			case "session":
				a.Delivery.Pages[0].SessionID = "other-session"
				c.rememberRequestCardLocked(&a)
			case "height":
				a.Delivery.Pages[0].Height++
				c.rememberRequestCardLocked(&a)
			case "removed":
				a.Delivery.Pages = nil
				c.rememberRequestCardLocked(&a)
			}
			mgr.mu.Unlock()

			if c.visualCurrent(owner, cache, page) {
				t.Fatal("obsolete publication owner remained current")
			}
			var m projects.Model
			c.bindVisualResults(nil, &m) // Prune only: nil window cannot start a reader/helper.
			select {
			case <-ctx.Done():
			default:
				t.Fatal("retired publication did not cancel its resource lifetime")
			}
		})
	}
}

func TestVisualPageUsesBoundModelAppearance(t *testing.T) {
	for _, dark := range []bool{false, true} {
		mgr, c, client, card, _ := cardOwnershipFixture(t)
		reference := card.Delivery.Pages[0]
		mgr.mu.Lock()
		owner, ok := c.visualOwnerLocked("request:A", reference, client)
		if !ok {
			mgr.mu.Unlock()
			t.Fatal("fixture publication was not admitted")
		}
		page := &visualPage{width: 320, dark: !dark, state: projects.VisualPagePreview{Available: true}}
		c.flow.visualResults = &visualResultsState{pages: map[visualOwner]*visualPage{owner: page}}
		mgr.mu.Unlock()

		model := projects.Model{SystemDark: dark}
		c.bindVisualResults("fake-window", &model)
		preview := model.VisualPage("request:A", reference, 320)
		mgr.mu.Lock()
		gotDark := page.dark
		mgr.mu.Unlock()
		if !preview.Available || gotDark != dark {
			t.Fatal("retained publication did not use the bound model appearance")
		}
	}
}

func TestQueueUsesDisplayedRevisionAndBeforeIdentity(t *testing.T) {
	queue := &projectflow.RequestQueue{Queued: []projectflow.RequestQueueEntry{{ID: "a", Revision: 1}, {ID: "b", Revision: 2}, {ID: "c", Revision: 3}, {ID: "d", Revision: 4}}}
	page := projectflow.RequestPage{Items: []projectflow.RequestCard{{ID: "b", Revision: 12}}}
	for _, tc := range []struct {
		id, action, before string
		revision           int
	}{{"b", "moveUp", "a", 12}, {"b", "moveDown", "d", 12}, {"c", "moveDown", "", 3}, {"a", "cancel", "", 1}, {"b", "enqueue", "", 12}} {
		change, err := QueueChange(page, nil, queue, tc.id, tc.action)
		if err != nil || change.RequestID != tc.id || change.RequestRevision != tc.revision || change.BeforeRequestID != tc.before {
			t.Fatalf("%s %s: %+v %v", tc.id, tc.action, change, err)
		}
		if (tc.action == "moveUp" || tc.action == "moveDown") && change.Action != "move" {
			t.Fatal("UI move direction leaked into Host protocol")
		}
	}
	detail := &projectflow.RequestCard{ID: "b", Revision: 20}
	change, err := QueueChange(page, detail, queue, "b", "resume")
	if err != nil || change.RequestRevision != 20 {
		t.Fatal("fresh selected detail lost its revision to the list")
	}
	for _, tc := range []struct{ id, action string }{{"a", "moveUp"}, {"d", "moveDown"}, {"missing", "moveUp"}, {"b", "invented"}} {
		if _, err := QueueChange(page, nil, queue, tc.id, tc.action); err == nil {
			t.Fatalf("invalid ordering accepted: %+v", tc)
		}
	}
	if _, err := QueueChange(page, nil, nil, "b", "enqueue"); err == nil {
		t.Fatal("missing Host queue projection accepted a write")
	}
	if queue.Queued[1].ID != "b" || queue.Queued[1].Revision != 2 {
		t.Fatal("building a command mutated retained queue state")
	}
}

func TestQueueSubmissionKeepsRenderedHostAndRoute(t *testing.T) {
	a, b := &dsh.Client{}, &dsh.Client{}
	f := flowState{client: a, generation: 3, navigation: 4, requestGeneration: 5}
	owner := QueueSubmission{Client: a, Generation: 3, Navigation: 4, RequestGeneration: 5}
	if !owner.Matches(&f, a) {
		t.Fatal("current rendered queue owner was rejected")
	}
	f.client = b
	if owner.Matches(&f, b) {
		t.Fatal("old button adopted a replacement Host with the same identities")
	}
	f.client = a
	f.navigation++
	if owner.Matches(&f, a) {
		t.Fatal("old button survived a close/reopen on the same Host")
	}
	f.navigation = 4
	f.requestGeneration++
	if owner.Matches(&f, a) {
		t.Fatal("old button survived replacement request data")
	}
	if (QueueSubmission{}).Matches(&flowState{}, nil) {
		t.Fatal("disconnected queue callback was writable")
	}
}

func TestSidebarProjectDirectoryNeedsKnownProjectWithoutDeliveryCard(t *testing.T) {
	f := flowState{listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "project"}}}}
	if _, err := assetReference(&f, "project", "directory", projectflow.Artifact{}, "", false); err != nil {
		t.Fatalf("sidebar directory incorrectly required a delivery: %v", err)
	}
	if _, err := assetReference(&f, "other", "directory", projectflow.Artifact{}, "", false); err == nil {
		t.Fatal("unknown Project directory was admitted")
	}
	if _, err := assetReference(&f, "project", "diff", projectflow.Artifact{}, "work", false); err == nil {
		t.Fatal("directory fix removed the delivery work guard")
	}
	f.requestProjectID = "project"
	f.requestCard = &projectflow.RequestCard{ID: "request", Work: []projectflow.RequestWork{{WorkstreamID: "work"}}}
	if _, err := assetReference(&f, "project", "diff", projectflow.Artifact{}, "foreign", false); err == nil {
		t.Fatal("foreign delivery work admitted")
	}
	if id, err := assetReference(&f, "project", "diff", projectflow.Artifact{}, "work", false); err != nil || id != "request" {
		t.Fatal("known delivery work rejected")
	}
}

func TestRecoveryExplicitSend(t *testing.T) {
	d := &dummyRecoveryTarget{}
	calls := 0
	err := SubmitRecoveryDraft(d, "verified original Lead request", func() {
		calls++
		if d.staged != "verified original Lead request" {
			t.Fatal(d.staged)
		}
	})
	if err != nil || calls != 1 {
		t.Fatalf("explicit recovery click must submit once: calls=%d err=%v", calls, err)
	}
	if d.snap.Text != "verified original Lead request" {
		t.Fatal("failed send must keep staged draft", d)
	}
}

func TestRecoveryPreservesExistingDraft(t *testing.T) {
	cases := []*dummyRecoveryTarget{
		{snap: LeadDraftSnapshot{Text: "existing"}},
		{snap: LeadDraftSnapshot{HasAttachments: true}},
		{snap: LeadDraftSnapshot{Sending: true}},
		{snap: LeadDraftSnapshot{Preparing: 1}},
		{snap: LeadDraftSnapshot{EditSeq: 1}},
		{snap: LeadDraftSnapshot{EditSessionID: "s1"}},
	}
	for _, d := range cases {
		calls := 0
		if err := SubmitRecoveryDraft(d, "recovery", func() { calls++ }); err == nil || calls != 0 {
			t.Fatalf("existing draft overwritten or sent: %+v", d.snap)
		}
	}
}
