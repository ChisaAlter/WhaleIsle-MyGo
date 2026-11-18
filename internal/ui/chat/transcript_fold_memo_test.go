package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func buildFoldMemoTestMessages() []session.Message {
	coords := func(turn, step int) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{"data":{"turn":%d,"step":%d}}`, turn, step))
	}
	toolRaw := func(turn, step int, callID, name string) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(
			`{"data":{"turn":%d,"step":%d,"call_id":%q,"name":%q,"status":"completed","arguments":"{\"path\":\"a.go\"}","output":"ok"}}`,
			turn, step, callID, name,
		))
	}
	return []session.Message{
		// Turn 0 (prefix turn)
		{ID: "u0", Seq: 1, Role: "user", Kind: session.MessageKindText, Body: "Turn zero question", Raw: coords(0, 0)},
		{ID: "p0", Seq: 2, Role: "thinking", Kind: session.MessageKindThinking, Body: "Prefix thinking process alpha", Raw: coords(0, 0)},
		{ID: "t0-1", Seq: 3, Role: "tool", Kind: session.MessageKindTool, Body: "read_file", Raw: toolRaw(0, 0, "call-0-1", "read_file")},
		{ID: "t0-2", Seq: 4, Role: "tool", Kind: session.MessageKindTool, Body: "grep_search", Raw: toolRaw(0, 0, "call-0-2", "grep_search")},
		{ID: "a0", SourceMessageID: "a0", Seq: 5, Role: "assistant", Kind: session.MessageKindText, Body: "Turn zero final answer", Raw: coords(0, 1)},
		{ID: "e0", Seq: 6, Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Raw: coords(0, 1)},
		// Turn 1 (prefix turn)
		{ID: "u1", Seq: 7, Role: "user", Kind: session.MessageKindText, Body: "Turn one question", Raw: coords(1, 0)},
		{ID: "p1", Seq: 8, Role: "thinking", Kind: session.MessageKindThinking, Body: "Prefix thinking process beta", Raw: coords(1, 0)},
		{ID: "a1", SourceMessageID: "a1", Seq: 9, Role: "assistant", Kind: session.MessageKindText, Body: "Turn one final answer", Raw: coords(1, 1)},
		{ID: "e1", Seq: 10, Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Raw: coords(1, 1)},
		// Turn 2 (active/tail turn with multiple tools + thinking + streaming reply)
		{ID: "u2", Seq: 11, Role: "user", Kind: session.MessageKindText, Body: "Tail turn question", Raw: coords(2, 0)},
		{ID: "p2", Seq: 12, Role: "thinking", Kind: session.MessageKindThinking, Body: "Tail reasoning summary\n\nTail unique thought marker", Raw: coords(2, 0)},
		{ID: "t2-1", Seq: 13, Role: "tool", Kind: session.MessageKindTool, Body: "read_file", Raw: toolRaw(2, 0, "call-2-1", "read_file")},
		{ID: "t2-2", Seq: 14, Role: "tool", Kind: session.MessageKindTool, Body: "read_file", Raw: toolRaw(2, 0, "call-2-2", "read_file")},
		{ID: "t2-3", Seq: 15, Role: "tool", Kind: session.MessageKindTool, Body: "read_file", Raw: toolRaw(2, 0, "call-2-3", "read_file")},
		{ID: "a2", SourceMessageID: "a2", Seq: 16, Role: "assistant", Kind: session.MessageKindText, Body: "Tail reply body", Streaming: true, Raw: coords(2, 1)},
	}
}

func TestTranscriptFoldDecoupledFromTurnPrefixMemo(t *testing.T) {
	snap := session.New().Snapshot()
	snap.SelectedID = "fold-memo-session"
	snap.PlanHistoryBinding = "fold-memo-host"
	snap.ClientReady = true
	snap.Messages = buildFoldMemoTestMessages()

	expanded := map[string]bool{}
	var viewMemo transcriptViewMemo
	m := Model{
		Snapshot:         snap,
		ThinkingExpanded: &expanded,
		Preferences: settings.ChatPreferences{
			TranscriptView: "standard",
			CollapseTiming: "never",
		},
	}

	msgs1, controls1, _, _ := viewMemo.resolve(m)
	if !viewMemo.turnPrefix.ready || len(viewMemo.turnPrefix.prefixMsgs) == 0 {
		t.Fatalf("expected turnPrefix memo to be active, got ready=%v prefixLen=%d", viewMemo.turnPrefix.ready, len(viewMemo.turnPrefix.prefixMsgs))
	}
	prefixPtr := &viewMemo.turnPrefix.prefixMsgs[0]
	msgsPtr := &msgs1[0]

	// 1. Toggling row-level thinking / tool / turn-end fold keys must hit the
	// projection cache cleanly without rebuilding turnPrefix.
	rowKeys := []string{
		"p0",
		"p2",
		"tool-args:call-0-1",
		"tool-result:call-2-1",
		"turn-end:0",
	}
	for _, k := range rowKeys {
		expanded[k] = true
		msgs2, _, _, _ := viewMemo.resolve(m)
		if &viewMemo.turnPrefix.prefixMsgs[0] != prefixPtr {
			t.Fatalf("toggling row fold key %q rebuilt turnPrefix", k)
		}
		if len(msgs2) == 0 || &msgs2[0] != msgsPtr {
			t.Fatalf("toggling row fold key %q missed whole-transcript memo cache", k)
		}
	}

	// 2. Find a prefix turn/group control and a tail control via turnPrefix metadata.
	var prefixCtrlKey, tailCtrlKey string
	if len(viewMemo.turnPrefix.prefixControlStates) > 0 {
		prefixCtrlKey = viewMemo.turnPrefix.prefixControlStates[0].key
	}
	for _, k := range viewMemo.turnPrefix.lastTailControlKeys {
		if strings.HasPrefix(k, "transcript-group:") || strings.HasPrefix(k, "transcript-turn:") {
			tailCtrlKey = k
			break
		}
	}

	if tailCtrlKey != "" {
		wasTailOpen := controls1[tailCtrlKey].open
		expanded[tailCtrlKey] = !wasTailOpen
		_, controlsAfterTail, _, _ := viewMemo.resolve(m)
		if &viewMemo.turnPrefix.prefixMsgs[0] != prefixPtr {
			t.Fatalf("toggling tail transcript control %q rebuilt turnPrefix", tailCtrlKey)
		}
		if controlsAfterTail[tailCtrlKey].open == wasTailOpen {
			t.Fatalf("tail transcript control %q did not update state", tailCtrlKey)
		}
	}

	if prefixCtrlKey == "" {
		t.Fatalf("expected at least one prefix transcript control in controls=%v", controls1)
	}
	wasPrefixOpen := controls1[prefixCtrlKey].open
	expanded[prefixCtrlKey] = !wasPrefixOpen
	_, controlsAfterPrefix, _, _ := viewMemo.resolve(m)
	if &viewMemo.turnPrefix.prefixMsgs[0] == prefixPtr {
		t.Fatalf("toggling prefix transcript control %q failed to rebuild turnPrefix", prefixCtrlKey)
	}
	if controlsAfterPrefix[prefixCtrlKey].open == wasPrefixOpen {
		t.Fatalf("prefix transcript control %q state did not flip", prefixCtrlKey)
	}
}

func TestTranscriptSingleThinkingCollapseUpdatesUIWithoutRebuildingPrefix(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale("zh") })

	snap := session.New().Snapshot()
	snap.SelectedID = "fold-ui-session"
	snap.PlanHistoryBinding = "fold-ui-host"
	snap.ClientReady = true
	snap.Messages = buildFoldMemoTestMessages()

	expanded := map[string]bool{}
	m := &Model{
		List:               &ui.ListState{FollowEnd: true},
		Scroll:             &ui.ScrollState{},
		Inspector:          &InspectorNavigation{SessionID: "fold-ui-session"},
		SuppressStatus:     true,
		FullColumnViewport: true,
		ContentWidth:       560,
		ThemeID:            theme.IDDeepseekLight,
		Snapshot:           snap,
		ThinkingExpanded:   &expanded,
		Preferences: settings.ChatPreferences{
			TranscriptView: "standard",
			CollapseTiming: "never",
		},
	}

	var viewMemo transcriptViewMemo
	viewMemo.resolve(*m)
	for _, k := range viewMemo.turnPrefix.lastTailControlKeys {
		if strings.HasPrefix(k, "transcript-group:") {
			expanded[k] = true
		}
	}
	viewMemo.resolve(*m)
	if !viewMemo.turnPrefix.ready || len(viewMemo.turnPrefix.prefixMsgs) == 0 {
		t.Fatalf("expected turnPrefix ready")
	}
	prefixPtr := &viewMemo.turnPrefix.prefixMsgs[0]

	now := time.Unix(12000, 0)
	tt := ui.NewTester(func(c *ui.Context) { View(c, *m) }, 760, 480)
	tt.SetTime(now)
	tt.Frame()

	header, ok := tt.Find(i18n.T(i18n.KeyChatThinking))
	if !ok {
		t.Fatalf("expected visible thinking toggle header; texts=%v", tt.Texts())
	}

	_, wasVisible := tt.Find("Tail unique thought marker")
	tt.ClickAt(header.X+header.W/2, header.Y+header.H/2)
	now = now.Add(time.Second)
	tt.SetTime(now)
	tt.Frame()
	_, nowVisible := tt.Find("Tail unique thought marker")
	if nowVisible == wasVisible {
		t.Fatalf("clicking thinking header did not toggle visible thinking body (was=%v now=%v)", wasVisible, nowVisible)
	}
	viewMemo.resolve(*m)
	if &viewMemo.turnPrefix.prefixMsgs[0] != prefixPtr {
		t.Fatal("toggling single thinking block in UI rebuilt turnPrefix")
	}

	header, ok = tt.Find(i18n.T(i18n.KeyChatThinking))
	if !ok {
		t.Fatalf("expected thinking header before second click; texts=%v", tt.Texts())
	}
	tt.ClickAt(header.X+header.W/2, header.Y+header.H/2)
	now = now.Add(time.Second)
	tt.SetTime(now)
	tt.Frame()
	_, backVisible := tt.Find("Tail unique thought marker")
	if backVisible != wasVisible {
		t.Fatalf("second click did not restore thinking visibility (want=%v got=%v)", wasVisible, backVisible)
	}
	viewMemo.resolve(*m)
	if &viewMemo.turnPrefix.prefixMsgs[0] != prefixPtr {
		t.Fatal("second thinking toggle in UI rebuilt turnPrefix")
	}
}
