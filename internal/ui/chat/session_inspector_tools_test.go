package chat

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func inspectorNestedToolMessages(t *testing.T, siblings int) []session.Message {
	t.Helper()
	messages := []session.Message{{ID: "root-source", Kind: session.MessageKindTool, Role: "tool", Seq: 10, EventType: "tool/call", Raw: json.RawMessage(`{"type":"tool/call","seq":10,"data":{"turn":0,"step":0,"callId":"opaque:root","name":"functions.read","arguments":"{}"}}`)}}
	add := func(id, parent string, seq int) {
		raw, _ := json.Marshal(map[string]any{"type": "tool/ptc-dispatch-start", "seq": seq, "data": map[string]any{"rootCallId": "opaque:root", "parentCallId": parent, "subCallId": id, "name": "functions.read", "arguments": map[string]any{"file_path": "same.txt"}}})
		messages = append(messages, session.Message{ID: "source:" + id, Kind: session.MessageKindTool, Role: "tool", Seq: float64(seq), EventType: "tool/ptc-dispatch-start", Raw: raw})
	}
	add("unrelated-child", "opaque:root", 11)
	add("opaque/grandchild", "unrelated-child", 12)
	for i := range siblings {
		add(fmt.Sprintf("sibling-%d", i), "opaque:root", 13+i)
	}
	return messages
}

func inspectorNestedToolFixture(t *testing.T, siblings int) (*ui.Tester, *InspectorNavigation, *Model, *ui.ScrollState) {
	t.Helper()
	n := &InspectorNavigation{SessionID: "s"}
	scroll := &ui.ScrollState{}
	expanded := map[string]bool{}
	m := &Model{Inspector: n, List: &ui.ListState{}, Scroll: scroll, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: inspectorNestedToolMessages(t, siblings)}, ThinkingExpanded: &expanded, SuppressStatus: true, ThemeID: theme.IDDeepseekLight, Preferences: settings.ChatPreferences{TranscriptView: "verbose"}, FullColumnViewport: true, ContentWidth: 450}
	tt := ui.NewTester(func(c *ui.Context) { View(c, *m) }, 650, 340)
	tt.SetTime(time.Unix(7100, 0))
	t.Cleanup(n.CancelReveal)
	return tt, n, m, scroll
}

func TestInspectorNestedToolPickerUsesClosestCallAndOwnerCoordinates(t *testing.T) {
	tt, n, m, _ := inspectorNestedToolFixture(t, 1)
	owner := "tool:opaque:root"
	before := n.reveal.rows[owner]
	n.Picking = true
	tt.Frame()
	if n.reveal.rows[owner] != before {
		t.Fatal("starting nested pick changed transcript geometry")
	}
	for _, id := range []string{"opaque:root", "unrelated-child", "opaque/grandchild", "sibling-0"} {
		key := inspectorToolKey{owner, id}
		r := n.reveal.toolRows[key]
		if r.H <= 0 || r.W <= 0 {
			t.Fatalf("missing painted call %q: %+v", id, n.reveal.toolRows)
		}
		// Each own header precedes its descendants, even when the subtree box
		// includes them. Identical titles never supply the call identity.
		x, y := r.X+5, r.Y+8
		tt.Move(x, y)
		if n.picker.previewCallID != id || n.picker.preview != inspectorChatIntersect(r, n.picker.area) {
			t.Fatalf("call %s shadowed by parent: picker=%+v rect=%+v", id, n.picker, r)
		}
		tt.ClickAt(x, y)
		if n.Picked == nil || n.Picked.ID != owner || n.Picked.CallID != id || n.Picked.AnchorSeq == nil || *n.Picked.AnchorSeq != 10 {
			t.Fatalf("nested pick replaced owner placement or child identity: %+v", n.Picked)
		}
		if len(*m.ThinkingExpanded) != 0 || n.reveal.rows[owner] != before {
			t.Fatal("nested pick opened underlying tool disclosure or reflowed Chat")
		}
		n.Picked = nil
	}
	child := n.reveal.toolRows[inspectorToolKey{owner, "unrelated-child"}]
	grand := n.reveal.toolRows[inspectorToolKey{owner, "opaque/grandchild"}]
	if child.Y+child.H < grand.Y+grand.H || child.H <= grand.H {
		t.Fatal("intermediate call box does not contain its descendant")
	}
	// Descendant indentation lies inside its parent's branch, outside the
	// descendant's own box, matching the nearest DOM ancestor.
	tt.Move(child.X+3, grand.Y+5)
	if n.picker.previewCallID != "unrelated-child" {
		t.Fatal("descendant indentation borrowed the leaf identity")
	}
	// A call disappearing without a viewport change retires the preview.
	m.Snapshot.Messages = m.Snapshot.Messages[:1]
	tt.Frame()
	if n.picker.previewCallID != "" || n.picker.preview.H > 0 {
		t.Fatal("removed call retained preview")
	}
}

