package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestRoomRuntimeCompleteUIExplicitOpenFixture(t *testing.T) {
	state := dshbotstore.RoomRuntimeSnapshot{Scope: "fixture", Loaded: true, Data: pluginrpc.DshbotRoomRuntime{Room: pluginrpc.DshbotRoomItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "room", Name: "Full room", SessionID: "room-session"}}, Catalog: pluginrpc.DshbotRoomCatalog{Revision: 9}, Raw: json.RawMessage(`{"futureRoot":"root extension"}`), Members: []pluginrpc.DshbotRoomRuntimeMember{{Name: "Alpha", BotID: "a", RoomSessionID: "room-session", SessionID: "member-session", Status: "pending", LastError: json.RawMessage(`{"code":"EXTERNAL_UNKNOWN","message":"full failure"}`), Pending: json.RawMessage(`{"kind":"approval","requestId":"approval-1"}`), Raw: json.RawMessage(`{"futureMember":"all details"}`)}}}}
	opens, reads := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		roomRuntimeView(c, RoomRuntimeModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnOpen: func(id string) {
			if id != "member-session" {
				t.Fatal("wrong member")
			}
			opens++
		}}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 2600)
	if opens != 0 || reads != 0 {
		t.Fatal("render activated runtime")
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"Full room", "room-session", "Alpha", "member-session", "pending", "EXTERNAL_UNKNOWN", "full failure", "approval-1", "all details", "root extension", "或自动回答", "可能恢复", i18n.T(i18n.KeyMiscAssistantNeedsAttention)} {
		if !strings.Contains(text, term) {
			t.Fatal("missing complete runtime or effect", term)
		}
	}
	if err := tt.Click(roomRuntimeOpenLabel()); err != nil {
		t.Fatal(err)
	}
	if opens != 1 {
		t.Fatal("explicit open missing")
	}
	state.Loaded = false
	state.Error = "group/runtime failed"
	tt.Frame()
	tt.Click(roomRuntimeOpenLabel())
	if opens != 1 {
		t.Fatal("failed stale runtime opened")
	}
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "group/runtime failed") {
		t.Fatal("runtime failure hidden")
	}
}
