package schedule

import (
	"encoding/json"
	"testing"

	schedstore "whaleisle/internal/store/schedule"
)

func draftFixture(at, title, prompt string) schedstore.Entry {
	raw, _ := json.Marshal(map[string]any{
		"id": "task", "kind": "daily", "title": title, "prompt": prompt,
		"scheduledAt": at, "time": "09:00:00", "timeZone": "UTC",
	})
	return schedstore.Entry{ID: "task", SessionID: "owner", Origin: "host", Title: title, Prompt: prompt, Status: "active", Raw: raw}
}

func TestDraftRebasePreservesEditsAndUpdatesExpected(t *testing.T) {
	before := draftFixture("2026-10-15T09:00:00Z", "original", "old prompt")
	after := draftFixture("2026-10-16T09:00:00Z", "remote title", "remote prompt")
	draft := newTaskDraft(before)
	draft.Prompt = "my edit"
	draft.rebase(after)
	if draft.Title != "remote title" || draft.Prompt != "my edit" || string(draft.row.Raw) != string(after.Raw) {
		t.Fatalf("rebased draft: %+v", draft)
	}
	if change, err := draft.change(); err != nil || change != nil {
		t.Fatalf("untouched timing was changed: %s, %v", change, err)
	}
	draft.Kind = "every"
	draft.Seconds = "600"
	newer := draftFixture("2026-10-17T09:00:00Z", "newest", "remote prompt")
	draft.rebase(newer)
	if draft.Kind != "every" || draft.Seconds != "600" || string(draft.row.Raw) != string(newer.Raw) {
		t.Fatalf("staged rule switch lost: %+v", draft)
	}
	newer.Status = "inactive"
	draft.rebase(newer)
	if draft.Kind != "daily" || draft.Prompt != "remote prompt" {
		t.Fatalf("ended task kept misleading draft: %+v", draft)
	}
}
