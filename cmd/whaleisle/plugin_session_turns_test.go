package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func turnPluginMessage(turn, seq int) session.Message {
	return session.Message{ID: fmt.Sprint(seq), Kind: session.MessageKindText, Role: "status", EventType: "turn/end", Seq: float64(seq), Raw: json.RawMessage(fmt.Sprintf(`{"type":"turn/end","seq":%d,"data":{"turn":%d,"reason":{"kind":"complete"}}}`, seq, turn))}
}
func turnPluginState() (pluginui.Snapshot, session.ViewState) {
	snap := session.ViewState{SelectedID: "s", ComposerHostGeneration: 3, ComposerSessionGeneration: 4, Messages: []session.Message{turnPluginMessage(0, 19), turnPluginMessage(2, 29)}}
	subject := pluginSessionSubject(snap)
	state := pluginui.Snapshot{Connected: true, Observation: true, Page: 7, Revision: 8, Subject: &subject, Outlets: []pluginui.SessionOutlet{
		{Handle: "a", Slot: "conversation.chat.turnTail", Label: "Content", Turn: 0, HasTurn: true, TurnTailSeq: 19, HasTurnTail: true, Available: true},
		{Handle: "b", Slot: "conversation.chat.turnTail", Label: "Content", Turn: 0, HasTurn: true, TurnTailSeq: 19, HasTurnTail: true, Available: true},
		{Handle: "c", Slot: "conversation.chat.turnTail", Label: "Content", Turn: 2, HasTurn: true, TurnTailSeq: 29, HasTurnTail: true, Available: true},
	}}
	return state, snap
}
func TestPluginSessionStaleTurnShowsPickerFeedback(t *testing.T) {
	sessions := session.New()
	sessions.Attach(&dsh.Client{}, nil)
	sessions.Select("s")
	target, err := sessions.CaptureComposerTarget("s")
	if err != nil {
		t.Fatal(err)
	}
	owner := &pluginui.Owner{}
	status := harness.Status{State: harness.Ready, Generation: 1}
	p := &nativePluginSessionState{visible: true, owner: owner, target: target, sessions: sessions, generation: 1, subject: pluginSessionSubject(sessions.Snapshot())}
	s := &mainState{sessions: sessions, bootStatus: status}
	s.nativePlugins.session = p
	request := sessionPluginTurnRequest{Page: 7, Revision: 8, Subject: p.subject, Tail: pluginui.SessionTurnTail{Turn: 0, Seq: 19}, Handle: "retired"}
	if s.showPluginSessionTurn(p, owner, request) == nil || !p.open || !p.turnExpired || p.error != "" {
		t.Fatal("stale turn silently ignored or leaked backend error")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		if p.turnExpired {
			ui.Text(c, i18n.T(i18n.KeySessionPluginsTurnExpired))
		}
	}, 390, 120)
	if !tt.HasText(i18n.T(i18n.KeySessionPluginsTurnExpired)) {
		t.Fatal("picker lacks user-visible stale feedback")
	}
}

func TestPluginSessionPickerRecoversExpiredMaskOnlyAfterSelection(t *testing.T) {
	for _, fail := range []bool{false, true} {
		masked := true
		var order []string
		tt := ui.NewTester(func(c *ui.Context) {
			if ui.Button(c, "Select original content").Clicked() {
				_ = selectSessionPluginSurface(func() error {
					order = append(order, "select")
					if fail {
						return errors.New("stale")
					}
					return nil
				}, func() { order = append(order, "unmask"); masked = false })
			}
		}, 390, 120)
		if err := tt.Click("Select original content"); err != nil {
			t.Fatal(err)
		}
		if fail {
			if !masked || !reflect.DeepEqual(order, []string{"select"}) {
				t.Fatal("failed selection exposed old content", order)
			}
		} else if masked || !reflect.DeepEqual(order, []string{"select", "unmask"}) {
			t.Fatal("picker retained expired mask", order)
		}
	}
}

