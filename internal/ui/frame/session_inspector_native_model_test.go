package frame

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func inspectorNativeFixture() (session.InspectorSnapshot, chat.InspectorNavigation) {
	raw := []string{
		`{"seq":0,"type":"turn/start","data":{"turn":0,"input":"inspect"}}`,
		`{"seq":1,"type":"step/start","data":{"turn":0,"step":0}}`,
		`{"seq":2,"type":"assistant/message","data":{"turn":0,"step":0,"reasoning":"why","large":9007199254740993}}`,
		`{"seq":3,"type":"step/end","data":{"turn":0,"step":0}}`,
		`{"seq":4,"type":"step/start","data":{"turn":0,"step":1}}`,
		`{"seq":5,"type":"assistant/message","data":{"turn":0,"step":1,"text":"answer"}}`,
		`{"seq":6,"type":"step/end","data":{"turn":0,"step":1}}`,
		`{"seq":7,"type":"turn/end","data":{"turn":0}}`,
	}
	source := session.InspectorSnapshot{Binding: "native-test", SessionID: "session"}
	for _, text := range raw {
		var event struct {
			Type string
			Seq  float64
		}
		_ = json.Unmarshal([]byte(text), &event)
		source.Records = append(source.Records, session.InspectorRecord{Key: "event:" + inspectorCoordinate(event.Seq), Type: event.Type, Seq: event.Seq, Raw: json.RawMessage(text)})
	}
	navigation := chat.InspectorNavigation{SessionID: "session", Nodes: []session.Message{
		{ID: "input", Kind: session.MessageKindText, Role: "user", Seq: 0, Raw: json.RawMessage(raw[0])},
		{ID: "reasoning", Kind: session.MessageKindThinking, Role: "assistant", Body: "why", Seq: 2, Raw: json.RawMessage(raw[2])},
		{ID: "response", Kind: session.MessageKindText, Role: "assistant", Body: "answer", Seq: 5, Raw: json.RawMessage(raw[5])},
		{ID: "process", Kind: "transcript-control", Seq: 7, Body: "Process", Raw: json.RawMessage(raw[7])},
	}, Groups: map[string][]string{"process": {"reasoning"}}}
	return source, navigation
}

func TestInspectorNativeChatRowsPreserveGroupOrderIdentityAndLocation(t *testing.T) {
	source, navigation := inspectorNativeFixture()
	rows := inspectorChatRows(source, &navigation)
	var keys []string
	for _, row := range rows {
		keys = append(keys, row.Key)
	}
	if !reflect.DeepEqual(keys, []string{"chat:input", "chat:process", "chat:reasoning", "chat:response"}) {
		t.Fatalf("group order: %v", keys)
	}
	if rows[1].Type != "group" || rows[1].Identity != "process" || rows[1].Disclosure != "open" || rows[1].ChatTarget.GroupKey != "process" || rows[1].Location != "" {
		t.Fatalf("group: %+v", rows[1])
	}
	if rows[2].Parent != "chat:process" || rows[2].Identity != "reasoning" || rows[2].Location != "0/0" || rows[2].ChatTarget.GroupPart != "reasoning" {
		t.Fatalf("member: %+v", rows[2])
	}
	if rows[0].Location != "0" || rows[3].Location != "0/1" || rows[3].Parent != "" {
		t.Fatalf("turn/step positions: %+v", rows)
	}
	zero, two := float64(0), float64(2)
	navigation.Nodes = append(navigation.Nodes, session.Message{ID: "live-block", Kind: session.MessageKindText, Role: "assistant", Streaming: true, ConversationTurn: &zero, ConversationStep: &two})
	rows = inspectorChatRows(source, &navigation)
	if rows[len(rows)-1].Location != "0/2" || rows[len(rows)-1].ChatTarget.AnchorSeq != nil {
		t.Fatal("live coordinates were lost or fabricated as durable event zero")
	}
	for _, message := range navigation.Nodes {
		raw, _ := json.Marshal(message)
		source.Chat = append(source.Chat, session.InspectorRecord{Key: "chat:" + message.ID, Raw: raw})
	}
	fallback := inspectorChatRows(source, &chat.InspectorNavigation{})
	if len(fallback) != len(navigation.Nodes) || fallback[1].Location != "0/0" || fallback[1].Identity != "reasoning" {
		t.Fatal("store-only fallback lost native message identity/coordinates")
	}
	coordinateFree := json.RawMessage(`{"seq":2,"type":"assistant/message","data":{"text":"actual loaded envelope"}}`)
	source.Records[2].Raw, navigation.Nodes[1].Raw = coordinateFree, coordinateFree
	navigation.Nodes = append(navigation.Nodes, session.Message{ID: "system-prompt", Kind: "system-prompt", Seq: 5, TurnSeq: 0})
	rows = inspectorChatRows(source, &navigation)
	byID := map[string]session.InspectorRecord{}
	for _, row := range rows {
		byID[row.ChatTarget.ID] = row
	}
	if byID["reasoning"].Location != "0/0" || byID["system-prompt"].Location != "0/1" {
		t.Fatal("real loaded event boundaries did not place coordinate-free/generated rows")
	}
	source.Records = append(source.Records, session.InspectorRecord{Key: "attempt:live/chunk:0", Type: "tool-call-delta", ChatTarget: session.InspectorChatTarget{CallID: "preparing", Turn: &zero, Step: &two}})
	navigation.Nodes = append(navigation.Nodes, session.Message{ID: "tool:preparing", Kind: session.MessageKindTool, Streaming: true, Tool: &session.ToolCallView{CallID: "preparing", State: "preparing"}})
	rows = inspectorChatRows(source, &navigation)
	if row := rows[len(rows)-1]; row.Location != "0/2" || row.ChatTarget.AnchorSeq != nil {
		t.Fatal("preparing tool did not inherit its exact live call's coordinates")
	}
	custom := session.InspectorRecord{Key: "custom-row", Parent: "custom-parent", Type: "native/custom", Identity: "native-identity", Raw: json.RawMessage(`{"value":[1,2]}`)}
	legacy := session.InspectorSnapshot{Chat: []session.InspectorRecord{source.Chat[0], custom}}
	legacyRows := inspectorChatRows(legacy, &chat.InspectorNavigation{})
	if len(legacyRows) != 2 || legacyRows[1].Parent != custom.Parent || legacyRows[1].Identity != custom.Identity || string(legacyRows[1].Raw) != string(custom.Raw) {
		t.Fatal("store-only records were dropped or reserialized")
	}
	if value, ok := inspectorNativeObjects(legacy, &chat.InspectorNavigation{})[custom.Key].Value.(json.RawMessage); !ok || string(value) != string(custom.Raw) {
		t.Fatal("non-Message record lost its native data tree")
	}
}