func TestInspectorNestedToolVisibleRevealUsesSubtreeWithoutOpeningBodies(t *testing.T) {
	tt, n, m, scroll := inspectorNestedToolFixture(t, 1)
	owner := "tool:opaque:root"
	before := n.reveal.rows[owner]
	for _, id := range []string{"unrelated-child", "opaque/grandchild", "opaque:root"} {
		n.Target = &InspectorTarget{AnchorSeq: pickToolNumber(12), CallID: id, RootCallID: "opaque:root"}
		tt.Frame()
		want := n.reveal.toolRows[inspectorToolKey{owner, id}]
		if n.Highlight != owner || n.reveal.callID != id || n.reveal.rect != want || n.reveal.started.IsZero() || scroll.Y != 0 {
			t.Fatalf("call=%s owner/child reveal mismatch: highlight=%s state=%+v want=%+v", id, n.Highlight, n.reveal, want)
		}
		if len(*m.ThinkingExpanded) != 0 || n.reveal.rows[owner] != before {
			t.Fatal("reveal opened call body or changed layout")
		}
	}
}

func TestInspectorNestedToolOffscreenRegionCentersWithinTallOwner(t *testing.T) {
	for _, id := range []string{"opaque:root", "opaque/grandchild", "sibling-24"} {
		t.Run(id, func(t *testing.T) {
			tt, n, m, scroll := inspectorNestedToolFixture(t, 25)
			owner := "tool:opaque:root"
			if id != "sibling-24" {
				m.List.ScrollToOffset(scroll.MaxY)
				tt.Frame()
			}
			before := scroll.Y
			n.Target = &InspectorTarget{CallID: id, RootCallID: "opaque:root", AnchorSeq: pickToolNumber(10)}
			tt.Frame()
			for ms := 16; n.reveal.started.IsZero() && ms <= 2000; ms += 16 {
				tt.SetTime(time.Unix(7100, 0).Add(time.Duration(ms) * time.Millisecond))
			}
			want := n.reveal.toolRows[inspectorToolKey{owner, id}]
			if want.H <= 0 || n.reveal.rect != want || n.reveal.started.IsZero() || n.reveal.pending || m.List.ScrollAnimating() {
				t.Fatalf("child remained offscreen in tall owner: id=%s before=%v after=%v max=%v state=%+v want=%+v", id, before, scroll.Y, scroll.MaxY, n.reveal, want)
			}
			if id == "opaque:root" {
				if scroll.Y != before {
					t.Fatal("partly visible root subtree scrolled unnecessarily")
				}
			} else if scroll.Y == before {
				t.Fatal("offscreen child did not scroll independently of visible owner")
			}
		})
	}
}

