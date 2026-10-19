package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func nativeStoreGraphFrame(t *testing.T, revisions map[string]string, omit ...string) []byte {
	t.Helper()
	names := []string{"ui-conversation", "connection", "api-session-controller", "file-upload", "locale", "ui-layout", "ui-renderer", "ui-session", "ui-settings", "ui-workspace", "ui-commands", "api-remotes", "ui-input-trigger", "ui-model-selection", "ui-permission-presets", "ui-message-feedback", "unrelated"}
	entries := []map[string]any{}
	ids := []string{}
	for _, name := range names {
		excluded := false
		for _, o := range omit {
			if name == o {
				excluded = true
			}
		}
		if excluded {
			continue
		}
		prefix := "@deepseek-ai/dsh-client-"
		if strings.HasPrefix(name, "api-") {
			prefix = "@deepseek-ai/dsh-"
		}
		id := prefix + name
		ids = append(ids, id)
		rev := "r1"
		if r, ok := revisions[name]; ok {
			rev = r
		}
		entries = append(entries, map[string]any{"id": id, "url": "/not-fetched/" + name, "rev": rev, "inject": []string{}, "external": []string{}})
	}
	raw, err := json.Marshal(map[string]any{"type": "graph", "graph": map[string]any{"rev": "graph", "release": map[string]any{"version": "fixture", "commit": "fixed"}, "entries": entries, "batches": []map[string]any{{"phase": "application", "url": "/not-fetched/batch", "rev": "batch", "entries": ids}}}})
	if err != nil {
		t.Fatal(err)
	}

	return raw
}
func nativeStoreGraph(t *testing.T, revisions map[string]string, omit ...string) dsh.NativeContributions {
	t.Helper()
	raw := nativeStoreGraphFrame(t, revisions, omit...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plugins/events" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", raw)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	got := make(chan dsh.NativeContributions, 2)
	go func() {
		defer close(done)
		_ = (&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}).WatchNativeContributions(ctx, func(v dsh.NativeContributions) {
			select {
			case got <- v:
			case <-ctx.Done():
			}
		})
	}()
	var value dsh.NativeContributions
	select {
	case value = <-got:
	case <-time.After(3 * time.Second):
		cancel()
		srv.Close()
		t.Fatal("local graph did not arrive")
	}
	cancel()
	<-done
	srv.Close()
	if !value.Known {
		t.Fatalf("invalid real graph fixture: %+v", value)
	}
	return value
}
func nativeSelectionStore(client *dsh.Client, graph dsh.NativeContributions) *Store {
	s := modelTestStore(client)
	s.setNativeContributionsLocked(graph)
	s.commandCatalog = commandCatalog{client: client, sessionID: "current", loaded: true, items: []api.ListResultItem2{{Name: "permission"}, {Name: "feedback"}}}
	s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}, {Label: "Workspace", Value: "workspace-write"}, {Label: "Full", Value: PermissionFullAccess}}
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 1)
	return s
}
func nativeSelectionOwner(s *Store, name string) NativeCommandOwner {
	v := s.Snapshot()
	g := v.NativeContributions
	return NativeCommandOwner{HostGeneration: v.ComposerHostGeneration, SessionGeneration: v.ComposerSessionGeneration, PermissionCatalogRevision: v.PermissionCatalogRevision, RegistrationGeneration: v.NativeContributionsGeneration, PortGeneration: v.NativeCommandGeneration(name), SessionID: v.SelectedID, Revision: g.CommandRevision(name), ReleaseVersion: g.ReleaseVersion, ReleaseCommit: g.ReleaseCommit}
}
func nativeResult(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("selection did not return its receipt")
		return nil
	}
}
func nativeRequest(r *http.Request) (string, map[string]json.RawMessage) {
	var q struct {
		RPCID   string `json:"rpcId"`
		Payload struct {
			Args map[string]json.RawMessage `json:"args"`
		} `json:"payload"`
	}
	_ = json.NewDecoder(r.Body).Decode(&q)
	return q.RPCID, q.Payload.Args
}

