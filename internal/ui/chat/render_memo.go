package chat

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"

	"whaleisle/internal/store/session"
	"whaleisle/internal/workflowrun"
)

// The transcript renderer asks two per-message questions on every pass: where a
// row sits in its turn, and which files an answer mentions. Both answered by
// decoding every message's raw JSON, so a 150-message session decoded its whole
// journal twice per message per render pass (tmp/stalls, round 2026-10-26).
// Coordinate entries own the exact source bytes they parsed. Size and message
// identity only select a candidate; neither proves the coordinates are unchanged.

type coordinateKey struct {
	id  string
	seq float64
	raw int
}

type coordinateValue struct {
	turn, step float64
	known      bool
}

type coordinateMemoEntry struct {
	coordinateValue
	raw string
}

type coordinateMemoStore struct {
	mu       sync.Mutex
	m        map[coordinateKey]coordinateMemoEntry
	rawBytes int
}

var coordinateMemo = coordinateMemoStore{m: map[coordinateKey]coordinateMemoEntry{}}

const memoLimit = 16384

// chatSourceByteLimit bounds the raw and body bytes of one transcript whose
// source copy projectChatMessageSources retains. A variable so a test can lower it.
var chatSourceByteLimit = 16 << 20

const coordinateMemoByteLimit = 8 << 20

func memoizedTranscriptCoordinates(msg session.Message) (float64, float64, bool) {
	if len(msg.Raw) > 0 {
		value := coordinateMemo.rawCoordinates(msg)
		if value.known {
			return value.turn, value.step, true
		}
	}
	// Cache only the raw record's own coordinates. A projected result's current
	// call chain can change independently, even at equal raw lengths.
	if msg.Tool != nil && msg.Tool.Call != nil {
		return memoizedTranscriptCoordinates(*msg.Tool.Call)
	}
	return 0, 0, false
}

func (memo *coordinateMemoStore) rawCoordinates(msg session.Message) coordinateValue {
	if turn, step, known, cached := session.LookupCachedCoordinates(msg.Seq, msg.Raw); cached {
		return coordinateValue{turn: turn, step: step, known: known}
	}
	key := coordinateKey{id: msg.ID, seq: msg.Seq, raw: len(msg.Raw)}
	memo.mu.Lock()
	if entry, ok := memo.m[key]; ok && bytes.Equal([]byte(entry.raw), msg.Raw) {
		memo.mu.Unlock()
		return entry.coordinateValue
	}
	memo.mu.Unlock()
	value := coordinateValue{}
	if len(msg.Raw) > 0 && (bytes.Contains(msg.Raw, []byte("turn")) || bytes.Contains(msg.Raw, []byte(`\u`))) {
		var event struct {
			Data struct {
				Turn *float64 `json:"turn"`
				Step float64  `json:"step"`
			} `json:"data"`
		}
		_ = json.Unmarshal(msg.Raw, &event)
		if event.Data.Turn != nil {
			value = coordinateValue{turn: *event.Data.Turn, step: event.Data.Step, known: true}
		}
	}
	// An oversized event is decoded but never retained and does not evict the
	// useful small entries. Cold misses own their bytes; warm hits allocate none.
	if len(msg.Raw) > coordinateMemoByteLimit {
		return value
	}
	entry := coordinateMemoEntry{coordinateValue: value, raw: string(msg.Raw)}
	memo.mu.Lock()
	previous, replacing := memo.m[key]
	retained := memo.rawBytes - len(previous.raw)
	if !replacing && len(memo.m) >= memoLimit || retained+len(entry.raw) > coordinateMemoByteLimit {
		memo.m = make(map[coordinateKey]coordinateMemoEntry)
		retained = 0
	}
	memo.m[key] = entry
	memo.rawBytes = retained + len(entry.raw)
	memo.mu.Unlock()
	return value
}

type mentionMemoKey struct {
	raw       string
	role      string
	streaming bool
}

type mentionMemoEntry struct {
	key     mentionMemoKey
	resolve func(string) string
}

type mentionMemoStore struct {
	mu          sync.Mutex
	source      []string
	entries     []mentionMemoEntry // Most recently used first, including nil results.
	sourceBytes int
	rawBytes    int
	generation  uint64
	ready       bool
}

const mentionMemoEntryLimit = 16

// This bounds retained journal and closing raw bytes, not the resolver's parsed
// path maps, slice headers or caller-retained functions.
const mentionMemoByteLimit = 8 << 20

var mentionMemo = mentionMemoStore{}

func memoizedMentionResolver(messages []session.Message, closing session.Message) func(string) string {
	return mentionMemo.resolver(messages, closing, buildMentionResolver)
}

