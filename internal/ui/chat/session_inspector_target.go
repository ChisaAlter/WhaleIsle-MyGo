package chat

import (
	"math"
	"sort"

	"whaleisle/internal/store/session"
)

type InspectorTarget = session.InspectorChatTarget

type inspectorRevealNode struct {
	ID, Kind, RootCallID string
	Seq                  float64
	Turn, Step           *float64
	Visible              bool
}

func inspectorTargetAnchor(target InspectorTarget) *float64 {
	if target.AnchorSeq != nil {
		return target.AnchorSeq
	}
	if target.Seq > 0 {
		return &target.Seq
	}
	return nil
}

func inspectorSameCoordinate(a, b *float64) bool { return a != nil && b != nil && *a == *b }

func inspectorNodeTarget(node inspectorRevealNode) InspectorTarget {
	return InspectorTarget{ID: node.ID, AnchorSeq: &node.Seq, Turn: node.Turn, Step: node.Step}
}

// Model candidates follow the original Inspector's tool/exact/kind/Step/Turn
// order. A requested occurrence keeps its identity even when it is not rendered.
func inspectorRevealTargets(target InspectorTarget, nodes []inspectorRevealNode) []InspectorTarget {
	target.AnchorSeq = inspectorTargetAnchor(target)
	reference, turn, step := target.AnchorSeq, target.Turn, target.Step
	visible := make([]inspectorRevealNode, 0, len(nodes))
	var explicit *inspectorRevealNode
	for i := range nodes {
		if nodes[i].Visible {
			visible = append(visible, nodes[i])
		}
		if target.ID != "" && explicit == nil && nodes[i].ID == target.ID {
			explicit = &nodes[i]
		}
	}
	primary := target
	if explicit != nil {
		if reference == nil {
			reference = &explicit.Seq
		}
		if turn == nil {
			turn = explicit.Turn
		}
		if step == nil {
			step = explicit.Step
		}
		primary.AnchorSeq, primary.Turn, primary.Step = reference, turn, step
	} else if target.ID == "" && reference != nil {
		var tool, exact, kind, inStep, inTurn *inspectorRevealNode
		root := target.RootCallID
		if root == "" {
			root = target.CallID
		}
		for i := range visible {
			n := &visible[i]
			if root != "" && tool == nil && n.Kind == "tool-call" && n.RootCallID == root {
				tool = n
			}
			if turn != nil && !inspectorSameCoordinate(turn, n.Turn) {
				continue
			}
			if inTurn == nil || math.Abs(n.Seq-*reference) < math.Abs(inTurn.Seq-*reference) {
				inTurn = n
			}
			if step != nil && !inspectorSameCoordinate(step, n.Step) {
				continue
			}
			if exact == nil && n.Seq == *reference {
				exact = n
			}
			if kind == nil && target.NodeKind != "" && n.Kind == target.NodeKind {
				kind = n
			}
			if inStep == nil || math.Abs(n.Seq-*reference) < math.Abs(inStep.Seq-*reference) {
				inStep = n
			}
		}
		for _, selected := range []*inspectorRevealNode{tool, exact, kind, inStep, inTurn} {
			if selected != nil {
				primary.ID = selected.ID
				if primary.Turn == nil {
					primary.Turn = selected.Turn
				}
				break
			}
		}
	}
	result := []InspectorTarget{primary}
	if reference == nil {
		return result
	}
	rank := func(node inspectorRevealNode) int {
		sameTurn := turn == nil || inspectorSameCoordinate(turn, node.Turn)
		if sameTurn && step != nil && inspectorSameCoordinate(step, node.Step) {
			return 0
		}
		if sameTurn && turn != nil {
			return 1
		}
		return 2
	}
	sort.SliceStable(visible, func(i, j int) bool {
		if a, b := rank(visible[i]), rank(visible[j]); a != b {
			return a < b
		}
		return math.Abs(visible[i].Seq-*reference) < math.Abs(visible[j].Seq-*reference)
	})
	seen := map[inspectorTargetKey]bool{inspectorTargetIdentity(primary): true}
	for _, node := range visible {
		candidate := inspectorNodeTarget(node)
		key := inspectorTargetIdentity(candidate)
		if !seen[key] {
			seen[key] = true
			result = append(result, candidate)
		}
	}
	return result
}

type inspectorTargetKey struct {
	id, call, root, kind, group, part string
	seq, turn, step                   float64
	hasSeq, hasTurn, hasStep          bool
}

func inspectorTargetIdentity(target InspectorTarget) inspectorTargetKey {
	key := inspectorTargetKey{id: target.ID, call: target.CallID, root: target.RootCallID, kind: target.NodeKind, group: target.GroupKey, part: target.GroupPart}
	if seq := inspectorTargetAnchor(target); seq != nil {
		key.seq, key.hasSeq = *seq, true
	}
	if target.Turn != nil {
		key.turn, key.hasTurn = *target.Turn, true
	}
	if target.Step != nil {
		key.step, key.hasStep = *target.Step, true
	}
	return key
}

