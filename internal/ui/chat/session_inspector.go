package chat

import (
	"github.com/egoist/mygo/ui"
	"math"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

// InspectorNavigation is an in-memory gesture owned by one visible inspector
// pane. It carries identities only and never fetches or changes session data.
type InspectorTarget struct {
	ID  string
	Seq float64
}
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
}

func inspectorMessageIndex(msgs []session.Message, target InspectorTarget) int {
	for i, m := range msgs {
		if target.ID != "" && (m.ID == target.ID || m.SourceMessageID == target.ID) {
			return i
		}
	}
	if target.Seq <= 0 {
		return -1
	}
	best := -1
	distance := math.MaxFloat64
	for i, m := range msgs {
		seq := m.Seq
		if m.Tool != nil && m.Tool.Call != nil {
			seq = m.Tool.Call.Seq
		}
		if seq <= 0 {
			continue
		}
		d := math.Abs(seq - target.Seq)
		if d < distance {
			best = i
			distance = d
		}
	}
	return best
}
func prepareSessionInspector(m Model, msgs []session.Message, controls map[string]transcriptControl, list *ui.ListState) {
	n := inspectorChatNavigation(m)
	if n == nil {
		return
	}
	r := &n.reveal
	if r.binding != m.Snapshot.PlanHistoryBinding || r.session != m.Snapshot.SelectedID {
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
		r.pending, r.waiting, r.locating, r.centerQueued, r.centerCommitted = false, false, false, false, false
		r.started, r.quietSince, r.settledAt, r.flashAfter = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	}
	n.Nodes = session.ProjectTurnDeliveryMessages(session.ProjectToolMessages(session.ProjectWorkflowMessages(session.WithSubmittedPlanCards(m.Snapshot.Messages), m.Snapshot.WorkflowRuns)))
	n.Groups = map[string][]string{}
	for _, msg := range msgs {
		if control, ok := controls[msg.ID]; ok {
			n.Nodes = append(n.Nodes, msg)
			n.Groups[msg.ID] = append([]string(nil), control.members...)
		}
	}
	if n.Target == nil {
		if r.pending && (r.centerQueued || r.centerList != nil && !r.centerCommitted) {
			r.centerQueued = false
			// Resolve the key immediately before List layout. Source prepends
			// or cancellation cannot leave a stale ordinal request in the list.
			if i := inspectorMessageIndex(msgs, InspectorTarget{ID: n.Highlight}); i >= 0 {
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
	// Reveal folded ancestors through the same native transcript state used by
	// their disclosure buttons, then resolve against the next visible projection.
	for _, control := range controls {
		match := false
		for _, id := range control.members {
			if id == target.ID {
				match = true
			}
		}
		if !match && target.Seq > 0 {
			for _, raw := range m.Snapshot.Messages {
				if raw.Seq == target.Seq {
					for _, id := range control.members {
						if id == raw.ID {
							match = true
						}
					}
				}
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
	if i := inspectorMessageIndex(msgs, target); i >= 0 {
		if msgs[i].Kind == session.MessageKindThinking && m.ThinkingExpanded != nil && !isThinkingExpanded(m.ThinkingExpanded, msgs[i].ID) {
			if *m.ThinkingExpanded == nil {
				*m.ThinkingExpanded = map[string]bool{}
			}
			(*m.ThinkingExpanded)[msgs[i].ID] = true
			n.Target = &target
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
func sessionInspectorChatControl(c *ui.Context, m Model, msg session.Message) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	n := m.Inspector
	if n == nil || n.SessionID != m.Snapshot.SelectedID || m.DisableDeveloperTools {
		return
	}
	if n.Picking {
		if stdButton(c, palette, historyNavigationText("选择此聊天节点", "Pick this chat node"), buttonGhost, false).Clicked() {
			n.Picked = &InspectorTarget{ID: msg.ID, Seq: msg.Seq}
			n.Picking = false
			n.CancelReveal()
		}
	}
}

func inspectorMessageIDs(messages []session.Message) []string {
	ids := make([]string, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
	}
	return ids
}
