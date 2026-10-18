package assistant

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestRoomCreateUICompleteReviewAndExplicitConfirmation(t *testing.T) {
	cmd := pluginrpc.DshbotRoomCreateCommand{ID: "room-fixture", SessionID: "canonical-fixture", Expected: pluginrpc.DshbotRoomCatalog{Revision: 12, Items: []pluginrpc.DshbotRoomItem{{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", Name: "Alpha", SessionID: "sa"}}, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", Name: "Beta", SessionID: "sb"}}}}, Draft: pluginrpc.DshbotRoomDraft{Name: "Full group name", Description: "Full group description", WorkspaceID: "workspace-fixture", ScratchCwd: "/fixture/cwd", MemberBotIDs: []string{"a", "b"}, MaxRounds: 2, MaxSpeaks: 7}}
	state := dshbotstore.RoomCreateSnapshot{Scope: "fixture", Loaded: true, Writable: true, Catalog: cmd.Expected, Draft: &dshbotstore.RoomCreateDraft{}, Review: &dshbotstore.RoomCreateReview{Command: cmd}}
	saves := 0
	tt := ui.NewTester(func(c *ui.Context) {
		roomCreateView(c, RoomCreateModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() {}, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 2800)
	if saves != 0 {
		t.Fatal("render created a room")
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "Full group name", "Full group description", "workspace-fixture", "/fixture/cwd", "canonical-fixture", "revision 12", "Alpha", "sa", "Beta", "sb", "2", "7", "模型", "费用", "不自动重试", "会话"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing reviewed value or effect", term)
		}
	}
	tt.Click(roomCreateConfirmLabel)
	if saves != 0 {
		t.Fatal("unacknowledged creation")
	}
	if err := tt.Click(roomCreateAcknowledgement); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(roomCreateConfirmLabel); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit create missing")
	}
}