func TestInspectorNestedToolInspectButtonReportsGrandchild(t *testing.T) {
	messages := session.ProjectToolMessages(inspectorNestedToolMessages(t, 1))
	msg := messages[0]
	expanded := map[string]bool{msg.ID + "|tool:opaque/grandchild": true}
	got := InspectorTarget{}
	tt := ui.NewTester(func(c *ui.Context) {
		ToolMessageView(c, msg, ToolCardStyle{FontSize: 12}, ToolImageGalleryOptions{ID: msg.ID, Expanded: &expanded, OnInspectCall: inspectCallHandler(func(callID, messageID string, seq float64) {
			got = InspectorTarget{CallID: callID, ID: messageID, Seq: seq}
		}, msg.ID, msg.Seq)})
	}, 650, 700)
	if err := tt.Click(i18n.T(i18n.KeyChatToolInspect)); err != nil {
		t.Fatal(err)
	}
	if got.CallID != "opaque/grandchild" || got.ID != msg.ID || got.Seq != 10 {
		t.Fatalf("nested renderer dropped selected identity: %+v", got)
	}
}

func TestInspectorNestedToolRevealOpensTranscriptAncestorsOnly(t *testing.T) {
	tt, n, m, _ := inspectorNestedToolFixture(t, 1)
	m.Preferences.TranscriptView = "standard"
	m.Snapshot.Messages = append(m.Snapshot.Messages,
		session.Message{ID: "answer", Seq: 50, Role: "assistant", Kind: session.MessageKindText, Body: "Answer", Raw: json.RawMessage(`{"data":{"turn":0,"step":1}}`)},
		session.Message{ID: "end", Seq: 51, Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Raw: json.RawMessage(`{"data":{"turn":0}}`)},
	)
	tt.Frame()
	owner := "tool:opaque:root"
	if n.reveal.toolRows[inspectorToolKey{owner, "opaque/grandchild"}].H > 0 {
		t.Fatal("closed transcript unexpectedly painted child")
	}
	n.Target = &InspectorTarget{CallID: "opaque/grandchild", RootCallID: "opaque:root", AnchorSeq: pickToolNumber(12)}
	tt.Frame()
	if !(*m.ThinkingExpanded)["transcript-turn:h:s:0"] || !(*m.ThinkingExpanded)["transcript-group:h:s:tool:opaque:root:10"] {
		t.Fatalf("child reveal did not unfold actual transcript ancestors: %+v", *m.ThinkingExpanded)
	}
	for key, open := range *m.ThinkingExpanded {
		if open && key != "transcript-turn:h:s:0" && key != "transcript-group:h:s:tool:opaque:root:10" {
			t.Fatalf("child reveal opened unrelated or own tool body: %s", key)
		}
	}
	for ms := 16; n.reveal.started.IsZero() && ms <= 2000; ms += 16 {
		tt.SetTime(time.Unix(7100, 0).Add(time.Duration(ms) * time.Millisecond))
	}
	if want := n.reveal.toolRows[inspectorToolKey{owner, "opaque/grandchild"}]; want.H <= 0 || n.reveal.rect != want {
		t.Fatalf("unfolded child did not own final flash: got=%+v want=%+v", n.reveal.rect, want)
	}
}