func TestInspectorNativeObjectReadersAndKeylessNavigation(t *testing.T) {
	source, navigation := inspectorNativeFixture()
	source.Chat = inspectorChatRows(source, &navigation)
	objects := inspectorNativeObjects(source, &navigation)
	want := map[string]string{"chat:reasoning": "node", "chat:reasoning/data": "nodeData", "chat:process": "group", "chat:process/data": "groupData", "turn:0": "turn", "turn:0/data": "turnData", "step:0/0": "step", "step:0/0/data": "stepData"}
	for key, kind := range want {
		if object, ok := objects[key]; !ok || object.Kind != kind || object.Value == nil {
			t.Fatalf("missing current %s reader: %+v", kind, object)
		}
	}
	message, ok := objects["chat:reasoning"].Value.(*session.Message)
	if !ok || message != &navigation.Nodes[1] {
		t.Fatal("Message reader is a JSON replacement, not the actual Chat snapshot")
	}
	turn := objects["turn:0"].Value.(*inspectorLoadedTurn)
	step := objects["step:0/0"].Value.(*inspectorLoadedStep)
	if len(turn.Events) != 8 || len(turn.Steps) != 2 || len(step.Events) != 3 || len(turn.End) == 0 || len(step.End) == 0 {
		t.Fatal("loaded timeline reader lost lifecycle events or multi-step membership")
	}
	if !json.Valid(objects["turn:0"].Raw) || objects["turn:0"].JSONError != "" {
		t.Fatal("current loaded timeline JSON export lost its event collection")
	}
	p := &SessionInspectorPaneState{Mode: "chat-node", selected: "chat:reasoning", breadcrumb: []string{"chat:reasoning"}, objects: objects, Filter: "excluded"}
	p.followInspectorReference("step:0/1/data", source.Chat)
	if p.navigation.Target == nil || p.navigation.Target.Step == nil || *p.navigation.Target.Step != 1 || p.selected != "chat:response" || p.Filter != "" || p.breadcrumb[len(p.breadcrumb)-1] != "step:0/1/data" {
		t.Fatalf("keyless data reference failed to locate Chat: %+v", p)
	}
	p.backInspectorReference(0, source.Chat)
	if p.selected != "chat:reasoning" || p.navigation.Target.ID != "reasoning" {
		t.Fatal("breadcrumb did not restore the Message reader and row")
	}
	navigation.Nodes[1].Body = "current publication"
	p.objects = inspectorNativeObjects(source, &navigation)
	if p.objects["chat:reasoning"].Value.(*session.Message).Body != "current publication" {
		t.Fatal("stable reference retained stale Message data")
	}
	source.Records = source.Records[2:3]
	partial := inspectorNativeObjects(source, &navigation)
	if _, invented := partial["turn:0"]; invented {
		t.Fatal("a missing Turn start was fabricated")
	}
	if _, invented := partial["step:0/0"]; invented {
		t.Fatal("a missing Step start was fabricated")
	}
}

type inspectorNativeMethodProbe struct {
	Calls  *int
	hidden int
}

func (p inspectorNativeMethodProbe) String() string { (*p.Calls)++; return "not a data field" }