func (memo *mentionMemoStore) resolver(messages []session.Message, closing session.Message, build func([]session.Message, session.Message) func(string) string) func(string) string {
	if len(closing.Raw) > mentionMemoByteLimit {
		return build(messages, closing)
	}
	memo.mu.Lock()
	if !memo.ready || !sameRawSources(memo.source, messages) {
		observedGeneration := memo.generation
		memo.mu.Unlock()
		sourceBytes := 0
		for _, message := range messages {
			if len(message.Raw) > mentionMemoByteLimit-len(closing.Raw)-sourceBytes {
				// An oversized source/closing pair leaves the useful journal warm.
				return build(messages, closing)
			}
			sourceBytes += len(message.Raw)
		}
		// Only a journal change owns a new source snapshot. Copying and resolver
		// construction both run outside the mutex.
		source := make([]string, len(messages))
		for i := range messages {
			source[i] = string(messages[i].Raw)
		}
		memo.mu.Lock()
		if memo.generation != observedGeneration && !sameRawSources(memo.source, messages) {
			memo.mu.Unlock()
			return build(messages, closing)
		}
		if !memo.ready || !sameRawSources(memo.source, messages) {
			memo.source, memo.sourceBytes, memo.rawBytes = source, sourceBytes, sourceBytes
			clear(memo.entries)
			memo.entries = memo.entries[:0]
			memo.generation++
			memo.ready = true
		}
	}
	if len(closing.Raw) > mentionMemoByteLimit-memo.sourceBytes {
		memo.mu.Unlock()
		return build(messages, closing)
	}
	if index := memo.findClosing(closing); index >= 0 {
		fn := memo.touch(index)
		memo.mu.Unlock()
		return fn
	}
	generation := memo.generation
	memo.mu.Unlock()

	fn := build(messages, closing)
	key := mentionMemoKey{raw: string(closing.Raw), role: closing.Role, streaming: closing.Streaming}
	memo.mu.Lock()
	defer memo.mu.Unlock()
	// A slower earlier view must not repopulate a newer journal's entries.
	if memo.generation != generation {
		return fn
	}
	if index := memo.findClosing(closing); index >= 0 {
		return memo.touch(index)
	}
	for len(memo.entries) >= mentionMemoEntryLimit || memo.rawBytes+len(key.raw) > mentionMemoByteLimit {
		last := len(memo.entries) - 1
		memo.rawBytes -= len(memo.entries[last].key.raw)
		memo.entries[last] = mentionMemoEntry{}
		memo.entries = memo.entries[:last]
	}
	memo.entries = append(memo.entries, mentionMemoEntry{})
	copy(memo.entries[1:], memo.entries[:len(memo.entries)-1])
	memo.entries[0] = mentionMemoEntry{key: key, resolve: fn}
	memo.rawBytes += len(key.raw)
	return fn
}

// findClosing and touch require memo.mu. Exact raw bytes, role and streaming
// state are the identity; IDs and equal lengths never imply the same answer.
func (memo *mentionMemoStore) findClosing(closing session.Message) int {
	for index, entry := range memo.entries {
		if entry.key.role == closing.Role && entry.key.streaming == closing.Streaming && bytes.Equal([]byte(entry.key.raw), closing.Raw) {
			return index
		}
	}
	return -1
}

func (memo *mentionMemoStore) touch(index int) func(string) string {
	entry := memo.entries[index]
	copy(memo.entries[1:index+1], memo.entries[:index])
	memo.entries[0] = entry
	return entry.resolve
}

func sameRawSources(source []string, messages []session.Message) bool {
	if len(source) != len(messages) {
		return false
	}
	for i := range messages {
		if !bytes.Equal([]byte(source[i]), messages[i].Raw) {
			return false
		}
	}
	return true
}

type chatProjectionKey struct {
	count    int
	total    int
	runs     int
	selected string
}

type chatProjectionStore struct {
	mu      sync.Mutex
	key     chatProjectionKey
	source  []chatMessageSource
	runs    []workflowProjectionSource
	project []session.Message
	ready   bool
	// revision is the last projection made for a store revision.
	revision        chatRevisionProjection
	streamingPrefix chatStreamingPrefixMemo
	turnPrefix      chatTurnPrefixMemo
}

type chatStreamingPrefixMemo struct {
	ready          bool
	gen            uint64
	turnPrefixGen  uint64
	visPrefixLen   int
	selected       string
	historyHasMore bool
	prefix         []session.Message
	runs           []workflowProjectionSource
	projectBuf     []session.Message
	visibleBuf     []session.Message
	todo           map[string]*todoBaseline
}

type chatTurnPrefixMemo struct {
	ready          bool
	gen            uint64
	selected       string
	historyHasMore bool
	prefix         []session.Message
	runs           []workflowProjectionSource
	maxSeq         float64
	turns          map[float64]struct{}
	turnSeqs       map[float64]struct{}
	toolCallIDs    map[string]struct{}
	planCallIDs    map[string]struct{}
	projectPrefix  []session.Message
	visiblePrefix  []session.Message
	todo           map[string]*todoBaseline
	todoLast       []session.PlanTodo
	todoKnown      bool
}

// chatRevisionProjection is reused while its store revision stands. The counts
// and edge identities only guard against a view whose Messages were replaced
// after the store projected it; the revision alone fixes the input.
type chatRevisionProjection struct {
	key             session.ProjectionKey
	prefixGen       uint64
	turnPrefixGen   uint64
	visPrefixLen    int
	messages, runs  int
	firstID, lastID string
	project         []session.Message
	// Derived lazily from the same revision.
	visible      []session.Message
	visibleReady bool
	todo         map[string]*todoBaseline
	todoReady    bool
	mentions     map[float64]func(string) string
	editSeq      float64
	editTextOnly bool
	editReady    bool
}

func (r chatRevisionProjection) matches(snap session.ViewState) bool {
	key := snap.HistoryProjectionKey()
	if !r.key.Valid() || r.key != key || r.messages != len(snap.Messages) || r.runs != len(snap.WorkflowRuns) {
		return false
	}
	n := len(snap.Messages)
	return n == 0 || r.firstID == snap.Messages[0].ID && r.lastID == snap.Messages[n-1].ID
}

func isPreparingToolTail(m session.Message) bool {
	if m.Kind != session.MessageKindTool || m.Tool == nil {
		return false
	}
	t := m.Tool
	return t.State == "preparing" && t.Call == nil && t.Result == nil &&
		len(t.Events) == 0 && len(t.Children) == 0 &&
		t.ParentCallID == "" && t.RootCallID == "" && t.ParentIssue == ""
}

func isStreamingTailMessage(m session.Message, runs []workflowrun.Run) bool {
	if !m.Streaming || m.Seq != 0 || len(m.Raw) != 0 {
		return false
	}
	preparingTool := isPreparingToolTail(m)
	if m.Kind != session.MessageKindText && m.Kind != session.MessageKindThinking && !preparingTool {
		return false
	}
	if (!preparingTool && m.Tool != nil) || m.Workflow != nil || m.TurnDelivery != nil || m.Command != nil || m.Submission != nil || m.Retry != nil || m.AutomaticCompaction != nil || m.Web != nil {
		return false
	}
	for _, r := range runs {
		if r.AnchorSeq == 0 {
			return false
		}
		for _, seq := range r.EventSeqs {
			if seq == 0 {
				return false
			}
		}
	}
	return true
}

