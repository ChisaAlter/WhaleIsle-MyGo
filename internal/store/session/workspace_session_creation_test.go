package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

type workspaceCreationCall struct {
	Method  string
	Request map[string]any
}

type workspaceCreationHost struct {
	t       *testing.T
	store   *Store
	client  *dsh.Client
	calls   []workspaceCreationCall
	handler func(workspaceCreationCall) (map[string]any, *dsh.Error)
}

func newWorkspaceCreationHost(t *testing.T) *workspaceCreationHost {
	t.Helper()
	h := &workspaceCreationHost{t: t, store: New()}
	h.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://workspace-fixture.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request map[string]any `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		call := workspaceCreationCall{Method: strings.TrimPrefix(r.URL.Path, "/api/"), Request: body.Payload.Args.Request}
		if call.Method != "session/create" && call.Method != "session/blankReuse" {
			raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{}}})
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		}
		h.calls = append(h.calls, call)
		var value map[string]any
		var failure *dsh.Error
		if h.handler != nil {
			value, failure = h.handler(call)
		} else {
			value = map[string]any{"sessionId": "new-target"}
		}
		result := map[string]any{"ok": true, "value": value}
		if failure != nil {
			result = map[string]any{"ok": false, "error": map[string]any{"code": failure.Code, "message": failure.Message, "details": json.RawMessage(failure.Details)}}
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": result})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	h.store.Attach(h.client, nil)
	h.store.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"scratchCwd":"/scratch","items":[{"workspaceId":"one","path":"/same","title":"One","sessionIds":["one-blank"]},{"workspaceId":"two","path":"/same","title":"Two","sessionIds":["two-blank"]}],"pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	h.store.mu.Lock()
	h.store.cwd = "/configured"
	h.store.workspaceID = "configured"
	h.store.selectedID = "current"
	h.store.workspaceLayout.view.Collapsed = map[string]bool{"one": true, "two": true, "::scratch": true}
	h.store.mu.Unlock()
	return h
}

func (h *workspaceCreationHost) capture(id string, excluded ...string) *WorkspaceSessionTarget {
	h.t.Helper()
	target, err := h.store.CaptureWorkspaceSessionTarget(h.store.Snapshot().WorkspaceLayout.HostGeneration, id, excluded...)
	if err != nil {
		h.t.Fatal(err)
	}
	return target
}

func TestWorkspaceSessionCreationUsesTargetIDAndOnlyItsEligibleBlank(t *testing.T) {
	h := newWorkspaceCreationHost(t)
	h.store.mu.Lock()
	h.store.items = []Item{
		{SessionID: "one-blank", Cwd: "/same", Blank: true},
		{SessionID: "two-archived", Cwd: "/same", Blank: true},
		{SessionID: "two-child", Cwd: "/same", Blank: true, Origin: "subagent"},
		{SessionID: "two-titled", Cwd: "/same", Blank: true, Title: "named"},
		{SessionID: "two-wrong-cwd", Cwd: "/wrong", Blank: true},
		{SessionID: "two-draft", Cwd: "/same", Blank: true},
		{SessionID: "two-blank", Cwd: "/same", Blank: true},
	}
	ws := h.store.workspaceMemberships["two"]
	ws.SessionIDs = append(ws.SessionIDs, "two-archived", "two-child", "two-titled", "two-wrong-cwd", "two-draft")
	h.store.workspaceMemberships["two"] = ws
	h.store.archivedSessionIDs = []string{"two-archived"}
	h.store.mu.Unlock()
	h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
		switch call.Method {
		case "session/blankReuse":
			return map[string]any{"reusable": true}, nil
		case "session/create":
			return map[string]any{"sessionId": "two-blank"}, nil
		default:
			t.Fatalf("unexpected call: %+v", call)
			return nil, nil
		}
	}
	target := h.capture("two", "two-draft")
	if got := h.store.Snapshot().WorkspaceLayout.Collapsed; got["two"] || !got["one"] {
		t.Fatalf("expanded wrong group: %v", got)
	}
	id, err := target.Create(context.Background())
	if err != nil || id != "two-blank" {
		t.Fatalf("create: %q %v", id, err)
	}
	want := []workspaceCreationCall{
		{Method: "session/blankReuse", Request: map[string]any{"sessionId": "two-blank"}},
		{Method: "session/create", Request: map[string]any{"workspaceId": "two", "sessionId": "two-blank"}},
	}
	if !reflect.DeepEqual(h.calls, want) {
		t.Fatalf("calls=%#v want=%#v", h.calls, want)
	}
	if snap := h.store.Snapshot(); snap.SelectedID != "current" || h.store.WorkspaceCwd() != "/configured" || h.store.WorkspaceID() == "two" {
		t.Fatal("creation changed selection or configured workspace", snap.SelectedID, h.store.WorkspaceCwd())
	}
	if !target.Select(id) || h.store.Snapshot().SelectedID != id || target.Select(id) {
		t.Fatal("captured receipt was not selected exactly once")
	}
}

