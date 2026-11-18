package main

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/recovery"
	"whaleisle/internal/store/session"
)

func TestSubmissionCheckpointKeepsNewerDraftAndTerminalIdentity(t *testing.T) {
	state := recovery.State{Drafts: map[string]json.RawMessage{"a": json.RawMessage(`{"Text":"before","SendSequence":0}`)}}
	record := session.SubmissionRecord{RequestID: "r1", SessionID: "a", DraftKey: "a", DraftSequence: 1,
		Phase: "prepared", DraftSnapshot: json.RawMessage(`{"Text":"","SendSequence":1,"Sends":[{"RequestID":"r1"}]}`)}
	if err := applySubmissionRecord(&state, "profile", record); err != nil {
		t.Fatal(err)
	}
	record.Phase, record.DraftSnapshot = "confirmed", nil
	if err := applySubmissionRecord(&state, "profile", record); err != nil {
		t.Fatal(err)
	}
	applyResumeOwned(&state, recovery.State{Drafts: map[string]json.RawMessage{"a": json.RawMessage(`{"Text":"before","SendSequence":0}`)}})
	var saved session.SubmissionRecovery
	if err := json.Unmarshal(state.Submissions, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Records["r1"].Phase != "confirmed" || string(state.Drafts["a"]) != `{"Text":"","SendSequence":1,"Sends":[{"RequestID":"r1"}]}` {
		t.Fatal("stale queued checkpoint resurrected sent text or removed terminal identity")
	}
	applyResumeOwned(&state, recovery.State{Drafts: map[string]json.RawMessage{"a": json.RawMessage(`{"Text":"newer","SendSequence":1}`)}})
	saved = session.SubmissionRecovery{}
	if err := json.Unmarshal(state.Submissions, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Records) != 0 || string(state.Drafts["a"]) != `{"Text":"newer","SendSequence":1}` {
		t.Fatal("settled checkpoint failed to retire tombstone and preserve newer input")
	}
}

func TestSubmissionCheckpointDoesNotRollBackFasterNextSend(t *testing.T) {
	state := recovery.State{}
	newer := json.RawMessage(`{"Text":"typed after two sends","SendSequence":2}`)
	applyResumeOwned(&state, recovery.State{Drafts: map[string]json.RawMessage{"a": newer}})
	for _, seq := range []uint64{2, 1} {
		record := session.SubmissionRecord{RequestID: string(rune('a' + seq)), SessionID: "a", DraftKey: "a", DraftSequence: seq,
			Phase: "prepared", DraftSnapshot: json.RawMessage(`{"Text":"older capture","SendSequence":1}`)}
		if err := applySubmissionRecord(&state, "profile", record); err != nil {
			t.Fatal(err)
		}
	}
	var saved session.SubmissionRecovery
	if err := json.Unmarshal(state.Submissions, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Records) != 2 || string(state.Drafts["a"]) != string(newer) {
		t.Fatal("out-of-order stage write lost a request or replaced a newer draft")
	}
}
