package pluginmanager

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNativeCancelledBatchSettlesSubmittedPrefixAndKeepsRemainingUnsent(t *testing.T) {
	op := action{kind: "install", mcpSpecs: []json.RawMessage{json.RawMessage(`{"id":"one","serverName":"one","transport":"stdio","command":"node","enabled":false}`), json.RawMessage(`{"id":"two","serverName":"two","transport":"stdio","command":"node","enabled":false}`)}}
	op.progress = newNativeProgress(op)
	op.progress.startNext()
	op.progress.completeNext()
	reads := 0
	client := Client{fixtureCall(func(_ context.Context, method string, _, out any) error {
		if method != "mcpServers/list" {
			t.Fatalf("reconciliation wrote or queried unsubmitted item: %s", method)
		}
		reads++
		return reply(out, map[string]any{"servers": []any{map[string]any{"id": "one", "spec": op.mcpSpecs[0]}}})
	})}
	result, err := client.reconcileNative(context.Background(), op)
	if err != nil || result.Application != "failed" || result.Stage != "partial" || !result.Changed || len(result.Warnings) != 2 || result.Warnings[1] != "MCP two: not submitted" || reads != 1 {
		t.Fatalf("partial outcome: %+v %v reads=%d", result, err, reads)
	}
}

func TestNativeUnconfirmedReviewCanFinishWithoutReplayingWrites(t *testing.T) {
	c := New()
	c.Attach(basicFixture(nil), nil)
	defer c.Detach()
	op := action{kind: "install", owner: c.state.Owner, options: InstallOptions{RequestID: "owned"}, mcpSpecs: []json.RawMessage{json.RawMessage(`{"id":"one"}`)}}
	op.progress = newNativeProgress(op)
	op.progress.startNext()
	c.nativeInstall = &op
	c.state.Install = InstallState{Open: true, RequestID: "owned", Phase: "unconfirmed", NativeCanFinish: true}
	a := c.Actions(op.owner)
	a.FinishNativeReview()
	state := c.Snapshot().Value
	if state.Install.Phase != "failed" || state.Install.Result == nil || state.Install.Result.Error.Code != "native-install/unconfirmed" || state.Install.Result.Warnings[0] != "MCP one: result unconfirmed" {
		t.Fatalf("uncertainty lost: %+v", state.Install)
	}
	a.CloseInstall()
	if c.Snapshot().Value.Install.RequestID != "" {
		t.Fatal("finished native review still locked installer")
	}
}
