package pluginui

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/coder/websocket/wsjson"
)

var errSessionUnavailable = errors.New("plugin UI: Session outlet is no longer available")

// SessionOutlet identifies one original renderer outlet inside the current
// retained Session. Multiple tool calls are distinct occurrences of one slot.
type SessionOutlet struct {
	Handle          string `json:"handle"`
	Slot            string `json:"slot"`
	Label           string `json:"label"`
	CallID          string `json:"callId,omitempty"`
	AgentID         string `json:"agentId,omitempty"`
	SourceRequestID string `json:"sourceRequestId,omitempty"`
	PendingKey      string `json:"pendingKey,omitempty"`
	EntryID         string `json:"entryId,omitempty"`
	Registrant      string `json:"registrant,omitempty"`
	Turn            int    `json:"turn,omitempty"`
	// HasTurn distinguishes a legitimate turn zero from absent metadata. Older
	// snapshots with a positive Turn are still unambiguous without this flag.
	HasTurn bool `json:"hasTurn,omitempty"`
	// TurnTailSeq is the canonical turn/end seq read from the original Turn's
	// turn-tail data, not the closing Assistant seq passed as owner.seq.
	TurnTailSeq uint64       `json:"turnTailSeq,omitempty"`
	HasTurnTail bool         `json:"hasTurnTail,omitempty"`
	Available   bool         `json:"available"`
	Native      []NativeNode `json:"native,omitempty"`
	Seq         float64      `json:"seq,omitempty"`
	HasSeq      bool         `json:"hasSeq,omitempty"`
}

var sessionSlotNames = map[string]bool{
	"main.conversation":                     true,
	"conversation.session.header.actions":   true,
	"conversation.session.header.utilities": true,
	"conversation.session.header.corner":    true,
	"conversation.session.header.lineage":   true,
	"conversation.session.header.leading":   true,
	"conversation.session.body":             true,
	"conversation.view":                     true,
	"conversation.composer":                 true,
	"conversation.composer.bar":             true,
	"conversation.chat.reasoning.body":      true,
	"conversation.chat.user.content":        true,
	"tool.call.body":                        true,
	"sidebar.right.pane.tab":                true,
	"conversation.composer.dock":            true,
	"conversation.input.dock":               true,
	"conversation.input.overlay":            true,
	"conversation.input.left":               true,
	"conversation.input.right":              true,
	"conversation.input.activity":           true,
	"conversation.input.plan":               true,
	"conversation.input.permission":         true,
	"conversation.input.model":              true,
	"conversation.input.managed":            true,
	"conversation.input.attachments":        true,
	"conversation.chat.node":                true,
	"conversation.chat.turnTail":            true,
	"conversation.chat.assistant-actions":   true,
	"conversation.chat.user-actions":        true,
	"conversation.chat.user-editor":         true,
	"conversation.chat.commandview":         true,
	"conversation.approval.detail":          true,
	"conversation.approval.actions":         true,
	"conversation.plan-review.actions":      true,
	"tool.call.toolview":                    true,
	"tool.call.images":                      true,
}

func validSessionOutlets(rows []SessionOutlet) bool {
	if len(rows) > 512 {
		return false
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.HasSeq && (row.Seq < 0 || row.Seq > 1<<53-1 || math.Trunc(row.Seq) != row.Seq) || !row.HasSeq && row.Seq != 0 {
			return false
		}
		if !validNativeNodes(row.Native) {
			return false
		}
		if row.Handle == "" || len(row.Handle) > 64 || seen[row.Handle] || !sessionSlotNames[row.Slot] || len(row.Label) > 8192 || len(row.CallID) > 512 || len(row.AgentID) > 8192 || len(row.SourceRequestID) > 256<<10 || len(row.PendingKey) > 8192 || len(row.EntryID) > 8192 || len(row.Registrant) > 8192 || row.Turn < 0 {
			return false
		}
		if row.TurnTailSeq > 1<<53-1 || row.HasTurnTail && (row.Slot != "conversation.chat.turnTail" || !row.HasTurn) || !row.HasTurnTail && row.TurnTailSeq != 0 {
			return false
		}
		seen[row.Handle] = true
	}
	return true
}

// SessionTurnTail is a durable completed boundary, never a display ordinal.
type SessionTurnTail struct {
	Turn int    `json:"turn"`
	Seq  uint64 `json:"seq"`
}

// ShowSessionOutlet requests an existing occurrence from one exact Session/page
// revision. The browser repeats its owner/connected-node checks before showing.
func (o *Owner) ShowSessionOutlet(page, revision uint64, subject Subject, handle string) error {
	return o.showSessionOutlet(page, revision, subject, handle, nil, "")
}

// ShowSessionTurnTail additionally binds a native transcript gesture to the
// canonical turn/end coordinates of that original rendered occurrence.
func (o *Owner) ShowSessionTurnTail(page, revision uint64, subject Subject, handle string, tail SessionTurnTail, request string) error {
	if request == "" || len(request) > 64 {
		return errSessionUnavailable
	}
	return o.showSessionOutlet(page, revision, subject, handle, &tail, request)
}