func sameBytesFast(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 || &a[0] == &b[0] {
		return true
	}
	return bytes.Equal(a, b)
}

func samePrefixMessage(a, b session.Message) bool {
	if a.ID != b.ID || a.Seq != b.Seq || a.Kind != b.Kind || a.Role != b.Role ||
		a.Body != b.Body || a.Streaming != b.Streaming || a.TurnSeq != b.TurnSeq ||
		a.TurnOpening != b.TurnOpening || a.EventType != b.EventType || a.EventTime != b.EventTime ||
		a.ImageRef != b.ImageRef || a.SourceMessageID != b.SourceMessageID ||
		a.Model != b.Model || a.Args != b.Args || a.ContextWaking != b.ContextWaking ||
		!sameProjectionNumber(a.ConversationTurn, b.ConversationTurn) ||
		!sameProjectionNumber(a.ConversationStep, b.ConversationStep) ||
		!sameBytesFast(a.Raw, b.Raw) || !sameBytesFast(a.ContentBlock, b.ContentBlock) ||
		!sameProjectionFiles(a.Files, b.Files) ||
		!sameProjectionTiming(a.TrajectoryTime, b.TrajectoryTime) ||
		!sameProjectionDelivery(a.TurnDelivery, b.TurnDelivery) {
		return false
	}
	if a.AutomaticCompaction != b.AutomaticCompaction || a.Retry != b.Retry ||
		a.Workflow != b.Workflow || a.Submission != b.Submission ||
		a.Tool != b.Tool || a.Command != b.Command || a.Web != b.Web {
		return reflect.DeepEqual(a, b)
	}
	return true
}

func (p *chatStreamingPrefixMemo) matches(snap session.ViewState) bool {
	n := len(snap.Messages)
	if !p.ready || n < 2 || len(p.prefix) != n-1 || p.selected != snap.SelectedID || p.historyHasMore != snap.HistoryHasMore {
		return false
	}
	if !sameWorkflowSources(p.runs, snap.WorkflowRuns) {
		return false
	}
	for i := 0; i < n-1; i++ {
		if !samePrefixMessage(p.prefix[i], snap.Messages[i]) {
			return false
		}
	}
	return true
}

func findLastSettledTurnEndSplit(messages []session.Message) int {
	for i := len(messages) - 2; i >= 0; i-- {
		if messages[i].EventType == "turn/end" && messages[i].Seq > 0 && !messages[i].Streaming {
			return i + 1
		}
	}
	return 0
}

func (p *chatTurnPrefixMemo) matchesPrefix(snap session.ViewState, splitIdx int) bool {
	if !p.ready || splitIdx <= 0 || len(p.prefix) != splitIdx || p.selected != snap.SelectedID || p.historyHasMore != snap.HistoryHasMore {
		return false
	}
	if !sameWorkflowSources(p.runs, snap.WorkflowRuns) {
		return false
	}
	for i := 0; i < splitIdx; i++ {
		if !samePrefixMessage(p.prefix[i], snap.Messages[i]) {
			return false
		}
	}
	return true
}

func (p *chatTurnPrefixMemo) canExtendTo(snap session.ViewState, splitIdx int) bool {
	oldLen := len(p.prefix)
	if !p.ready || oldLen <= 0 || oldLen >= splitIdx || p.selected != snap.SelectedID || p.historyHasMore != snap.HistoryHasMore {
		return false
	}
	if !sameWorkflowSources(p.runs, snap.WorkflowRuns) {
		return false
	}
	for i := 0; i < oldLen; i++ {
		if !samePrefixMessage(p.prefix[i], snap.Messages[i]) {
			return false
		}
	}
	return true
}

func recordToolViewCallIDs(v *session.ToolCallView, dst map[string]struct{}) {
	if v == nil {
		return
	}
	if v.CallID != "" {
		dst[v.CallID] = struct{}{}
	}
	if v.ParentCallID != "" {
		dst[v.ParentCallID] = struct{}{}
	}
	if v.RootCallID != "" {
		dst[v.RootCallID] = struct{}{}
	}
	for _, child := range v.Children {
		recordToolViewCallIDs(child, dst)
	}
}

func hasCrossPrefixToolView(v *session.ToolCallView, prefixIDs map[string]struct{}) bool {
	if v == nil || len(prefixIDs) == 0 {
		return false
	}
	if v.CallID != "" {
		if _, ok := prefixIDs[v.CallID]; ok {
			return true
		}
	}
	if v.ParentCallID != "" {
		if _, ok := prefixIDs[v.ParentCallID]; ok {
			return true
		}
	}
	if v.RootCallID != "" {
		if _, ok := prefixIDs[v.RootCallID]; ok {
			return true
		}
	}
	for _, child := range v.Children {
		if hasCrossPrefixToolView(child, prefixIDs) {
			return true
		}
	}
	return false
}

func (p *chatTurnPrefixMemo) recordSegmentMetadata(rawSeg, projSeg []session.Message) {
	for _, msg := range rawSeg {
		if msg.Seq > p.maxSeq {
			p.maxSeq = msg.Seq
		}
		if msg.TurnSeq > 0 {
			p.turnSeqs[msg.TurnSeq] = struct{}{}
		}
		if turn, _, ok := memoizedTranscriptCoordinates(msg); ok {
			p.turns[turn] = struct{}{}
		}
		if msg.EventType == "tool/call" && bytes.Contains(msg.Raw, []byte("submit_plan")) {
			var ev struct {
				Data struct {
					CallID string `json:"callId"`
					Name   string `json:"name"`
				} `json:"data"`
			}
			if json.Unmarshal(msg.Raw, &ev) == nil && ev.Data.Name == "submit_plan" && ev.Data.CallID != "" {
				p.planCallIDs[ev.Data.CallID] = struct{}{}
				p.toolCallIDs[ev.Data.CallID] = struct{}{}
			}
		}
	}
	for _, msg := range projSeg {
		if msg.TurnSeq > 0 {
			p.turnSeqs[msg.TurnSeq] = struct{}{}
		}
		if turn, _, ok := memoizedTranscriptCoordinates(msg); ok {
			p.turns[turn] = struct{}{}
		}
		if msg.Tool != nil {
			recordToolViewCallIDs(msg.Tool, p.toolCallIDs)
		}
	}
}

