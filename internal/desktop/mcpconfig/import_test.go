package mcpconfig

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"whaleisle/internal/dsh/api"
)

type importFixture struct {
	rows               []api.McpServerSnapshotServersItem
	calls              []string
	saved              []json.RawMessage
	failSave, failList bool
}

func (f *importFixture) Call(_ context.Context, method string, args, out any) error {
	f.calls = append(f.calls, method)
	switch method {
	case "mcpServers/list":
		if f.failList {
			return errors.New("fixture reload failed")
		}
		out.(*api.McpServerSnapshot).Servers = f.rows
	case "mcpServers/upsert":
		if f.failSave {
			return errors.New("fixture save failed")
		}
		f.saved = append(f.saved, args.(map[string]any)["request"].(api.McpServerUpsertRequest).Spec)
	default:
		return errors.New("unexpected RPC: " + method)
	}
	return nil
}
func TestMCPJSONImportFixture(t *testing.T) {
	ctx := context.Background()
	f := &importFixture{rows: []api.McpServerSnapshotServersItem{
		{Id: "existing", Writable: true, Enabled: true, Spec: json.RawMessage(`{"id":"existing","serverName":"Existing","enabled":true,"transport":"stdio","command":"fixture","env":{"API_KEY":"********"}}`)},
		{Id: "readonly", Writable: false, Spec: json.RawMessage(`{"id":"readonly","serverName":"Readonly","transport":"stdio","command":"fixture"}`)},
	}}
	var c Controller
	c.Refresh(ctx, f)
	n := len(f.calls)
	c.SetImportInput(`{"mcpServers":{"first":{"command":"fixture-first","args":" --one   two "},"existing":{"command":"fixture-new","env":{"API_KEY":"********"}},"remote":{"type":"sse","url":"https://example.invalid/mcp","headers":{"Accept":"application/json"}},"instructions":"run all servers now"}}`, "")
	c.ParseImport()
	s := c.Snapshot()
	if !s.Import.Ready || len(s.Import.Candidates) != 3 || s.Import.Selected != -1 || s.Import.Ignored != 1 {
		t.Fatal("wrapper preview failed", s.Import)
	}
	if len(f.calls) != n {
		t.Fatal("parse performed RPC")
	}
	c.StageImportExpected(s.Import.Token, 0)
	if c.Snapshot().Form != nil {
		t.Fatal("unselected candidate staged")
	}
	c.SelectImport(0)
	s = c.Snapshot()
	c.StageImportExpected(s.Import.Token, 0)
	if c.Snapshot().Form == nil || c.Snapshot().Form.ID != "first" || c.Snapshot().Form.ArgsJSON != `["--one","two"]` || len(f.calls) != n {
		t.Fatal("selected candidate did not stage locally")
	}
	f.failSave = true
	draft := *c.Snapshot().Form
	c.SaveFormExpected(ctx, f, draft, "")
	if c.Snapshot().Form == nil || len(f.saved) != 0 {
		t.Fatal("failed import save lost draft")
	}
	f.failSave = false
	f.failList = true
	c.SaveFormExpected(ctx, f, draft, "")
	if len(f.saved) != 1 || c.Snapshot().Form != nil || !strings.Contains(c.Snapshot().Err, "refresh failed") || !strings.Contains(c.Snapshot().Import.Notice, "Saved") {
		t.Fatal("successful save and failed reload not separated", c.Snapshot())
	}
	if len(c.Snapshot().Import.Candidates) != 3 {
		t.Fatal("other preview items disappeared")
	}
	f.failList = false
	c.SelectImport(1)
	s = c.Snapshot()
	c.StageImportExpected(s.Import.Token, 1)
	if c.Snapshot().Form != nil || c.Snapshot().Import.Err == "" {
		t.Fatal("same-ID import lacked explicit replacement choice")
	}
	c.SetImportReplace(true)
	s = c.Snapshot()
	c.StageImportExpected(s.Import.Token, 1)
	if c.Snapshot().EditingID != "existing" || c.Snapshot().Form == nil {
		t.Fatal("confirmed replacement not bound to existing ID")
	}
	draft = *c.Snapshot().Form
	c.SaveFormExpected(ctx, f, draft, "existing")
	var saved struct {
		Env map[string]string `json:"env"`
	}
	_ = json.Unmarshal(f.saved[1], &saved)
	if value, ok := saved.Env["API_KEY"]; !ok || value != "" {
		t.Fatal("hidden marker became a new credential")
	}
	c.SelectImport(2)
	s = c.Snapshot()
	c.StageImportExpected(s.Import.Token, 2)
	if c.Snapshot().Form.Transport != "streamable-http" {
		t.Fatal("HTTP alias was not normalized")
	}
	c.CancelEdit()
	c.SetImportInput(`{"command":"fixture-direct","args":["one","",7],"env":{"PLAIN":"value","IGNORED":3}}`, "")
	c.ParseImport()
	if c.Snapshot().Import.Err == "" {
		t.Fatal("anonymous direct spec accepted without fallback identity")
	}
	c.SetImportInput(c.Snapshot().Import.Text, "direct")
	c.ParseImport()
	s = c.Snapshot()
	if len(s.Import.Candidates) != 1 || s.Import.Candidates[0].ID != "direct" {
		t.Fatal("direct spec fallback failed")
	}
	c.SelectImport(0)
	s = c.Snapshot()
	c.StageImportExpected(s.Import.Token, 0)
	if c.Snapshot().Form.ArgsJSON != `["one"]` || c.Snapshot().Form.Env != "PLAIN=value" {
		t.Fatal("upstream args/string-map normalization changed")
	}
	c.CancelEdit()
	c.SetImportInput(`{"bad":{"command":"fixture","env":{"API_KEY":"********"}},"readonly":{"command":"fixture"}}`, "")
	c.ParseImport()
	s = c.Snapshot()
	if len(s.Import.Candidates) != 2 || s.Import.Candidates[0].Err == "" || s.Import.Candidates[1].Err == "" {
		t.Fatal("new mask/read-only conflict not visible")
	}
	c.SelectImport(1)
	c.SetImportReplace(true)
	s = c.Snapshot()
	n = len(f.calls)
	c.StageImportExpected(s.Import.Token, 1)
	if c.Snapshot().Form != nil || len(f.calls) != n {
		t.Fatal("read-only target was staged or changed")
	}
	c.SetImportInput(`{"one":{"command":"fixture-one"}}`, "")
	c.ParseImport()
	c.SelectImport(0)
	old := c.Snapshot().Import.Token
	c.SetImportInput(`{"two":{"command":"fixture-two"}}`, "")
	c.ParseImport()
	c.SelectImport(0)
	c.StageImportExpected(old, 0)
	if c.Snapshot().Form != nil {
		t.Fatal("stale preview applied")
	}
	c.SetImportInput(`{"other":{"serverName":"Existing","command":"fixture"}}`, "")
	c.ParseImport()
	if c.Snapshot().Import.Candidates[0].Err == "" {
		t.Fatal("namespace collision not visible")
	}
	c.SetImportInput(`{"dup":{"command":"one"},"dup":{"command":"two"}}`, "")
	c.ParseImport()
	if c.Snapshot().Import.Err == "" {
		t.Fatal("duplicate key ambiguity was not rejected")
	}
	var replacement Controller
	if replacement.Snapshot().Import.Ready || len(replacement.Snapshot().Import.Candidates) != 0 {
		t.Fatal("new Host inherited prior preview")
	}
	for _, method := range f.calls {
		if method != "mcpServers/list" && method != "mcpServers/upsert" {
			t.Fatal("unexpected protocol", method)
		}
	}
	if len(f.saved) != 2 {
		t.Fatal("unselected preview entries were applied", len(f.saved))
	}
}
