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
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

func modelTestCatalog() api.ModelCatalog {
	var cat api.ModelCatalog
	_ = json.Unmarshal([]byte(`{"default":{"provider":"p","model":"m1"},"groups":[{"id":"p","name":"Provider","models":[{"id":"m1","name":"One"},{"id":"m2","name":"Two","reasoning":{"defaultEffort":"adapter-deep","efforts":[{"id":"adapter-light","name":"Light"},{"id":"adapter-deep","name":"Deep"}]}},{"id":"m3","name":"Three","reasoning":{"efforts":[{"id":"dynamic","name":"Dynamic"}]}}]}]}`), &cat)
	return cat
}
func modelTestStore(client *dsh.Client) *Store {
	s := New()
	s.client = client
	s.selectedID = "current"
	cat := modelTestCatalog()
	s.modelOptions = flattenModelCatalog(cat)
	s.modelState.CatalogReady = true
	s.modelState.Default = cat.Default
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"lastUsed":null,"next":{"provider":"p","model":"m1"}}}`), 10)
	return s
}
func waitModelSnapshot(t *testing.T, s *Store, pred func(ViewState) bool) ViewState {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot()
		if pred(v) {
			return v
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("model state did not settle: %+v", s.Snapshot())
	return ViewState{}
}
func modelTestReply(w http.ResponseWriter, rpcID string, value any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": rpcID, "result": map[string]any{"ok": true, "value": value}})
}
func modelTestRequest(r *http.Request) (string, map[string]any) {
	var q struct {
		RPCID   string
		Payload struct {
			Args struct{ Request map[string]any }
		}
	}
	_ = json.NewDecoder(r.Body).Decode(&q)
	return q.RPCID, q.Payload.Args.Request
}

func TestModelProjectionUsesCurrentSession(t *testing.T) {
	s := modelTestStore(nil)
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[],"projections":{"asOfSeq":12,"values":{"modelSelection":{"lastUsed":null,"next":{"provider":"p","model":"m2","reasoningEffort":"adapter-light"}}}}}`))
	snap := s.Snapshot()
	if snap.SelectedModel != "Provider — Two" || snap.SelectedModelEffort != "Light" || snap.SelectedProvider != "p" {
		t.Fatalf("wrong current route: %+v", snap)
	}
	s.mu.Lock()
	s.selectSessionLocked("other")
	s.mu.Unlock()
	if v := s.Snapshot(); v.SelectedModel != "" || v.ModelSelectionKnown || v.SelectedModelEffort != "" || len(v.ModelLabels) != 0 {
		t.Fatalf("other session inherited model: %+v", v)
	}
	s.applyFollowFrame(json.RawMessage(`{"type":"snapshot","records":[],"projections":{"asOfSeq":1,"values":{"modelSelection":{"lastUsed":null,"next":null}}}}`))
	if got := s.Snapshot().SelectedModel; got != "Provider — One" {
		t.Fatalf("explicit default = %q", got)
	}
}

func TestModelCatalogPreservesOpaqueRoutesAndEfforts(t *testing.T) {
	cat := modelTestCatalog()
	cat.Groups = append(cat.Groups, api.ModelCatalogGroupsItem{Id: "a/b", Name: "Same", Models: []api.ModelCatalogGroupsItemModelsItem{{Id: "c", Name: "Same"}}}, api.ModelCatalogGroupsItem{Id: "a", Name: "Same", Models: []api.ModelCatalogGroupsItemModelsItem{{Id: "b/c", Name: "Same"}}})
	opts := flattenModelCatalog(cat)
	if len(opts) != 5 || opts[3].Label == opts[4].Label {
		t.Fatalf("opaque routes or display names collapsed: %+v", opts)
	}
	if opts[1].Reasoning == nil || *opts[1].Reasoning.DefaultEffort != "adapter-deep" {
		t.Fatal("dynamic default effort dropped")
	}
	*cat.Groups[0].Models[1].Reasoning.DefaultEffort = "changed"
	if *opts[1].Reasoning.DefaultEffort != "adapter-deep" {
		t.Fatal("catalog aliases retained state")
	}
	levels := modelEffortOptions(opts[2])
	if len(levels) != 2 || levels[0].Value != nil || *levels[1].Value != "dynamic" {
		t.Fatalf("provider default options: %+v", levels)
	}
}