func (p *chatTurnPrefixMemo) canIsolateRawSegment(segment []session.Message, runs []workflowrun.Run) bool {
	for _, r := range runs {
		if r.AnchorSeq == 0 || r.AnchorSeq > p.maxSeq {
			return false
		}
		for _, seq := range r.EventSeqs {
			if seq == 0 || seq > p.maxSeq {
				return false
			}
		}
	}
	for _, msg := range segment {
		if msg.Seq < 0 || (msg.Seq > 0 && msg.Seq <= p.maxSeq) {
			return false
		}
		if msg.TurnSeq > 0 {
			if _, exists := p.turnSeqs[msg.TurnSeq]; exists {
				return false
			}
		}
		if turn, _, ok := memoizedTranscriptCoordinates(msg); ok {
			if _, exists := p.turns[turn]; exists {
				return false
			}
		}
		if len(p.planCallIDs) > 0 && msg.EventType == "tool/result" {
			for planID := range p.planCallIDs {
				if bytes.Contains(msg.Raw, []byte(planID)) {
					return false
				}
			}
		}
	}
	return true
}

func (p *chatTurnPrefixMemo) canIsolateProjectedSegment(projSeg []session.Message) bool {
	if len(p.toolCallIDs) == 0 {
		return true
	}
	for _, msg := range projSeg {
		if msg.Tool != nil && hasCrossPrefixToolView(msg.Tool, p.toolCallIDs) {
			return false
		}
	}
	return true
}

func ensureTurnPrefixMemo(snap session.ViewState, splitIdx int) (chatTurnPrefixMemo, bool) {
	cp := chatProjectionOf(snap)
	if splitIdx <= 0 || splitIdx > len(snap.Messages) {
		return chatTurnPrefixMemo{}, false
	}
	cp.mu.Lock()
	if cp.turnPrefix.matchesPrefix(snap, splitIdx) {
		p := cp.turnPrefix
		cp.mu.Unlock()
		return p, true
	}
	if cp.turnPrefix.canExtendTo(snap, splitIdx) {
		base := cp.turnPrefix
		oldSplit := len(base.prefix)
		delta := snap.Messages[oldSplit:splitIdx]
		if base.canIsolateRawSegment(delta, snap.WorkflowRuns) {
			cp.mu.Unlock()
			deltaProj := buildChatMessages(session.ViewState{Messages: delta, SelectedID: snap.SelectedID})
			if base.canIsolateProjectedSegment(deltaProj) {
				for i := range deltaProj {
					deltaProj[i].TurnDelivery = ownProjectionDelivery(deltaProj[i].TurnDelivery)
					deltaProj[i].Files = ownProjectionFiles(deltaProj[i].Files)
				}
				deltaVis := visibleChatMessages(deltaProj)
				cp.mu.Lock()
				if cp.turnPrefix.gen == base.gen && cp.turnPrefix.canExtendTo(snap, splitIdx) {
					p := &cp.turnPrefix
					p.prefix = append(p.prefix[:len(p.prefix):len(p.prefix)], delta...)
					p.prefix = p.prefix[:len(p.prefix):len(p.prefix)]
					p.projectPrefix = append(p.projectPrefix[:len(p.projectPrefix):len(p.projectPrefix)], deltaProj...)
					p.projectPrefix = p.projectPrefix[:len(p.projectPrefix):len(p.projectPrefix)]
					p.visiblePrefix = append(p.visiblePrefix[:len(p.visiblePrefix):len(p.visiblePrefix)], deltaVis...)
					p.visiblePrefix = p.visiblePrefix[:len(p.visiblePrefix):len(p.visiblePrefix)]
					p.todo, p.todoLast, p.todoKnown = extendTodoBaselinesState(p.todo, p.todoLast, p.todoKnown, delta, snap.HistoryHasMore)
					p.recordSegmentMetadata(delta, deltaProj)
					p.gen++
					if p.gen == 0 {
						p.gen = 1
					}
					out := *p
					cp.mu.Unlock()
					return out, true
				}
				cp.mu.Unlock()
			}
		} else {
			cp.mu.Unlock()
		}
	} else {
		cp.mu.Unlock()
	}

	var maxSeq float64
	for i := 0; i < splitIdx; i++ {
		if snap.Messages[i].Seq > maxSeq {
			maxSeq = snap.Messages[i].Seq
		}
	}
	for _, r := range snap.WorkflowRuns {
		if r.AnchorSeq == 0 || r.AnchorSeq > maxSeq {
			return chatTurnPrefixMemo{}, false
		}
		for _, seq := range r.EventSeqs {
			if seq == 0 || seq > maxSeq {
				return chatTurnPrefixMemo{}, false
			}
		}
	}

	prefixSnap := snap
	prefixSnap.Messages = snap.Messages[:splitIdx]
	projPrefix := buildChatMessages(prefixSnap)
	for i := range projPrefix {
		projPrefix[i].TurnDelivery = ownProjectionDelivery(projPrefix[i].TurnDelivery)
		projPrefix[i].Files = ownProjectionFiles(projPrefix[i].Files)
	}
	visPrefix := visibleChatMessages(projPrefix)
	todo, todoLast, todoKnown := todoBaselinesStateFor(prefixSnap.Messages, snap.HistoryHasMore)
	ownedPrefix := make([]session.Message, splitIdx)
	copy(ownedPrefix, prefixSnap.Messages)
	runs := ownWorkflowSources(snap.WorkflowRuns)

	next := chatTurnPrefixMemo{
		ready:          true,
		selected:       snap.SelectedID,
		historyHasMore: snap.HistoryHasMore,
		prefix:         ownedPrefix[:len(ownedPrefix):len(ownedPrefix)],
		runs:           runs,
		turns:          make(map[float64]struct{}, 64),
		turnSeqs:       make(map[float64]struct{}, 64),
		toolCallIDs:    make(map[string]struct{}, 64),
		planCallIDs:    make(map[string]struct{}),
		projectPrefix:  projPrefix[:len(projPrefix):len(projPrefix)],
		visiblePrefix:  visPrefix[:len(visPrefix):len(visPrefix)],
		todo:           todo,
		todoLast:       todoLast,
		todoKnown:      todoKnown,
	}
	next.recordSegmentMetadata(ownedPrefix, projPrefix)

	cp.mu.Lock()
	gen := cp.turnPrefix.gen + 1
	if gen == 0 {
		gen = 1
	}
	next.gen = gen
	cp.turnPrefix = next
	cp.mu.Unlock()
	return next, true
}

