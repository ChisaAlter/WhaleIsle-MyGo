package mcpconfig

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"whaleisle/internal/dsh/api"
)

type healthFixtureReply struct {
	rows []api.McpServerSnapshotServersItem
	err  error
}

type healthFixtureRequest struct {
	ctx    context.Context
	method string
	args   any
	reply  chan healthFixtureReply
}

type healthFixtureCaller struct {
	calls chan healthFixtureRequest
	abort chan struct{}
}

// Deliberately allow a reply after cancellation to prove late-result rejection.
func (f *healthFixtureCaller) Call(ctx context.Context, method string, args, out any) error {
	request := healthFixtureRequest{ctx: ctx, method: method, args: args, reply: make(chan healthFixtureReply, 1)}
	select {
	case f.calls <- request:
	case <-f.abort:
		return context.Canceled
	}
	var reply healthFixtureReply
	select {
	case reply = <-request.reply:
	case <-f.abort:
		return context.Canceled
	}
	if reply.err != nil {
		return reply.err
	}
	switch method {
	case "mcpServers/list":
		out.(*api.McpServerSnapshot).Servers = reply.rows
	case "mcpServers/setEnabled":
		*out.(*json.RawMessage) = json.RawMessage(`{"ok":true}`)
	default:
		return errors.New("unexpected fixture RPC: " + method)
	}
	return nil
}

func healthFixtureRows(phase string, enabled bool) []api.McpServerSnapshotServersItem {
	return []api.McpServerSnapshotServersItem{{
		Id: "fixture", Enabled: enabled, Writable: true, FiberPhase: phase,
		Spec:       json.RawMessage(`{"id":"fixture","serverName":"Fixture","transport":"stdio","command":"synthetic-never-executed"}`),
		Connection: &api.McpServerSnapshotServersItemConnection{Health: phase, Tools: []string{"fixture_tool"}},
	}}
}

