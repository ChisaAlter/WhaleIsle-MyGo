package mcpconfig

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"whaleisle/internal/dsh/api"
)

type fixture struct {
	rows           []api.McpServerSnapshotServersItem
	fail           string
	calls          []string
	saved          api.McpServerUpsertRequest
	block, entered chan struct{}
}

func (f *fixture) Call(_ context.Context, method string, args, out any) error {
	f.calls = append(f.calls, method)
	if f.block != nil && method == "mcpServers/upsert" {
		close(f.entered)
		<-f.block
	}
	if method == f.fail {
		return errors.New("fixture rejected")
	}
	switch method {
	case "mcpServers/list":
		out.(*api.McpServerSnapshot).Servers = f.rows
	case "mcpServers/upsert":
		f.saved = args.(map[string]any)["request"].(api.McpServerUpsertRequest)
		f.rows[0].Spec = f.saved.Spec
	case "mcpServers/delete":
		f.rows = f.rows[1:]
	}
	return nil
}
func TestManagedMCPFixture(t *testing.T) {
	ctx := context.Background()
	f := &fixture{rows: []api.McpServerSnapshotServersItem{
		{Id: "demo", Origin: "managed", Writable: true, Enabled: true, Spec: json.RawMessage(`{"id":"demo","enabled":true,"serverName":"demo","transport":"stdio","command":"mock","env":{"FIXTURE":"only"},"futureOption":42}`)},
		{Id: "built-in", Origin: "composition", Spec: json.RawMessage(`{"id":"built-in","transport":"stdio","command":"mock"}`)},
	}}
	var c Controller
	c.Refresh(ctx, f)
	if !c.Snapshot().Ready {
		t.Fatal("list not ready")
	}
	c.Edit("built-in")
	if c.Snapshot().EditingID != "" {
		t.Fatal("read-only row opened")
	}
	n := len(f.calls)
	c.SetEnabled(ctx, f, "built-in", true)
	c.RequestDelete("built-in")
	c.Delete(ctx, f)
	if len(f.calls) != n {
		t.Fatal("read-only mutation reached host")
	}
	c.Edit("demo")
	original := c.Snapshot().Draft
	c.SetDraft(`{"id":"renamed","transport":"stdio","command":"mock"}`)
	c.Save(ctx, f)
	if len(f.calls) != n {
		t.Fatal("editing renamed id must not create second server")
	}
	c.SetDraft(original)
	f.fail = "mcpServers/upsert"
	c.Save(ctx, f)
	if c.Snapshot().Draft != original || c.Snapshot().Err == "" {
		t.Fatal("failed save lost draft or error")
	}
	f.fail = ""
	f.block = make(chan struct{})
	f.entered = make(chan struct{})
	done := make(chan struct{})
	go func() { c.Save(ctx, f); close(done) }()
	<-f.entered
	n = len(f.calls)
	c.Save(ctx, f)
	c.RequestDelete("demo")
	c.Delete(ctx, f)
	if len(f.calls) != n {
		t.Fatal("duplicate mutation while busy")
	}
	c.SetDraft(original + " ") // A newer draft must survive an in-flight save.
	close(f.block)
	<-done
	f.block = nil
	if c.Snapshot().Draft != original+" " {
		t.Fatal("save erased newer draft")
	}
	var saved map[string]any
	json.Unmarshal(f.saved.Spec, &saved)
	if saved["futureOption"] != float64(42) {
		t.Fatal("unknown host field lost")
	}
	c.CancelEdit()
	if c.Snapshot().Draft != "" {
		t.Fatal("cancel retained editor")
	}
	c.RequestDelete("demo")
	c.CancelDelete()
	n = len(f.calls)
	c.Delete(ctx, f)
	if len(f.calls) != n {
		t.Fatal("cancelled delete reached host")
	}
	c.RequestDelete("demo")
	f.fail = "mcpServers/delete"
	c.Delete(ctx, f)
	if c.Snapshot().DeletingID != "demo" || c.Snapshot().Err == "" {
		t.Fatal("failed delete lost target or error")
	}
	f.fail = ""
	c.Delete(ctx, f)
	if c.Snapshot().DeletingID != "" || len(c.Snapshot().Rows) != 1 {
		t.Fatal("delete did not refresh list")
	}
}
