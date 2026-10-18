package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const roomTextFixtureJSON = `{"id":"room","kind":"room","sessionId":"canonical-room","name":"Fixture group","title":"","description":"preserve full description","workspaceId":"workspace-1","sectionId":"section-1","pinned":false,"hidden":false,"pinOrder":0,"readInitialized":true,"lastSeenSeq":4,"createdAt":1,"updatedAt":2,"objective":"","objectiveMaxRounds":16,"memoryReview":true,"notify":true,"allowedSenderIds":[],"memberBotIds":["a","b"],"maxRounds":3,"maxSpeaks":10,"avatar":{"kind":"blob","shape":"circle","color":"blue","crop":"","dataUrl":""},"model":{"provider":"","model":"","reasoningEffort":""},"capabilities":{"tools":{"mode":"all","names":[],"future":{"guard":true}},"skills":{"mode":"all","names":[]},"mcp":{"mode":"all","names":[]},"extra":{"nested":[1,2]}},"holds":{},"inbox":[],"futureTop":{"retain":{"bytes":"value"}}}`

func roomTextFixtureStatus() pluginrpc.DshbotRoomEditStatus {
	var p map[string]json.RawMessage
	json.Unmarshal([]byte(roomTextFixtureJSON), &p)
	var row pluginrpc.DshbotRoomItem
	json.Unmarshal([]byte(roomTextFixtureJSON), &row)
	return pluginrpc.DshbotRoomEditStatus{Profile: p, Item: row, Sections: []string{"section-1"}, Catalog: pluginrpc.DshbotRoomCatalog{Revision: 7, Items: []pluginrpc.DshbotRoomItem{row, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", Kind: "bot", SessionID: "sa"}}, {DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "sb"}}}}}
}

func TestRoomTextUICompleteReviewAndExplicitImpact(t *testing.T) {
	s := roomTextFixtureStatus()
	cmd := pluginrpc.DshbotRoomEditCommand{Expected: s, Values: pluginrpc.DshbotRoomEditValues{MemberBotIDs: s.Item.MemberBotIDs, MaxRounds: s.Item.MaxRounds, MaxSpeaks: s.Item.MaxSpeaks, Text: &pluginrpc.DshbotRoomTextValues{Name: "Reviewed new group", Description: "First new line\nSecond new line"}}}
	state := dshbotstore.RoomEditSnapshot{Scope: "room", Loaded: true, Writable: true, Status: s, Draft: &dshbotstore.RoomEditDraft{Name: cmd.Values.Text.Name, Description: cmd.Values.Text.Description, MemberBotIDs: s.Item.MemberBotIDs, MaxRounds: "3", MaxSpeaks: "10"}, Review: &dshbotstore.RoomEditReview{Command: cmd}, Warnings: []string{"complete rename warning"}}
	saves := 0
	tt := ui.NewTester(func(c *ui.Context) {
		roomEditView(c, RoomEditModel{State: state, HostLabel: "room-host", OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}, false, ui.RGB(255, 255, 255), ui.RGB(120, 120, 120), ui.RGB(40, 40, 40), ui.RGB(80, 80, 80), ui.RGB(70, 130, 210))
	}, 1200, 7000)
	text := strings.Join(tt.Texts(), "\n")
	for _, s := range []string{"Reviewed new group", "preserve full description", "First new line", "Second new line", "canonical-room", "revision 7", "没有独立 title", "presentation", "rename", "system prompt", "requestSync", "不自动打开", "complete rename warning"} {
		if !strings.Contains(text, s) {
			t.Fatal("missing room text/effect", s)
		}
	}
	tt.Click(roomEditConfirmLabel)
	if saves != 0 {
		t.Fatal("render/unconfirmed save")
	}
	if e := tt.Click(roomEditAcknowledgement); e != nil {
		t.Fatal(e)
	}
	if e := tt.Click(roomEditConfirmLabel); e != nil {
		t.Fatal(e)
	}
	if saves != 1 {
		t.Fatal("explicit save failed")
	}
}
