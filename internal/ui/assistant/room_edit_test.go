package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestRoomEditUIFullReviewAndExplicitSave(t *testing.T) {
	cmd := pluginrpc.DshbotRoomEditCommand{Expected: pluginrpc.DshbotRoomEditStatus{Catalog: pluginrpc.DshbotRoomCatalog{Revision: 7, Items: []pluginrpc.DshbotRoomItem{{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", Name: "Alpha", SessionID: "sa"}}, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", Name: "Beta", SessionID: "sb"}}}}, Item: pluginrpc.DshbotRoomItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "room", Kind: "room", Name: "Full room name", SessionID: "canonical-room", WorkspaceID: "workspace-1", MemberBotIDs: []string{"a", "b"}}, MaxRounds: 3, MaxSpeaks: 10}, Profile: map[string]json.RawMessage{"futureTop": json.RawMessage(`{"opaque":"retain all nested fields"}`), "memberBotIds": json.RawMessage(`["a","b"]`), "maxRounds": json.RawMessage(`3`), "maxSpeaks": json.RawMessage(`10`)}}, Values: pluginrpc.DshbotRoomEditValues{MemberBotIDs: []string{"b", "a"}, MaxRounds: 2, MaxSpeaks: 4}}
	state := dshbotstore.RoomEditSnapshot{Scope: "fixture", Loaded: true, Writable: true, Status: cmd.Expected, Draft: &dshbotstore.RoomEditDraft{MemberBotIDs: cmd.Values.MemberBotIDs, MaxRounds: "2", MaxSpeaks: "4"}, Review: &dshbotstore.RoomEditReview{Command: cmd}}
	saves := 0
	tt := ui.NewTester(func(c *ui.Context) {
		roomEditView(c, RoomEditModel{State: state, HostLabel: "https://fixture.invalid", OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 3400)
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "revision 7", "Full room name", "canonical-room", "workspace-1", "Alpha", "sa", "Beta", "sb", "3 → 2", "10 → 4", "Full before", "Full after", "retain all nested fields", "费用", "审计", "不启动或停止"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing review or effect", term)
		}
	}
	tt.Click(roomEditConfirmLabel)
	if saves != 0 {
		t.Fatal("saved without acknowledgment")
	}
	if err := tt.Click(roomEditAcknowledgement); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(roomEditConfirmLabel); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("missing explicit save")
	}
}