func TestPluginSessionTurnRowsRequireUniqueRawCompletedBoundary(t *testing.T) {
	state, snap := turnPluginState()
	zero := pluginui.SessionTurnTail{Turn: 0, Seq: 19}
	if got := pluginSessionTurnRows(state, snap); len(got) != 2 || len(got[zero]) != 2 {
		t.Fatalf("actual turn zero not matched: %+v", got)
	}
	for _, tc := range []string{"missing-turn", "missing-seq", "raw-type", "fraction", "negative", "unsafe", "row-seq", "streaming", "duplicate-turn", "duplicate-seq", "raw-removed", "epoch", "unobserved", "unavailable", "no-canonical"} {
		t.Run(tc, func(t *testing.T) {
			st, ss := turnPluginState()
			m := &ss.Messages[0]
			switch tc {
			case "missing-turn":
				m.Raw = json.RawMessage(`{"type":"turn/end","seq":19,"data":{}}`)
			case "missing-seq":
				m.Raw = json.RawMessage(`{"type":"turn/end","data":{"turn":0}}`)
			case "raw-type":
				m.Raw = json.RawMessage(`{"type":"assistant/message","seq":19,"data":{"turn":0}}`)
			case "fraction":
				m.Raw = json.RawMessage(`{"type":"turn/end","seq":19,"data":{"turn":0.5}}`)
			case "negative":
				m.Raw = json.RawMessage(`{"type":"turn/end","seq":19,"data":{"turn":-1}}`)
			case "unsafe":
				m.Raw = json.RawMessage(`{"type":"turn/end","seq":19,"data":{"turn":9007199254740992}}`)
			case "row-seq":
				m.Seq = 18
			case "streaming":
				m.Streaming = true
			case "duplicate-turn":
				ss.Messages = append(ss.Messages, turnPluginMessage(0, 39))
			case "duplicate-seq":
				ss.Messages = append(ss.Messages, session.Message{Seq: 19})
			case "raw-removed":
				m.Raw = nil
			case "epoch":
				ss.ComposerSessionGeneration++
			case "unobserved":
				st.Observation = false
			case "unavailable":
				st.Unavailable = "session-version-incompatible"
			case "no-canonical":
				st.Outlets[0].HasTurnTail = false
				st.Outlets[1].HasTurnTail = false
			}
			if len(pluginSessionTurnRows(st, ss)[zero]) != 0 {
				t.Fatal("unverified boundary gained a native entry")
			}
		})
	}
	// Same ID, seq and raw byte length must not hit the transcript coordinate memo.
	snap.Messages[0].Raw = json.RawMessage(`{"type":"turn/end","seq":19,"data":{"turn":1,"reason":{"kind":"complete"}}}`)
	if len(pluginSessionTurnRows(state, snap)[zero]) != 0 {
		t.Fatal("same-length raw coordinate mutation reused cached turn")
	}
}
func TestPluginSessionTurnRowsReorderPrependAndDuplicateLabels(t *testing.T) {
	state, snap := turnPluginState()
	tail := pluginui.SessionTurnTail{Turn: 0, Seq: 19}
	before := pluginSessionTurnRows(state, snap)[tail]
	state.Outlets[0], state.Outlets[1] = state.Outlets[1], state.Outlets[0]
	snap.Messages = append([]session.Message{turnPluginMessage(9, 9)}, snap.Messages...)
	after := pluginSessionTurnRows(state, snap)[tail]
	labels := map[string]string{}
	for _, item := range before {
		labels[item.Outlet.Handle] = item.Accessible
	}
	for _, item := range after {
		if labels[item.Outlet.Handle] != item.Accessible {
			t.Fatal("history prepend/reorder renamed occurrence")
		}
	}
	var calls []string
	tt := ui.NewTester(func(c *ui.Context) {
		renderSessionPluginTurnOutlets(c, after, func(row pluginui.SessionOutlet) { calls = append(calls, row.Handle) })
	}, 390, 300)
	for _, item := range after {
		label := fmt.Sprintf(i18n.T(i18n.KeySessionPluginsOpenTurnContent), item.Title) + " · " + item.Attribution
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(calls, []string{"b", "a"}) {
		t.Fatalf("wrong handles: %v", calls)
	}
}
func TestPluginSessionTurnEntriesUseActualChatFooterCoordinates(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	state, snap := turnPluginState()
	state.Outlets = append(state.Outlets[:1], state.Outlets[2])
	rows := pluginSessionTurnRows(state, snap)
	var received []string
	m := chat.Model{Snapshot: snap, List: &ui.ListState{FollowEnd: true}, SuppressStatus: true, FullColumnViewport: true, ThemeID: theme.IDDeepseekLight}
	m.TurnContributions = func(c *ui.Context, view session.ViewState, msg session.Message) {
		tail, ok := pluginSessionTurnEnd(msg)
		if ok {
			renderSessionPluginTurnOutlets(c, rows[tail], func(row pluginui.SessionOutlet) {
				received = append(received, fmt.Sprintf("%d:%s", tail.Seq, row.Handle))
			})
		}
	}
	tt := ui.NewTester(func(c *ui.Context) { chat.View(c, m) }, 390, 500)
	for _, item := range sessionPluginOutletItems(state.Outlets) {
		label := fmt.Sprintf(i18n.T(i18n.KeySessionPluginsOpenTurnContent), item.Title) + " · " + item.Attribution
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(received, []string{"19:a", "29:c"}) {
		t.Fatalf("same-label footers retargeted: %v", received)
	}
}

func TestPluginSessionTurnRequestNeverSubstitutesNewRevisionOrOwner(t *testing.T) {
	state, snap := turnPluginState()
	request := sessionPluginTurnRequest{Page: state.Page, Revision: state.Revision, Subject: *state.Subject, Tail: pluginui.SessionTurnTail{Turn: 0, Seq: 19}, Handle: "a"}
	for _, mode := range []string{"current", "owner-reused", "raw-removed", "same-id-epoch", "stale-revision", "switch-during-send"} {
		t.Run(mode, func(t *testing.T) {
			ss := snap
			ss.Messages = append([]session.Message(nil), snap.Messages...)
			current := true
			shown, opened := 0, 0
			switch mode {
			case "owner-reused":
				current = false
			case "raw-removed":
				ss.Messages = nil
			case "same-id-epoch":
				ss.ComposerSessionGeneration++
			}
			err := request.invoke(func() bool { return current }, func() session.ViewState { return ss }, func(got sessionPluginTurnRequest) error {
				shown++
				if got != request {
					t.Fatal("captured revision/handle replaced")
				}
				if mode == "stale-revision" {
					return errors.New("stale")
				}
				if mode == "switch-during-send" {
					current = false
				}
				return nil
			}, func() { opened++ })
			if mode == "current" {
				if err != nil || shown != 1 || opened != 1 {
					t.Fatal(err, shown, opened)
				}
			} else if err == nil || opened != 0 {
				t.Fatal("stale request opened a dialog", err, shown, opened)
			}
		})
	}
}
func TestPluginSessionTurnPendingRequiresOwnAcknowledgementAndExpires(t *testing.T) {
	state, _ := turnPluginState()
	request := sessionPluginTurnRequest{Page: 7, Revision: 8, Subject: *state.Subject, Tail: pluginui.SessionTurnTail{Turn: 0, Seq: 19}, Handle: "a", Nonce: "new-request"}
	now := time.Unix(10, 0)
	pending := sessionPluginTurnPending{Request: request, Until: now.Add(3 * time.Second)}
	state.Revision = 9
	state.Surface = "conversation.chat.turnTail"
	state.SessionSurfaceHandle = "a"
	state.SessionSurfaceRequest = "old-request"
	if ready, expired := pending.settled(state, now); ready || expired {
		t.Fatal("unrelated higher revision prematurely settled")
	}
	state.SessionSurfaceRequest = request.Nonce
	if ready, expired := pending.settled(state, now); !ready || expired {
		t.Fatal("own acknowledgement refused")
	}
	for _, mode := range []string{"same-revision", "old-page", "disconnected", "removed", "retargeted", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			st := state
			st.Outlets = append([]pluginui.SessionOutlet(nil), state.Outlets...)
			at := now
			switch mode {
			case "same-revision":
				st.Revision = 8
			case "old-page":
				st.Page++
			case "disconnected":
				st.Connected = false
			case "removed":
				st.Outlets = nil
			case "retargeted":
				st.Outlets[0].TurnTailSeq++
			case "timeout":
				st.SessionSurfaceRequest = "old"
				at = pending.Until
			}
			ready, expired := pending.settled(st, at)
			if ready {
				t.Fatal("foreign acknowledgement accepted")
			}
			if mode != "same-revision" && !expired {
				t.Fatal("invalid or dropped selection kept waiting")
			}
		})
	}
}
