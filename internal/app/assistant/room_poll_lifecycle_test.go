package assistantctl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
)

func TestRoomPollUnmountCancelsOwnedReadersAndTimers(t *testing.T) {
	svc := NewService(HostDeps{})
	groupContext, cancelGroup := context.WithCancel(context.Background())
	contactsContext, cancelContacts := context.WithCancel(context.Background())
	defer cancelGroup()
	defer cancelContacts()
	groupTimer, contactsTimer := time.NewTimer(time.Hour), time.NewTimer(time.Hour)
	defer groupTimer.Stop()
	defer contactsTimer.Stop()

	live := &svc.assistantGroupLive
	live.scope, live.visible, live.rendered, live.busy, live.loaded = "group-owned", true, true, true, true
	live.cancel, live.timer = cancelGroup, groupTimer
	live.contacts.scope, live.contacts.visible, live.contacts.rendered = "page-owned", true, true
	live.contacts.busy, live.contacts.loaded = true, true
	live.contacts.cancel, live.contacts.timer = cancelContacts, contactsTimer
	live.contacts.rows = map[string]contactRoomRuntime{"room": {err: "previous read error"}}
	groupGeneration, contactsGeneration := live.generation, live.contacts.generation

	svc.BeginGroupRuntimeFrame()
	svc.FinishGroupRuntimeFrame() // Neither original page painted.

	if groupContext.Err() != context.Canceled || contactsContext.Err() != context.Canceled {
		t.Fatal("hidden page retained an owned read")
	}
	if groupTimer.Stop() || contactsTimer.Stop() {
		t.Fatal("hidden page retained a scheduled repaint")
	}
	if live.visible || live.busy || live.loaded || live.scope != "" || live.contacts.visible || live.contacts.busy || live.contacts.loaded || live.contacts.scope != "" || len(live.contacts.rows) != 0 {
		t.Fatal("unmounted runtime state can be reused as current")
	}
	if live.generation == groupGeneration || live.contacts.generation == contactsGeneration {
		t.Fatal("old results were not retired on unmount")
	}
}

func TestRoomPollFrameKeepsOnlyThePageThatActuallyRendered(t *testing.T) {
	for _, keepGroup := range []bool{false, true} {
		t.Run(map[bool]string{true: "group", false: "bot-page"}[keepGroup], func(t *testing.T) {
			svc := NewService(HostDeps{})
			groupContext, cancelGroup := context.WithCancel(context.Background())
			contactsContext, cancelContacts := context.WithCancel(context.Background())
			defer cancelGroup()
			defer cancelContacts()

			live := &svc.assistantGroupLive
			live.scope, live.visible, live.cancel = "group", true, cancelGroup
			live.contacts.scope, live.contacts.visible, live.contacts.cancel = "page", true, cancelContacts

			svc.BeginGroupRuntimeFrame()
			if keepGroup {
				live.rendered = true
			} else {
				live.contacts.rendered = true
			}
			svc.FinishGroupRuntimeFrame()

			if keepGroup {
				if groupContext.Err() != nil || contactsContext.Err() != context.Canceled {
					t.Fatal("group visibility leaked into Bot page ownership")
				}
			} else if contactsContext.Err() != nil || groupContext.Err() != context.Canceled {
				t.Fatal("Bot page visibility leaked into group ownership")
			}
		})
	}
}

type fakeGroupReplyTarget struct {
	replied []string
}

func (f *fakeGroupReplyTarget) BeginGroupReply(threadID string) bool {
	f.replied = append(f.replied, threadID)
	return true
}

func TestGroupMembersAndBeginGroupReply(t *testing.T) {
	var runtimeCalls atomic.Int32
	memberPair, _ := json.Marshal([]string{"sess-room-1", "bot-a"})
	memberSessID := "dshbot-group-member-v3-" + base64.RawURLEncoding.EncodeToString(memberPair)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID  string `json:"rpcId"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		switch r.URL.Path {
		case "/dshbot/describe":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": req.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"view": map[string]any{
							"ns":       "dsh-bot",
							"revision": 1,
							"value": map[string]any{
								"items": []map[string]any{
									{"id": "bot-a", "kind": "bot", "name": "Alice", "title": "Worker", "sessionId": "sess-a"},
									{"id": "room-1", "kind": "room", "name": "WarRoom", "sessionId": "sess-room-1", "memberBotIds": []string{"bot-a"}},
								},
							},
						},
					},
				},
			})
		case "/dshbot/group/runtime":
			runtimeCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": req.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"roomId":    "room-1",
						"sessionId": "sess-room-1",
						"members": []map[string]any{
							{"roomSessionId": "sess-room-1", "botId": "bot-a", "sessionId": memberSessID, "status": "idle"},
						},
					},
				},
			})
		default:
			http.Error(w, "unexpected "+r.URL.Path, 404)
		}
	}))
	defer srv.Close()

	cfgStore, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := cfgStore.Update(func(c *config.Config) { c.DshbotEnabled = true }); err != nil {
		t.Fatal(err)
	}
	roster, err := dshbotstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			NS:       "dsh-bot",
			Revision: 1,
			Value: pluginrpc.DshbotCatalogValue{
				Items: []pluginrpc.DshbotCatalogItem{
					{ID: "bot-a", Kind: "bot", Name: "Alice", Title: "Worker", SessionID: "sess-a"},
					{ID: "room-1", Kind: "room", Name: "WarRoom", SessionID: "sess-room-1", MemberBotIDs: []string{"bot-a"}},
				},
			},
		},
	}, false, srv.URL)

	sessions := session.New()
	sessions.Select("sess-room-1")
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL, Cookie: "c=1"}}
	ep := harness.Endpoint{Origin: srv.URL, Cookie: "c=1"}

	svc := NewService(HostDeps{
		Config:        func() *config.Store { return cfgStore },
		Sessions:      func() *session.Store { return sessions },
		CurrentClient: func() *dsh.Client { return client },
		BootEndpoint:  func() (harness.Endpoint, harness.State, int) { return ep, harness.Ready, 1 },
		Window:        func() any { return "win" },
		WindowUpdate:  func(_ any, fn func()) { fn() },
	})
	t.Cleanup(svc.SuspendAll)
	svc.AttachStores(nil, nil, roster)

	isRoom, members := svc.GroupMembers("sess-room-1")
	if !isRoom || len(members) != 1 || members[0].ID != "bot-a" {
		t.Fatalf("GroupMembers got (%v, %+v)", isRoom, members)
	}
	if isBotRoom, _ := svc.GroupMembers("sess-a"); isBotRoom {
		t.Fatal("personal bot session must not match as room")
	}

	// Wait briefly for the async room/runtime fetch to converge.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtimeCalls.Load() > 0 {
			_, updated := svc.GroupMembers("sess-room-1")
			if len(updated) == 1 && updated[0].RuntimeStatus == "idle" {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, updated := svc.GroupMembers("sess-room-1")
	if len(updated) != 1 || updated[0].RuntimeStatus != "idle" {
		t.Fatalf("expected runtime status 'idle', got %+v", updated)
	}

	target := &fakeGroupReplyTarget{}
	svc.BeginGroupReply(sessions, target, "other-session", "thread-1")
	if len(target.replied) != 0 {
		t.Fatal("BeginGroupReply acted on non-selected session")
	}
}