func TestNativeModelSelectionReceiptWaitsForCommittedProjection(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	selected := make(chan struct{}, 1)
	projection := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, req := modelTestRequest(r)
		if req["sessionId"] != "current" {
			t.Error("selection crossed captured session")
		}
		switch r.URL.Path {
		case "/api/session/selectModel":
			calls.Add(1)
			if req["provider"] != "p" || req["model"] != "m2" || req["reasoningEffort"] != "adapter-deep" {
				t.Errorf("opaque route/default effort lost: %#v", req)
			}
			selected <- struct{}{}
			modelTestReply(w, id, api.SessionSelectModelValue{Selected: api.ModelCatalogDefault{Provider: "p", Model: "m2", ReasoningEffort: func() *string { x := "adapter-deep"; return &x }()}})
		case "/api/session/projections":
			projection <- struct{}{}
			<-release
			modelTestReply(w, id, map[string]any{"asOfSeq": 12, "values": map[string]any{"modelSelection": map[string]any{"next": map[string]any{"provider": "p", "model": "m2", "reasoningEffort": "adapter-deep"}}}})
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	s := nativeSelectionStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, graph)
	owner := nativeSelectionOwner(s, "model")
	result := make(chan error, 1)
	effort := "adapter-deep"
	go func() {
		result <- s.SelectNativeCommandModel(context.Background(), owner, api.ModelCatalogDefault{Provider: "p", Model: "m2", ReasoningEffort: &effort})
	}()
	select {
	case <-selected:
	case <-time.After(3 * time.Second):
		t.Fatal("model RPC not made")
	}
	select {
	case <-projection:
	case <-time.After(3 * time.Second):
		t.Fatal("projection read not made")
	}
	select {
	case err := <-result:
		t.Fatalf("receipt preceded projection: %v", err)
	default:
	}
	if !s.Snapshot().ModelSelectionBusy || s.Snapshot().SelectedModel != "Provider — One" {
		t.Fatal("selection committed optimistically")
	}
	close(release)
	if err := nativeResult(t, result); err != nil {
		t.Fatal(err)
	}
	v := s.Snapshot()
	if v.ModelSelectionBusy || v.SelectedModel != "Provider — Two" || v.SelectedModelEffort != "Deep" || calls.Load() != 1 {
		t.Fatalf("wrong committed selection: %+v", v)
	}
	// The ordinary public picker shares this core and does not reset this route's effort.
	s.SelectModelForSessionID(context.Background(), "current", "Provider — Two")
	if calls.Load() != 1 {
		t.Fatal("ordinary picker reset the command's committed effort")
	}
}
func TestNativeModelSelectionFailureKeepsProjectionAndCanRetry(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := modelTestRequest(r)
		switch r.URL.Path {
		case "/api/session/selectModel":
			if calls.Add(1) == 1 {
				json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": id, "result": map[string]any{"ok": false, "error": map[string]any{"code": "session/writer-held", "message": "busy"}}})
				return
			}
			modelTestReply(w, id, api.SessionSelectModelValue{Selected: api.ModelCatalogDefault{Provider: "p", Model: "m3"}})
		case "/api/session/projections":
			modelTestReply(w, id, map[string]any{"asOfSeq": 12, "values": map[string]any{"modelSelection": map[string]any{"next": map[string]any{"provider": "p", "model": "m3"}}}})
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := nativeSelectionStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, graph)
	o := nativeSelectionOwner(s, "model")
	selection := api.ModelCatalogDefault{Provider: "p", Model: "m3"}
	if err := s.SelectNativeCommandModel(context.Background(), o, selection); err == nil {
		t.Fatal("writer refusal reported success")
	}
	if v := s.Snapshot(); v.ModelSelectionBusy || v.SelectedModel != "Provider — One" || v.ModelsErr == "" {
		t.Fatal("failure changed projection or left busy state")
	}
	if err := s.SelectNativeCommandModel(context.Background(), o, selection); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().SelectedModel != "Provider — Three" || calls.Load() != 2 {
		t.Fatal("retry did not use original selection core")
	}
}
func TestNativeOwnerFencesRejectBeforeAnyMutationRPC(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	rebuilt := nativeStoreGraph(t, map[string]string{"ui-model-selection": "r2", "ui-permission-presets": "r2"})
	unrelated := nativeStoreGraph(t, map[string]string{"unrelated": "r2"})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); http.Error(w, "must not dispatch", 500) }))
	defer srv.Close()
	for _, name := range []string{"model", "permission", "file", "feedback"} {
		for _, change := range []string{"host", "follow", "release", "registration-aba", "unknown", "target-entry", "permission-catalog"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				s := nativeSelectionStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, graph)
				o := nativeSelectionOwner(s, name)
				s.mu.Lock()
				switch change {
				case "host":
					s.composerHostGeneration++
				case "follow":
					s.followGeneration++
				case "release":
					changed := s.nativeContributions.snapshot
					changed.ReleaseCommit = "another"
					s.setNativeContributionsLocked(changed)
				case "registration-aba":
					s.resetNativeContributionsLocked()
					s.setNativeContributionsLocked(graph)
				case "unknown":
					s.setNativeContributionsLocked(dsh.NativeContributions{})
				case "target-entry":
					s.setNativeContributionsLocked(rebuilt)
				case "permission-catalog":
					_ = s.beginPermissionCatalogLocked(context.Background())
				}
				s.mu.Unlock()
				// Only the corresponding code closure and permission catalog retire owners.
				shouldRetire := change != "target-entry" && change != "permission-catalog" || change == "target-entry" && (name == "model" || name == "permission") || change == "permission-catalog" && name == "permission"
				if s.NativeCommandCurrent(o, name) == shouldRetire {
					t.Fatalf("wrong owner validity after %s", change)
				}
				if shouldRetire && name == "model" {
					if err := s.SelectNativeCommandModel(context.Background(), o, api.ModelCatalogDefault{Provider: "p", Model: "m3"}); err == nil {
						t.Fatal("retired model dispatched")
					}
				}
				if shouldRetire && name == "permission" {
					if err := s.SelectNativeCommandPermission(context.Background(), o, "workspace-write", false); err == nil {
						t.Fatal("retired permission dispatched")
					}
				}
			})
		}
	}
	s := nativeSelectionStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, graph)
	o := nativeSelectionOwner(s, "model")
	s.setNativeContributionsLocked(unrelated)
	if !s.NativeCommandCurrent(o, "model") {
		t.Fatal("unrelated entry rebuild retired a live port")
	}
	if calls.Load() != 0 {
		t.Fatalf("retired owner made %d mutation RPCs", calls.Load())
	}
}
func TestNativeModelLateReceiptCannotSurviveSameGraphWithdrawalABA(t *testing.T) {
	frame := nativeStoreGraphFrame(t, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	frames := make(chan []byte, 2)
	var projections atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/plugins/events" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: %s\n\n", frame)
			w.(http.Flusher).Flush()
			for {
				select {
				case raw := <-frames:
					fmt.Fprintf(w, "data: %s\n\n", raw)
					w.(http.Flusher).Flush()
				case <-r.Context().Done():
					return
				}
			}
		}
		id, _ := modelTestRequest(r)
		if r.URL.Path == "/api/session/selectModel" {
			close(started)
			<-release
			modelTestReply(w, id, api.SessionSelectModelValue{Selected: api.ModelCatalogDefault{Provider: "p", Model: "m3"}})
			return
		}
		projections.Add(1)
		http.Error(w, "stale projection read", 500)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s := nativeSelectionStore(client, dsh.NativeContributions{})
	events := s.BeginEvents(ctx, client)
	s.WatchNativeContributionsForEvents(events)
	waitModelSnapshot(t, s, func(v ViewState) bool { return v.NativeContributions.Known })
	o := nativeSelectionOwner(s, "model")
	result := make(chan error, 1)
	go func() {
		result <- s.SelectNativeCommandModel(ctx, o, api.ModelCatalogDefault{Provider: "p", Model: "m3"})
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("model request not started")
	}
	// No UI frame occurs between the real watcher's withdrawal and restoration.
	frames <- []byte(`{"type":"graph","graph":{}}`)
	waitModelSnapshot(t, s, func(v ViewState) bool { return !v.NativeContributions.Known })
	frames <- frame
	restored := waitModelSnapshot(t, s, func(v ViewState) bool { return v.NativeContributions.Known })
	if restored.NativeContributionsGeneration == o.RegistrationGeneration {
		t.Fatal("real withdrawal/recovery producer reused its registration generation")
	}
	close(release)
	if err := nativeResult(t, result); err == nil {
		t.Fatal("withdrawn same-graph owner returned a success receipt")
	}
	if projections.Load() != 0 || s.Snapshot().SelectedModel != "Provider — One" || s.Snapshot().ModelSelectionBusy {
		t.Fatal("old delivery read/committed projection or retained busy")
	}
	if s.NativeCommandCurrent(o, "model") {
		t.Fatal("same-graph restore revived old model owner")
	}
}
func TestNativePermissionRiskAndHostAcknowledgement(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	var commands atomic.Int32
	var projections atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, args := nativeRequest(r)
		switch r.URL.Path {
		case "/api/commands/execute":
			commands.Add(1)
			if string(args["agentId"]) != `"current"` || string(args["line"]) != `"/permission danger-full-access"` || string(args["submittedAttachments"]) != "[]" {
				t.Errorf("wrong captured permission command: %v", args)
			}
			modelTestReply(w, id, map[string]any{"result": map[string]any{"kind": "success"}})
		case "/api/session/projections":
			projections.Add(1)
			modelTestReply(w, id, map[string]any{"asOfSeq": 3, "values": map[string]any{"permissions": map[string]any{"currentValue": PermissionFullAccess}}})
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	s := nativeSelectionStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, graph)
	o := nativeSelectionOwner(s, "permission")
	if err := s.SelectNativeCommandPermission(context.Background(), o, PermissionFullAccess, false); err == nil || commands.Load() != 0 {
		t.Fatal("unacknowledged permission reached Host")
	}
	if err := s.SelectNativeCommandPermission(context.Background(), o, PermissionFullAccess, true); err != nil {
		t.Fatal(err)
	}
	if commands.Load() != 1 || projections.Load() != 1 || s.Snapshot().SelectedPermission != "Full" || s.permissionProjection.Pending {
		t.Fatal("permission command lacked confirmed projection receipt")
	}
}
func TestNativeCatalogOptionsAreReadOnlyAndKeepFailuresOpaque(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := nativeRequest(r)
		switch r.URL.Path {
		case "/api/session/modelCatalog":
			cat := modelTestCatalog()
			cat.Failures = []api.ModelCatalogFailuresItem{{Id: "failed-provider", Name: "Failed provider", Message: "catalog unavailable"}}
			modelTestReply(w, id, cat)
		case "/api/permissionPresets/catalog":
			modelTestReply(w, id, api.PermissionCatalog{DefaultOptions: []api.PermissionCatalogOptionsItem{{Value: "default-only", Name: "Future only"}}, Options: []api.PermissionCatalogOptionsItem{{Value: "workspace-write", Name: "Workspace"}, {Value: PermissionFullAccess, Name: "Full access"}}})
		default:
			writes.Add(1)
			http.Error(w, "read-only popup made mutation", 500)
		}
	}))
	defer srv.Close()
	s := nativeSelectionStore(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, graph)
	rows, err := s.LoadNativeCommandOptions(context.Background(), nativeSelectionOwner(s, "model"), "model")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || rows[3].Failure == "" || rows[3].Model != nil || rows[3].Detail != "catalog unavailable" {
		t.Fatalf("provider failure became a selection: %+v", rows)
	}
	if rows[1].Model == nil || rows[1].Model.ReasoningEffort == nil || *rows[1].Model.ReasoningEffort != "adapter-deep" {
		t.Fatal("options omitted adapter default effort")
	}
	rows, err = s.LoadNativeCommandOptions(context.Background(), nativeSelectionOwner(s, "permission"), "permission")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != "workspace-write" || !rows[1].Confirmation {
		t.Fatal("permission popup used defaults or omitted risk metadata")
	}
	if writes.Load() != 0 {
		t.Fatal("inspecting options selected a model or permission")
	}
}

