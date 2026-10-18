package mcpconfig

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/dsh/api"
)

type formFixture struct {
	rows             []api.McpServerSnapshotServersItem
	calls            []string
	saved            json.RawMessage
	fail             bool
	entered, release chan struct{}
}

func (f *formFixture) Call(_ context.Context, method string, args, out any) error {
	f.calls = append(f.calls, method)
	switch method {
	case "mcpServers/list":
		out.(*api.McpServerSnapshot).Servers = f.rows
	case "mcpServers/upsert":
		if f.entered != nil {
			close(f.entered)
			<-f.release
		}
		if f.fail {
			return errors.New("fixture save rejected")
		}
		f.saved = append(json.RawMessage(nil), args.(map[string]any)["request"].(api.McpServerUpsertRequest).Spec...)
	default:
		return errors.New("unexpected RPC: " + method)
	}
	return nil
}
func TestMCPStructuredEditorFixture(t *testing.T) {
	ctx := context.Background()
	f := &formFixture{rows: []api.McpServerSnapshotServersItem{
		{Id: "demo", Writable: true, Spec: json.RawMessage(`{"id":"demo","serverName":"Demo","enabled":true,"transport":"stdio","command":"fixture-bin","args":["path with spaces",""],"env":{"API_KEY":"********","PLAIN":"original"},"reconnect":{"enabled":true,"maxAttempts":3},"future":9007199254740993}`)},
		{Id: "readonly", Writable: false, Spec: json.RawMessage(`{"id":"readonly","transport":"stdio","command":"fixture"}`)},
	}}
	var c Controller
	c.Refresh(ctx, f)
	c.OpenForm()
	form := *c.Snapshot().Form
	form.ID = "new_server"
	form.ServerName = "NewServer"
	form.Command = "fixture-command"
	form.ArgsJSON = `["one two", ""]`
	form.Env = "PLAIN=value=with=equals"
	form.Timeout = "2500"
	c.SetForm(form)
	c.SaveFormExpected(ctx, f, form, "")
	if c.Snapshot().Form != nil {
		t.Fatal("successful form not closed")
	}
	var saved map[string]json.RawMessage
	_ = json.Unmarshal(f.saved, &saved)
	var args []string
	_ = json.Unmarshal(saved["args"], &args)
	if !reflect.DeepEqual(args, []string{"one two", ""}) || string(saved["toolCallTimeoutMs"]) != "2500" {
		t.Fatal("stdio fields not serialized exactly", string(f.saved))
	}
	c.OpenForm()
	bad := *c.Snapshot().Form
	bad.ID = "bad id"
	bad.ServerName = "bad/name"
	bad.Transport = "streamable-http"
	bad.URL = "file:///fixture"
	bad.Timeout = "0"
	bad.Headers = "bad header=value"
	c.SetForm(bad)
	n := len(f.calls)
	c.SaveFormExpected(ctx, f, bad, "")
	if len(f.calls) != n || len(c.Snapshot().FormErrors) < 5 || c.Snapshot().Form.ID != "bad id" {
		t.Fatal("invalid form changed host or lost draft", c.Snapshot())
	}
	bad.ID = "remote"
	bad.ServerName = "Remote"
	bad.URL = "https://example.invalid/mcp"
	bad.Timeout = "42"
	bad.Headers = "Accept=application/json"
	bad.Enabled = false
	c.SetForm(bad)
	c.ShowJSON()
	if c.Snapshot().Form != nil || !strings.Contains(c.Snapshot().Draft, `"streamable-http"`) {
		t.Fatal("valid form did not switch to original JSON editor")
	}
	c.OpenForm()
	if c.Snapshot().Form.URL != bad.URL || c.Snapshot().Form.Enabled {
		t.Fatal("JSON-to-form lost HTTP fields")
	}
	c.CancelEdit()
	c.Edit("demo")
	c.OpenForm()
	existing := *c.Snapshot().Form
	if !strings.Contains(existing.ArgsJSON, "path with spaces") || !strings.Contains(existing.Env, "API_KEY=********") {
		t.Fatal("original record projection lost fields")
	}
	existing.Command = "fixture-updated"
	c.SetForm(existing)
	f.fail = true
	c.SaveFormExpected(ctx, f, existing, "demo")
	if c.Snapshot().Form == nil || c.Snapshot().Form.Command != "fixture-updated" || c.Snapshot().Err != "fixture save rejected" {
		t.Fatal("failed save lost form")
	}
	f.fail = false
	c.SaveFormExpected(ctx, f, existing, "demo")
	_ = json.Unmarshal(f.saved, &saved)
	var env map[string]string
	_ = json.Unmarshal(saved["env"], &env)
	var reconnect struct {
		MaxAttempts int `json:"maxAttempts"`
	}
	_ = json.Unmarshal(saved["reconnect"], &reconnect)
	if env["API_KEY"] != "" || string(saved["future"]) != "9007199254740993" || reconnect.MaxAttempts != 3 {
		t.Fatal("mask or advanced-field preservation failed", string(f.saved))
	}
	c.Edit("demo")
	c.OpenForm()
	existing = *c.Snapshot().Form
	existing.ID = "renamed"
	c.SetForm(existing)
	n = len(f.calls)
	c.SaveFormExpected(ctx, f, existing, "demo")
	if len(f.calls) != n || c.Snapshot().FormErrors["id"] == "" {
		t.Fatal("edit identity was not bound")
	}
	c.CancelEdit()
	c.OpenForm()
	mask := *c.Snapshot().Form
	mask.ID = "new"
	mask.ServerName = "New"
	mask.Command = "fixture"
	mask.Env = "API_KEY=********"
	c.SetForm(mask)
	n = len(f.calls)
	c.SaveFormExpected(ctx, f, mask, "")
	if len(f.calls) != n || c.Snapshot().FormErrors["env"] == "" {
		t.Fatal("new placeholder accepted as credential")
	}
	c.CancelEdit()
	c.Edit("demo")
	c.OpenForm()
	changed := *c.Snapshot().Form
	changed.Transport = "streamable-http"
	changed.URL = "https://example.invalid"
	changed.Headers = "Authorization=********"
	c.SetForm(changed)
	n = len(f.calls)
	c.SaveFormExpected(ctx, f, changed, "demo")
	if len(f.calls) != n || c.Snapshot().FormErrors["headers"] == "" {
		t.Fatal("cross-transport placeholder accepted")
	}
	c.CancelEdit()
	c.Edit("demo")
	c.OpenForm()
	older := *c.Snapshot().Form
	newer := older
	newer.Command = "newer"
	c.SetForm(newer)
	n = len(f.calls)
	c.SaveFormExpected(ctx, f, older, "demo")
	if len(f.calls) != n {
		t.Fatal("queued stale form reached host")
	}
	f.entered = make(chan struct{})
	f.release = make(chan struct{})
	done := make(chan struct{})
	go func() { c.SaveFormExpected(ctx, f, newer, "demo"); close(done) }()
	<-f.entered
	newest := newer
	newest.Command = "newest"
	c.SetForm(newest)
	n = len(f.calls)
	c.SaveFormExpected(ctx, f, newest, "demo")
	if len(f.calls) != n {
		t.Fatal("duplicate save while pending")
	}
	close(f.release)
	<-done
	if c.Snapshot().Form == nil || c.Snapshot().Form.Command != "newest" {
		t.Fatal("completed save erased newer form")
	}
	c.CancelEdit()
	c.Edit("readonly")
	if c.Snapshot().EditingID != "" {
		t.Fatal("read-only form target opened")
	}
	c.SetDraft(`{"id":"bad","transport":"stdio","command":"fixture","env":{"PLAIN":"line\nbreak"}}`)
	c.OpenForm()
	if c.Snapshot().Form != nil || c.Snapshot().Err == "" {
		t.Fatal("unrepresentable map silently changed")
	}
}
