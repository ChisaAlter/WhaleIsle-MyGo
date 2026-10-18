package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestChildVisibleImageRequests(t *testing.T) {
	calls := []string{}
	reads := map[string]session.AgentResourceRead{}
	opts := ReadOnlyResourceOptions{Scope: "owner/child/token", Reads: reads, OnRead: func(ref string, more bool) {
		if more {
			t.Error("unexpected pagination")
		}
		calls = append(calls, ref)
	}}
	requests := childImageRequests{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Height(120).Width(400).Children(func() {
			ui.Column(c).Children(func() {
				childAutoImages(c, session.Message{Body: "[image] attachment:visible\n[image] https://invalid.test/no.png"}, opts, requests, ToolCardStyle{})
				ui.Box(c).Height(600)
				childAutoImages(c, session.Message{Body: "[image] attachment:offscreen"}, opts, requests, ToolCardStyle{})
			})
		})
	}, 500, 200)
	tt.Frame()
	if len(calls) != 1 || calls[0] != "attachment:visible" {
		t.Fatalf("visible-only requests: %v", calls)
	}
	tt.Frame()
	if len(calls) != 1 {
		t.Fatal("repeated paint duplicated request")
	}
	reads["attachment:visible"] = session.AgentResourceRead{Error: "synthetic failure"}
	tt.Frame()
	if len(calls) != 1 {
		t.Fatal("failure retried automatically")
	}
	requests.request("attachment:second", opts)
	requests.request("attachment:third", opts)
	requests.request("attachment:fourth", opts)
	if len(calls) != 3 {
		t.Fatal("concurrency budget not enforced", calls)
	}
	if childImageVisible(ui.Rect{W: 10, H: 10}, ui.Rect{}) {
		t.Fatal("modal-blocked clip admitted")
	}
	capped := childImageRequests{}
	for i := 0; i < 32; i++ {
		capped[string(rune(i+1))] = true
	}
	capped.request("attachment:limit", opts)
	if len(calls) != 3 {
		t.Fatal("total budget exceeded")
	}
}
