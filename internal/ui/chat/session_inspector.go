package chat

import (
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/store/session"
)

// InspectorNavigation is an in-memory gesture owned by one visible inspector
// pane. It carries identities only and never fetches or changes session data.
type InspectorNavigation struct {
	Nodes     []session.Message
	Groups    map[string][]string
	SessionID string
	Picking   bool
	Picked    *InspectorTarget
	Target    *InspectorTarget
	Highlight string
	Notice    string
	reveal    inspectorChatReveal
	picker    inspectorChatPicker
}

func prepareSessionInspector(m Model, msgs []session.Message, controls map[string]transcriptControl, list *ui.ListState, projected ...[]session.Message) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	r := &n.reveal
	if r.binding != m.Snapshot.PlanHistoryBinding || r.session != m.Snapshot.SelectedID {
		if r.binding != "" || r.session != "" {
			n.CancelPicking()
			n.picker.area = ui.Rect{}
		}
		r.cancelCenter()
		*r = inspectorChatReveal{binding: m.Snapshot.PlanHistoryBinding, session: m.Snapshot.SelectedID}
		n.Highlight = ""
	}
	if n.Picking {
		n.CancelReveal()
	}
	if n.Target != nil && n.Target != r.request {
		r.cancelCenter()
		n.Highlight = ""
		r.request = n.Target
		r.rejected = nil
		r.pending, r.waiting, r.locating, r.centerQueued, r.centerCommitted = false, false, false, false, false
		r.started, r.quietSince, r.settledAt, r.flashAfter = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	}
	if len(projected) > 0 {
		// Production Chat already projected these exact source occurrences.
		// Copy before appending folded headers, retaining the source slice.
		n.Nodes = append([]session.Message(nil), projected[0]...)
	} else {
		n.Nodes = session.ProjectTurnDeliveryMessages(session.ProjectToolMessages(session.ProjectWorkflowMessages(session.WithSubmittedPlanCards(m.Snapshot.Messages), m.Snapshot.WorkflowRuns)))
	}
	n.Groups = map[string][]string{}
	for _, msg := range msgs {
		if control, ok := controls[msg.ID]; ok {
			n.Nodes = append(n.Nodes, msg)
			n.Groups[msg.ID] = append([]string(nil), control.members...)
		}
	}
	if n.Target == nil && r.pending && r.request != nil && r.request.LiveAssistant && inspectorMessageExactIndex(n.Nodes, n.Highlight) < 0 {
		if rebound := inspectorRebindLiveTarget(*r.request, n.Nodes); rebound.ID != r.request.ID {
			n.Target = &rebound
			r.cancelCenter()
			r.centerQueued, r.centerCommitted = false, false
			r.quietSince, r.settledAt, r.flashAfter = time.Time{}, time.Time{}, time.Time{}
		}
	}
	if n.Target == nil {
		if r.pending && (r.centerQueued || r.centerList != nil && !r.centerCommitted) {
			r.centerQueued = false
			// Resolve the key immediately before List layout. Source prepends
			// or cancellation cannot leave a stale ordinal request in the list.
			if i := inspectorMessageExactIndex(msgs, n.Highlight); i >= 0 {
				r.centerList = list
				scale := r.scale
				if scale <= 0 {
					scale = 1
				}
				list.ScrollToAnimated(i, ui.Center, ui.ScrollAnimation{Duration: func(d float32) time.Duration { return inspectorChatScrollDuration(d * scale) }, Ease: inspectorChatScrollEase})
				r.waiting = true
				r.lastScroll, r.scrollMoved = r.offset, false
				r.quietSince, r.settledAt, r.flashAfter = time.Time{}, time.Time{}, time.Time{}
			} else {
				r.cancelCenter()
				r.waiting = false
			}
		}
		return
	}
	target := *n.Target
	// Resolve before folding, so a raw tool result or Assistant part unfolds
	// the same occurrence that will be highlighted after the next projection.
	loaded := append([]session.Message(nil), n.Nodes...)
	seen := make(map[string]bool, len(loaded))
	for _, msg := range loaded {
		seen[msg.ID] = true
	}
	for _, msg := range msgs {
		if !seen[msg.ID] {
			loaded = append(loaded, msg)
			seen[msg.ID] = true
		}
	}
	target = inspectorRebindLiveTarget(target, loaded)
	displayed := visibleChatMessages(loaded)
	candidates := inspectorRevealTargets(target, inspectorMessageNodes(loaded, displayed))
	occurrences := inspectorRevealOccurrences(loaded, candidates)
	usable := make(map[string]bool, len(displayed))
	for _, msg := range displayed {
		usable[msg.ID] = true
	}
	owner := ""
	for _, id := range occurrences {
		if usable[id] && !r.rejected[id] {
			owner = id
			break
		}
	}
	for _, control := range controls {
		match := false
		for _, id := range control.members {
			if id == owner {
				match = true
			}
		}
		if match && !control.open && m.ThinkingExpanded != nil {
			if *m.ThinkingExpanded == nil {
				*m.ThinkingExpanded = map[string]bool{}
			}
			(*m.ThinkingExpanded)[control.key] = true
			return
		}
	}
	n.Target = nil
	r.pending = true
	r.waiting, r.locating = false, false
	n.Notice = ""
	if i := inspectorMessageExactIndex(msgs, owner); i >= 0 {
		if msgs[i].Kind == session.MessageKindThinking && m.ThinkingExpanded != nil && !isThinkingExpanded(m.ThinkingExpanded, msgs[i].ID) {
			if *m.ThinkingExpanded == nil {
				*m.ThinkingExpanded = map[string]bool{}
			}
			(*m.ThinkingExpanded)[msgs[i].ID] = true
			n.Target = r.request
			r.pending = false
			return
		}
		n.Highlight = msgs[i].ID
		r.locating = true
	} else {
		// The original reveals the displayed Chat column when no loaded
		// occurrence can provide a usable box, without adding a status row.
		n.Highlight = ""
	}
}

// PickTarget keeps a transcript header a group identity. Its displayed sequence
// is presentation metadata; log mode resolves placement from the first member.
func (n *InspectorNavigation) PickTarget(msg session.Message) InspectorTarget {
	if _, ok := n.Groups[msg.ID]; ok {
		return InspectorTarget{GroupKey: msg.ID}
	}
	return msg.InspectorTarget()
}

func inspectorMessageIDs(messages []session.Message) []string {
	ids := make([]string, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
	}
	return ids
}