func (o *Owner) showSessionOutlet(page, revision uint64, subject Subject, handle string, tail *SessionTurnTail, request string) error {
	if o == nil {
		return errSessionUnavailable
	}
	o.mu.Lock()
	valid := !o.closed && !o.incompatible && o.state.Unavailable == "" && o.state.Connected && o.state.Page == page && o.state.Revision == revision && o.state.Subject != nil && o.state.Subject.equal(subject) && o.selected.equal(subject) && subject.Kind == "session"
	found := false
	for _, row := range o.state.Outlets {
		if row.Handle == handle && row.Available && (tail == nil || row.Slot == "conversation.chat.turnTail" && row.HasTurnTail && row.HasTurn && row.Turn == tail.Turn && row.TurnTailSeq == tail.Seq) {
			found = true
			break
		}
	}
	// Capture the validated page itself: a replacement connection must never
	// receive a gesture with the previous page's coincidentally equal revision.
	target := o.page
	o.mu.Unlock()
	if !valid || !found || target == nil || target.id != page {
		return errSessionUnavailable
	}
	command := map[string]any{"type": "sessionSurface", "revision": revision, "subject": subject, "handle": handle}
	if tail != nil {
		command["turnTail"] = *tail
		command["request"] = request
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, target.socket, command); err != nil {
		o.disconnect(target)
		return errSessionUnavailable
	}
	return nil
}

type ConversationView struct {
	Handle string `json:"handle"`
	ID     string `json:"id"`
	Label  string `json:"label"`
	Target string `json:"target,omitempty"`
}

func validConversationViews(views []ConversationView) bool {
	if len(views) > 512 {
		return false
	}
	seen := map[string]bool{}
	for _, v := range views {
		if v.Handle == "" || len(v.Handle) > 8192 || v.ID == "" || len(v.ID) > 8192 || len(v.Label) > 8192 || len(v.Target) > 8192 || seen[v.Handle] {
			return false
		}
		seen[v.Handle] = true
	}
	return true
}

func (o *Owner) SelectConversationView(snapshot Snapshot, handle string) error {
	o.mu.Lock()
	page := o.page
	valid := !o.closed && !o.incompatible && snapshot.Subject != nil && snapshot.Subject.Kind == "session" && o.selected.equal(*snapshot.Subject) && o.state.Page == snapshot.Page && o.state.Revision == snapshot.Revision
	o.mu.Unlock()
	if !valid {
		return errSessionUnavailable
	}
	for _, view := range snapshot.ConversationViews {
		if view.Handle == handle {
			return o.sendToPage(page, map[string]any{"type": "conversationView", "revision": snapshot.Revision, "subject": snapshot.Subject, "handle": handle})
		}
	}
	return errSessionUnavailable
}

type RightPaneTab struct {
	Handle string `json:"handle"`
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
}

func validRightPaneTabs(rows []RightPaneTab, outlets []SessionOutlet) bool {
	if len(rows) > 512 {
		return false
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.ID == "" || len(row.ID) > 8192 || len(row.Label) > 8192 || len(row.Kind) > 8192 || seen[row.Handle] {
			return false
		}
		found := false
		for _, outlet := range outlets {
			if outlet.Handle == row.Handle && outlet.Slot == "sidebar.right.pane.tab" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
		seen[row.Handle] = true
	}
	return true
}
func (o *Owner) SelectRightPaneTab(snapshot Snapshot, handle string) error {
	o.mu.Lock()
	page := o.page
	valid := !o.closed && !o.incompatible && snapshot.Subject != nil && snapshot.Subject.Kind == "session" && o.selected.equal(*snapshot.Subject) && o.state.Page == snapshot.Page && o.state.Revision == snapshot.Revision
	o.mu.Unlock()
	if !valid {
		return errSessionUnavailable
	}
	for _, tab := range snapshot.RightPaneTabs {
		if tab.Handle == handle {
			return o.sendToPage(page, map[string]any{"type": "rightPaneTab", "revision": snapshot.Revision, "subject": snapshot.Subject, "handle": handle})
		}
	}
	return errSessionUnavailable
}

// NativeRightPaneTab keeps the public shell integration name stable.
type NativeRightPaneTab = RightPaneTab

func (o *Owner) CloseRightPaneTab(snapshot Snapshot, handle string) error {
	o.mu.Lock()
	page := o.page
	valid := !o.closed && !o.incompatible && snapshot.Subject != nil && snapshot.Subject.Kind == "session" && o.selected.equal(*snapshot.Subject) && o.state.Page == snapshot.Page && o.state.Revision == snapshot.Revision
	o.mu.Unlock()
	if !valid {
		return errSessionUnavailable
	}
	for _, tab := range snapshot.RightPaneTabs {
		if tab.Handle == handle {
			return o.sendToPage(page, map[string]any{"type": "closeRightPaneTab", "revision": snapshot.Revision, "subject": snapshot.Subject, "handle": handle})
		}
	}
	return errSessionUnavailable
}
