package chat

import (
	"bytes"
	"encoding/json"
	"math"
	"path"
	"sort"
	"strings"
	"sync"

	"whaleisle/internal/store/session"
)

type mentionEvent struct {
	sessionLevel bool
	Type         string          `json:"type"`
	Seq          float64         `json:"seq"`
	SurfaceOp    json.RawMessage `json:"surfaceOp"`
	Data         struct {
		Turn        *float64        `json:"turn"`
		CallID      string          `json:"callId"`
		Name        string          `json:"name"`
		Arguments   json.RawMessage `json:"arguments"`
		Meta        json.RawMessage `json:"meta"`
		IsError     bool            `json:"isError"`
		Error       json.RawMessage `json:"error"`
		Interrupted bool            `json:"interrupted"`
		Files       []struct {
			Path string `json:"path"`
		} `json:"files"`
		Message struct {
			IsError bool `json:"isError"`
			Source  struct {
				CallID string `json:"callId"`
			} `json:"source"`
		} `json:"message"`
	} `json:"data"`
}

type mentionEventMemoEntry struct {
	raw   string
	event mentionEvent
	ok    bool
}

const mentionEventMemoByteLimit = 16 << 20

var mentionEventMemo = struct {
	mu       sync.Mutex
	m        map[coordinateKey]mentionEventMemoEntry
	rawBytes int
}{
	m: make(map[coordinateKey]mentionEventMemoEntry),
}

func decodeMentionEvent(m session.Message) (mentionEvent, bool) {
	if len(m.Raw) == 0 {
		return mentionEvent{}, false
	}
	cacheable := !math.IsNaN(m.Seq) && len(m.Raw) <= mentionEventMemoByteLimit
	key := coordinateKey{id: m.ID, seq: m.Seq, raw: len(m.Raw)}
	if cacheable {
		mentionEventMemo.mu.Lock()
		if entry, exists := mentionEventMemo.m[key]; exists && bytes.Equal([]byte(entry.raw), m.Raw) {
			mentionEventMemo.mu.Unlock()
			return entry.event, entry.ok
		}
		mentionEventMemo.mu.Unlock()
	}
	var e mentionEvent
	ok := json.Unmarshal(m.Raw, &e) == nil
	if ok && e.Type == "tool/ptc-dispatch" && e.Data.Turn == nil {
		var coordinates struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(m.Raw, &coordinates)
		e.sessionLevel = string(coordinates.Data["turn"]) == "null"
	}
	if cacheable {
		entry := mentionEventMemoEntry{raw: string(m.Raw), event: e, ok: ok}
		mentionEventMemo.mu.Lock()
		prev, replacing := mentionEventMemo.m[key]
		retained := mentionEventMemo.rawBytes - len(prev.raw)
		if (!replacing && len(mentionEventMemo.m) >= memoLimit) || retained+len(entry.raw) > mentionEventMemoByteLimit {
			mentionEventMemo.m = make(map[coordinateKey]mentionEventMemoEntry)
			retained = 0
		}
		mentionEventMemo.m[key] = entry
		mentionEventMemo.rawBytes = retained + len(entry.raw)
		mentionEventMemo.mu.Unlock()
	}
	return e, ok
}

// FileMentionResolver uses recorded turn locations and completed mutation facts.
// PTC inherits the wire location index, never a projection's TurnSeq, rootCallId,
// selected turn, or a path guessed from closing prose.
// FileMentionResolver answers, for one closing answer, which files it mentions.
// The per-pass answers are memoized by render_memo.go.
func FileMentionResolver(messages []session.Message, closing session.Message) func(string) string {
	return memoizedMentionResolver(messages, closing)
}

func buildMentionResolver(messages []session.Message, closing session.Message) func(string) string {
	return mentionResolverFrom(groupMentionFacts(messages), closing)
}

// buildMentionProjectedResolver answers from the events one projection holds, so
// the rows of that projection share a single grouping of them.
func buildMentionProjectedResolver(key session.ProjectionKey, messages []session.Message, closing session.Message) func(string) string {
	return mentionResolverFrom(mentionFactsFor(key, messages), closing)
}

// mentionFactsMemo keeps the last projection's events grouped by turn. Every
// visible row asks for its own closing message, and each answer needs the facts
// of one turn: walking and copying the whole journal per row costs a long
// session tens of megabytes in the frame it opens.
var mentionFactsMemo struct {
	mu     sync.Mutex
	key    session.ProjectionKey
	byTurn map[float64][]mentionEvent
	ready  bool
}

