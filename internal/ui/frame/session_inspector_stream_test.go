package frame

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func inspectorStreamRows(state string) []session.InspectorRecord {
	return []session.InspectorRecord{
		{Key: "turn", Type: "turn/start", Disclosure: "open"},
		{Key: "step", Parent: "turn", Type: "step/start", Disclosure: "open"},
		{Key: "assistant", Parent: "step", Type: "assistant/live"},
		{Key: "block", Parent: "assistant", Type: "block-start", Identity: "0", Disclosure: state, CollapsedSummary: "blockType: reasoning · first second", Raw: json.RawMessage(`{"type":"block-start","index":0,"blockType":"reasoning"}`)},
		{Key: "delta", Parent: "block", Type: "reasoning-delta", Identity: "1", Raw: json.RawMessage(`{"type":"reasoning-delta","index":0,"text":"first second"}`)},
		{Key: "finish", Parent: "assistant", Type: "finish", Identity: "2", Raw: json.RawMessage(`{"type":"finish","reason":{"kind":"stop"}}`)},
	}
}

func TestInspectorStreamDisclosureChoicesFollowLifecycle(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	rows := inspectorStreamRows("open")
	p := SessionInspectorPaneState{}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 900, 600)
	if !tt.HasText("reasoning-delta") || !tt.HasText("finish") {
		t.Fatal("unfinished block was not expanded")
	}
	if err := tt.Click("Collapse children 0"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("reasoning-delta") || !tt.HasText("blockType: reasoning · first second") || !tt.HasText("finish") {
		t.Fatal("manual reasoning fold lost its summary or swallowed finish")
	}
	if err := tt.Click("Expand children 0"); err != nil {
		t.Fatal(err)
	}
	if p.collapsed["block"].state != "open" || p.collapsed["block"].collapsed {
		t.Fatal("choice did not record the current block lifecycle")
	}
	rows[3].Disclosure = "closed"
	tt.Frame()
	if tt.HasText("reasoning-delta") || !tt.HasText("blockType: reasoning · first second") {
		t.Fatal("completed block reused its earlier open-state override")
	}
	if err := tt.Click("Expand children 0"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("reasoning-delta") || p.collapsed["block"].state != "closed" || p.collapsed["block"].collapsed {
		t.Fatal("manual completed-block expansion was not retained")
	}
	if err := tt.Click("Collapse children turn"); err != nil {
		t.Fatal(err)
	}
	rows = append(rows, session.InspectorRecord{Key: "turn-end", Parent: "turn", Type: "turn/end"})
	tt.Frame()
	if tt.HasText("step/start") || tt.HasText("turn/end") {
		t.Fatal("Turn end reset a manual fold of the unchanged open Turn header")
	}
}

func TestInspectorFilteredStreamUsesFilteredDisclosureState(t *testing.T) {
	rows := inspectorStreamRows("closed")
	p := SessionInspectorPaneState{}
	p.applyInspectorFilter("reasoning-delta")
	filtered := inspectorTypeRows(rows, p.Filter)
	if len(filtered) != 5 || filtered[3].Disclosure != "open" || len(inspectorDisclosedRows(filtered, p.collapsed)) != 5 {
		t.Fatal("filter failed to reveal the matching closed block")
	}
	p.setInspectorCollapsed(filtered[3], true)
	rows[3].Disclosure = "open"
	if len(inspectorVisibleRows(rows, p.Filter, p.collapsed)) != 4 {
		t.Fatal("source lifecycle invalidated the filtered open-state manual fold")
	}
	rows[3].Disclosure = "closed"
	if len(inspectorVisibleRows(rows, p.Filter, p.collapsed)) != 4 {
		t.Fatal("completed source block ignored filtered manual fold")
	}
	p.applyInspectorFilter("reasoning-delta")
	if len(inspectorVisibleRows(rows, p.Filter, p.collapsed)) != 5 {
		t.Fatal("same-query confirmation failed to reset manual filtered fold")
	}
	p.applyInspectorFilter("block-start")
	filtered = inspectorTypeRows(rows, p.Filter)
	_, branches := inspectorDepths(filtered)
	if filtered[3].Disclosure != "" || branches["block"] {
		t.Fatal("matching block with excluded descendants retained an empty lifecycle arrow")
	}
	p.applyInspectorFilter("")
	if len(inspectorVisibleRows(rows, p.Filter, p.collapsed)) != 5 {
		t.Fatal("clearing filter did not restore closed lifecycle default")
	}
	if !p.revealInspectorRow("delta", rows) || len(inspectorVisibleRows(rows, p.Filter, p.collapsed)) != 6 || p.collapsed["block"].state != "closed" {
		t.Fatal("reveal did not explicitly open its closed block ancestor")
	}
}

func TestInspectorLiveRowsNavigateByActualChatIdentity(t *testing.T) {
	rows := inspectorStreamRows("open")
	for i := range rows {
		if rows[i].Key == "assistant" || rows[i].Key == "block" || rows[i].Key == "delta" {
			rows[i].ChatID, rows[i].ChatPickID = "actual-live-message", "actual-live-message"
		}
	}
	p := SessionInspectorPaneState{}
	p.selectInspectorRow(rows[4])
	if p.navigation.Target == nil || p.navigation.Target.ID != "actual-live-message" {
		t.Fatal("raw live chunk did not target its own native Chat block")
	}
	empty := session.InspectorRecord{Key: "empty-header", Type: "block-start", ChatID: "actual-live-message"}
	rows = append([]session.InspectorRecord{empty}, rows...)
	if got := inspectorPickRecord(rows, chat.InspectorTarget{ID: "actual-live-message"}); got != "block" {
		t.Fatalf("forward owner fallback hijacked exact reverse pick: %s", got)
	}
	rows = append(rows, session.InspectorRecord{Key: "event:9", Type: "assistant/message", Seq: 9})
	rows[3].Seq = 9
	if got := inspectorPickRecord(rows, chat.InspectorTarget{Seq: 9}); got != "event:9" {
		t.Fatalf("canonical message pick chose retained live prefix: %s", got)
	}
	if got := inspectorPickRecord(rows, chat.InspectorTarget{ID: "actual-live-message"}); got != "block" {
		t.Fatalf("live pick did not prefer its block header: %s", got)
	}
}

func TestInspectorSettledAttemptStillOwnsDataAndChatReference(t *testing.T) {
	source := session.InspectorSnapshot{Records: []session.InspectorRecord{
		{Key: "turn", Type: "turn/start", Seq: 1},
		{Key: "step", Type: "step/start", Seq: 2},
		{Key: "attempt:stable", Parent: "step", Type: "assistant/attempt", Seq: 3, Raw: json.RawMessage(`{"type":"assistant/attempt","seq":3,"data":{"turn":1,"step":0,"stream":[]}}`)},
		{Key: "attempt:stable/chunk:0", Parent: "attempt:stable", Type: "finish", Seq: 3, Raw: json.RawMessage(`{"type":"finish"}`)},
	}, Chat: []session.InspectorRecord{{Key: "chat:actual", Seq: 3, Type: "text"}}}
	objects := inspectorObjects(source)
	if _, ok := objects["attempt:stable/data"]; !ok || len(objects["chat:actual"].Links) != 3 || objects["chat:actual"].Links[0] != "attempt:stable" {
		t.Fatal("settled attempt alias lost root/Data references or chunk replaced its owning event")
	}
}

func TestInspectorEmptyStreamBlockStillOwnsDisclosure(t *testing.T) {
	rows := []session.InspectorRecord{{Key: "block", Type: "block-start", Disclosure: "open", CollapsedSummary: "blockType: reasoning"}}
	_, branches := inspectorDepths(rows)
	if !branches["block"] {
		t.Fatal("fresh block-start lost its lifecycle disclosure before the first delta")
	}
	filtered := inspectorTypeRows(rows, "block-start")
	_, branches = inspectorDepths(filtered)
	if branches["block"] {
		t.Fatal("filter retained an empty branch without matching descendants")
	}
}
