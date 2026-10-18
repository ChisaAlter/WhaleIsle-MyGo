package chat

import (
	"github.com/egoist/mygo/ui"
	"math"
	"whaleisle/internal/store/session"
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
	n := m.Inspector
	if n == nil || n.SessionID != m.Snapshot.SelectedID || m.DisableDeveloperTools {
		return
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
	if i := inspectorMessageIndex(msgs, target); i >= 0 {
		list.ScrollIntoView(i)
		n.Highlight = msgs[i].ID
		n.Notice = ""
	} else {
		n.Notice = "No matching loaded chat content"
	}
}
func sessionInspectorChatControl(c *ui.Context, m Model, msg session.Message) {
	n := m.Inspector
	if n == nil || n.SessionID != m.Snapshot.SelectedID || m.DisableDeveloperTools {
		return
	}
	if n.Picking {
		if ui.Button(c, historyNavigationText("选择此聊天节点", "Pick this chat node")).Clicked() {
			n.Picked = &InspectorTarget{ID: msg.ID, Seq: msg.Seq}
			n.Picking = false
			n.Highlight = msg.ID
		}
	}
	if n.Highlight == msg.ID {
		ui.Text(c, historyNavigationText("已定位检查器节点", "Inspector target")).FontSize(11)
	}
}

func inspectorMessageIDs(messages []session.Message) []string {
	ids := make([]string, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
	}
	return ids
}