func projectRangeWithTurnPrefix(
	snap session.ViewState,
	limit int,
	splitIdx int,
) ([]session.Message, []session.Message, map[string]*todoBaseline, uint64, int, bool) {
	p, ok := ensureTurnPrefixMemo(snap, splitIdx)
	if !ok {
		return nil, nil, nil, 0, 0, false
	}
	tail := snap.Messages[splitIdx:limit]
	if len(tail) == 0 {
		return p.projectPrefix, p.visiblePrefix, p.todo, p.gen, len(p.visiblePrefix), true
	}
	if !p.canIsolateRawSegment(tail, snap.WorkflowRuns) {
		return nil, nil, nil, 0, 0, false
	}
	tailProj := buildChatMessages(session.ViewState{Messages: tail, SelectedID: snap.SelectedID})
	if !p.canIsolateProjectedSegment(tailProj) {
		return nil, nil, nil, 0, 0, false
	}
	for i := range tailProj {
		tailProj[i].TurnDelivery = ownProjectionDelivery(tailProj[i].TurnDelivery)
		tailProj[i].Files = ownProjectionFiles(tailProj[i].Files)
	}
	tailVis := visibleChatMessages(tailProj)
	todo := extendTodoBaselines(p.todo, p.todoLast, p.todoKnown, tail, snap.HistoryHasMore)
	projOut := make([]session.Message, len(p.projectPrefix)+len(tailProj))
	copy(projOut, p.projectPrefix)
	copy(projOut[len(p.projectPrefix):], tailProj)
	visOut := make([]session.Message, len(p.visiblePrefix)+len(tailVis))
	copy(visOut, p.visiblePrefix)
	copy(visOut[len(p.visiblePrefix):], tailVis)
	return projOut, visOut, todo, p.gen, len(p.visiblePrefix), true
}

// chatProjections keeps the transcript memos of the sessions drawn most
// recently, one chatProjectionStore per SelectedID. A workspace can draw the
// main conversation and a child agent's chat in the same pass; with a single
// memo each call evicted the other's projection, so both rebuilt their whole
// transcript on every frame. A slot is only a place to look: every hit is still
// validated against the view's own revision key and content.
var chatProjections struct {
	sync.Mutex
	slots []chatProjectionSlot // most recently used first
}

type chatProjectionSlot struct {
	selected string
	store    *chatProjectionStore
}

const chatProjectionSlots = 3

func chatProjectionOf(snap session.ViewState) *chatProjectionStore {
	chatProjections.Lock()
	defer chatProjections.Unlock()
	for i, slot := range chatProjections.slots {
		if slot.selected == snap.SelectedID {
			copy(chatProjections.slots[1:i+1], chatProjections.slots[:i])
			chatProjections.slots[0] = slot
			return slot.store
		}
	}
	slot := chatProjectionSlot{selected: snap.SelectedID, store: &chatProjectionStore{}}
	chatProjections.slots = append([]chatProjectionSlot{slot}, chatProjections.slots...)
	if len(chatProjections.slots) > chatProjectionSlots {
		chatProjections.slots[chatProjectionSlots] = chatProjectionSlot{}
		chatProjections.slots = chatProjections.slots[:chatProjectionSlots]
	}
	return slot.store
}

func (s *chatProjectionStore) revisionPrefixGen(snap session.ViewState) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revision.matches(snap) {
		return s.revision.prefixGen
	}
	return 0
}

func (s *chatProjectionStore) revisionTurnPrefix(snap session.ViewState) (uint64, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revision.matches(snap) {
		return s.revision.turnPrefixGen, s.revision.visPrefixLen
	}
	return 0, 0
}

