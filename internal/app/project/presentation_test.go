package projectctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

func TestPresentationFrameRetiresObserversAndRetainsMaterialDraft(t *testing.T) {
	client := &dsh.Client{}
	mgr := NewManager(HostDeps{HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} }})
	mgr.primary.flow.client = client
	o := &projects.ObserverState{ProjectID: "project", Section: "tasks", HistoryKind: "completed", RequestModes: map[string]projects.RequestMode{"request": {Mode: "card", Tab: "results"}}}
	o.Scroll.Y = 120
	finish := mgr.BeginPresentationFrame()
	c := mgr.Observer(o)
	ctx, cancel := context.WithCancel(context.Background())
	draft := &projectflow.MaterialDraft{ProjectID: "project", Draft: "unsaved", Saved: "saved", Loaded: true}
	mgr.mu.Lock()
	c.flow.materials = map[string]*projectflow.MaterialDraft{"notes": draft}
	c.flow.materialKey = "notes"
	c.flow.refreshLoop = &refreshLoop{cancel: cancel, client: client}
	mgr.mu.Unlock()
	finish()
	select {
	case <-ctx.Done():
		t.Fatal("visible observer was stopped")
	default:
	}
	finish = mgr.BeginPresentationFrame()
	finish()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("closed observer kept its refresh loop")
	}
	mgr.mu.Lock()
	remaining := len(mgr.observers)
	mgr.mu.Unlock()
	if remaining != 0 || len(mgr.observerReadings) != 1 || o.OnLoadCatalog != nil {
		t.Fatal("closed observer retained active ownership")
	}
	nextState := &projects.ObserverState{ProjectID: "project", Section: "tasks"}
	reopened := mgr.Observer(nextState)
	if reopened == c || reopened.flow.materials["notes"] != draft || !draft.Dirty() {
		t.Fatal("reopening lost material draft or reused retired controller")
	}
	if nextState.Scroll.Y != 120 || nextState.HistoryKind != "completed" || nextState.RequestModes["request"].Tab != "results" {
		t.Fatal("reopening lost reading position")
	}
	mgr.Close()
}

func TestPresentationFrameRetiresUnshownPublishedPage(t *testing.T) {
	mgr, c, client, card, _ := cardOwnershipFixture(t)
	finish := mgr.BeginPresentationFrame()
	mgr.mu.Lock()
	owner, _ := c.visualOwnerLocked("request:A", card.Delivery.Pages[0], client)
	ctx, cancel := context.WithCancel(context.Background())
	c.flow.visualResults = &visualResultsState{pages: map[visualOwner]*visualPage{owner: {ctx: ctx, cancel: cancel, presented: mgr.presentationFrame}}}
	mgr.primaryPresented = mgr.presentationFrame
	mgr.mu.Unlock()
	finish()
	finish = mgr.BeginPresentationFrame()
	mgr.mu.Lock()
	mgr.primaryPresented = mgr.presentationFrame
	mgr.mu.Unlock()
	finish()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("unshown published page retained its renderer lifetime")
	}
	if len(c.flow.visualResults.pages) != 0 {
		t.Fatal("unshown page stayed cached with live ownership")
	}
}

func TestWindowHiddenReleasesPublishedPageWithoutLosingCard(t *testing.T) {
	mgr, c, client, card, _ := cardOwnershipFixture(t)
	mgr.mu.Lock()
	owner, _ := c.visualOwnerLocked("request:A", card.Delivery.Pages[0], client)
	ctx, cancel := context.WithCancel(context.Background())
	c.flow.visualResults = &visualResultsState{pages: map[visualOwner]*visualPage{owner: {ctx: ctx, cancel: cancel, opened: true}}}
	mgr.mu.Unlock()
	mgr.SetWindowHidden(true)
	select {
	case <-ctx.Done():
	default:
		t.Fatal("hidden window retained its published page renderer")
	}
	if c.flow.requestCards[requestCardKey(card.ProjectID, card.ID)].ID != card.ID {
		t.Fatal("hiding the window discarded the observed requirement card")
	}
	mgr.SetWindowHidden(false)
	var model projects.Model
	c.bindVisualResults("win", &model)
	preview := model.VisualPage("request:A", card.Delivery.Pages[0], 500)
	if !preview.Available || preview.Open == nil || preview.Opened || preview.Loading {
		t.Fatal("restoring the window lost its page entry or reopened its renderer")
	}
	mgr.Close()
}

