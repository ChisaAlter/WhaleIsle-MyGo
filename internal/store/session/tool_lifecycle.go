package session

import (
	"encoding/json"
	"strings"
)

// ToolCallView is a read-only callId projection. Every source event remains in
// Events as well as in the store journal; Call and Result retain full payloads.
// Missing parents stay visible as roots until an older history page arrives.
type ToolCallView struct {
	CallID       string
	ParentCallID string
	RootCallID   string
	Name         string
	Arguments    string
	State        string // running, ok, error, stopped
	ParentIssue  string
	Call         *Message
	Result       *Message
	Events       []Message
	Children     []*ToolCallView
}

type toolLifecyclePart struct {
	id, parent, root, name, args, state string
	terminal                            bool
}

type toolLifecycleNode struct {
	view          *ToolCallView
	first, parent int
}

// ProjectToolMessages groups only recognized, unambiguous wire identities. It
// does not edit the journal, cursor, stream settlement, user edits or commands.
// The chat uses this after all existing projections; other consumers retain the
// original Messages. Equal tool names, arguments and output never imply identity.
func ProjectToolMessages(messages []Message) []Message {
	nodes := make([]toolLifecycleNode, 0)
	byID := make(map[string]int)
	owners := make(map[int]int)
	for i, msg := range messages {
		if msg.Kind != MessageKindTool || msg.Tool != nil {
			continue
		}
		part, ok := readToolLifecyclePart(msg)
		if !ok {
			continue
		}
		index, exists := byID[part.id]
		if !exists {
			index = len(nodes)
			byID[part.id] = index
			nodes = append(nodes, toolLifecycleNode{view: &ToolCallView{CallID: part.id, Name: "tool", State: "running"}, first: i, parent: -1})
		}
		owners[i] = index
		v := nodes[index].view
		source := cloneToolSource(msg)
		v.Events = append(v.Events, source)
		if part.terminal {
			if v.Result == nil || source.Seq >= v.Result.Seq {
				v.Result = &source
				v.State = part.state
			}
		} else if v.Call == nil || source.Seq < v.Call.Seq {
			v.Call = &source
		}
	}
	if len(nodes) == 0 {
		return append([]Message(nil), messages...)
	}
	for i := range nodes {
		v := nodes[i].view
		// Starts own the call name/arguments. PTC result-only windows carry them too.
		source := v.Call
		if source == nil {
			source = v.Result
		}
		if source != nil {
			p, _ := readToolLifecyclePart(*source)
			v.ParentCallID, v.RootCallID = p.parent, p.root
			v.Arguments = p.args
			if p.name != "" {
				v.Name = p.name
			}
		}
		parent := v.ParentCallID
		if parent == "" && v.RootCallID != v.CallID {
			parent = v.RootCallID
		}
		if parent == "" {
			continue
		}
		if parent == v.CallID {
			v.ParentIssue = "self parent"
			continue
		}
		if p, ok := byID[parent]; ok {
			nodes[i].parent = p
		} else {
			v.ParentIssue = "missing parent"
		}
	}
	// Cut a cycle once per component without recursive traversal. All nodes and
	// their raw relation metadata remain available, including corrupt histories.
	colors := make([]uint8, len(nodes))
	for i := range nodes {
		if colors[i] != 0 {
			continue
		}
		path := []int{}
		cur := i
		for cur >= 0 && colors[cur] == 0 {
			colors[cur] = 1
			path = append(path, cur)
			cur = nodes[cur].parent
		}
		if cur >= 0 && colors[cur] == 1 {
			cut := path[len(path)-1]
			nodes[cut].parent = -1
			nodes[cut].view.ParentIssue = "cyclic parent"
		}
		for _, n := range path {
			colors[n] = 2
		}
	}
	for i := range nodes {
		if p := nodes[i].parent; p >= 0 {
			nodes[p].view.Children = append(nodes[p].view.Children, nodes[i].view)
		}
	}
	// Resolve roots with path compression; cap neither data nor depth. Rendering
	// is iterative too, so a very deep valid tree cannot overflow the call stack.
	roots := make([]int, len(nodes))
	for i := range roots {
		roots[i] = -1
	}
	anchors := make(map[int]int)
	for i := range nodes {
		path := []int{}
		cur := i
		for roots[cur] < 0 && nodes[cur].parent >= 0 {
			path = append(path, cur)
			cur = nodes[cur].parent
		}
		root := cur
		if roots[cur] >= 0 {
			root = roots[cur]
		} else {
			roots[cur] = cur
		}
		roots[i] = root
		for _, n := range path {
			roots[n] = root
		}
		first := nodes[i].first
		if previous, ok := anchors[root]; !ok || first < previous {
			anchors[root] = first
		}
	}
	at := make(map[int]int, len(anchors))
	for root, index := range anchors {
		at[index] = root
	}
	out := make([]Message, 0, len(messages))
	for i, msg := range messages {
		if _, owned := owners[i]; !owned {
			out = append(out, msg)
			continue
		}
		root, anchor := at[i]
		if !anchor {
			continue
		}
		v := nodes[root].view
		source := v.Call
		if source == nil {
			source = v.Result
		}
		row := cloneToolSource(*source)
		row.ID = "tool:" + v.CallID
		row.Tool = v
		row.Body = v.Name
		row.Args = v.Arguments
		out = append(out, row)
	}
	return out
}

