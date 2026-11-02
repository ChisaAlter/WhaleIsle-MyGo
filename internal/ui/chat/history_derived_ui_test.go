package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func historyDerivedChatFixture() session.ViewState {
	first, end := 1000.0, 1600.0
	v := session.ViewState{SelectedID: "memo-session", PlanHistoryBinding: "host-a/follow", Messages: []session.Message{{Seq: 1, Raw: json.RawMessage(`{"type":"turn/start","seq":1,"data":{"turn":1}}`)}, {Seq: 2, Raw: json.RawMessage(`{"type":"tool/call","seq":2,"data":{"turn":1,"callId":"schedule-1","name":"schedule_create"}}`)}}}
	receipt := `{"id":"task-1","title":"Fixture task","prompt":"Check fixture","kind":"after","scheduledAt":"2030-01-02T03:04:05Z","afterSeconds":60}`
	raw, _ := json.Marshal(map[string]any{"type": "tool/result", "seq": 3, "data": map[string]any{"turn": 1, "message": map[string]any{"source": map[string]string{"callId": "schedule-1"}, "content": []map[string]string{{"type": "text", "text": receipt}}}}})
	v.Messages = append(v.Messages, session.Message{Seq: 3, Raw: raw}, session.Message{Seq: 4, Raw: json.RawMessage(`{"type":"assistant/message","seq":4,"data":{"turn":1,"usage":{"outputTokens":300}}}`), TrajectoryTime: &session.TrajectoryTiming{FirstTokenAt: &first, CompletedAt: &end}}, session.Message{Seq: 5, Raw: json.RawMessage(`{"type":"turn/end","seq":5,"data":{"turn":1}}`)})
	return v
}

func TestHistoryDerivedChatScheduleContextOwnsSourceAndReceipts(t *testing.T) {
	v := historyDerivedChatFixture()
	var cards map[float64][]schedstore.Entry
	var owner *session.HistoryDerivedMemo
	var firstOwner *session.HistoryDerivedMemo
	tt := ui.NewTester(func(c *ui.Context) {
		root := ui.Column(c).Fill()
		owner = ui.Local(root, "history-derived", func() session.HistoryDerivedMemo { return session.HistoryDerivedMemo{} })
		if firstOwner == nil {
			firstOwner = owner
		}
		cards = prepareScheduleContext(Model{Snapshot: v}, owner)
	}, 640, 360)
	for range 3 {
		tt.Frame()
	}
	if owner != firstOwner {
		t.Fatal("UI owner did not survive frames")
	}
	if !reflect.DeepEqual(cards, session.ScheduleCards(v.Messages, v.SelectedID)) {
		t.Fatal("schedule context differs from pure oracle")
	}
	cards[5][0].Title = "external mutation"
	cards[5][0].Raw[0] = '!'
	tt.Frame()
	if cards[5][0].Title != "Fixture task" || cards[5][0].Raw[0] != '{' {
		t.Fatal("external card edits contaminated later UI read")
	}
	v.Messages[2].Streaming = true
	tt.Frame()
	if len(cards) != 0 {
		t.Fatal("streaming result retained a completed schedule receipt")
	}
	v.Messages[2].Streaming = false
	v.SelectedID = "same-history-new-session"
	v.PlanHistoryBinding = "host-b/follow"
	tt.Frame()
	if len(cards[5]) != 1 || cards[5][0].SessionID != v.SelectedID {
		t.Fatal("new owner retained old receipt identity")
	}
}

func TestHistoryDerivedChatPerformanceChangesWithoutReplacingSlice(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	v := historyDerivedChatFixture()
	tt := ui.NewTester(func(c *ui.Context) { PerformanceUsageView(c, v, "compact", c.Theme().TextMuted) }, 640, 160)
	assertSpeed := func(speed float64) {
		t.Helper()
		if !tt.HasText(fmt.Sprintf(i18n.T(i18n.KeyChatStatusSpeed), speed)) {
			t.Fatal("current performance not rendered", tt.Texts())
		}
	}
	assertSpeed(500)
	for range 3 {
		tt.Frame()
		assertSpeed(500)
	}
	raw := v.Messages[3].Raw
	at := bytes.Index(raw, []byte("300"))
	copy(raw[at:at+3], "600")
	tt.Frame()
	assertSpeed(1000)
	*v.Messages[3].TrajectoryTime.CompletedAt = 2200
	tt.Frame()
	assertSpeed(500)
	v.Messages[3].Streaming = true
	tt.Frame()
	if len(tt.Texts()) != 0 {
		t.Fatal("untimed streaming row retained performance", tt.Texts())
	}
}