func TestInspectorNativeTypedTreeLazyPagingMapKeysBytesAndCycles(t *testing.T) {
	values := make([]int, 123)
	state := inspectorObjectTreeState{}
	object := inspectorObject{Value: values}
	rows := state.rows(object, nil)
	if len(rows) != 52 || !rows[len(rows)-1].more || rows[0].label != "[]int(123)" {
		t.Fatalf("first page: %d %+v", len(rows), rows[0])
	}
	state.more("$")
	if rows = state.rows(object, nil); len(rows) != 102 || !rows[len(rows)-1].more {
		t.Fatal("More did not add exactly one native collection page")
	}
	state.toggle("$")
	if len(state.rows(object, nil)) != 1 {
		t.Fatal("collapsed branch eagerly enumerated descendants")
	}
	state = inspectorObjectTreeState{expanded: map[string]bool{"$/0": true}}
	rows = state.rows(inspectorObject{Value: map[int]string{7: "seven"}}, nil)
	if rows[0].label != "map[int]string(1)" || rows[2].name != "Key" || rows[2].label != "7" || rows[3].label != `"seven"` {
		t.Fatalf("map key/value type lost: %+v", rows)
	}
	rows = (&inspectorObjectTreeState{}).rows(inspectorObject{Value: []byte{0, 255}}, nil)
	if len(rows) != 3 || rows[0].label != "[]uint8(2)" || rows[2].label != "255" {
		t.Fatal("bytes were JSON-base64 encoded")
	}
	cycle := map[string]any{}
	cycle["self"] = cycle
	rows = state.rows(inspectorObject{Value: cycle}, nil)
	if rows[len(rows)-1].circular != "$" {
		t.Fatalf("cycle did not link to its ancestor: %+v", rows)
	}
	state = inspectorObjectTreeState{expanded: map[string]bool{"$/nested": true}}
	rows = state.rows(inspectorObject{Value: json.RawMessage(`{"nested":{"large":9007199254740993},"array":[1,2]}`)}, nil)
	if len(rows) != 4 || rows[2].label != "9007199254740993" || rows[3].expanded {
		t.Fatalf("typed JSON precision or lazy disclosure lost: %+v", rows)
	}
	rows = state.rows(inspectorObject{Value: json.RawMessage(`{broken`)}, nil)
	if !strings.Contains(rows[0].label, "Invalid JSON") {
		t.Fatal("invalid source data disappeared")
	}
	calls := 0
	rows = (&inspectorObjectTreeState{}).rows(inspectorObject{Value: inspectorNativeMethodProbe{Calls: &calls}}, nil)
	if calls != 0 || rows[len(rows)-1].label != "<unexported>" {
		t.Fatal("inspection invoked a method or misrepresented an inaccessible field")
	}
	fn := func() { calls++ }
	label, expandable := inspectorValueDescription(fn)
	if calls != 0 || expandable || label != "[func()]" {
		t.Fatal("function inspection invoked or hid its real type")
	}
	if raw, failure := inspectorObjectJSON(fn); len(raw) != 0 || failure == "" {
		t.Fatal("unserializable runtime value advertised an empty JSON export")
	}
}

func TestInspectorNativeGroupAndTypedReferenceControls(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	source, navigation := inspectorNativeFixture()
	p := SessionInspectorPaneState{Mode: "chat-node", navigation: navigation}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, source, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 640)
	if err := tt.Click("Collapse children process"); err != nil {
		t.Fatal(err)
	}
	if _, visible := tt.Find("Inspector record chat:reasoning"); visible {
		t.Fatal("native Group did not collapse its members")
	}
	p.navigation.Picked = &chat.InspectorTarget{ID: "reasoning", GroupPart: "reasoning"}
	tt.Frame()
	if _, visible := tt.Find("Inspector record chat:reasoning"); !visible || p.selected != "chat:reasoning" {
		t.Fatal("pick did not reveal the folded member")
	}
	if err := tt.Click("Open Loaded Turn · 0"); err != nil {
		t.Fatal(err)
	}
	if p.breadcrumb[len(p.breadcrumb)-1] != "turn:0" || p.navigation.Target.Turn == nil {
		t.Fatal("typed Turn reference did not navigate")
	}
	if err := tt.Click("Open Loaded Turn data · 0"); err != nil {
		t.Fatal(err)
	}
	if p.breadcrumb[len(p.breadcrumb)-1] != "turn:0/data" {
		t.Fatal("typed Data reference did not navigate")
	}
	if err := tt.Click("Message · reasoning"); err != nil {
		t.Fatal(err)
	}
	if len(p.breadcrumb) != 1 || p.selected != "chat:reasoning" {
		t.Fatal("breadcrumb did not restore the source object")
	}
}

func TestInspectorHistoryErrorDismissal(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	p := SessionInspectorPaneState{}
	state := session.ViewState{HistoryPageErr: "page unavailable"}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{}, state, theme.Default().Light, nil)
	}, 500, 480)
	if err := tt.Click("Dismiss error"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("page unavailable") {
		t.Fatal("history error was not dismissed")
	}
	state.HistoryPageErr = ""
	tt.Frame()
	state.HistoryPageErr = "page unavailable"
	tt.Frame()
	if !tt.HasText("page unavailable") {
		t.Fatal("a new failed page request was silently dismissed")
	}
}