func TestMCPHealthRefreshFixture(t *testing.T) {
	if HealthRefreshInterval != 2*time.Second {
		t.Fatal("original polling interval changed")
	}
	f := &healthFixtureCaller{calls: make(chan healthFixtureRequest, 32), abort: make(chan struct{})}
	t.Cleanup(func() { close(f.abort) })
	notify := make(chan struct{}, 32)
	wait := func(done <-chan struct{}) {
		t.Helper()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("fixture synchronization timed out")
		}
	}
	next := func(method string) healthFixtureRequest {
		t.Helper()
		select {
		case request := <-f.calls:
			if request.method != method || (method == "mcpServers/list" && request.args != nil) {
				t.Fatalf("unexpected request: %s %#v", request.method, request.args)
			}
			return request
		case <-time.After(5 * time.Second):
			t.Fatal("fixture did not receive expected RPC")
			return healthFixtureRequest{}
		}
	}
	start := func(c *Controller) (chan time.Time, *healthRefreshLoop) {
		t.Helper()
		ticks := make(chan time.Time)
		c.observeHealthRefresh(true, f, func() { notify <- struct{}{} }, func() (<-chan time.Time, func()) {
			return ticks, func() {}
		})
		c.mu.Lock()
		loop := c.healthLoop
		c.mu.Unlock()
		if loop == nil {
			t.Fatal("ready visible controller did not start")
		}
		t.Cleanup(c.StopHealthRefresh)
		return ticks, loop
	}
	tick := func(ticks chan time.Time) healthFixtureRequest {
		t.Helper()
		select {
		case ticks <- time.Time{}:
		case <-time.After(5 * time.Second):
			t.Fatal("fixture loop did not accept tick")
		}
		return next("mcpServers/list")
	}

	// No initial loading or hidden/offline polling, including no timer creation.
	var unloaded Controller
	factory := func() (<-chan time.Time, func()) { t.Fatal("ineligible timer created"); return nil, nil }
	unloaded.observeHealthRefresh(true, f, nil, factory)
	unloaded.state.Ready = true
	unloaded.observeHealthRefresh(false, f, nil, factory)
	unloaded.observeHealthRefresh(true, nil, nil, factory)

	rows := healthFixtureRows("connecting", true)
	c := &Controller{state: Snapshot{
		Ready: true, Rows: rows, Draft: "unsaved JSON", EditingID: "fixture", DeletingID: "fixture",
		Form: &FormDraft{ID: "fixture", Command: "unsaved command"}, FormErrors: map[string]string{"command": "unsaved validation"},
		Import: ImportState{Text: "unapplied pasted JSON", Notice: "previous import result"},
		Query:  "fixture", EnabledFilter: "enabled", ExpandedID: "fixture", Err: "explicit action error", LastAction: "explicit action result",
		Tests: map[string]TestDiagnostic{"fixture": {Err: "prior explicit test failure", spec: string(rows[0].Spec)}},
	}}
	before := c.Snapshot()
	ticks, loop := start(c)
	c.observeHealthRefresh(true, f, nil, factory) // repeated frame is idempotent
	request := tick(ticks)
	if c.Snapshot().Busy || c.refreshHealth(loop, f) {
		t.Fatal("background read blocked actions or started overlapping list")
	}
	request.reply <- healthFixtureReply{rows: healthFixtureRows("connected", true)}
	wait(notify)
	after := c.Snapshot()
	if after.Rows[0].FiberPhase != "connected" || after.Draft != before.Draft || after.EditingID != before.EditingID || after.DeletingID != before.DeletingID || !reflect.DeepEqual(after.Form, before.Form) || !reflect.DeepEqual(after.FormErrors, before.FormErrors) || !reflect.DeepEqual(after.Import, before.Import) || after.Err != before.Err || after.LastAction != before.LastAction || !reflect.DeepEqual(after.Tests, before.Tests) || after.Query != before.Query || after.EnabledFilter != before.EnabledFilter || after.ExpandedID != before.ExpandedID {
		t.Fatal("passive refresh modified editor, import, filter, or explicit action state")
	}
	request = tick(ticks)
	request.reply <- healthFixtureReply{err: errors.New("synthetic list failure")}
	wait(notify)
	after = c.Snapshot()
	if !after.HealthRefreshFailed || after.Rows[0].FiberPhase != "connected" || after.Err != before.Err || after.LastAction != before.LastAction {
		t.Fatal("passive failure erased rows or explicit feedback")
	}
	request = tick(ticks)
	request.reply <- healthFixtureReply{rows: healthFixtureRows("reconnecting", true)}
	wait(notify)
	if c.Snapshot().HealthRefreshFailed {
		t.Fatal("successful later list retained stale refresh failure")
	}

	// An explicit action cancels the earlier read, wins over its late reply,
	// and suppresses ticks until both the operation and its own reload finish.
	stale := tick(ticks)
	operationDone := make(chan struct{})
	go func() { c.SetEnabled(context.Background(), f, "fixture", false); close(operationDone) }()
	operation := next("mcpServers/setEnabled")
	wait(stale.ctx.Done())
	if !c.Snapshot().Busy || c.refreshHealth(loop, f) {
		t.Fatal("poll ran during explicit action")
	}
	operation.reply <- healthFixtureReply{}
	reload := next("mcpServers/list")
	if c.refreshHealth(loop, f) {
		t.Fatal("poll ran during action reload")
	}
	reload.reply <- healthFixtureReply{rows: healthFixtureRows("disabled", false)}
	wait(operationDone)
	stale.reply <- healthFixtureReply{rows: healthFixtureRows("stale-pre-write", true)}
	request = tick(ticks) // receiving this proves the stale reply was processed
	if c.Snapshot().Rows[0].FiberPhase != "disabled" || c.Snapshot().LastAction != "setEnabled: fixture" {
		t.Fatal("old poll overwrote successful operation")
	}
	c.SetDraft("newer draft during list")
	request.reply <- healthFixtureReply{rows: healthFixtureRows("current", false)}
	wait(notify)
	if c.Snapshot().Draft != "newer draft during list" || c.Snapshot().LastAction != "setEnabled: fixture" {
		t.Fatal("poll overwrote a newer draft or successful action feedback")
	}

	// Hide/reopen while a canceled caller has not returned: no second read slot.
	stale = tick(ticks)
	c.ObserveHealthRefresh(false, f, nil)
	wait(stale.ctx.Done())
	newTicks, newLoop := start(c)
	if c.refreshHealth(newLoop, f) {
		t.Fatal("reopening overlapped a canceled list still returning")
	}
	stale.reply <- healthFixtureReply{rows: healthFixtureRows("stale-hidden", true)}
	wait(loop.done)
	if c.Snapshot().Rows[0].FiberPhase != "current" {
		t.Fatal("hidden result published")
	}
	request = tick(newTicks)
	request.reply <- healthFixtureReply{rows: healthFixtureRows("restored", false)}
	wait(notify)

	// Host replacement uses a fresh controller and cancels the old captured Host.
	stale = tick(newTicks)
	c.StopHealthRefresh()
	wait(stale.ctx.Done())
	replacement := &Controller{state: Snapshot{Ready: true, Rows: healthFixtureRows("new-host", true)}}
	replacementTicks, replacementLoop := start(replacement)
	request = tick(replacementTicks)
	request.reply <- healthFixtureReply{rows: healthFixtureRows("new-host-connected", true)}
	wait(notify)
	stale.reply <- healthFixtureReply{rows: healthFixtureRows("old-host-late", true)}
	wait(newLoop.done)
	if c.Snapshot().Rows[0].FiberPhase != "restored" || replacement.Snapshot().Rows[0].FiberPhase != "new-host-connected" {
		t.Fatal("Host cancellation or isolation failed")
	}
	replacement.StopHealthRefresh()
	wait(replacementLoop.done)
	select {
	case extra := <-f.calls:
		t.Fatalf("unexpected extra RPC: %s", extra.method)
	default:
	}
}