func inspectorMessageNodes(messages, displayed []session.Message) []inspectorRevealNode {
	visible := make(map[string]bool, len(displayed))
	for _, msg := range displayed {
		visible[msg.ID] = true
	}
	nodes := make([]inspectorRevealNode, 0, len(messages))
	for _, msg := range messages {
		target := msg.InspectorTarget()
		seq := msg.Seq
		if anchor := inspectorTargetAnchor(target); anchor != nil {
			seq = *anchor
		}
		kind := target.NodeKind
		if kind == "" {
			kind = string(msg.Kind)
		}
		node := inspectorRevealNode{ID: msg.ID, Kind: kind, Seq: seq, Turn: target.Turn, Step: target.Step, Visible: visible[msg.ID] && msg.Kind != transcriptControlKind}
		if msg.Tool != nil {
			node.RootCallID = msg.Tool.CallID
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// Live native block IDs retire at settlement; the original Assistant Step's
// identity survives. Rebind only a request carrying that explicit live owner.
func inspectorRebindLiveTarget(target InspectorTarget, messages []session.Message) InspectorTarget {
	if !target.LiveAssistant || target.ID == "" || inspectorMessageExactIndex(messages, target.ID) >= 0 || target.Turn == nil || target.Step == nil {
		return target
	}
	for _, msg := range messages {
		candidate := msg.InspectorTarget()
		if candidate.NodeKind == "assistant-step" && inspectorSameCoordinate(candidate.Turn, target.Turn) && inspectorSameCoordinate(candidate.Step, target.Step) {
			target.ID = msg.ID
			return target
		}
	}
	return target
}

// Resolve occurrence specificity separately from model ordering. Exact native
// IDs precede durable aliases; a selected part can refine a shared Assistant.
func inspectorRevealOccurrences(messages []session.Message, targets []InspectorTarget) []string {
	byID := make(map[string]int, len(messages))
	aliases := map[string][]int{}
	calls := map[string][]string{}
	for i, msg := range messages {
		if _, exists := byID[msg.ID]; !exists {
			byID[msg.ID] = i
		}
		if msg.SourceMessageID != "" {
			aliases[msg.SourceMessageID] = append(aliases[msg.SourceMessageID], i)
		}
		if msg.Tool != nil {
			stack := []*session.ToolCallView{msg.Tool}
			visited := map[*session.ToolCallView]bool{}
			for len(stack) > 0 {
				last := len(stack) - 1
				tool := stack[last]
				stack = stack[:last]
				if tool == nil || visited[tool] {
					continue
				}
				visited[tool] = true
				calls[tool.CallID] = append(calls[tool.CallID], msg.ID)
				stack = append(stack, tool.Children...)
			}
		}
	}
	var result []string
	seen := map[string]bool{}
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	addCall := func(call string) {
		if call != "" {
			for _, id := range calls[call] {
				add(id)
			}
		}
	}
	metadata := map[int]InspectorTarget{}
	info := func(i int) InspectorTarget {
		if value, ok := metadata[i]; ok {
			return value
		}
		value := messages[i].InspectorTarget()
		metadata[i] = value
		return value
	}
	for _, target := range targets {
		addCall(target.CallID)
		if target.ID != "" {
			selected, exists := byID[target.ID]
			if target.GroupPart != "" {
				source := target.ID
				if exists {
					if info(selected).GroupPart == target.GroupPart {
						add(messages[selected].ID)
					}
					if messages[selected].SourceMessageID != "" {
						source = messages[selected].SourceMessageID
					}
				}
				for _, i := range aliases[source] {
					if info(i).GroupPart == target.GroupPart {
						add(messages[i].ID)
					}
				}
				if exists && messages[selected].Streaming {
					a := info(selected)
					for i, msg := range messages {
						if msg.Streaming {
							b := info(i)
							if a.NodeKind == "assistant-step" && b.NodeKind == "assistant-step" && inspectorSameCoordinate(a.Turn, b.Turn) && inspectorSameCoordinate(a.Step, b.Step) && b.GroupPart == target.GroupPart {
								add(msg.ID)
							}
						}
					}
				}
			}
			if exists {
				add(messages[selected].ID)
			}
			for _, i := range aliases[target.ID] {
				add(messages[i].ID)
			}
		}
		addCall(target.RootCallID)
		if target.GroupKey != "" {
			if i, ok := byID[target.GroupKey]; ok {
				add(messages[i].ID)
			}
		}
	}
	return result
}

func inspectorMessageExactIndex(messages []session.Message, id string) int {
	if id != "" {
		for i, msg := range messages {
			if msg.ID == id {
				return i
			}
		}
	}
	return -1
}

func inspectorMessageIndex(messages []session.Message, target InspectorTarget) int {
	nodes := inspectorMessageNodes(messages, messages)
	for _, id := range inspectorRevealOccurrences(messages, inspectorRevealTargets(target, nodes)) {
		for i, msg := range messages {
			if msg.ID == id {
				return i
			}
		}
	}
	return -1
}
