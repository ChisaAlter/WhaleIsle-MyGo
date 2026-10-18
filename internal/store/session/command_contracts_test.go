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
)

func TestCommandHistoryPairsAcrossPagesAndKeepsJournal(t *testing.T) {
	st := New()
	st.selectedID = "one"
	st.applyPersistenceEvent(json.RawMessage(`{"type":"command/done","seq":9,"data":{"commandId":"cmd-1","kind":"success","text":"first\nsecond","sourceEventSeq":8}}`))
	orphan := st.Snapshot().Messages
	if len(orphan) != 1 || orphan[0].Command == nil || orphan[0].Command.State != "success" {
		t.Fatalf("done-only page disappeared: %+v", orphan)
	}
	run, ok := classifyPersistenceEvent(json.RawMessage(`{"type":"command/run","seq":5,"time":100,"data":{"commandId":"cmd-1","name":"inspect","args":" details"}}`))
	if !ok {
		t.Fatal("command/run was dropped")
	}
	st.mu.Lock()
	st.messages = append([]Message{run}, st.messages...)
	st.mu.Unlock()
	st.applyPersistenceEvent(json.RawMessage(`{"type":"command/done","seq":9,"data":{"commandId":"cmd-1","kind":"success","text":"first\nsecond","sourceEventSeq":8}}`))
	snap := st.Snapshot()
	if len(snap.Messages) != 1 || snap.Messages[0].Seq != 5 || snap.Messages[0].ID != orphan[0].ID {
		t.Fatalf("paging duplicated or moved the command identity: %+v", snap.Messages)
	}
	view := snap.Messages[0].Command
	if view.Name != "inspect" || view.Args != " details" || view.Text != "first\nsecond" || view.SourceEventSeq == nil || *view.SourceEventSeq != 8 {
		t.Fatalf("lost command contract: %+v", view)
	}
	view.Text = "mutated UI"
	*view.SourceEventSeq = 99
	if got := st.Snapshot().Messages[0].Command; got.Text != "first\nsecond" || *got.SourceEventSeq != 8 {
		t.Fatalf("snapshot leaked mutable command data: %+v", got)
	}
	if len(st.messages) != 3 || oldestPersistenceSeq(st.messages) != 5 || maxPersistenceSeq(st.messages) != 9 {
		t.Fatal("projection changed the journal or paging cursor")
	}
	st.applyPersistenceEvent(json.RawMessage(`{"type":"command/run","seq":10,"data":{"commandId":"cmd-2","name":"compact"}}`))
	if got := st.Snapshot().Messages[1].Command.State; got != "running" {
		t.Fatalf("unsettled state = %q", got)
	}
	st.applyPersistenceEvent(json.RawMessage(`{"type":"command/done","seq":11,"data":{"commandId":"cmd-2","kind":"error","text":"try again"}}`))
	if got := st.Snapshot().Messages[1].Command; got.State != "error" || got.Text != "try again" {
		t.Fatalf("failure lost: %+v", got)
	}
	if _, ok := classifyPersistenceEvent(json.RawMessage(`{"type":"command/done","seq":12,"data":{"kind":"success"}}`)); ok {
		t.Fatal("accepted a lifecycle event without command identity")
	}
}

func TestCommandSubmissionUsesArraysAndCatalogOwnership(t *testing.T) {
	var executeCount atomic.Int32
	var promptCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args map[string]json.RawMessage `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		args := body.Payload.Args
		switch {
		case strings.HasSuffix(r.URL.Path, "/commands/list"):
			rpcOK(w, body.RPCID, []any{map[string]any{"name": "inspect", "description": "Inspect", "input": map[string]any{"hint": "Details"}}})
		case strings.HasSuffix(r.URL.Path, "/commands/execute"):
			executeCount.Add(1)
			if string(args["submittedAttachments"]) != "[]" {
				t.Errorf("attachment-free command requires [], got %s", args["submittedAttachments"])
			}
			var id, line string
			_ = json.Unmarshal(args["agentId"], &id)
			_ = json.Unmarshal(args["line"], &line)
			if id != "original" {
				t.Errorf("changed command target: %q", id)
			}
			kind := "success"
			if line == "/fail" {
				kind = "error"
			}
			rpcOK(w, body.RPCID, map[string]any{"commandId": "cmd-test", "result": map[string]any{"kind": kind, "text": "retryable failure"}})
		case strings.HasSuffix(r.URL.Path, "/session/prompt"):
			promptCount.Add(1)
			var req struct {
				SessionID string `json:"sessionId"`
				Content   []struct {
					Text string `json:"text"`
				} `json:"content"`
			}
			_ = json.Unmarshal(args["request"], &req)
			if req.SessionID != "original" || len(req.Content) != 1 || req.Content[0].Text != "/my_skill brief" {
				t.Errorf("unclaimed skill token was changed: %+v", req)
			}
			rpcOK(w, body.RPCID, map[string]any{"accepted": true})
		default:
			t.Errorf("unexpected call: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.selectedID = "original"
	if err := st.SubmitComposer(context.Background(), "original", "/inspect\nline one\nline two", nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := st.SubmitComposer(context.Background(), "original", "/my_skill brief", nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := st.ExecuteCommandLine(context.Background(), "/fail"); err == nil || !strings.Contains(err.Error(), "retryable failure") {
		t.Fatalf("application error was swallowed: %v", err)
	}
	if executeCount.Load() != 2 || promptCount.Load() != 1 {
		t.Fatalf("wrong dispatch counts: commands=%d prompts=%d", executeCount.Load(), promptCount.Load())
	}
}

func TestCommandCatalogRejectsLateRepliesAndStaleEvents(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if calls.Add(1) == 1 {
			entered <- struct{}{}
			<-release
			rpcOK(w, body["rpcId"], []any{map[string]any{"name": "obsolete"}})
			return
		}
		rpcOK(w, body["rpcId"], []any{map[string]any{"name": "fresh"}})
	}))
	defer srv.Close()
	defer close(release)
	wake := make(chan struct{}, 16)
	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, func() {
		select {
		case wake <- struct{}{}:
		default:
		}
	})
	st.selectedID = "original"
	owner := st.BeginEvents(context.Background(), client)
	st.EnsureSlashCommands(context.Background())
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("catalog did not start")
	}
	st.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "commands/change"})
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		snap := st.Snapshot()
		if len(snap.SlashCommands) == 1 && snap.SlashCommands[0].Name == "fresh" && !snap.SlashCommandsLoading {
			break
		}
		select {
		case <-wake:
		case <-deadline.C:
			t.Fatalf("catalog did not recover: %+v", snap)
		}
	}
	st.mu.Lock()
	st.eventsGeneration++
	st.mu.Unlock()
	st.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "commands/change"})
	if snap := st.Snapshot(); len(snap.SlashCommands) != 1 || snap.SlashCommands[0].Name != "fresh" {
		t.Fatal("superseded stream cleared the current catalog")
	}
	st.mu.Lock()
	st.selectSessionLocked("other")
	st.mu.Unlock()
	if snap := st.Snapshot(); len(snap.SlashCommands) != 0 {
		t.Fatal("catalog leaked into a different session")
	}
	st.Detach()
}