// projectChatMessages returns the transcript projection for this snapshot. A
// view the store projected at an unchanged revision reuses the last projection
// outright: transcripts with tool, retry or workflow rows never qualify for the
// source memo below, so every frame re-decoded the whole journal
// (tmp/perf-20261029).
//
// The returned slice belongs to the memo. While a reply streams, its last row is
// rewritten in place for the next revision, so callers read it within the pass,
// on the UI thread, and copy what they need before any goroutine gets hold of it.
func projectChatMessages(snap session.ViewState) []session.Message {
	cp := chatProjectionOf(snap)
	key := snap.HistoryProjectionKey()
	if !key.Valid() {
		return projectChatMessageSources(snap)
	}
	cp.mu.Lock()
	cached := cp.revision
	cp.mu.Unlock()
	if cached.matches(snap) {
		return cached.project
	}
	splitIdx := findLastSettledTurnEndSplit(snap.Messages)
	if n := len(snap.Messages); n >= 2 && isStreamingTailMessage(snap.Messages[n-1], snap.WorkflowRuns) {
		tail := snap.Messages[n-1]
		tail.TurnDelivery = ownProjectionDelivery(tail.TurnDelivery)
		tail.Files = ownProjectionFiles(tail.Files)
		cp.mu.Lock()
		hit := cp.streamingPrefix.matches(snap)
		if hit {
			p := &cp.streamingPrefix
			p.projectBuf[len(p.projectBuf)-1] = tail
			p.visibleBuf[len(p.visibleBuf)-1] = tail
			out := p.projectBuf
			vis := p.visibleBuf
			todo := p.todo
			prefixGen := p.gen
			turnPrefixGen := p.turnPrefixGen
			visPrefixLen := p.visPrefixLen
			cp.revision = chatRevisionProjection{
				key:           key,
				prefixGen:     prefixGen,
				turnPrefixGen: turnPrefixGen,
				visPrefixLen:  visPrefixLen,
				messages:      n,
				runs:          len(snap.WorkflowRuns),
				firstID:       snap.Messages[0].ID,
				lastID:        snap.Messages[n-1].ID,
				project:       out,
				visible:       vis,
				visibleReady:  true,
				todo:          todo,
				todoReady:     true,
			}
			cp.mu.Unlock()
			return out
		}
		cp.mu.Unlock()

		var projPrefix, visPrefix []session.Message
		var todo map[string]*todoBaseline
		var turnPrefixGen uint64
		var visPrefixLen int
		usedTurnPrefix := false
		if splitIdx > 0 && splitIdx <= n-1 {
			projPrefix, visPrefix, todo, turnPrefixGen, visPrefixLen, usedTurnPrefix = projectRangeWithTurnPrefix(snap, n-1, splitIdx)
		}
		if !usedTurnPrefix {
			prefixSnap := snap
			prefixSnap.Messages = snap.Messages[:n-1]
			projPrefix = buildChatMessages(prefixSnap)
			for i := range projPrefix {
				projPrefix[i].TurnDelivery = ownProjectionDelivery(projPrefix[i].TurnDelivery)
				projPrefix[i].Files = ownProjectionFiles(projPrefix[i].Files)
			}
			visPrefix = visibleChatMessages(projPrefix)
			todo = todoBaselinesFor(prefixSnap.Messages, snap.HistoryHasMore)
		}
		projectBuf := make([]session.Message, len(projPrefix)+1)
		copy(projectBuf, projPrefix)
		projectBuf[len(projPrefix)] = tail
		visibleBuf := make([]session.Message, len(visPrefix)+1)
		copy(visibleBuf, visPrefix)
		visibleBuf[len(visPrefix)] = tail
		ownedPrefix := make([]session.Message, n-1)
		copy(ownedPrefix, snap.Messages[:n-1])
		runs := ownWorkflowSources(snap.WorkflowRuns)

		cp.mu.Lock()
		prefixGen := cp.streamingPrefix.gen + 1
		if prefixGen == 0 {
			prefixGen = 1
		}
		cp.streamingPrefix = chatStreamingPrefixMemo{
			ready:          true,
			gen:            prefixGen,
			turnPrefixGen:  turnPrefixGen,
			visPrefixLen:   visPrefixLen,
			selected:       snap.SelectedID,
			historyHasMore: snap.HistoryHasMore,
			prefix:         ownedPrefix,
			runs:           runs,
			projectBuf:     projectBuf,
			visibleBuf:     visibleBuf,
			todo:           todo,
		}
		cp.revision = chatRevisionProjection{
			key:           key,
			prefixGen:     prefixGen,
			turnPrefixGen: turnPrefixGen,
			visPrefixLen:  visPrefixLen,
			messages:      n,
			runs:          len(snap.WorkflowRuns),
			firstID:       snap.Messages[0].ID,
			lastID:        snap.Messages[n-1].ID,
			project:       projectBuf,
			visible:       visibleBuf,
			visibleReady:  true,
			todo:          todo,
			todoReady:     true,
		}
		cp.mu.Unlock()
		return projectBuf
	}
	if n := len(snap.Messages); splitIdx > 0 && splitIdx < n {
		if projOut, visOut, todo, turnPrefixGen, visPrefixLen, ok := projectRangeWithTurnPrefix(snap, n, splitIdx); ok {
			cp.mu.Lock()
			cp.revision = chatRevisionProjection{
				key:           key,
				turnPrefixGen: turnPrefixGen,
				visPrefixLen:  visPrefixLen,
				messages:      n,
				runs:          len(snap.WorkflowRuns),
				firstID:       snap.Messages[0].ID,
				lastID:        snap.Messages[n-1].ID,
				project:       projOut,
				visible:       visOut,
				visibleReady:  true,
				todo:          todo,
				todoReady:     true,
			}
			cp.mu.Unlock()
			return projOut
		}
	}
	// Own the nested delivery data, as the source memo does, in a copy so the
	// view's own rows are never written.
	out := append([]session.Message(nil), projectChatMessageSources(snap)...)
	for i := range out {
		out[i].TurnDelivery = ownProjectionDelivery(out[i].TurnDelivery)
		out[i].Files = ownProjectionFiles(out[i].Files)
	}
	next := chatRevisionProjection{key: key, messages: len(snap.Messages), runs: len(snap.WorkflowRuns), project: out}
	if n := len(snap.Messages); n > 0 {
		next.firstID, next.lastID = snap.Messages[0].ID, snap.Messages[n-1].ID
	}
	cp.mu.Lock()
	cp.revision = next
	cp.mu.Unlock()
	return out
}

// visibleProjectedMessages is visibleChatMessages(projectChatMessages(snap)),
// filtered once per store revision.
func visibleProjectedMessages(snap session.ViewState) []session.Message {
	cp := chatProjectionOf(snap)
	msgs := projectChatMessages(snap)
	if !snap.HistoryProjectionKey().Valid() {
		return visibleChatMessages(msgs)
	}
	cp.mu.Lock()
	if r := cp.revision; r.matches(snap) && r.visibleReady {
		cp.mu.Unlock()
		return r.visible
	}
	cp.mu.Unlock()
	out := visibleChatMessages(msgs)
	cp.mu.Lock()
	if r := &cp.revision; r.matches(snap) {
		r.visible, r.visibleReady = out, true
	}
	cp.mu.Unlock()
	return out
}