func TestWorkspaceSessionFreshAndHostDeclinedBlank(t *testing.T) {
	for _, mode := range []string{"fresh", "declined", "rejected-by-create"} {
		t.Run(mode, func(t *testing.T) {
			h := newWorkspaceCreationHost(t)
			if mode != "fresh" {
				h.store.mu.Lock()
				h.store.items = []Item{{SessionID: "two-blank", Cwd: "/same", Blank: true}}
				h.store.mu.Unlock()
			}
			h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
				if call.Method == "session/blankReuse" {
					return map[string]any{"reusable": mode == "rejected-by-create"}, nil
				}
				if call.Method != "session/create" || call.Request["workspaceId"] != "two" || call.Request["cwd"] != nil {
					t.Fatalf("bad create: %+v", call)
				}
				if (call.Request["sessionId"] != nil) != (mode == "rejected-by-create") {
					t.Fatalf("bad reuse request: %+v", call)
				}
				return map[string]any{"sessionId": "host-new"}, nil
			}
			target := h.capture("two")
			id, err := target.Create(context.Background())
			if err != nil || id != "host-new" {
				t.Fatalf("%q %v", id, err)
			}
			snap := h.store.Snapshot()
			if snap.SelectedID != "current" {
				t.Fatal("creation selected", snap.SelectedID)
			}
			members := snap.Workspaces[1].SessionIDs
			if !containsWorkspaceCreationID(members, "host-new") {
				t.Fatalf("receipt missing from group: %v", members)
			}
			if !target.Select(id) {
				t.Fatal("valid selection failed")
			}
		})
	}
}

func containsWorkspaceCreationID(ids []string, wanted string) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}

func TestWorkspaceSessionCreateErrorsAndPartialAttachment(t *testing.T) {
	for _, mode := range []string{"reuse-error", "create-error", "attach-error", "empty-id"} {
		t.Run(mode, func(t *testing.T) {
			h := newWorkspaceCreationHost(t)
			if mode == "reuse-error" {
				h.store.mu.Lock()
				h.store.items = []Item{{SessionID: "two-blank", Cwd: "/same", Blank: true}}
				h.store.mu.Unlock()
			}
			h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
				switch mode {
				case "reuse-error":
					return nil, &dsh.Error{Code: "offline", Message: "inspection failed", Details: json.RawMessage(`{}`)}
				case "create-error":
					return nil, &dsh.Error{Code: "workspace/not-found", Message: "missing workspace", Details: json.RawMessage(`{}`)}
				case "attach-error":
					return nil, &dsh.Error{Code: "session/workspace-attach-failed", Message: "attach failed", Details: json.RawMessage(`{"sessionId":"published","workspaceId":"two"}`)}
				default:
					return map[string]any{"sessionId": ""}, nil
				}
			}
			target := h.capture("two")
			id, err := target.Create(context.Background())
			if id != "" || err == nil || !target.ReportFailure(err) {
				t.Fatal("failed call not reported", id, err)
			}
			if h.store.Snapshot().SelectedID != "current" || !strings.Contains(h.store.Snapshot().LastAction, "session") {
				t.Fatal("failure changed selection or omitted feedback")
			}
			if mode == "reuse-error" && (len(h.calls) != 1 || h.calls[0].Method != "session/blankReuse") {
				t.Fatalf("reuse error led to create: %v", h.calls)
			}
			if mode == "attach-error" {
				snap := h.store.Snapshot()
				found := false
				for _, item := range snap.Items {
					if item.SessionID == "published" {
						found = true
					}
				}
				if !found || containsWorkspaceCreationID(snap.Workspaces[1].SessionIDs, "published") {
					t.Fatal("partial receipt falsely attached or missing")
				}
			}
		})
	}
}

func TestWorkspaceSessionCreateOwnerTransitions(t *testing.T) {
	for _, phase := range []string{"blankReuse", "create"} {
		for _, transition := range []string{"selection", "selection-aba", "host-replacement", "host-aba", "same-client-events", "workspace-change", "workspace-aba", "workspace-picker", "workspace-removed", "cancel"} {
			t.Run(phase+"/"+transition, func(t *testing.T) {
				h := newWorkspaceCreationHost(t)
				if phase == "blankReuse" {
					h.store.mu.Lock()
					h.store.items = []Item{{SessionID: "two-blank", Cwd: "/same", Blank: true}}
					h.store.mu.Unlock()
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
					if (phase == "blankReuse" && call.Method == "session/blankReuse") || (phase == "create" && call.Method == "session/create") {
						s := h.store
						switch transition {
						case "selection":
							s.mu.Lock()
							s.selectSessionLocked("later")
							s.mu.Unlock()
						case "selection-aba":
							s.mu.Lock()
							s.selectSessionLocked("later")
							s.selectSessionLocked("current")
							s.mu.Unlock()
						case "host-replacement":
							s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}, nil)
						case "host-aba":
							s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}, nil)
							s.Attach(h.client, nil)
						case "same-client-events":
							s.BeginEvents(context.Background(), h.client)
						case "workspace-change":
							s.SetWorkspaceCwd("/new-workspace")
						case "workspace-aba":
							s.SetWorkspaceCwd("/new-workspace")
							s.SetWorkspaceCwd("/configured")
						case "workspace-picker":
							s.OpenWorkspacePicker()
						case "workspace-removed":
							s.mu.Lock()
							delete(s.workspaceMemberships, "two")
							s.mu.Unlock()
						case "cancel":
							cancel()
						}
						if phase == "blankReuse" {
							return map[string]any{"reusable": true}, nil
						}
						return map[string]any{"sessionId": "created-before-change"}, nil
					}
					t.Fatalf("unexpected RPC %+v", call)
					return nil, nil
				}
				target := h.capture("two")
				id, err := target.Create(ctx)
				if err == nil || id != "" || target.Select("created-before-change") || target.ReportFailure(errors.New("old error")) {
					t.Fatal("stale target accepted", id, err)
				}
				if len(h.calls) != 1 {
					t.Fatalf("stale target issued follow-up mutation: %v", h.calls)
				}
				if got := h.store.Snapshot().SelectedID; got == "created-before-change" {
					t.Fatal("stale creation stole selection")
				}
			})
		}
	}
}