func TestHiddenObserverKeepsAcceptedActionUntilCompletion(t *testing.T) {
	client := &dsh.Client{}
	mgr := NewManager(HostDeps{HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} }})
	mgr.primary.flow.client = client
	o := &projects.ObserverState{ProjectID: "project"}
	finish := mgr.BeginPresentationFrame()
	c := mgr.Observer(o)
	c.flow.busy = true // An accepted business action, not a cancellable page read.
	finish()
	finish = mgr.BeginPresentationFrame()
	finish()
	if mgr.observers[o] != c || c.flow.observerOpen || !c.flow.busy {
		t.Fatal("closing the observer discarded its in-flight business action")
	}
	finish = mgr.BeginPresentationFrame()
	replacement := &projects.ObserverState{ProjectID: "project"}
	if mgr.Observer(replacement) != c || !c.flow.busy || len(mgr.observers) != 1 {
		t.Fatal("reopening during an accepted action created a second idle owner")
	}
	finish()
	c.flow.busy = false // Its normal completion callback publishes the outcome.
	c.flow.err = "owned completion"
	finish = mgr.BeginPresentationFrame()
	finish()
	if len(mgr.observers) != 0 {
		t.Fatal("completed hidden action retained a controller")
	}
	reopened := mgr.Observer(o)
	if reopened.flow.err != "owned completion" {
		t.Fatal("hidden action result was lost")
	}
	mgr.Close()
}

func TestHiddenObserverCompletesAcceptedWorkStop(t *testing.T) {
	for _, refreshFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("refresh-fails-%t", refreshFails), func(t *testing.T) {
			started, release, completed := make(chan context.Context, 1), make(chan struct{}), make(chan struct{})
			var stops atomic.Int32
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:19876"}}
			work := projectflow.RequestWork{WorkstreamID: "work", DelegationRef: "delegation", WorkerSessionID: "worker"}
			card := projectflow.RequestCard{ID: "request", ProjectID: "project", Revision: 1, Work: []projectflow.RequestWork{work}}
			client.HTTP = &http.Client{Transport: requestDetailFixture(func(req *http.Request) (*http.Response, error) {
				var envelope struct {
					RPCID string `json:"rpcId"`
				}
				if err := json.NewDecoder(req.Body).Decode(&envelope); err != nil {
					return nil, err
				}
				var value any
				switch req.URL.Path {
				case "/dsh-project/requests/get":
					value = map[string]any{"card": card}
				case "/dsh-project/workstreams/get":
					value = map[string]any{"workstream": map[string]any{"id": "work", "projectId": "project", "currentDelegationRef": "delegation", "request": map[string]any{"id": "request", "revision": 1}}}
				case "/dsh-project/stop":
					stops.Add(1)
					started <- req.Context()
					<-release
					value = map[string]any{"note": "stopped"}
				case "/dsh-project/detail":
					if refreshFails {
						return nil, fmt.Errorf("owned refresh failure")
					}
					value = map[string]any{"project": map[string]any{"id": "project", "coordinatorSessionId": "lead", "activity": map[string]any{"state": "idle"}}}
				case "/api/session/projections":
					value = map[string]any{"values": map[string]any{"agentTeam": map[string]any{"members": []any{}, "tasks": []any{}}}}
				case "/api/session/list":
					value = map[string]any{"items": []any{}}
				default:
					return nil, fmt.Errorf("unexpected endpoint: %s", req.URL.Path)
				}
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.RPCID, "result": map[string]any{"ok": true, "value": value}})
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}, nil
			})}
			mgr := NewManager(HostDeps{HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} }, UpdateWindow: func(_ any, fn func()) { fn(); close(completed) }})
			mgr.primary.flow.client = client
			mgr.primary.flow.listing = projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "project", CoordinatorSessionID: "lead", Lifecycle: "ready"}}}
			o := &projects.ObserverState{ProjectID: "project", CoordinatorSessionID: "lead"}
			finish := mgr.BeginPresentationFrame()
			c := mgr.Observer(o)
			c.flow.requestProjectID = "project"
			c.rememberRequestCardLocked(&card)
			state := c.revertStateLocked()
			c.runRequestWorkAction("win", state, requestRevision{"request", 1}, work, "stop")
			var operationContext context.Context
			select {
			case operationContext = <-started:
			case <-time.After(time.Second):
				t.Fatal("stop never reached the Host")
			}
			finish()
			finish = mgr.BeginPresentationFrame()
			finish()
			if operationContext.Err() != nil || mgr.observers[o] != c || !c.flow.busy {
				t.Fatal("closing the observer retired its accepted stop")
			}
			close(release)
			select {
			case <-completed:
			case <-time.After(time.Second):
				t.Fatal("stop completion never arrived")
			}
			if c.flow.busy || stops.Load() != 1 {
				t.Fatal("hidden stop remained busy or was repeated")
			}
			if refreshFails && !strings.Contains(c.flow.err, "owned refresh failure") {
				t.Fatalf("lost hidden stop refresh failure: %q", c.flow.err)
			}
			if !refreshFails && (c.flow.team == nil || c.flow.listing.Projects[0].Activity.State != "idle") {
				t.Fatal("hidden stop did not update its Project snapshot")
			}
			finish = mgr.BeginPresentationFrame()
			finish()
			if len(mgr.observers) != 0 {
				t.Fatal("completed stop retained its observer controller")
			}
			mgr.Close()
		})
	}
}

