package mcpconfig

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"whaleisle/internal/dsh/api"
)

type catalogFixture struct {
	rows                 []api.McpServerSnapshotServersItem
	calls                []string
	testError, listError error
	entered, release     chan struct{}
	result               string
}

func (f *catalogFixture) Call(_ context.Context, method string, args, out any) error {
	f.calls = append(f.calls, method)
	switch method {
	case "mcpServers/list":
		if f.listError != nil {
			return f.listError
		}
		out.(*api.McpServerSnapshot).Servers = f.rows
	case "mcpServers/test":
		if got := args.(map[string]any)["request"].(api.McpServerIdRequest).Id; got != "local" && got != "remote" {
			return errors.New("unexpected test target")
		}
		if f.entered != nil {
			close(f.entered)
			<-f.release
		}
		if f.testError != nil {
			return f.testError
		}
		*out.(*json.RawMessage) = json.RawMessage(f.result)
	default:
		return errors.New("unexpected mutating RPC: " + method)
	}
	return nil
}
func TestMCPCatalogFixture(t *testing.T) {
	ctx := context.Background()
	f := &catalogFixture{
		rows: []api.McpServerSnapshotServersItem{
			{Id: "local", Origin: "managed", Writable: true, Enabled: true, Spec: json.RawMessage(`{"id":"local","serverName":"Workspace","transport":"stdio","command":"fixture-bin","env":{"HIDDEN":"secret-value"}}`), Connection: &api.McpServerSnapshotServersItemConnection{Health: "connected", Tools: []string{"mcp__Workspace__read"}}},
			{Id: "remote", Origin: "composition", Enabled: false, Spec: json.RawMessage(`{"id":"remote","serverName":"Docs","transport":"streamable-http","url":"https://example.invalid/mcp","headers":{"HIDDEN":"secret-value"}}`)},
		},
		result: `{"ok":true,"serverName":"Workspace","transport":"stdio","toolCount":2,"toolNames":["read","search"],"elapsedMs":12}`,
	}
	var c Controller
	c.Refresh(ctx, f)
	c.SetQuery(" WORKSPACE ")
	if got := c.Snapshot().CatalogRows(); len(got) != 1 || got[0].Id != "local" {
		t.Fatal("name search", got)
	}
	c.SetQuery("fixture-bin")
	if len(c.Snapshot().CatalogRows()) != 1 {
		t.Fatal("command search")
	}
	c.SetQuery("example.invalid")
	if got := c.Snapshot().CatalogRows(); len(got) != 1 || got[0].Id != "remote" {
		t.Fatal("URL search")
	}
	c.SetQuery("secret-value")
	if len(c.Snapshot().CatalogRows()) != 0 {
		t.Fatal("private env/header values entered search index")
	}
	c.SetQuery("")
	c.SetEnabledFilter("disabled")
	if got := c.Snapshot().CatalogRows(); len(got) != 1 || got[0].Id != "remote" {
		t.Fatal("disabled filter")
	}
	c.SetEnabledFilter("enabled")
	if got := c.Snapshot().CatalogRows(); len(got) != 1 || got[0].Id != "local" {
		t.Fatal("enabled filter")
	}
	c.SetEnabledFilter("unexpected")
	if len(c.Snapshot().CatalogRows()) != 2 {
		t.Fatal("unknown filter should normalize to all")
	}
	c.Inspect("local")
	if c.Snapshot().ExpandedID != "local" {
		t.Fatal("inspection did not open")
	}
	c.Inspect("local")
	c.Inspect("missing")
	if c.Snapshot().ExpandedID != "" || !reflect.DeepEqual(f.calls, []string{"mcpServers/list"}) {
		t.Fatal("browsing triggered RPC", f.calls)
	}
	s := c.Snapshot()
	s.Rows[0].Connection.Tools[0] = "changed"
	if c.Snapshot().Rows[0].Connection.Tools[0] != "mcp__Workspace__read" {
		t.Fatal("snapshot leaked live tools")
	}
	c.RowAction(ctx, f, "local", "test")
	result, ok := c.Snapshot().Tests["local"]
	if !ok || result.Err != "" || result.Result.ToolCount != 2 || !reflect.DeepEqual(result.Result.ToolNames, []string{"read", "search"}) {
		t.Fatal("test details missing", result)
	}
	result.Result.ToolNames[0] = "changed"
	if c.Snapshot().Tests["local"].Result.ToolNames[0] != "read" {
		t.Fatal("snapshot leaked test tools")
	}
	c.Refresh(ctx, f)
	if len(c.Snapshot().Tests) != 1 {
		t.Fatal("normal refresh erased diagnostics")
	}
	f.testError = errors.New("fixture test failure")
	c.RowAction(ctx, f, "local", "test")
	if c.Snapshot().Tests["local"].Err != "fixture test failure" || len(c.Snapshot().Tests["local"].Result.ToolNames) != 0 {
		t.Fatal("failed retest retained old success")
	}
	c.Refresh(ctx, f)
	if c.Snapshot().Tests["local"].Err == "" {
		t.Fatal("refresh erased row failure")
	}
	f.testError = nil
	f.listError = errors.New("fixture reload failure")
	c.RowAction(ctx, f, "local", "test")
	if len(c.Snapshot().Tests["local"].Result.ToolNames) != 2 || c.Snapshot().Err == "" {
		t.Fatal("post-test refresh failure hid test result")
	}
	f.listError = nil
	f.rows[0].Spec = json.RawMessage(`{"id":"local","serverName":"Workspace","transport":"stdio","command":"changed-bin"}`)
	c.Refresh(ctx, f)
	if len(c.Snapshot().Tests) != 0 {
		t.Fatal("changed spec retained stale test result")
	}
	f.entered = make(chan struct{})
	f.release = make(chan struct{})
	done := make(chan struct{})
	go func() { c.RowAction(ctx, f, "remote", "test"); close(done) }()
	<-f.entered
	before := len(f.calls)
	c.RowAction(ctx, f, "remote", "test")
	c.Refresh(ctx, f)
	if len(f.calls) != before {
		t.Fatal("duplicate RPC while test pending")
	}
	close(f.release)
	<-done
	if c.Snapshot().Tests["remote"].Err != "" {
		t.Fatal("explicit read-only composition test unavailable")
	}
	c.Inspect("remote")
	f.rows = f.rows[:1]
	c.Refresh(ctx, f)
	if _, ok := c.Snapshot().Tests["remote"]; ok || c.Snapshot().ExpandedID != "" {
		t.Fatal("removed row retained diagnostics")
	}
}
