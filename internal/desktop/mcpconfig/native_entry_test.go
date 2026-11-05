package mcpconfig

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestAddAndEditNativeEntryPointsKeepIdentityAndDraftBoundaries(t *testing.T) {
	f := &formFixture{rows: []api.McpServerSnapshotServersItem{{Id: "old", Writable: true, Spec: json.RawMessage(`{"id":"old","transport":"stdio","command":"fixture","future":{"keep":true}}`)}, {Id: "readonly", Writable: false, Spec: json.RawMessage(`{"id":"readonly","command":"fixture"}`)}}}
	var c Controller
	c.Refresh(context.Background(), f)
	c.OpenFormFor("old")
	snap := c.Snapshot()
	if snap.Form == nil || snap.EditingID != "old" || snap.Form.ID != "old" || !strings.Contains(snap.Form.base, "future") {
		t.Fatal("edit did not open original structured identity")
	}
	c.NewForm()
	snap = c.Snapshot()
	if snap.Form == nil || snap.EditingID != "" || snap.Form.ID != "" || snap.Draft != "" || strings.Contains(snap.Form.base, "future") {
		t.Fatal("Add reused old edit draft")
	}
	c.OpenFormFor("readonly")
	if c.Snapshot().EditingID != "" {
		t.Fatal("read-only record edited")
	}
	if len(f.calls) != 1 {
		t.Fatal("opening form launched external server or wrote config", f.calls)
	}
	c.ShowJSON()
	snap = c.Snapshot()
	if !snap.EditorOpen || snap.Form != nil || !json.Valid([]byte(snap.Draft)) || !strings.Contains(snap.Draft, `"command": ""`) {
		t.Fatal("blank Add cannot switch to JSON without a complete server", snap)
	}
	c.Save(context.Background(), f)
	if c.Snapshot().Err == "" || c.Snapshot().Draft != snap.Draft || len(f.calls) != 1 {
		t.Fatal("invalid blank Save lost draft or mutated Host")
	}
	c.SetDraft(`{"id":"unfinished","transport":"stdio","command":42}`)
	c.OpenForm()
	if after := c.Snapshot(); after.Form != nil || !after.EditorOpen || after.Err == "" || after.Draft != `{"id":"unfinished","transport":"stdio","command":42}` {
		t.Fatal("failed JSON-to-form conversion closed editor or lost draft", after)
	}
	c.CancelEdit()
	if c.Snapshot().EditorOpen || len(f.calls) != 1 {
		t.Fatal("cancel did not close editor without writing Host")
	}
}