func TestInspectorNestedToolColdOwnerReducedMotionAndInterruption(t *testing.T) {
	for _, reduced := range []bool{false, true} {
		t.Run(fmt.Sprint(reduced), func(t *testing.T) {
			tt, n, m, scroll := inspectorNestedToolFixture(t, 25)
			// Cold target sits below enough ordinary rows to be unmaterialized.
			tools := m.Snapshot.Messages
			m.Snapshot.Messages = nil
			for i := range 25 {
				m.Snapshot.Messages = append(m.Snapshot.Messages, session.Message{ID: fmt.Sprintf("before-%d", i), Role: "user", Kind: session.MessageKindText, Body: "Filler"})
			}
			m.Snapshot.Messages = append(m.Snapshot.Messages, tools...)
			m.List.ScrollToOffset(0)
			tt.Frame()
			owner := "tool:opaque:root"
			if n.reveal.layoutRows[owner].H > 0 {
				t.Fatal("target fixture was not cold")
			}
			tt.SetPreferences(ui.Preferences{ReduceMotion: reduced, TextScale: 1})
			n.Target = &InspectorTarget{CallID: "sibling-24", RootCallID: "opaque:root", AnchorSeq: pickToolNumber(10)}
			tt.Frame()
			if !reduced {
				for ms := 16; n.reveal.started.IsZero() && ms <= 4000; ms += 16 {
					tt.SetTime(time.Unix(7100, 0).Add(time.Duration(ms) * time.Millisecond))
				}
			}
			want := n.reveal.toolRows[inspectorToolKey{owner, "sibling-24"}]
			if want.H <= 0 || scroll.Y <= 0 || m.List.ScrollAnimating() || n.reveal.pending {
				t.Fatalf("cold child not brought into view: reduced=%v offset=%v state=%+v box=%+v", reduced, scroll.Y, n.reveal, want)
			}
			if reduced {
				if !n.reveal.started.IsZero() || n.Highlight != "" {
					t.Fatal("reduced motion retained flash")
				}
			} else if n.reveal.rect != want || n.reveal.started.IsZero() {
				t.Fatal("cold child flashed its containing owner")
			}
		})
	}
	// Interrupt a measured child scroll before its endpoint; the user's wheel
	// wins and no delayed region request may restart afterward.
	tt, n, m, scroll := inspectorNestedToolFixture(t, 25)
	n.Target = &InspectorTarget{CallID: "sibling-24", RootCallID: "opaque:root", AnchorSeq: pickToolNumber(10)}
	tt.Frame()
	tt.SetTime(time.Unix(7100, 0).Add(100 * time.Millisecond))
	if !m.List.ScrollAnimating() || scroll.Y <= 0 {
		t.Fatal("region animation did not start")
	}
	tt.Scroll(300, 180, 0, -15)
	stopped := scroll.Y
	tt.SetTime(time.Unix(7100, 0).Add(2 * time.Second))
	if scroll.Y != stopped || m.List.ScrollAnimating() || n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatal("interrupted region navigation resumed")
	}
}

func TestInspectorNestedToolPendingRevealFollowsReparentedOrphan(t *testing.T) {
	for _, running := range []bool{false, true} {
		t.Run(fmt.Sprint(running), func(t *testing.T) {
			tt, n, m, _ := inspectorNestedToolFixture(t, 25)
			messages := m.Snapshot.Messages
			m.Snapshot.Messages = messages[1:]
			m.List.ScrollToOffset(0)
			tt.Frame()
			n.Target = &InspectorTarget{CallID: "sibling-24", RootCallID: "opaque:root", AnchorSeq: pickToolNumber(37)}
			tt.Frame()
			if running {
				tt.SetTime(time.Unix(7100, 0).Add(50 * time.Millisecond))
			}
			if n.Highlight != "tool:sibling-24" || !n.reveal.pending {
				t.Fatal("missing parent fixture did not start an orphan reveal")
			}
			m.Snapshot.Messages = messages
			tt.Frame()
			for ms := 80; n.reveal.started.IsZero() && ms < 4000; ms += 16 {
				tt.SetTime(time.Unix(7100, 0).Add(time.Duration(ms) * time.Millisecond))
			}
			want := n.reveal.toolRows[inspectorToolKey{"tool:opaque:root", "sibling-24"}]
			if want.H <= 0 || n.Highlight != "tool:opaque:root" || n.reveal.callID != "sibling-24" || n.reveal.rect != want || n.reveal.started.IsZero() || n.reveal.pending {
				t.Fatalf("older parent lost child or reused orphan region: running=%v highlight=%s state=%+v want=%+v", running, n.Highlight, n.reveal, want)
			}
		})
	}
}