func TestNativePortRemoveReaddAndRevisionABAOnlyRetireTheirOwnPort(t *testing.T) {
	graph := nativeStoreGraph(t, nil)
	absent := nativeStoreGraph(t, nil, "ui-model-selection")
	rebuilt := nativeStoreGraph(t, map[string]string{"ui-model-selection": "r2"})
	s := nativeSelectionStore(&dsh.Client{}, graph)
	model, feedback := nativeSelectionOwner(s, "model"), nativeSelectionOwner(s, "feedback")
	s.mu.Lock()
	s.setNativeContributionsLocked(absent)
	s.setNativeContributionsLocked(graph)
	s.mu.Unlock()
	if s.NativeCommandCurrent(model, "model") {
		t.Fatal("remove/readd revived the model popup owner")
	}
	if !s.NativeCommandCurrent(feedback, "feedback") {
		t.Fatal("model removal retired unrelated feedback action")
	}
	model = nativeSelectionOwner(s, "model")
	s.mu.Lock()
	s.setNativeContributionsLocked(rebuilt)
	s.setNativeContributionsLocked(graph)
	s.mu.Unlock()
	if s.NativeCommandCurrent(model, "model") {
		t.Fatal("revision ABA revived the old model popup owner")
	}
	if !s.NativeCommandCurrent(feedback, "feedback") {
		t.Fatal("model revision ABA retired unrelated feedback action")
	}
}