func TestWorkspaceSessionTargetRevalidatesBeforeSelection(t *testing.T) {
	for _, transition := range []string{"selection-aba", "host-aba", "workspace-cwd", "workspace-aba", "workspace-picker", "removed"} {
		t.Run(transition, func(t *testing.T) {
			h := newWorkspaceCreationHost(t)
			target := h.capture("two")
			id, err := target.Create(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			switch transition {
			case "selection-aba":
				h.store.mu.Lock()
				h.store.selectSessionLocked("later")
				h.store.selectSessionLocked("current")
				h.store.mu.Unlock()
			case "host-aba":
				h.store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}, nil)
				h.store.Attach(h.client, nil)
			case "workspace-cwd":
				h.store.SetWorkspaceCwd("/later")
			case "workspace-aba":
				h.store.SetWorkspaceCwd("/later")
				h.store.SetWorkspaceCwd("/configured")
			case "workspace-picker":
				h.store.OpenWorkspacePicker()
			case "removed":
				h.store.mu.Lock()
				delete(h.store.workspaceMemberships, "two")
				h.store.mu.Unlock()
			}
			if target.Select(id) {
				t.Fatal("late result selected")
			}
		})
	}
}

func TestWorkspaceSessionScratchDelegatesToNoDirectory(t *testing.T) {
	h := newWorkspaceCreationHost(t)
	h.store.mu.Lock()
	h.store.items = []Item{{SessionID: "scratch-blank", Cwd: "/scratch", Blank: true}}
	h.store.mu.Unlock()
	h.handler = func(call workspaceCreationCall) (map[string]any, *dsh.Error) {
		if call.Method != "session/blankReuse" || call.Request["sessionId"] != "scratch-blank" {
			t.Fatalf("scratch escaped existing no-directory path: %+v", call)
		}
		return map[string]any{"reusable": true}, nil
	}
	target := h.capture("::scratch")
	id, err := target.Create(context.Background())
	if err != nil || id != "scratch-blank" || h.store.Snapshot().WorkspaceLayout.Collapsed["::scratch"] {
		t.Fatal("scratch creation or expansion", id, err)
	}
	if !target.Select(id) || h.store.Snapshot().SelectedID != id {
		t.Fatal("scratch selection rejected")
	}
}

func TestWorkspaceSessionScratchAdmissionDoesNotRetargetNewHost(t *testing.T) {
	h := newWorkspaceCreationHost(t)
	generation := h.store.Snapshot().WorkspaceLayout.HostGeneration
	replacement := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://replacement.invalid"}}
	h.store.Attach(replacement, nil)
	if _, err := h.store.createNoDirectorySession(context.Background(), h.client, generation); err == nil {
		t.Fatal("scratch admission retargeted a new Host")
	}
	if len(h.calls) != 0 {
		t.Fatalf("stale admission called Host: %v", h.calls)
	}
}

func TestWorkspaceGroupExpansionChecksRenderedHost(t *testing.T) {
	h := newWorkspaceCreationHost(t)
	generation := h.store.Snapshot().WorkspaceLayout.HostGeneration
	if h.store.ExpandWorkspaceGroupForHost(generation+1, "::scratch") {
		t.Fatal("stale host expanded")
	}
	if !h.store.Snapshot().WorkspaceLayout.Collapsed["::scratch"] {
		t.Fatal("stale action altered group")
	}
	if !h.store.ExpandWorkspaceGroupForHost(generation, "::scratch") || h.store.Snapshot().WorkspaceLayout.Collapsed["::scratch"] {
		t.Fatal("scratch did not expand")
	}
	if !h.store.ExpandWorkspaceGroupForHost(generation, "::scratch") || h.store.Snapshot().WorkspaceLayout.Collapsed["::scratch"] {
		t.Fatal("repeated expand toggled")
	}
	if _, err := h.store.CaptureWorkspaceSessionTarget(generation, "missing"); err == nil {
		t.Fatal("unknown workspace accepted")
	}
}
