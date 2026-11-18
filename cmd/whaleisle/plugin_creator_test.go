package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func TestPluginCreatorDraftWaitsForConfirmedNewTargetAndKeepsPreviousDraft(t *testing.T) {
	var target string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var frame struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args map[string]json.RawMessage `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
			t.Error(err)
			return
		}
		var value any
		switch strings.TrimPrefix(r.URL.Path, "/api/") {
		case "agentPresets/list":
			value = map[string]any{"presets": []map[string]any{{"id": "cordis"}}}
		case "session/create":
			var request struct {
				SessionID string `json:"sessionId"`
			}
			_ = json.Unmarshal(frame.Payload.Args["request"], &request)
			target = request.SessionID
			value = map[string]any{"sessionId": target}
		case "agentPresets/select":
			if string(frame.Payload.Args["agentId"]) != `"`+target+`"` {
				t.Error("selected old conversation")
			}
			value = "cordis"
		default:
			t.Errorf("unexpected operation %s", r.URL.Path)
			http.Error(w, "unexpected", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": frame.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer server.Close()
	sessions := session.New()
	sessions.Select("previous")
	sessions.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	sessions.SetWorkspaceCwd("/fixture")
	old := composer.NewDraftState("keep original draft")
	old.Attachments = []session.ComposerAttachment{{Path: "/fixture/keep.txt", Name: "keep.txt"}}
	s := &mainState{sessions: sessions, composerDrafts: map[string]*composer.DraftState{"previous": old}}
	op := &pluginDraftCreation{source: sessions, prompt: pluginCreatorDraftPrompt("leftbar"), route: s.captureMainRoute()}
	s.pluginDraftCreator = op
	op.task = session.NewCreatorTask(sessions, func() bool { return s.pluginDraftCurrent(op) })
	op.task.ObserveVisible(true)
	op.task.Prepare()
	ticket := op.task.Snapshot().Ticket
	if !op.task.Create(context.Background(), ticket) {
		t.Fatal(op.task.Snapshot())
	}
	if old.Text != "keep original draft" || sessions.SelectedSessionID() != "previous" {
		t.Fatal("preparation changed original draft/selection")
	}
	s.mu.Lock()
	s.setMainRouteLocked(routeMarket)
	s.mu.Unlock()
	s.finishCreatorPluginDraft(nil, op, ticket, true)
	if sessions.SelectedSessionID() != "previous" || s.composerDrafts[target] != nil {
		t.Fatal("late Creator result stole navigation or seeded a draft after departure")
	}
	// A new explicit use of the same retained result is allowed from the new
	// route, while the callback belonging to the departed route was rejected.
	op.route = s.captureMainRoute()
	s.finishCreatorPluginDraft(nil, op, ticket, true)
	if sessions.SelectedSessionID() != target || target == "previous" {
		t.Fatal("did not select confirmed fresh target")
	}
	if old.Text != "keep original draft" || len(old.Attachments) != 1 || old.Attachments[0].Path != "/fixture/keep.txt" {
		t.Fatal("original draft or attachments changed")
	}
	if draft := s.composerDrafts[target]; draft == nil || draft.Text != op.prompt {
		t.Fatal("missing prepared creator draft")
	}
	s.finishCreatorPluginDraft(nil, op, ticket, true)
	if s.composerDrafts[target].Text != op.prompt {
		t.Fatal("replayed completion duplicated template")
	}
}
