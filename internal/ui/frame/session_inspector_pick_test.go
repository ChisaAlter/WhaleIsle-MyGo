package frame

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func pickNumber(n float64) *float64 { return &n }

func TestInspectorAcceptedMaskImmediatelyReleasesUnderlyingDisclosure(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	p := SessionInspectorPaneState{Mode: "chat-node", navigation: chat.InspectorNavigation{SessionID: "s"}}
	expanded := map[string]bool{}
	model := chat.Model{Inspector: &p.navigation, List: &ui.ListState{}, ThinkingExpanded: &expanded, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: []session.Message{
		{ID: "question", Role: "user", Kind: session.MessageKindText, Body: "Question"},
		{ID: "reason", Role: "assistant", Kind: session.MessageKindThinking, Body: "Details"},
		{ID: "answer", Role: "assistant", Kind: session.MessageKindText, Body: "Answer"},
	}}, ThemeID: theme.IDDeepseekLight, SuppressStatus: true, FullColumnViewport: true, ContentWidth: 400}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			ui.Column(c).Width(600).FillHeight().Children(func() { chat.View(c, model) })
			ui.Column(c).Fill().Children(func() {
				source := session.InspectorSnapshot{}
				for _, node := range p.navigation.Nodes {
					raw, _ := json.Marshal(node)
					source.Chat = append(source.Chat, session.InspectorRecord{Key: "chat:" + node.ID, Type: string(node.Kind), Raw: raw, ChatTarget: p.navigation.PickTarget(node)})
				}
				inspectorPresentation(c, &p, source, session.ViewState{}, theme.Default().Light, nil)
			})
		})
	}, 1300, 500)
	label := i18n.T(i18n.KeyChatProcessAnalyzing)
	for range 2 {
		p.toggleInspectorPicking()
		tt.Frame()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		if p.navigation.Picking || p.navigation.Picked != nil || p.selected == "" {
			t.Fatal("accepted handoff retained the mask")
		}
		for _, open := range expanded {
			if open {
				t.Fatal("pick activated underlying disclosure")
			}
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		open := false
		for _, value := range expanded {
			open = open || value
		}
		if !open {
			t.Fatal("next click was swallowed after successful selection")
		}
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInspectorReversePickInstalledReference(t *testing.T) {
	raw, err := os.ReadFile("testdata/inspector-picks.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		LogCases []struct {
			Name    string
			Target  chat.InspectorTarget
			Entries []struct {
				Type  string
				Event struct {
					Type string
					Seq  float64
					Data struct {
						Turn, Step        *float64
						AttemptID, CallID string
					}
				}
			}
			Rows []struct {
				Key, Parent string
				Value       json.RawMessage
			}
			SettledRoots []json.RawMessage
			Expected     string
		}
		ChatCases []struct {
			Name     string
			Target   chat.InspectorTarget
			Rows     [][]json.RawMessage
			Expected string
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, c := range fixture.LogCases {
		t.Run(c.Name, func(t *testing.T) {
			aliases := map[float64]string{}
			for _, alias := range c.SettledRoots {
				var pair []json.RawMessage
				_ = json.Unmarshal(alias, &pair)
				var seq float64
				var key string
				_ = json.Unmarshal(pair[0], &seq)
				_ = json.Unmarshal(pair[1], &key)
				aliases[seq] = key
			}
			entries := []inspectorPickLogEntry{}
			rows := []session.InspectorRecord{}
			for _, entry := range c.Entries {
				ev := entry.Event
				key := aliases[ev.Seq]
				if key == "" {
					key = "event:" + strconv.FormatFloat(ev.Seq, 'f', -1, 64)
				}
				kind := ev.Type
				target := chat.InspectorTarget{AnchorSeq: pickNumber(ev.Seq), CallID: ev.Data.CallID, Turn: ev.Data.Turn, Step: ev.Data.Step}
				if entry.Type == "transient" {
					key, kind = "attempt:"+ev.Data.AttemptID, "assistant/live"
				}
				entries = append(entries, inspectorPickLogEntry{key: key, kind: kind, seq: target.AnchorSeq, target: target})
			}
			for _, row := range c.Rows {
				var chunk struct{ Type string }
				_ = json.Unmarshal(row.Value, &chunk)
				rows = append(rows, session.InspectorRecord{Key: row.Key, Parent: row.Parent, Type: chunk.Type, Raw: row.Value})
			}
			if got := inspectorPickLogPart(rows, inspectorPickLogRoot(entries, c.Target), c.Target.GroupPart); got != c.Expected {
				t.Fatalf("installed pick differs: got %q, want %q", got, c.Expected)
			}
		})
	}
	for _, c := range fixture.ChatCases {
		t.Run(c.Name, func(t *testing.T) {
			rows := []session.InspectorRecord{}
			for _, pair := range c.Rows {
				var key string
				var target chat.InspectorTarget
				_ = json.Unmarshal(pair[0], &key)
				_ = json.Unmarshal(pair[1], &target)
				rows = append(rows, session.InspectorRecord{Key: key, ChatTarget: target})
			}
			if got := inspectorPickChatRecord(rows, c.Target); got != c.Expected {
				t.Fatalf("installed Chat pick differs: got %q want %q", got, c.Expected)
			}
		})
	}
}

func TestInspectorLogPickEnrichesCurrentGroupFirstMember(t *testing.T) {
	zero, one, nine := pickNumber(0), pickNumber(1), pickNumber(9)
	p := SessionInspectorPaneState{navigation: chat.InspectorNavigation{
		Groups: map[string][]string{"process": {"first", "second"}, "empty": {}, "missing-first": {"unavailable", "second"}},
		Nodes: []session.Message{
			{ID: "first", Seq: 4, Kind: session.MessageKindThinking, Raw: json.RawMessage(`{"type":"assistant/message","seq":4,"data":{"turn":0,"step":0}}`)},
			{ID: "second", Seq: 6, Role: "assistant", Raw: json.RawMessage(`{"type":"assistant/message","seq":6,"data":{"turn":1,"step":1}}`)},
			{ID: "turn-node", Seq: 7, Raw: json.RawMessage(`{"type":"custom/turn","seq":7,"data":{"turn":0}}`)},
			{ID: "session-node", Seq: 8},
		},
	}}
	for _, tc := range []struct {
		name               string
		in                 chat.InspectorTarget
		anchor, turn, step *float64
		kind, part         string
	}{
		{"first-member", chat.InspectorTarget{GroupKey: "process"}, pickNumber(4), zero, zero, "assistant-step", ""},
		{"node-over-group", chat.InspectorTarget{ID: "second", GroupKey: "process", AnchorSeq: nine, Turn: nine, Step: nine, GroupPart: "response"}, pickNumber(6), one, one, "assistant-step", "response"},
		{"stale-node-does-not-retry-group", chat.InspectorTarget{ID: "stale", GroupKey: "process", AnchorSeq: nine, Turn: nine}, nine, nine, nil, "", ""},
		{"first-unresolved-does-not-retry-second", chat.InspectorTarget{GroupKey: "missing-first"}, nil, nil, nil, "", ""},
		{"empty-group", chat.InspectorTarget{GroupKey: "empty"}, nil, nil, nil, "", ""},
		{"absent-group", chat.InspectorTarget{GroupKey: "absent"}, nil, nil, nil, "", ""},
		{"turn-location-preserves-input-step", chat.InspectorTarget{ID: "turn-node", Turn: nine, Step: nine}, pickNumber(7), zero, nine, "", ""},
		{"session-location-preserves-input-location", chat.InspectorTarget{ID: "session-node", Turn: nine, Step: nine}, pickNumber(8), nine, nine, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := p.inspectorLogPickTarget(tc.in)
			equal := func(a, b *float64) bool { return a == nil && b == nil || inspectorPickCoordinate(a, b) }
			if !equal(got.AnchorSeq, tc.anchor) || !equal(got.Turn, tc.turn) || !equal(got.Step, tc.step) || got.NodeKind != tc.kind || got.GroupPart != tc.part {
				t.Fatalf("wrong current-node enrichment: %+v", got)
			}
		})
	}
}

func TestInspectorNativeLiveBlocksAndPreparingToolsKeepActualOwner(t *testing.T) {
	zero := pickNumber(0)
	rows := []session.InspectorRecord{
		{Key: "event:0", Type: "turn/start", Seq: 0, ChatTarget: chat.InspectorTarget{Turn: zero}},
		{Key: "attempt:live", Type: "assistant/live", ChatTarget: chat.InspectorTarget{Turn: zero, Step: zero}},
		{Key: "attempt:live/chunk:0", Parent: "attempt:live", Type: "block-start", ChatID: "second"},
		{Key: "attempt:live/chunk:1", Parent: "attempt:live", Type: "block-start", ChatPickID: "first"},
		{Key: "attempt:live/chunk:2", Parent: "attempt:live", Type: "block-start", ChatPickID: "second"},
		{Key: "attempt:live/chunk:3", Parent: "attempt:live", Type: "tool-call-delta", ChatTarget: chat.InspectorTarget{CallID: "preparing"}},
	}
	second := session.Message{ID: "second", Role: "assistant", Streaming: true, ConversationTurn: zero, ConversationStep: zero}
	tool := session.Message{ID: "tool:preparing", Kind: session.MessageKindTool, Streaming: true, Tool: &session.ToolCallView{CallID: "preparing", State: "preparing"}}
	p := SessionInspectorPaneState{navigation: chat.InspectorNavigation{Nodes: []session.Message{second, tool}, Groups: map[string][]string{"tools": {"tool:preparing"}}}}
	if got := p.inspectorPickRecord(rows, chat.InspectorTarget{GroupKey: "tools"}); got != "attempt:live" {
		t.Fatalf("group first preparing member lost actual stream owner: %q", got)
	}
	if got := p.inspectorPickRecord(rows, second.InspectorTarget()); got != "attempt:live/chunk:2" {
		t.Fatalf("exact second live block lost to Step fallback or borrowed ChatID: %q", got)
	}
	if got := p.inspectorPickRecord(rows, tool.InspectorTarget()); got != "attempt:live" {
		t.Fatalf("preparing call lost actual stream owner: %q", got)
	}
	if got := p.inspectorPickRecord(rows, chat.InspectorTarget{CallID: "preparing"}); got != "" {
		t.Fatalf("unowned call invented a source anchor: %q", got)
	}
	if got := p.inspectorPickRecord(rows, chat.InspectorTarget{AnchorSeq: zero}); got != "event:0" {
		t.Fatalf("explicit zero anchor lost to unanchored live root: %q", got)
	}
	rows = append(rows, session.InspectorRecord{Key: "event:4", Type: "assistant/message", Seq: 4, ChatTarget: chat.InspectorTarget{Turn: zero, Step: zero}})
	rows[1].Seq, rows[1].ChatTarget.AnchorSeq = 4, pickNumber(4)
	if got := p.inspectorPickRecord(rows, chat.InspectorTarget{AnchorSeq: pickNumber(4)}); got != "event:4" {
		t.Fatalf("retained live prefix shadowed canonical settlement: %q", got)
	}
}

func TestInspectorReverseReasoningOpensAncestorsWithoutRevealingChat(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	zero := pickNumber(0)
	rows := []session.InspectorRecord{
		{Key: "event:0", Type: "turn/start", Seq: 0, ChatTarget: chat.InspectorTarget{Turn: zero}},
		{Key: "event:1", Parent: "event:0", Type: "step/start", Seq: 1, ChatTarget: chat.InspectorTarget{Turn: zero, Step: zero}},
		{Key: "attempt:stable", Parent: "event:1", Type: "assistant/attempt", Seq: 4, ChatTarget: chat.InspectorTarget{Turn: zero, Step: zero}},
		{Key: "attempt:stable/chunk:0", Parent: "attempt:stable", Type: "block-start", Raw: json.RawMessage(`{"type":"block-start","index":9,"blockType":"reasoning"}`)},
		{Key: "attempt:stable/chunk:1", Parent: "attempt:stable/chunk:0", Type: "reasoning-delta"},
	}
	p := SessionInspectorPaneState{Filter: "assistant", collapsed: map[string]inspectorDisclosure{"event:0": {collapsed: true}, "event:1": {collapsed: true}, "attempt:stable": {collapsed: true}, "attempt:stable/chunk:0": {collapsed: true}}, navigation: chat.InspectorNavigation{SessionID: "s"}}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 600)
	p.navigation.Picked = &chat.InspectorTarget{AnchorSeq: pickNumber(4), GroupPart: "reasoning"}
	tt.Frame()
	if p.selected != "attempt:stable/chunk:0" || p.Filter != "" || p.navigation.Target != nil || p.navigation.Picking || !tt.HasText("block-start") || tt.HasText("reasoning-delta") {
		t.Fatalf("reasoning pick lost root/part/disclosures: selected=%s filter=%s navigation=%+v", p.selected, p.Filter, p.navigation)
	}
	for _, key := range []string{"event:0", "event:1", "attempt:stable"} {
		if p.collapsed[key].collapsed {
			t.Fatalf("ancestor %s was not opened", key)
		}
	}
}
