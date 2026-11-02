package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"whaleisle/internal/dsh"
)

type workspaceRefreshCall struct {
	method  string
	request map[string]any
	agentID string
	ctx     context.Context
	settled <-chan struct{}
}

type workspaceRefreshBody struct {
	*strings.Reader
	closed chan struct{}
}

func (b workspaceRefreshBody) Close() error {
	close(b.closed)
	return nil
}

type workspaceRefreshHost struct {
	store      *Store
	client     *dsh.Client
	mu         sync.Mutex
	calls      []workspaceRefreshCall
	beforeCall func(workspaceRefreshCall)
	handle     func(workspaceRefreshCall) (any, error)
	notify     chan struct{}
	callNotify chan struct{}
}

func newWorkspaceRefreshHost(t *testing.T) *workspaceRefreshHost {
	t.Helper()
	h := &workspaceRefreshHost{store: New(), notify: make(chan struct{}, 64), callNotify: make(chan struct{}, 1)}
	h.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://workspace-refresh.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request map[string]any `json:"request"`
					AgentID string         `json:"agentId"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		settled := make(chan struct{})
		call := workspaceRefreshCall{method: strings.TrimPrefix(r.URL.Path, "/api/"), request: body.Payload.Args.Request, agentID: body.Payload.Args.AgentID, ctx: r.Context(), settled: settled}
		h.mu.Lock()
		beforeCall := h.beforeCall
		h.mu.Unlock()
		if beforeCall != nil {
			beforeCall(call)
		}
		h.mu.Lock()
		h.calls = append(h.calls, call)
		handle := h.handle
		h.mu.Unlock()
		select {
		case h.callNotify <- struct{}{}:
		default:
		}
		value, err := workspaceRefreshValue(call)
		if handle != nil {
			value, err = handle(call)
		}
		if err != nil {
			close(settled)
			return nil, err
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
		if err != nil {
			close(settled)
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: workspaceRefreshBody{Reader: strings.NewReader(string(raw)), closed: settled}}, nil
	})}}
	h.store.Attach(h.client, func() {
		select {
		case h.notify <- struct{}{}:
		default:
		}
	})
	h.store.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"project","path":"/project","title":"Project","sessionIds":[]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	h.store.mu.Lock()
	h.store.cwd = "/configured"
	h.store.selectedID = "source"
	h.store.mu.Unlock()
	return h
}

func workspaceRefreshValue(call workspaceRefreshCall) (any, error) {
	switch call.method {
	case "session/create":
		return map[string]any{"sessionId": "created"}, nil
	case "session/projections":
		return map[string]any{"asOfSeq": 5, "values": map[string]any{"title": "Created title", "contextPressure": map[string]any{"projectedTokens": 250, "contextWindow": 1000}}}, nil
	case "skills/list":
		return map[string]any{"skills": []any{map[string]any{"name": "created-skill", "description": "created", "modelInvocable": true}}}, nil
	case "skillInventory/list":
		return map[string]any{"skills": []any{map[string]any{"name": "created-catalog", "userInvocable": true, "modelInvocable": true}}}, nil
	case "messageFeedback/list":
		return map[string]any{"ok": true, "value": map[string]any{"items": []any{map[string]any{"messageId": "created-message", "rating": "positive", "version": "v1"}}}}, nil
	case "goals/get":
		return map[string]any{}, nil
	default:
		return nil, fmt.Errorf("unexpected RPC %s", call.method)
	}
}

func (h *workspaceRefreshHost) callsSnapshot() []workspaceRefreshCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]workspaceRefreshCall(nil), h.calls...)
}

func (h *workspaceRefreshHost) create(t *testing.T) (*WorkspaceSessionTarget, string) {
	t.Helper()
	target, err := h.store.CaptureWorkspaceSessionTarget(h.store.Snapshot().WorkspaceLayout.HostGeneration, "project")
	if err != nil {
		t.Fatal(err)
	}
	id, err := target.Create(context.Background())
	if err != nil || id != "created" {
		t.Fatalf("create: %q %v", id, err)
	}
	return target, id
}

func (h *workspaceRefreshHost) await(t *testing.T, condition func() bool) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for !condition() {
		select {
		case <-h.notify:
		case <-h.callNotify:
		case <-timer.C:
			t.Fatalf("timed out waiting for selected views; calls=%+v snapshot=%+v feedback=%+v", h.callsSnapshot(), h.store.Snapshot(), h.store.Feedback())
		}
	}
}

func TestWorkspaceSessionCreationRefreshUnselected(t *testing.T) {
	for _, transition := range []string{"leave-unselected", "wrong-receipt", "canceled-create", "stale-selection"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newWorkspaceRefreshHost(t)
				target, err := h.store.CaptureWorkspaceSessionTarget(h.store.Snapshot().WorkspaceLayout.HostGeneration, "project")
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if transition == "canceled-create" {
					h.handle = func(call workspaceRefreshCall) (any, error) {
						if call.method == "session/create" {
							cancel()
						}
						return workspaceRefreshValue(call)
					}
				}
				id, createErr := target.Create(ctx)
				if transition == "canceled-create" {
					if createErr == nil || target.Select("created") {
						t.Fatal("canceled creation was accepted")
					}
				} else if createErr != nil {
					t.Fatal(createErr)
				} else if transition == "wrong-receipt" && target.Select("unrelated") {
					t.Fatal("wrong receipt was accepted")
				} else if transition == "stale-selection" {
					h.store.mu.Lock()
					h.store.selectSessionLocked("new-selection")
					h.store.mu.Unlock()
					if target.Select(id) {
						t.Fatal("stale receipt was accepted")
					}
				}
				// Include any goroutine launched erroneously by a rejected selection.
				synctest.Wait()
				for _, call := range h.callsSnapshot() {
					if call.method != "session/create" {
						t.Fatalf("unselected creation launched %s", call.method)
					}
				}
			})
		})
	}
}

func TestWorkspaceSessionCreationRefreshesSelectedViews(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newWorkspaceRefreshHost(t)
		target, id := h.create(t)
		if target.Select("wrong") {
			t.Fatal("selected an unrelated receipt")
		}
		synctest.Wait()
		if calls := h.callsSnapshot(); len(calls) != 1 || calls[0].method != "session/create" {
			t.Fatalf("creation without selection refreshed views: %+v", calls)
		}
		goalEntered := make(chan struct{})
		releaseGoal := make(chan struct{})
		releaseGoalResponse := make(chan struct{})
		var releaseOnce, responseOnce sync.Once
		defer releaseOnce.Do(func() { close(releaseGoal) })
		defer responseOnce.Do(func() { close(releaseGoalResponse) })
		h.beforeCall = func(call workspaceRefreshCall) {
			if call.method == "goals/get" {
				close(goalEntered)
				<-releaseGoal
			}
		}
		h.handle = func(call workspaceRefreshCall) (any, error) {
			if call.method == "goals/get" {
				// RPC arrival must wake await even before a goal response can notify.
				<-releaseGoalResponse
			}
			return workspaceRefreshValue(call)
		}
		if !target.Select(id) {
			t.Fatal("accepted creation receipt was not selected")
		}
		wait(t, goalEntered)
		synctest.Wait()
		h.await(t, func() bool {
			v, f := h.store.Snapshot(), h.store.Feedback()
			foundTitle := false
			for _, item := range v.Items {
				foundTitle = foundTitle || item.SessionID == id && item.Title == "Created title"
			}
			return foundTitle && v.ContextOccupancy != nil && v.ContextOccupancy.UsedTokens == 250 && len(v.SkillOptions) == 1 && v.SkillOptions[0].Name == "created-skill" && len(v.SkillCatalog.Skills) == 1 && v.SkillCatalog.Skills[0].Name == "created-catalog" && f.Ready && f.Items["created-message"].Version == "v1"
		})
		// Drain every earlier notification: only the gated goal arrival can wake
		// the final wait. Its response remains blocked until that wait succeeds.
		for len(h.notify) > 0 {
			<-h.notify
		}
		for len(h.callNotify) > 0 {
			<-h.callNotify
		}
		armed := false
		h.await(t, func() bool {
			if !armed {
				armed = true
				releaseOnce.Do(func() { close(releaseGoal) })
				return false
			}
			for _, call := range h.callsSnapshot() {
				if call.method == "goals/get" && call.agentID == id {
					return true
				}
			}
			return false
		})
		responseOnce.Do(func() { close(releaseGoalResponse) })
		synctest.Wait()
		methods := map[string]int{}
		for _, call := range h.callsSnapshot() {
			methods[call.method]++
			switch call.method {
			case "session/projections", "skills/list", "skillInventory/list", "messageFeedback/list":
				if call.request["sessionId"] != id {
					t.Errorf("%s sent to wrong session: %+v", call.method, call.request)
				}
			case "goals/get":
				if call.agentID != id {
					t.Errorf("goal sent to wrong session: %+v", call)
				}
			}
		}
		for _, method := range []string{"session/projections", "skills/list", "skillInventory/list", "messageFeedback/list", "goals/get"} {
			if methods[method] != 1 {
				t.Errorf("%s called %d times, want 1", method, methods[method])
			}
		}
	})
}

// The notification hook runs synchronously at the publication boundary, after
// selecting the receipt but before any refresh adapter can claim the selection.
func TestWorkspaceSessionCreationRefreshAdmissionFence(t *testing.T) {
	for _, transition := range []string{"switch", "away-and-back", "same-client-events", "host-aba"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newWorkspaceRefreshHost(t)
				target, id := h.create(t)
				var once sync.Once
				h.store.Attach(h.client, func() {
					once.Do(func() {
						switch transition {
						case "switch", "away-and-back":
							h.store.mu.Lock()
							h.store.selectSessionLocked("other")
							if transition == "away-and-back" {
								h.store.selectSessionLocked(id)
							}
							h.store.mu.Unlock()
						case "same-client-events":
							h.store.BeginEvents(context.Background(), h.client)
						case "host-aba":
							h.store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}, nil)
							h.store.Attach(h.client, nil)
						}
					})
				})
				if !target.Select(id) {
					t.Fatal("selection was not accepted")
				}
				// Drain any goroutines incorrectly launched after synchronous admission.
				synctest.Wait()
				for _, call := range h.callsSnapshot() {
					switch call.method {
					case "session/projections", "skills/list", "skillInventory/list", "messageFeedback/list":
						t.Fatalf("stale callback refreshed views after %s: %+v", transition, call)
					}
				}
				if h.store.Feedback().Loading {
					t.Fatal("stale callback marked feedback loading")
				}
			})
		})
	}
}

func TestWorkspaceSessionCreationRefreshInflightFence(t *testing.T) {
	for _, transition := range []string{"switch", "away-and-back", "same-client-events", "host-aba"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newWorkspaceRefreshHost(t)
				entered := make(chan workspaceRefreshCall, 4)
				release := make(chan struct{})
				var releaseOnce sync.Once
				defer releaseOnce.Do(func() { close(release) })
				h.handle = func(call workspaceRefreshCall) (any, error) {
					switch call.method {
					case "session/projections", "skills/list", "skillInventory/list", "messageFeedback/list":
						entered <- call
						<-release
					}
					return workspaceRefreshValue(call)
				}
				target, id := h.create(t)
				if !target.Select(id) {
					t.Fatal("selection rejected")
				}
				active := map[string]workspaceRefreshCall{}
				for len(active) < 4 {
					select {
					case call := <-entered:
						active[call.method] = call
					case <-time.After(5 * time.Second):
						t.Fatalf("not all refresh requests started: %+v", active)
					}
				}
				s := h.store
				switch transition {
				case "switch", "away-and-back":
					s.mu.Lock()
					s.selectSessionLocked("other")
					if transition == "away-and-back" {
						s.selectSessionLocked(id)
					}
					s.mu.Unlock()
				case "same-client-events":
					s.BeginEvents(context.Background(), h.client)
				case "host-aba":
					s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}, nil)
					s.Attach(h.client, nil)
				}
				releaseOnce.Do(func() { close(release) })
				for _, call := range active {
					wait(t, call.settled)
				}
				// Feedback's timeout context closes after its settlement, independently
				// of the transport response body.
				wait(t, active["messageFeedback/list"].ctx.Done())
				// Body closure precedes projection and skill result handling.
				synctest.Wait()
				s.mu.Lock()
				defer s.mu.Unlock()
				for _, item := range s.items {
					if item.SessionID == id && item.Title == "Created title" {
						t.Fatal("old projection changed the later session title")
					}
				}
				if len(s.skillOptions) != 0 || len(s.skillCatalog.Skills) != 0 || s.feedback.Ready || len(s.feedback.Items) != 0 {
					t.Fatalf("old refresh changed later owner: skills=%+v catalog=%+v feedback=%+v", s.skillOptions, s.skillCatalog, s.feedback)
				}
			})
		})
	}
}

func TestWorkspaceSessionCreationRefreshFallbackFence(t *testing.T) {
	for _, transition := range []string{"switch", "away-and-back", "same-client-events", "host-aba"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newWorkspaceRefreshHost(t)
				entered := make(chan workspaceRefreshCall, 2)
				release := make(chan struct{})
				var releaseOnce sync.Once
				defer releaseOnce.Do(func() { close(release) })
				h.handle = func(call workspaceRefreshCall) (any, error) {
					if call.method == "skills/list" || (call.method == "skillInventory/list" && call.request["sessionId"] == "created") {
						entered <- call
						<-release
						return nil, fmt.Errorf("fixture failure")
					}
					return workspaceRefreshValue(call)
				}
				target, id := h.create(t)
				if !target.Select(id) {
					t.Fatal("selection rejected")
				}
				var active []workspaceRefreshCall
				for len(active) < 2 {
					select {
					case call := <-entered:
						active = append(active, call)
					case <-time.After(5 * time.Second):
						t.Fatal("initial scoped skill requests did not start")
					}
				}
				switch transition {
				case "switch", "away-and-back":
					h.store.mu.Lock()
					h.store.selectSessionLocked("other")
					if transition == "away-and-back" {
						h.store.selectSessionLocked(id)
					}
					h.store.mu.Unlock()
				case "same-client-events":
					h.store.BeginEvents(context.Background(), h.client)
				case "host-aba":
					h.store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}, nil)
					h.store.Attach(h.client, nil)
				}
				releaseOnce.Do(func() { close(release) })
				for _, call := range active {
					wait(t, call.settled)
				}
				// Error return precedes fallback admission. Wait for the refresh
				// goroutines to finish their admission checks before asserting.
				synctest.Wait()
				var inventories int
				for _, call := range h.callsSnapshot() {
					if call.method == "skillInventory/list" {
						inventories++
						if call.request["sessionId"] != id {
							t.Errorf("stale global fallback: %+v", call)
						}
					}
				}
				if inventories != 1 {
					t.Fatalf("stale skill fallback made %d inventory requests; want initial scoped request only", inventories)
				}
			})
		})
	}
}

func TestWorkspaceSessionCreationRefreshCancellation(t *testing.T) {
	for _, phase := range []string{"before-admission", "in-flight"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := newWorkspaceRefreshHost(t)
				target, id := h.create(t)
				if !target.Select(id) {
					t.Fatal("selection rejected")
				}
				h.await(t, func() bool {
					v := h.store.Snapshot()
					titleReady := false
					for _, item := range v.Items {
						titleReady = titleReady || item.SessionID == id && item.Title == "Created title"
					}
					return titleReady && len(v.SkillOptions) == 1 && len(v.SkillCatalog.Skills) == 1 && h.store.Feedback().Ready
				})
				// The initial goal request is independent of the other views.
				synctest.Wait()
				s := h.store
				s.mu.Lock()
				owner := &workspaceSelectedOwner{client: h.client, hostGeneration: s.eventsGeneration, composerGeneration: s.composerHostGeneration, followGeneration: s.followGeneration, sessionID: id, cwd: s.cwd}
				s.clearFeedbackLocked()
				s.invalidateSkillOptionsLocked()
				for i := range s.items {
					if s.items[i].SessionID == id {
						s.items[i].Title = ""
					}
				}
				s.mu.Unlock()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if phase == "before-admission" {
					before := len(h.callsSnapshot())
					cancel()
					s.refreshProjectionsForOwner(ctx, owner)
					s.refreshSkillsForOwner(ctx, owner)
					s.refreshFeedbackForOwner(ctx, owner)
					synctest.Wait()
					if len(h.callsSnapshot()) != before || s.Feedback().Loading {
						t.Fatal("canceled refresh was admitted")
					}
					return
				}
				entered := make(chan workspaceRefreshCall, 4)
				release := make(chan struct{})
				var releaseOnce sync.Once
				defer releaseOnce.Do(func() { close(release) })
				h.mu.Lock()
				h.handle = func(call workspaceRefreshCall) (any, error) {
					switch call.method {
					case "session/projections", "skills/list", "skillInventory/list", "messageFeedback/list":
						entered <- call
						<-release
					}
					return workspaceRefreshValue(call)
				}
				h.mu.Unlock()
				s.refreshProjectionsForOwner(ctx, owner)
				s.refreshSkillsForOwner(ctx, owner)
				s.refreshFeedbackForOwner(ctx, owner)
				active := map[string]workspaceRefreshCall{}
				for len(active) < 4 {
					select {
					case call := <-entered:
						active[call.method] = call
					case <-time.After(5 * time.Second):
						t.Fatal("initial refresh calls did not start")
					}
				}
				cancel()
				releaseOnce.Do(func() { close(release) })
				for _, call := range active {
					wait(t, call.settled)
				}
				wait(t, active["messageFeedback/list"].ctx.Done())
				synctest.Wait()
				// Canceled owner results must not be folded, including those from a
				// transport that returns a response despite context cancellation.
				s.mu.Lock()
				defer s.mu.Unlock()
				for _, item := range s.items {
					if item.SessionID == id && item.Title == "Created title" {
						t.Fatal("canceled projection applied")
					}
				}
				if len(s.skillOptions) != 0 || len(s.skillCatalog.Skills) != 0 || s.feedback.Ready {
					t.Fatalf("canceled refresh applied: skills=%+v catalog=%+v feedback=%+v", s.skillOptions, s.skillCatalog, s.feedback)
				}
			})
		})
	}
}

func TestWorkspaceSessionCreationRefreshFeedbackReconnect(t *testing.T) {
	h := newWorkspaceRefreshHost(t)
	entered := make(chan workspaceRefreshCall, 2)
	releaseOld, releaseNew := make(chan struct{}), make(chan struct{})
	var onceOld, onceNew sync.Once
	defer onceOld.Do(func() { close(releaseOld) })
	defer onceNew.Do(func() { close(releaseNew) })
	var feedbackCount int
	h.handle = func(call workspaceRefreshCall) (any, error) {
		if call.method != "messageFeedback/list" {
			return workspaceRefreshValue(call)
		}
		h.mu.Lock()
		feedbackCount++
		index := feedbackCount
		h.mu.Unlock()
		entered <- call
		if index == 1 {
			<-releaseOld
			return workspaceRefreshValue(call)
		}
		<-releaseNew
		return map[string]any{"ok": true, "value": map[string]any{"items": []any{map[string]any{"messageId": "new-message", "rating": "negative", "version": "v2"}}}}, nil
	}
	target, id := h.create(t)
	if !target.Select(id) {
		t.Fatal("selection rejected")
	}
	var old workspaceRefreshCall
	select {
	case old = <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("old feedback request did not start")
	}
	h.store.BeginEvents(context.Background(), h.client)
	h.store.RefreshFeedback(context.Background())
	var current workspaceRefreshCall
	select {
	case current = <-entered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("new Host feedback refresh was blocked by old loading request")
	}
	onceOld.Do(func() { close(releaseOld) })
	wait(t, old.ctx.Done())
	if !h.store.Feedback().Loading {
		t.Fatal("old request cleared new Host loading state")
	}
	onceNew.Do(func() { close(releaseNew) })
	wait(t, current.ctx.Done())
	if got := h.store.Feedback(); !got.Ready || got.Items["new-message"].Version != "v2" || got.Items["created-message"].Version != "" {
		t.Fatalf("new Host feedback not retained: %+v", got)
	}
}

func TestWorkspaceSessionCreationRefreshFeedbackCancelBeforeLaunch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newWorkspaceRefreshHost(t)
		target, id := h.create(t)
		if !target.Select(id) {
			t.Fatal("selection rejected")
		}
		h.await(t, func() bool {
			v := h.store.Snapshot()
			titleReady := false
			for _, item := range v.Items {
				titleReady = titleReady || item.SessionID == id && item.Title == "Created title"
			}
			return titleReady && len(v.SkillOptions) == 1 && len(v.SkillCatalog.Skills) == 1 && h.store.Feedback().Ready
		})
		// Other initial refresh notifications must finish before the callback is replaced.
		synctest.Wait()
		s := h.store
		s.mu.Lock()
		owner := &workspaceSelectedOwner{client: h.client, hostGeneration: s.eventsGeneration, composerGeneration: s.composerHostGeneration, followGeneration: s.followGeneration, sessionID: id, cwd: s.cwd}
		s.clearFeedbackLocked()
		s.mu.Unlock()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var seenLoading atomic.Bool
		var retired atomic.Bool
		s.Attach(h.client, func() {
			if s.Feedback().Loading {
				if seenLoading.CompareAndSwap(false, true) {
					cancel()
				}
			} else if seenLoading.Load() {
				retired.Store(true)
			}
		})
		before := len(h.callsSnapshot())
		s.refreshFeedbackForOwner(ctx, owner)
		synctest.Wait()
		if len(h.callsSnapshot()) != before {
			t.Fatal("canceled feedback started an RPC")
		}
		if !seenLoading.Load() || s.Feedback().Loading {
			t.Fatal("canceled feedback did not retire its loading state")
		}
		if !retired.Load() {
			t.Fatal("feedback loading retirement was not published")
		}
	})
}

func TestWorkspaceSessionCreationRefreshGoalHostRebind(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newWorkspaceRefreshHost(t)
		entered := make(chan workspaceRefreshCall, 1)
		release := make(chan struct{})
		var releaseOnce sync.Once
		defer releaseOnce.Do(func() { close(release) })
		h.handle = func(call workspaceRefreshCall) (any, error) {
			if call.method == "goals/get" {
				entered <- call
				<-release
				return map[string]any{"id": "obsolete", "objective": "old host goal", "revision": 1, "phase": "active"}, nil
			}
			return workspaceRefreshValue(call)
		}
		target, id := h.create(t)
		if !target.Select(id) {
			t.Fatal("selection rejected")
		}
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("goal request did not start")
		}
		h.store.BeginEvents(context.Background(), h.client)
		h.store.mu.Lock()
		h.store.planBoard.Goal = &PlanGoal{ID: "current", Objective: "new Host goal"}
		h.store.mu.Unlock()
		releaseOnce.Do(func() { close(release) })
		// Wait for the goal goroutine to consume the response and apply its fence.
		synctest.Wait()
		h.store.mu.Lock()
		defer h.store.mu.Unlock()
		if goal := h.store.planBoard.Goal; goal == nil || goal.ID != "current" {
			t.Fatalf("old Host goal overwrote new Host state: %+v", goal)
		}
	})
}

func TestWorkspaceSessionCreationRefreshReentrantFeedbackNavigation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newWorkspaceRefreshHost(t)
		target, id := h.create(t)
		var switched bool
		h.store.Attach(h.client, func() {
			h.store.mu.Lock()
			defer h.store.mu.Unlock()
			if h.store.feedback.Loading && !switched {
				switched = true
				h.store.selectSessionLocked("new-selection")
			}
		})
		if !target.Select(id) || !switched {
			t.Fatal("selection did not reach feedback loading boundary")
		}
		// Let any erroneously launched feedback goroutine reach the mock transport.
		synctest.Wait()
		for _, call := range h.callsSnapshot() {
			if call.method == "messageFeedback/list" {
				t.Fatalf("feedback request started after its selection was revoked: %+v", call)
			}
		}
	})
}
