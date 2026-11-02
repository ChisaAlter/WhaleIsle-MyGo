package chat

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"whaleisle/internal/store/session"
)

func TestDeliveryProjectionMemoWarmHitAndIndependentOutput(t *testing.T) {
	snap := deliveryMemoSnapshot()
	original := snap.Messages[0].TurnDelivery
	first := projectChatMessages(snap)
	second := projectChatMessages(snap)
	if len(first) != 1 || len(second) != 1 || &first[0] != &second[0] {
		t.Fatal("unchanged delivery metadata rebuilt the chat projection")
	}
	if first[0].TurnDelivery == original || first[0].TurnDelivery.Changes == original.Changes || &first[0].TurnDelivery.Files[0] == &original.Files[0] {
		t.Fatal("memoized output retains caller-owned delivery data")
	}
	cp := chatProjectionOf(snap)
	cp.mu.Lock()
	if len(cp.source) != 1 || cp.source[0].message.TurnDelivery != nil {
		cp.mu.Unlock()
		t.Fatal("plain source value retains a caller-owned delivery pointer")
	}
	cp.mu.Unlock()
	original.Turn = 9
	original.Changes.Index = 8
	original.Files[0].Path = "/synthetic/report-b.txt"
	if first[0].TurnDelivery.Turn != 1 || first[0].TurnDelivery.Changes.Index != 3 || first[0].TurnDelivery.Files[0].Path != "/synthetic/report-a.txt" {
		t.Fatal("input mutation altered an already returned projection")
	}
	current := projectChatMessages(snap)
	if &current[0] == &first[0] || current[0].TurnDelivery.Turn != 9 || current[0].TurnDelivery.Changes.Index != 8 || current[0].TurnDelivery.Files[0].Path != "/synthetic/report-b.txt" {
		t.Fatal("owned source identity did not invalidate for in-place delivery mutation")
	}
}

func TestDeliveryProjectionMemoEveryNestedFieldInvalidates(t *testing.T) {
	cases := []struct {
		name   string
		change func(*session.TurnDelivery)
	}{
		{"turn", func(d *session.TurnDelivery) { d.Turn++ }},
		{"changes-kind", func(d *session.TurnDelivery) { d.Changes.Kind = "changed" }},
		{"changes-turn", func(d *session.TurnDelivery) { d.Changes.Turn++ }},
		{"changes-seq", func(d *session.TurnDelivery) { d.Changes.Seq++ }},
		{"changes-index", func(d *session.TurnDelivery) { d.Changes.Index++ }},
		{"changes-nil", func(d *session.TurnDelivery) { d.Changes = nil }},
		{"files-kind", func(d *session.TurnDelivery) { d.Files[0].Target.Kind = "different" }},
		{"files-turn", func(d *session.TurnDelivery) { d.Files[0].Target.Turn++ }},
		{"files-seq", func(d *session.TurnDelivery) { d.Files[0].Target.Seq++ }},
		{"files-index", func(d *session.TurnDelivery) { d.Files[0].Target.Index++ }},
		{"files-path-same-length", func(d *session.TurnDelivery) { d.Files[0].Path = "/synthetic/report-b.txt" }},
		{"files-description-same-length", func(d *session.TurnDelivery) { d.Files[0].Description = "report B" }},
		{"files-count", func(d *session.TurnDelivery) { d.Files = append(d.Files, d.Files[0]) }},
		{"files-empty", func(d *session.TurnDelivery) { d.Files = []session.TurnPresentedFile{} }},
		{"files-nil", func(d *session.TurnDelivery) { d.Files = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := deliveryMemoSnapshot()
			snap.SelectedID = "delivery-field-" + tc.name
			first := projectChatMessages(snap)
			wantBefore := deliveryMemoValue()
			tc.change(snap.Messages[0].TurnDelivery)
			current := projectChatMessages(snap)
			if &first[0] == &current[0] || !reflect.DeepEqual(current[0].TurnDelivery, snap.Messages[0].TurnDelivery) {
				t.Fatal("changed delivery field reused stale output")
			}
			if !reflect.DeepEqual(first[0].TurnDelivery, wantBefore) {
				t.Fatal("caller-owned delivery mutation modified prior output")
			}
			if again := projectChatMessages(snap); &again[0] != &current[0] {
				t.Fatal("unchanged mutated delivery did not stay warm")
			}
		})
	}
}

func TestDeliveryProjectionMemoNilEmptyAndEqualReplacement(t *testing.T) {
	snap := deliveryMemoSnapshot()
	snap.SelectedID = "delivery-nil-empty"
	first := projectChatMessages(snap)
	snap.Messages[0].TurnDelivery = deliveryMemoValue()
	if next := projectChatMessages(snap); &first[0] != &next[0] {
		t.Fatal("equal independently owned metadata did not hit")
	}
	states := []*session.TurnDelivery{nil, {}, {Changes: &session.TurnFileTarget{}}, {Files: []session.TurnPresentedFile{}}, {Files: nil}}
	for index, state := range states {
		snap.Messages[0].TurnDelivery = state
		current := projectChatMessages(snap)
		if &first[0] == &current[0] || !reflect.DeepEqual(state, current[0].TurnDelivery) {
			t.Fatalf("nil/empty state %d reused prior projection", index)
		}
		if again := projectChatMessages(snap); &again[0] != &current[0] {
			t.Fatalf("nil/empty state %d did not stay warm", index)
		}
		first = current
	}
}

func TestDeliveryProjectionMemoRawAndContentRemainAuthoritative(t *testing.T) {
	snap := deliveryMemoSnapshot()
	snap.SelectedID = "delivery-raw-content"
	snap.Messages[0].Raw = json.RawMessage(`{"x":"a"}`)
	snap.Messages[0].ContentBlock = json.RawMessage(`{"text":"a"}`)
	first := projectChatMessages(snap)
	copy(snap.Messages[0].Raw, `{"x":"b"}`)
	current := projectChatMessages(snap)
	if &first[0] == &current[0] {
		t.Fatal("same-length raw edit did not invalidate delivery projection")
	}
	copy(snap.Messages[0].ContentBlock, `{"text":"b"}`)
	next := projectChatMessages(snap)
	if &current[0] == &next[0] {
		t.Fatal("same-length content edit did not invalidate delivery projection")
	}
}

func TestDeliveryProjectionMemoConcurrentIndependentReadOnlyInputs(t *testing.T) {
	const workers = 8
	const rounds = 16
	var pending sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		pending.Add(1)
		go func(worker int) {
			defer pending.Done()
			snap := deliveryMemoSnapshot()
			snap.Messages[0].TurnDelivery.Turn = uint64(worker + 1)
			for range rounds {
				out := projectChatMessages(snap)
				if len(out) != 1 || out[0].TurnDelivery.Turn != uint64(worker+1) || out[0].TurnDelivery == snap.Messages[0].TurnDelivery {
					t.Errorf("worker %d received a foreign or caller-owned delivery", worker)
					return
				}
			}
		}(worker)
	}
	pending.Wait()
}