func TestDeliveryFileCloseCancelsReadAndLateResultCannotClearNextOperation(t *testing.T) {
	for _, action := range []string{"close", "back-list"} {
		t.Run(action, func(t *testing.T) {
			started, release, canceled, applied := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
			var opened atomic.Bool
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:19876"}}
			client.HTTP = &http.Client{Transport: requestDetailFixture(func(req *http.Request) (*http.Response, error) {
				var envelope struct {
					RPCID string `json:"rpcId"`
				}
				_ = json.NewDecoder(req.Body).Decode(&envelope)
				close(started)
				<-req.Context().Done()
				close(canceled)
				<-release
				body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"canonicalPath": "C:/fixture/file.txt", "root": "C:/fixture"}}})
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}, nil
			})}
			mgr := NewManager(HostDeps{HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} }, UpdateWindow: func(_ any, fn func()) { fn(); close(applied) }})
			c := mgr.Primary()
			file := projectflow.Artifact{Path: "file.txt", WorkstreamID: "work", DelegationRef: "delegation"}
			d := &deliveryState{browser: projectflow.NewDeliveryBrowser(projectflow.DeliveryScope{Host: fmt.Sprintf("%p", client), ProjectID: "project"}, "Project")}
			r, _ := d.browser.BeginPage("refresh")
			d.browser.CompletePage(r, projectflow.DeliveryHistoryPage{Items: []projectflow.HistoricalDelivery{{ID: "delivery", ProjectID: "project", RequestID: "request", RequestRevision: 1, Status: "delivered", Artifacts: []projectflow.Artifact{file}}}}, nil)
			if err := d.browser.Select("delivery"); err != nil {
				t.Fatal(err)
			}
			c.flow = flowState{client: client, expanded: true, deliveries: d}
			c.openDeliveryFile("win", d, file, func(projectflow.FileTarget) { opened.Store(true) })
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("file resolution never started")
			}
			c.runDeliveries("win", d, action, "", projectflow.DeliveryReportRef{})
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("close did not cancel resolution")
			}
			mgr.mu.Lock()
			if c.flow.busy {
				mgr.mu.Unlock()
				t.Fatal("close left project busy")
			}
			_, nextCancel := context.WithCancel(context.Background())
			next := c.beginReadLocked(nextCancel)
			mgr.mu.Unlock()
			close(release)
			select {
			case <-applied:
			case <-time.After(time.Second):
				t.Fatal("late callback never completed")
			}
			mgr.mu.Lock()
			if !c.flow.busy || c.flow.readOperation != next || opened.Load() {
				mgr.mu.Unlock()
				t.Fatal("late callback affected next operation or opened stale file")
			}
			c.finishReadLocked(next)
			mgr.mu.Unlock()
		})
	}
}