func TestInspectorNestedToolPartialClipAndDeepAncestry(t *testing.T) {
	tt, n, m, _ := inspectorNestedToolFixture(t, 12)
	owner, leaf := "tool:opaque:root", "deep-8"
	for i := range 9 {
		parent := "opaque/grandchild"
		if i > 0 {
			parent = fmt.Sprintf("deep-%d", i-1)
		}
		raw, _ := json.Marshal(map[string]any{"type": "tool/ptc-dispatch-start", "seq": 100 + i, "data": map[string]any{"rootCallId": "opaque:root", "parentCallId": parent, "subCallId": fmt.Sprintf("deep-%d", i), "name": "functions.read", "arguments": map[string]any{}}})
		m.Snapshot.Messages = append(m.Snapshot.Messages, session.Message{ID: fmt.Sprintf("deep-source-%d", i), Seq: float64(100 + i), Kind: session.MessageKindTool, Role: "tool", EventType: "tool/ptc-dispatch-start", Raw: raw})
	}
	tt.Frame()
	box := n.reveal.toolFull[inspectorToolKey{owner, leaf}]
	if box.H <= 0 {
		t.Fatal("deep ancestry was not measured")
	}
	// A positive clipped intersection reveals in place even when the call's
	// top is above the viewport; it does not center the whole owner.
	m.List.ScrollToOffset(box.Y - n.reveal.viewport.Y + 10)
	tt.Frame()
	y := m.Scroll.Y
	n.Target = &InspectorTarget{CallID: leaf, RootCallID: "opaque:root", AnchorSeq: pickToolNumber(108)}
	tt.Frame()
	want := n.reveal.toolRows[inspectorToolKey{owner, leaf}]
	if want.H <= 0 || want.H >= n.reveal.toolFull[inspectorToolKey{owner, leaf}].H || n.reveal.rect != want || m.Scroll.Y != y {
		t.Fatalf("partial deep call was centered or unclipped: box=%+v want=%+v state=%+v", box, want, n.reveal)
	}
	n.Picking = true
	tt.Frame()
	tt.Move(want.X+4, want.Y+want.H/2)
	if n.picker.previewCallID != leaf || n.picker.preview != want {
		t.Fatal("indentation cap caused ancestor to shadow deeper call")
	}
	tt.SetSize(580, 320)
	if n.picker.preview.H > 0 {
		t.Fatal("nested preview retained geometry across resize")
	}
}

func TestInspectorNestedToolPendingRegionSurvivesOwnerSourceKeyChange(t *testing.T) {
	tt, n, m, _ := inspectorNestedToolFixture(t, 25)
	messages := m.Snapshot.Messages
	// Enable the real production source key, which includes event type/seq.
	m.HistoryStore = session.New()
	result := session.Message{ID: "root-result", Seq: 40, Kind: session.MessageKindTool, Role: "tool", EventType: "tool/result", Raw: json.RawMessage(`{"type":"tool/result","seq":40,"data":{"turn":0,"step":0,"message":{"toolCallId":"opaque:root","source":{"callId":"opaque:root"},"content":[]}}}`)}
	m.Snapshot.Messages = append(append([]session.Message(nil), messages[1:]...), result)
	m.List.ScrollToOffset(0)
	tt.Frame()
	n.Target = &InspectorTarget{CallID: "sibling-24", RootCallID: "opaque:root", AnchorSeq: pickToolNumber(37)}
	tt.Frame()
	tt.SetTime(time.Unix(7100, 0).Add(60 * time.Millisecond))
	if !m.List.ScrollAnimating() || n.reveal.centerRegion == nil || n.reveal.ownerEvent != "tool/result" {
		t.Fatal("result-only parent did not start keyed child-region motion")
	}
	// Restoring the earlier start retains owner ID but changes the list key.
	m.Snapshot.Messages = append([]session.Message{messages[0]}, m.Snapshot.Messages...)
	tt.Frame()
	for ms := 80; n.reveal.started.IsZero() && ms < 4000; ms += 16 {
		tt.SetTime(time.Unix(7100, 0).Add(time.Duration(ms) * time.Millisecond))
	}
	want := n.reveal.toolRows[inspectorToolKey{"tool:opaque:root", "sibling-24"}]
	if want.H <= 0 || n.reveal.rect != want || n.reveal.started.IsZero() || n.reveal.pending || n.reveal.ownerEvent != "tool/call" {
		t.Fatalf("source-key replacement abandoned pending child: state=%+v want=%+v", n.reveal, want)
	}
}

func pickToolNumber(v float64) *float64 { return &v }
