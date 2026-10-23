package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

type inspectorToolKey struct{ owner, call string }

type inspectorToolBox struct {
	call *session.ToolCallView
	rect ui.Rect
}

// A call's native wrapper contains its card and complete descendant subtree.
type inspectorToolTree struct {
	owner session.Message
	boxes []*inspectorToolBox
}

type inspectorChatRegion struct{ top, height float32 }

func inspectorToolInMessage(msg session.Message, callID string) *session.ToolCallView {
	if callID == "" {
		return nil
	}
	stack := []*session.ToolCallView{msg.Tool}
	seen := map[*session.ToolCallView]bool{}
	for len(stack) > 0 {
		call := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if call == nil || seen[call] {
			continue
		}
		seen[call] = true
		if call.CallID == callID {
			return call
		}
		for i := len(call.Children) - 1; i >= 0; i-- {
			stack = append(stack, call.Children[i])
		}
	}
	return nil
}

// Full layout geometry can place a child outside the clip of a tall virtual
// item. Only branches belonging to a painted owner enter the pointer hit list.
func inspectorChatToolGeometry(n *InspectorNavigation) {
	r := &n.reveal
	clear(r.toolFull)
	clear(r.toolRows)
	clear(n.picker.tools)
	n.picker.tools = n.picker.tools[:0]
	for _, tree := range r.toolTrees {
		// Reverse preorder places descendants before their containing ancestors.
		for i := len(tree.boxes) - 1; i >= 0; i-- {
			box := tree.boxes[i]
			key := inspectorToolKey{tree.owner.ID, box.call.CallID}
			r.toolFull[key] = box.rect
			if r.rows[tree.owner.ID].W <= 0 {
				continue
			}
			clipped := inspectorChatIntersect(box.rect, r.viewport)
			if clipped.W > 0 && clipped.H > 0 {
				r.toolRows[key] = clipped
				n.picker.tools = append(n.picker.tools, inspectorChatPick{Message: tree.owner, call: box.call, box: clipped})
			}
		}
	}
}

func inspectorChatRevealBox(n *InspectorNavigation) ui.Rect {
	r := &n.reveal
	if r.callID != "" {
		if box, exists := r.toolFull[inspectorToolKey{n.Highlight, r.callID}]; exists {
			return inspectorChatIntersect(box, r.viewport)
		}
	}
	return inspectorChatIntersect(r.rows[n.Highlight], r.viewport)
}

func inspectorChatRevealRegion(n *InspectorNavigation) *inspectorChatRegion {
	r := &n.reveal
	if r.callID == "" {
		return nil
	}
	box, exists := r.toolFull[inspectorToolKey{n.Highlight, r.callID}]
	owner := r.layoutRows[n.Highlight]
	if !exists || box.W <= 0 || box.H <= 0 || owner.H <= 0 {
		return nil
	}
	return &inspectorChatRegion{top: max(float32(0), box.Y-owner.Y), height: box.H}
}