// projectedTodoBaselines is todoBaselinesFor the view's loaded window, indexed
// once per store revision (which also fixes HistoryHasMore).
func projectedTodoBaselines(snap session.ViewState) map[string]*todoBaseline {
	cp := chatProjectionOf(snap)
	if !snap.HistoryProjectionKey().Valid() {
		return todoBaselinesFor(snap.Messages, snap.HistoryHasMore)
	}
	projectChatMessages(snap) // records this view's revision
	cp.mu.Lock()
	if r := cp.revision; r.matches(snap) && r.todoReady {
		cp.mu.Unlock()
		return r.todo
	}
	cp.mu.Unlock()
	out := todoBaselinesFor(snap.Messages, snap.HistoryHasMore)
	cp.mu.Lock()
	if r := &cp.revision; r.matches(snap) {
		r.todo, r.todoReady = out, true
	}
	cp.mu.Unlock()
	return out
}

func projectedFileMentionResolver(snap session.ViewState, closing session.Message) func(string) string {
	cp := chatProjectionOf(snap)
	if !snap.HistoryProjectionKey().Valid() || closing.Seq <= 0 || closing.Streaming {
		return FileMentionResolver(snap.Messages, closing)
	}
	projectChatMessages(snap)
	cp.mu.Lock()
	if r := cp.revision; r.matches(snap) && r.mentions != nil {
		if fn, ok := r.mentions[closing.Seq]; ok {
			cp.mu.Unlock()
			return fn
		}
	}
	cp.mu.Unlock()
	fn := buildMentionResolver(snap.Messages, closing)
	cp.mu.Lock()
	if r := &cp.revision; r.matches(snap) {
		if r.mentions == nil {
			r.mentions = map[float64]func(string) string{}
		}
		r.mentions[closing.Seq] = fn
	}
	cp.mu.Unlock()
	return fn
}

func projectedLatestEditable(snap session.ViewState) (seq float64, textOnly bool) {
	cp := chatProjectionOf(snap)
	if !snap.HistoryProjectionKey().Valid() {
		seq = session.LatestEditableMessage(snap.Messages)
		probe := snap
		probe.SelectedID = "_"
		probe.PromptBusy = false
		probe.MessageEditBusy = false
		return seq, seq > 0 && session.CanEditMessage(probe, seq)
	}
	projectChatMessages(snap)
	cp.mu.Lock()
	if r := cp.revision; r.matches(snap) && r.editReady {
		cp.mu.Unlock()
		return r.editSeq, r.editTextOnly
	}
	cp.mu.Unlock()
	seq = session.LatestEditableMessage(snap.Messages)
	if seq > 0 {
		probe := snap
		probe.SelectedID = "_"
		probe.PromptBusy = false
		probe.MessageEditBusy = false
		textOnly = session.CanEditMessage(probe, seq)
	}
	cp.mu.Lock()
	if r := &cp.revision; r.matches(snap) {
		r.editSeq, r.editTextOnly, r.editReady = seq, textOnly, true
	}
	cp.mu.Unlock()
	return seq, textOnly
}

// projectChatMessageSources rebuilds the projection only when a message body,
// raw record or workflow set changes. The chat view used to re-project every
// message on every render pass.
func projectChatMessageSources(snap session.ViewState) []session.Message {
	cp := chatProjectionOf(snap)
	total := 0
	cacheable := true
	for i := range snap.Messages {
		total += len(snap.Messages[i].Raw) + len(snap.Messages[i].Body)
		cacheable = cacheable && plainProjectionMessage(snap.Messages[i])
	}
	// Complex presentation metadata has its own mutable lifecycle. Until it has
	// an explicit owned identity, rebuild rather than reuse a stale projection.
	if !cacheable {
		return buildChatMessages(snap)
	}
	// The source copy below keeps every record's bytes a second time. Past the
	// limit, rebuild per call instead of pinning that much memory; the coordinate
	// and mention memos are bounded the same way.
	if total > chatSourceByteLimit {
		out := buildChatMessages(snap)
		for i := range out {
			out[i].TurnDelivery = ownProjectionDelivery(out[i].TurnDelivery)
			out[i].Files = ownProjectionFiles(out[i].Files)
		}
		return out
	}
	key := chatProjectionKey{count: len(snap.Messages), total: total, runs: len(snap.WorkflowRuns), selected: snap.SelectedID}
	cp.mu.Lock()
	if cp.ready && cp.key == key && sameMessageSources(cp.source, snap.Messages) && sameWorkflowSources(cp.runs, snap.WorkflowRuns) {
		out := cp.project
		cp.mu.Unlock()
		return out
	}
	cp.mu.Unlock()
	out := buildChatMessages(snap)
	// Existing delivery rows pass through the builders unchanged. Own their
	// nested data so later caller edits cannot change an already returned view.
	for i := range out {
		out[i].TurnDelivery = ownProjectionDelivery(out[i].TurnDelivery)
		out[i].Files = ownProjectionFiles(out[i].Files)
	}
	source := make([]chatMessageSource, len(snap.Messages))
	for i, msg := range snap.Messages {
		source[i] = chatMessageSource{message: plainProjectionValue(msg), raw: string(msg.Raw), content: string(msg.ContentBlock), timing: ownProjectionTiming(msg.TrajectoryTime), delivery: ownProjectionDelivery(msg.TurnDelivery), files: ownProjectionFiles(msg.Files)}
	}
	runs := ownWorkflowSources(snap.WorkflowRuns)
	cp.mu.Lock()
	cp.key, cp.source, cp.runs, cp.project, cp.ready = key, source, runs, out, true
	cp.mu.Unlock()
	return out
}

func buildChatMessages(snap session.ViewState) []session.Message {
	return session.ProjectTurnDeliveryMessages(session.ProjectToolMessages(session.ProjectWorkflowMessages(session.WithSubmittedPlanCards(snap.Messages), snap.WorkflowRuns)))
}

type chatMessageSource struct {
	message  session.Message
	raw      string
	content  string
	timing   *session.TrajectoryTiming
	delivery *session.TurnDelivery
	files    []string
}

