package session

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"sync"
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
	State        string // preparing, running, ok, error, stopped
	ParentIssue  string
	Call         *Message
	Result       *Message
	Events       []Message
	Children     []*ToolCallView
}

type toolLifecyclePart struct {
	id, parent, root, name, args, state string
	terminal                            bool
	eventType                           string
	seq                                 float64
	time, turn, step                    *float64
	surfaceOp                           json.RawMessage
	streamValid, jsonValid              bool
}

type toolLifecycleNode struct {
	view                 *ToolCallView
	first, parent        int
	callPart, resultPart toolLifecyclePart
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
				nodes[index].resultPart = part
			}
		} else if v.Call == nil || source.Seq < v.Call.Seq {
			v.Call = &source
			nodes[index].callPart = part
		}
	}
	if len(nodes) == 0 {
		return append([]Message(nil), messages...)
	}
	for i := range nodes {
		v := nodes[i].view
		// Starts own the call name/arguments. PTC result-only windows carry them too.
		if v.Call != nil || v.Result != nil {
			p := nodes[i].callPart
			if v.Call == nil {
				p = nodes[i].resultPart
			}
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

type toolLifecycleMemoKey struct {
	seq    float64
	rawLen int
}

type toolLifecycleMemoEntry struct {
	raw  string
	part toolLifecyclePart
	ok   bool
}

var toolLifecycleMemo = struct {
	mu       sync.Mutex
	m        map[toolLifecycleMemoKey]toolLifecycleMemoEntry
	rawBytes int
}{
	m: make(map[toolLifecycleMemoKey]toolLifecycleMemoEntry),
}

const (
	toolLifecycleMemoLimit     = 16384
	toolLifecycleMemoByteLimit = 16 << 20
)

func readToolLifecycleMemo(seq float64, raw json.RawMessage) (toolLifecyclePart, bool) {
	if len(raw) == 0 {
		return toolLifecyclePart{}, false
	}
	cacheable := !math.IsNaN(seq) && len(raw) <= toolLifecycleMemoByteLimit
	key := toolLifecycleMemoKey{seq: seq, rawLen: len(raw)}
	if cacheable {
		toolLifecycleMemo.mu.Lock()
		if entry, exists := toolLifecycleMemo.m[key]; exists && bytes.Equal([]byte(entry.raw), raw) {
			toolLifecycleMemo.mu.Unlock()
			return entry.part, entry.ok
		}
		toolLifecycleMemo.mu.Unlock()
	}
	part, ok := decodeToolLifecycleRaw(raw)
	if cacheable {
		entry := toolLifecycleMemoEntry{raw: string(raw), part: part, ok: ok}
		toolLifecycleMemo.mu.Lock()
		prev, replacing := toolLifecycleMemo.m[key]
		retained := toolLifecycleMemo.rawBytes - len(prev.raw)
		if (!replacing && len(toolLifecycleMemo.m) >= toolLifecycleMemoLimit) || retained+len(entry.raw) > toolLifecycleMemoByteLimit {
			toolLifecycleMemo.m = make(map[toolLifecycleMemoKey]toolLifecycleMemoEntry)
			retained = 0
		}
		toolLifecycleMemo.m[key] = entry
		toolLifecycleMemo.rawBytes = retained + len(entry.raw)
		toolLifecycleMemo.mu.Unlock()
	}
	return part, ok
}

func readToolLifecyclePart(msg Message) (toolLifecyclePart, bool) {
	part, ok := readToolLifecycleMemo(msg.Seq, msg.Raw)
	if !ok || (msg.EventType != "" && msg.EventType != part.eventType) {
		return toolLifecyclePart{}, false
	}
	return part, true
}

// LookupCachedCoordinates returns turn/step coordinates when raw is already
// memoized by the tool lifecycle cache, avoiding a second JSON unmarshal in chat.
func LookupCachedCoordinates(seq float64, raw json.RawMessage) (turn, step float64, known, cached bool) {
	if len(raw) == 0 || math.IsNaN(seq) || len(raw) > toolLifecycleMemoByteLimit {
		return 0, 0, false, false
	}
	key := toolLifecycleMemoKey{seq: seq, rawLen: len(raw)}
	toolLifecycleMemo.mu.Lock()
	entry, exists := toolLifecycleMemo.m[key]
	if !exists || !bytes.Equal([]byte(entry.raw), raw) {
		toolLifecycleMemo.mu.Unlock()
		return 0, 0, false, false
	}
	part := entry.part
	toolLifecycleMemo.mu.Unlock()
	if !part.jsonValid {
		return 0, 0, false, true
	}
	if part.turn != nil {
		return *part.turn, promptCoordinate(part.step), true, true
	}
	return 0, 0, false, true
}

func decodeToolLifecycleRaw(raw json.RawMessage) (toolLifecyclePart, bool) {
	var ev struct {
		Type      string          `json:"type"`
		Seq       float64         `json:"seq"`
		Time      *float64        `json:"time"`
		SurfaceOp json.RawMessage `json:"surfaceOp"`
		Data      struct {
			Turn         *float64        `json:"turn"`
			Step         *float64        `json:"step"`
			Stream       json.RawMessage `json:"stream"`
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
	if json.Unmarshal(raw, &ev) != nil {
		return toolLifecyclePart{}, false
	}
	d := ev.Data
	p := toolLifecyclePart{
		name:        d.Name,
		state:       "running",
		eventType:   ev.Type,
		seq:         ev.Seq,
		time:        cloneFloat64Ptr(ev.Time),
		turn:        cloneFloat64Ptr(d.Turn),
		step:        cloneFloat64Ptr(d.Step),
		surfaceOp:   cloneRawMessage(ev.SurfaceOp),
		streamValid: validRawJSONArrayOrNull(d.Stream),
		jsonValid:   true,
	}
	switch ev.Type {
	case "tool/call":
		p.id = d.CallID
	case "tool/result":
		p.id = d.Message.ToolCallID
		if p.id == "" {
			p.id = d.Message.Source.CallID
		}
		if d.Message.ToolCallID != "" && d.Message.Source.CallID != "" && d.Message.ToolCallID != d.Message.Source.CallID {
			return p, false
		}
		p.terminal = true
	case "tool/ptc-dispatch-start", "tool/ptc-dispatch":
		p.id, p.parent, p.root = d.SubCallID, d.ParentCallID, d.RootCallID
		p.terminal = ev.Type == "tool/ptc-dispatch"
	default:
		return p, false
	}
	if strings.TrimSpace(p.id) == "" {
		return p, false
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