func TestModelProjectionUnknownMissingAndRetainedSelection(t *testing.T) {
	s := modelTestStore(nil)
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"lastUsed":null,"next":{"provider":"gone","model":"removed","reasoningEffort":"custom"}}}`), 20)
	v := s.Snapshot()
	if v.SelectedModel != "gone/removed" || v.SelectedProvider != "gone" || v.SelectedModelEffort != "custom" || len(v.ModelEffortLabels) != 0 {
		t.Fatalf("removed route lost: %+v", v)
	}
	s.applyModelControlLocked(json.RawMessage(`{"type":"projection","sessionId":"other","key":"modelSelection","seq":100,"value":{"next":null}}`))
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"next":null}}`), 19)
	if s.Snapshot().SelectedProvider != "gone" {
		t.Fatal("stale or wrong-session projection won")
	}
	s.applyModelValuesLocked(json.RawMessage(`{}`), 21)
	if v = s.Snapshot(); v.ModelSelectionKnown || v.SelectedModel != "" {
		t.Fatal("missing projection guessed a global route")
	}
	s.applyPersistenceEvent(json.RawMessage(`{"type":"model/selection","seq":22,"data":{"provider":"p","model":"m2","reasoningEffort":"adapter-light"}}`))
	if s.Snapshot().SelectedModelEffort != "Light" {
		t.Fatal("durable selection event not folded")
	}
	s.items = []Item{{SessionID: "current", Origin: "subagent"}}
	s.fire()
	if len(s.Snapshot().ModelLabels) != 0 {
		t.Fatal("addressed subagent exposed Agent-bound model picker")
	}
}

func TestModelRefreshDoesNotOverwriteSessionSelection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := modelTestRequest(r)
		if r.URL.Path != "/api/session/modelCatalog" {
			t.Errorf("unexpected method %s", r.URL.Path)
		}
		modelTestReply(w, id, modelTestCatalog())
	}))
	defer srv.Close()
	s := modelTestStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"next":{"provider":"p","model":"m2","reasoningEffort":"adapter-light"}}}`), 12)
	s.modelOptions = nil
	s.RefreshModelCatalog(context.Background())
	v := waitModelSnapshot(t, s, func(v ViewState) bool { return len(v.ModelLabels) == 3 })
	if v.SelectedModel != "Provider — Two" || v.SelectedModelEffort != "Light" {
		t.Fatalf("catalog overwrote session: %+v", v)
	}
}

func TestModelSelectionWireFailureRetryAndEffort(t *testing.T) {
	var attempts atomic.Int32
	requests := make(chan map[string]any, 8)
	var selected api.ModelCatalogDefault
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, req := modelTestRequest(r)
		if req["sessionId"] != "current" {
			t.Errorf("wrong target: %#v", req)
		}
		switch r.URL.Path {
		case "/api/session/selectModel":
			requests <- req
			for key := range req {
				if key != "sessionId" && key != "provider" && key != "model" && key != "reasoningEffort" {
					t.Errorf("guessed/global field %q", key)
				}
			}
			if attempts.Add(1) == 1 {
				json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": id, "result": map[string]any{"ok": false, "error": map[string]any{"code": "session/writer-held", "message": "busy"}}})
				return
			}
			raw, _ := json.Marshal(req)
			selected = api.ModelCatalogDefault{}
			json.Unmarshal(raw, &selected)
			modelTestReply(w, id, api.SessionSelectModelValue{Selected: selected})
		case "/api/session/projections":
			modelTestReply(w, id, map[string]any{"asOfSeq": 10 + attempts.Load(), "values": map[string]any{"modelSelection": map[string]any{"lastUsed": nil, "next": selected}}})
		default:
			t.Errorf("unexpected method %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := modelTestStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Two")
	v := waitModelSnapshot(t, s, func(v ViewState) bool { return !v.ModelSelectionBusy && v.ModelsErr != "" })
	if v.SelectedModel != "Provider — One" {
		t.Fatal("rejected selection changed committed label")
	}
	first := <-requests
	if first["reasoningEffort"] != "adapter-deep" {
		t.Fatalf("model selection omitted catalog effort: %#v", first)
	}
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Two")
	v = waitModelSnapshot(t, s, func(v ViewState) bool { return !v.ModelSelectionBusy && v.SelectedModel == "Provider — Two" })
	if v.SelectedModelEffort != "Deep" || v.ModelsErr != "" {
		t.Fatalf("retry failed: %+v", v)
	}
	<-requests
	s.SelectModelEffortForSessionID(context.Background(), "current", "Light")
	v = waitModelSnapshot(t, s, func(v ViewState) bool { return !v.ModelSelectionBusy && v.SelectedModelEffort == "Light" })
	req := <-requests
	if req["provider"] != "p" || req["model"] != "m2" || req["reasoningEffort"] != "adapter-light" {
		t.Fatalf("effort selection route: %#v", req)
	}
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Two")
	if attempts.Load() != 3 {
		t.Fatal("same route reset its chosen effort")
	}
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Three")
	waitModelSnapshot(t, s, func(v ViewState) bool { return !v.ModelSelectionBusy && v.SelectedModel == "Provider — Three" })
	req = <-requests
	if _, ok := req["reasoningEffort"]; ok {
		t.Fatal("provider default must omit reasoningEffort")
	}
	s.SelectModelEffortForSessionID(context.Background(), "current", "Dynamic")
	waitModelSnapshot(t, s, func(v ViewState) bool { return !v.ModelSelectionBusy && v.SelectedModelEffort == "Dynamic" })
	<-requests
	s.SelectModelEffortForSessionID(context.Background(), "current", storelocale.T(storelocale.KeyStoreSessionProviderDefault))
	waitModelSnapshot(t, s, func(v ViewState) bool {
		return !v.ModelSelectionBusy && v.SelectedModelEffort == storelocale.T(storelocale.KeyStoreSessionProviderDefault)
	})
	req = <-requests
	if _, ok := req["reasoningEffort"]; ok {
		t.Fatal("reset to provider default must omit field")
	}
}

func TestModelSelectionStaleResponseCannotCrossSessionGeneration(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := modelTestRequest(r)
		if calls.Add(1) != 1 {
			http.Error(w, "stale follow-up", 500)
			return
		}
		close(started)
		<-release
		modelTestReply(w, id, api.SessionSelectModelValue{Selected: api.ModelCatalogDefault{Provider: "p", Model: "m2"}})
		close(finished)
	}))
	defer srv.Close()
	s := modelTestStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Two")
	<-started
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Three")
	if calls.Load() != 1 {
		t.Fatal("pending selection allowed competing write")
	}
	s.mu.Lock()
	s.selectSessionLocked("other")
	s.selectSessionLocked("current")
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"next":{"provider":"p","model":"m3"}}}`), 25)
	s.mu.Unlock()
	close(release)
	<-finished
	// A second projections call would itself be a stale-session RPC. Give the
	// completion a deterministic notification by waiting for the HTTP goroutine.
	time.Sleep(20 * time.Millisecond)
	v := s.Snapshot()
	if v.SelectedModel != "Provider — Three" || v.ModelSelectionBusy || strings.Contains(v.LastAction, "模型已更新") {
		t.Fatalf("late response won: %+v", v)
	}
	if calls.Load() != 1 {
		t.Fatal("late response fetched another session's projection")
	}
}

func TestModelSelectionRejectsUnconfirmedProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := modelTestRequest(r)
		if r.URL.Path == "/api/session/selectModel" {
			modelTestReply(w, id, map[string]any{"selected": map[string]any{"provider": "p", "model": "m2", "reasoningEffort": "adapter-deep"}})
		} else {
			modelTestReply(w, id, map[string]any{"asOfSeq": 10, "values": map[string]any{"modelSelection": map[string]any{"next": map[string]any{"provider": "p", "model": "m1"}}}})
		}
	}))
	defer srv.Close()
	s := modelTestStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Two")
	v := waitModelSnapshot(t, s, func(v ViewState) bool { return !v.ModelSelectionBusy && v.ModelsErr != "" })
	if v.SelectedModel != "Provider — One" {
		t.Fatal("unconfirmed response changed projection")
	}
}

func TestModelSnapshotChoicesRetainIdentityAndStaleEffortIsRejected(t *testing.T) {
	s := modelTestStore(&dsh.Client{})
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"next":{"provider":"p","model":"m2","reasoningEffort":"adapter-light"}}}`), 12)
	view := s.Snapshot()
	choice := view.ModelChoices["Provider — Two"]
	if choice.Provider != "p" || choice.Model != "m2" || choice.ReasoningEffort == nil || *choice.ReasoningEffort != "adapter-light" {
		t.Fatalf("snapshot lost full choice: %+v", choice)
	}
	effort := view.ModelEffortChoices["Deep"]
	s.modelOptions[1].Label = "renamed after menu opened"
	if view.ModelChoices["Provider — Two"].Model != "m2" {
		t.Fatal("menu choice depended on live display label")
	}
	s.applyModelValuesLocked(json.RawMessage(`{"modelSelection":{"next":{"provider":"p","model":"m3"}}}`), 13)
	s.SelectModelEffortSelectionForSessionID(context.Background(), "current", effort)
	if s.modelState.Pending || s.Snapshot().SelectedModel != "Provider — Three" {
		t.Fatal("stale effort choice switched the current model")
	}
	*choice.ReasoningEffort = "modified snapshot"
	if view.ModelEffortChoices["Light"].ReasoningEffort == nil || *view.ModelEffortChoices["Light"].ReasoningEffort != "adapter-light" {
		t.Fatal("choice pointers alias")
	}
}