func plainProjectionMessage(msg session.Message) bool {
	return msg.AutomaticCompaction == nil && msg.Retry == nil && msg.Workflow == nil && msg.Submission == nil && msg.Tool == nil && msg.Command == nil && msg.ConversationTurn == nil && msg.ConversationStep == nil && msg.Web == nil
}

func plainProjectionValue(msg session.Message) session.Message {
	msg.Raw = nil
	msg.ContentBlock = nil
	msg.TrajectoryTime = nil
	msg.TurnDelivery = nil
	msg.Files = nil
	return msg
}

func sameMessageSources(source []chatMessageSource, messages []session.Message) bool {
	if len(source) != len(messages) {
		return false
	}
	for i, msg := range messages {
		previous := source[i]
		a, b := previous.message, msg
		if a.ContextWaking != b.ContextWaking || a.ID != b.ID || a.SourceMessageID != b.SourceMessageID || a.ImageRef != b.ImageRef || a.Kind != b.Kind || a.Role != b.Role || a.Body != b.Body || a.Seq != b.Seq || a.EventType != b.EventType || a.TurnSeq != b.TurnSeq || a.TurnOpening != b.TurnOpening || a.Streaming != b.Streaming || a.Model != b.Model || a.Args != b.Args || a.EventTime != b.EventTime || !bytes.Equal([]byte(previous.raw), msg.Raw) || !bytes.Equal([]byte(previous.content), msg.ContentBlock) || !sameProjectionTiming(previous.timing, msg.TrajectoryTime) || !sameProjectionDelivery(previous.delivery, msg.TurnDelivery) || !sameProjectionFiles(previous.files, msg.Files) {
			return false
		}
	}
	return true
}

func ownProjectionFiles(value []string) []string {
	if value == nil {
		return nil
	}
	owned := make([]string, len(value))
	copy(owned, value)
	return owned
}

func sameProjectionFiles(a, b []string) bool {
	if (a == nil) != (b == nil) || len(a) != len(b) {
		return false
	}
	if len(a) == 0 || &a[0] == &b[0] {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ownProjectionDelivery(value *session.TurnDelivery) *session.TurnDelivery {
	if value == nil {
		return nil
	}
	owned := *value
	if value.Changes != nil {
		changes := *value.Changes
		owned.Changes = &changes
	}
	if value.Files != nil {
		owned.Files = make([]session.TurnPresentedFile, len(value.Files))
		copy(owned.Files, value.Files)
	}
	return &owned
}

func sameProjectionDelivery(a, b *session.TurnDelivery) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Turn != b.Turn || (a.Changes == nil) != (b.Changes == nil) || a.Changes != nil && *a.Changes != *b.Changes || (a.Files == nil) != (b.Files == nil) || len(a.Files) != len(b.Files) {
		return false
	}
	for i := range a.Files {
		if a.Files[i] != b.Files[i] {
			return false
		}
	}
	return true
}

func sameProjectionNumber(a, b *float64) bool {
	return a == b || a != nil && b != nil && *a == *b
}

func ownProjectionNumber(value *float64) *float64 {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}

func ownProjectionTiming(value *session.TrajectoryTiming) *session.TrajectoryTiming {
	if value == nil {
		return nil
	}
	owned := *value
	owned.StartedAt = ownProjectionNumber(value.StartedAt)
	owned.CompletedAt = ownProjectionNumber(value.CompletedAt)
	owned.FirstTokenAt = ownProjectionNumber(value.FirstTokenAt)
	return &owned
}

func sameProjectionTiming(a, b *session.TrajectoryTiming) bool {
	return a == b || a != nil && b != nil && a.OperationID == b.OperationID && a.Closed == b.Closed && a.Invalid == b.Invalid && sameProjectionNumber(a.StartedAt, b.StartedAt) && sameProjectionNumber(a.CompletedAt, b.CompletedAt) && sameProjectionNumber(a.FirstTokenAt, b.FirstTokenAt)
}

type workflowProjectionSource struct {
	value  workflowrun.Run
	events []string
}

func ownWorkflowSources(runs []workflowrun.Run) []workflowProjectionSource {
	owned := make([]workflowProjectionSource, len(runs))
	for i, run := range runs {
		value := run
		value.Events = nil
		value.EventSeqs = append([]float64(nil), run.EventSeqs...)
		value.Phases = make([]workflowrun.Phase, len(run.Phases))
		for j, phase := range run.Phases {
			value.Phases[j] = phase
			if phase.Name != nil {
				name := *phase.Name
				value.Phases[j].Name = &name
			}
			value.Phases[j].Members = append([]workflowrun.Member(nil), phase.Members...)
		}
		events := make([]string, len(run.Events))
		for j := range run.Events {
			events[j] = string(run.Events[j])
		}
		owned[i] = workflowProjectionSource{value: value, events: events}
	}
	return owned
}

func sameWorkflowSources(source []workflowProjectionSource, runs []workflowrun.Run) bool {
	if len(source) != len(runs) {
		return false
	}
	for i, run := range runs {
		previous := source[i]
		a := previous.value
		if a.ID != run.ID || a.Name != run.Name || a.Status != run.Status || a.AnchorSeq != run.AnchorSeq || len(a.Phases) != len(run.Phases) || len(a.EventSeqs) != len(run.EventSeqs) || len(previous.events) != len(run.Events) {
			return false
		}
		for j, phase := range run.Phases {
			p := a.Phases[j]
			if p.Key != phase.Key || (p.Name == nil) != (phase.Name == nil) || p.Name != nil && *p.Name != *phase.Name || len(p.Members) != len(phase.Members) {
				return false
			}
			for k := range phase.Members {
				if p.Members[k] != phase.Members[k] {
					return false
				}
			}
		}
		for j := range run.EventSeqs {
			if a.EventSeqs[j] != run.EventSeqs[j] {
				return false
			}
		}
		for j := range run.Events {
			if !bytes.Equal([]byte(previous.events[j]), run.Events[j]) {
				return false
			}
		}
	}
	return true
}