func mentionFactsFor(key session.ProjectionKey, messages []session.Message) map[float64][]mentionEvent {
	if !key.Valid() {
		return groupMentionFacts(messages)
	}
	mentionFactsMemo.mu.Lock()
	if mentionFactsMemo.ready && mentionFactsMemo.key == key {
		byTurn := mentionFactsMemo.byTurn
		mentionFactsMemo.mu.Unlock()
		return byTurn
	}
	mentionFactsMemo.mu.Unlock()
	byTurn := groupMentionFacts(messages)
	mentionFactsMemo.mu.Lock()
	if !mentionFactsMemo.ready || mentionFactsMemo.key != key {
		mentionFactsMemo.key, mentionFactsMemo.byTurn, mentionFactsMemo.ready = key, byTurn, true
	}
	mentionFactsMemo.mu.Unlock()
	return byTurn
}

// groupMentionFacts reads the journal once and files each event under the turn
// that owns it. A tool dispatch without a turn of its own belongs to the turn
// open when it arrives, which is the same cursor each row used to keep.
func groupMentionFacts(messages []session.Message) map[float64][]mentionEvent {
	var events []mentionEvent
	for _, m := range messages {
		if e, ok := decodeMentionEvent(m); ok && e.Seq > 0 {
			events = append(events, e)
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Seq < events[j].Seq })
	byTurn := map[float64][]mentionEvent{}
	var currentTurn *float64
	for _, e := range events {
		// ConversationLocationIndex updates its cursor from explicit Turn/Step
		// coordinates and retires it at the matching turn/end.
		if t := e.Data.Turn; t != nil && *t >= 0 && *t <= 9007199254740991 && math.Trunc(*t) == *t {
			currentTurn = t
		}
		turn := e.Data.Turn
		if e.Type == "tool/ptc-dispatch" && turn == nil && !e.sessionLevel {
			turn = currentTurn
		}
		if turn != nil {
			byTurn[*turn] = append(byTurn[*turn], e)
		}
		if e.Type == "turn/end" && currentTurn != nil && e.Data.Turn != nil && *currentTurn == *e.Data.Turn {
			currentTurn = nil
		}
	}
	return byTurn
}