func cloneToolSource(msg Message) Message {
	msg.Raw = append(json.RawMessage(nil), msg.Raw...)
	msg.Files = append([]string(nil), msg.Files...)
	msg.Tool = nil
	return msg
}

func readToolLifecyclePart(msg Message) (toolLifecyclePart, bool) {
	var ev struct {
		Type string `json:"type"`
		Data struct {
			CallID       string          `json:"callId"`
			SubCallID    string          `json:"subCallId"`
			ParentCallID string          `json:"parentCallId"`
			RootCallID   string          `json:"rootCallId"`
			Name         string          `json:"name"`
			Arguments    json.RawMessage `json:"arguments"`
			IsError      bool            `json:"isError"`
			Error        *struct {
				Code string `json:"code"`
			} `json:"error"`
			Message struct {
				ToolCallID string `json:"toolCallId"`
				IsError    bool   `json:"isError"`
				Source     struct {
					CallID string `json:"callId"`
				} `json:"source"`
			} `json:"message"`
		} `json:"data"`
	}
	if json.Unmarshal(msg.Raw, &ev) != nil || (msg.EventType != "" && msg.EventType != ev.Type) {
		return toolLifecyclePart{}, false
	}
	d := ev.Data
	p := toolLifecyclePart{name: d.Name, state: "running"}
	switch ev.Type {
	case "tool/call":
		p.id = d.CallID
	case "tool/result":
		p.id = d.Message.ToolCallID
		if p.id == "" {
			p.id = d.Message.Source.CallID
		}
		if d.Message.ToolCallID != "" && d.Message.Source.CallID != "" && d.Message.ToolCallID != d.Message.Source.CallID {
			return toolLifecyclePart{}, false
		}
		p.terminal = true
	case "tool/ptc-dispatch-start", "tool/ptc-dispatch":
		p.id, p.parent, p.root = d.SubCallID, d.ParentCallID, d.RootCallID
		p.terminal = ev.Type == "tool/ptc-dispatch"
	default:
		return toolLifecyclePart{}, false
	}
	if strings.TrimSpace(p.id) == "" {
		return toolLifecyclePart{}, false
	}
	if len(d.Arguments) > 0 && string(d.Arguments) != "null" {
		if json.Unmarshal(d.Arguments, &p.args) != nil {
			p.args = string(d.Arguments)
		}
	}
	if p.terminal {
		p.state = "ok"
		if d.IsError || d.Message.IsError || d.Error != nil {
			p.state = "error"
		}
		if d.Error != nil && d.Error.Code == "interrupted" {
			p.state = "stopped"
		}
	}
	return p, true
}