func mentionResolverFrom(byTurn map[float64][]mentionEvent, closing session.Message) func(string) string {
	if closing.Role != "assistant" || closing.Streaming {
		return nil
	}
	owner, ok := decodeMentionEvent(closing)
	if !ok || owner.Type != "assistant/message" || owner.Data.Turn == nil || owner.Seq <= 0 || owner.Data.Interrupted {
		return nil
	}
	// A turn's events are in seq order, and an answer can only mention what it
	// could already read: the first event past the closing message ends it.
	facts := byTurn[*owner.Data.Turn]
	preceding := facts
	for i, e := range facts {
		if e.Seq > owner.Seq {
			preceding = facts[:i]
			break
		}
	}
	paths := map[string]map[string]bool{}
	add := func(target, argument string) {
		if target == "" {
			return
		}
		for _, token := range []string{target, path.Base(strings.ReplaceAll(target, `\`, "/")), argument} {
			if token == "" {
				continue
			}
			if paths[token] == nil {
				paths[token] = map[string]bool{}
			}
			paths[token][target] = true
		}
	}
	type call struct {
		path string
		seq  float64
	}
	calls := map[string]call{}
	for _, e := range preceding {
		if e.Type == "tool/call" && e.Data.CallID != "" {
			p := mutationMentionPath(e.Data.Name, rootMentionArguments(e.Data.Arguments))
			// Duplicate call IDs are not enough authority to choose one file.
			if _, exists := calls[e.Data.CallID]; exists {
				p = ""
			}
			calls[e.Data.CallID] = call{p, e.Seq}
		}
	}
	for _, e := range preceding {
		switch e.Type {
		case "deliverables/presented":
			for _, f := range e.Data.Files {
				if strings.TrimSpace(f.Path) != "" {
					add(f.Path, "")
				}
			}
		case "tool/ptc-dispatch":
			if !e.Data.IsError {
				argument := mutationMentionPath(e.Data.Name, string(e.Data.Arguments))
				if argument != "" {
					add(mutationResultMentionPath(argument, e.Data.Meta), argument)
				}
			}
		case "tool/result":
			if string(e.SurfaceOp) != `"append"` || e.Data.Message.IsError || (len(e.Data.Error) > 0 && string(e.Data.Error) != "null") {
				continue
			}
			c := calls[e.Data.Message.Source.CallID]
			if c.path != "" && c.seq < e.Seq {
				add(mutationResultMentionPath(c.path, e.Data.Meta), c.path)
			}
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return func(value string) string {
		// Exact paths, basenames and argument aliases all participate in
		// ambiguity, as they do in producedFileMentions.
		if matches := paths[value]; len(matches) == 1 {
			for target := range matches {
				return target
			}
		}
		return ""
	}
}

func rootMentionArguments(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return ""
	}
	return text
}

func absoluteMentionPath(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
}

func mutationResultMentionPath(argument string, raw json.RawMessage) string {
	var meta map[string]json.RawMessage
	if json.Unmarshal(raw, &meta) != nil || meta == nil {
		return argument
	}
	value, exists := meta["path"]
	if !exists {
		return argument
	}
	var target string
	if json.Unmarshal(value, &target) == nil && absoluteMentionPath(target) {
		return target
	}
	if absoluteMentionPath(argument) {
		return argument
	}
	return ""
}

func mutationMentionPath(name, raw string) string {
	var a map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &a) != nil || a == nil {
		return ""
	}
	str := func(key string) (string, bool) {
		var s string
		v, ok := a[key]
		if !ok || string(v) == "null" || json.Unmarshal(v, &s) != nil {
			return "", false
		}
		return s, true
	}
	valid := false
	key := "file_path"
	switch name {
	case "write":
		_, valid = str("content")
	case "edit":
		old, ok := str("old_string")
		next, nok := str("new_string")
		valid = ok && nok && old != "" && old != next
		if v, exists := a["replace_all"]; exists {
			var b bool
			valid = valid && string(v) != "null" && json.Unmarshal(v, &b) == nil
		}
	case "str_replace_editor":
		key = "path"
		command, _ := str("command")
		switch command {
		case "create":
			_, valid = str("file_text")
		case "str_replace":
			old, ok := str("old_str")
			_, nok := str("new_str")
			_, exists := a["new_str"]
			valid = ok && old != "" && (!exists || nok)
		case "insert":
			var line int
			v, exists := a["insert_line"]
			_, ok := str("new_str")
			valid = exists && string(v) != "null" && json.Unmarshal(v, &line) == nil && line >= 0 && ok
		}
	}
	p, ok := str(key)
	if !valid || !ok || strings.TrimSpace(p) == "" {
		return ""
	}
	return p
}

var toolMentionRevisionMemo struct {
	mu      sync.Mutex
	entries []toolMentionRevisionEntry
}

type toolMentionRevisionEntry struct {
	key session.ProjectionKey
	rev uint64
}

func fnv1aString(h uint64, s string) uint64 {
	const prime64 = 1099511628211
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	h ^= 0xff
	h *= prime64
	return h
}

func fileMentionToolRevision(messages []session.Message) uint64 {
	h := uint64(14695981039346656037)
	for _, msg := range messages {
		if msg.Kind != session.MessageKindTool {
			continue
		}
		h = fnv1aString(h, msg.ID)
		h = fnv1aString(h, msg.Body)
		h = fnv1aString(h, string(msg.Raw))
	}
	return h
}

func combineFileMentionRevision(toolRev uint64, closing session.Message) uint64 {
	h := toolRev
	h = fnv1aString(h, closing.ID)
	h = fnv1aString(h, closing.Role)
	h = fnv1aString(h, string(closing.Raw))
	if h == 0 {
		return 1
	}
	return h
}

func fileMentionRevision(messages []session.Message, closing session.Message) uint64 {
	return combineFileMentionRevision(fileMentionToolRevision(messages), closing)
}

func projectedFileMentionRevision(snap session.ViewState, closing session.Message) uint64 {
	key := snap.HistoryProjectionKey()
	if !key.Valid() {
		return fileMentionRevision(snap.Messages, closing)
	}
	toolMentionRevisionMemo.mu.Lock()
	for _, e := range toolMentionRevisionMemo.entries {
		if e.key == key {
			toolMentionRevisionMemo.mu.Unlock()
			return combineFileMentionRevision(e.rev, closing)
		}
	}
	toolMentionRevisionMemo.mu.Unlock()

	toolRev := fileMentionToolRevision(snap.Messages)
	toolMentionRevisionMemo.mu.Lock()
	const maxEntries = 8
	for i, e := range toolMentionRevisionMemo.entries {
		if e.key == key {
			toolMentionRevisionMemo.entries[i].rev = toolRev
			toolMentionRevisionMemo.mu.Unlock()
			return combineFileMentionRevision(toolRev, closing)
		}
	}
	if len(toolMentionRevisionMemo.entries) >= maxEntries {
		copy(toolMentionRevisionMemo.entries, toolMentionRevisionMemo.entries[1:])
		toolMentionRevisionMemo.entries[len(toolMentionRevisionMemo.entries)-1] = toolMentionRevisionEntry{key: key, rev: toolRev}
	} else {
		toolMentionRevisionMemo.entries = append(toolMentionRevisionMemo.entries, toolMentionRevisionEntry{key: key, rev: toolRev})
	}
	toolMentionRevisionMemo.mu.Unlock()
	return combineFileMentionRevision(toolRev, closing)
}
