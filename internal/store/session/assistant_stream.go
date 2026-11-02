package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// assistantStreamState joins process-local attempts to their named durable
// settlement. It is owned by the selected follow generation, never by text.
type assistantStreamState struct {
	active, retained *assistantStreamAttempt
	pending          map[float64]json.RawMessage
	published        map[float64]bool
	revision         float64
	rebaseline       bool
}

type assistantStreamAttempt struct {
	trajectoryFirstToken *float64
	inspectorAfterSeq    *float64
	tools                toolPreparingState
	AttemptID            string            `json:"attemptId"`
	StartedAfterSeq      float64           `json:"startedAfterSeq"`
	Turn                 float64           `json:"turn"`
	Step                 float64           `json:"step"`
	NextIndex            int               `json:"nextIndex"`
	Stream               []json.RawMessage `json:"stream"`
	blocks               map[int]assistantStreamBlock
	rows                 map[int]string
	settlementSeq        float64
	liveUsage            map[string]float64
	liveUsageStreamLen   int
	liveUsageInvalid     bool
}

type assistantStreamBaseline struct {
	Revision      float64                 `json:"revision"`
	ActiveAttempt *assistantStreamAttempt `json:"activeAttempt"`
}

type assistantStreamBlock struct {
	kind, text string
	imageRef   string
	raw        json.RawMessage
	time       float64
}

type assistantStreamEvent struct {
	Type      string          `json:"type"`
	Seq       float64         `json:"seq"`
	Time      float64         `json:"time"`
	SurfaceOp json.RawMessage `json:"surfaceOp"`
	Data      struct {
		Turn        float64           `json:"turn"`
		Step        float64           `json:"step"`
		Interrupted bool              `json:"interrupted"`
		Stream      json.RawMessage   `json:"stream"`
		Text        string            `json:"text"`
		Content     []json.RawMessage `json:"content"`
		Message     struct {
			ID      string            `json:"id"`
			Content []json.RawMessage `json:"content"`
			Source  struct {
				Model string `json:"model"`
			} `json:"source"`
		} `json:"message"`
	} `json:"data"`
}

func validRawJSONArrayOrNull(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] == '['
}

func (s *Store) resetAssistantStreamLocked() {
	s.retireAssistantAttemptLocked(s.assistantStream.active)
	s.retireAssistantAttemptLocked(s.assistantStream.retained)
	s.assistantStream = assistantStreamState{}
}

func (s *Store) rebaselineAssistantLocked() bool {
	s.invalidateViewLocked()
	s.assistantStream.rebaseline = true
	return false
}

func (s *Store) retireAssistantAttemptLocked(attempt *assistantStreamAttempt) {
	s.invalidateViewLocked()
	if attempt == nil {
		return
	}
	ids := make(map[string]bool, len(attempt.rows))
	for _, id := range attempt.rows {
		ids[id] = true
	}
	kept := s.messages[:0]
	for _, msg := range s.messages {
		if !ids[msg.ID] {
			kept = append(kept, msg)
		}
	}
	s.messages = kept
}

// acceptAssistantDurableLocked is called only for followed events. Historical
// replacement/prepend is already durable and must not settle a live attempt.
func (s *Store) acceptAssistantDurableLocked(raw json.RawMessage) bool {
	s.invalidateViewLocked()
	var ev assistantStreamEvent
	if json.Unmarshal(raw, &ev) != nil || !validRawJSONArrayOrNull(ev.Data.Stream) || ev.Type == "" {
		return false
	}
	st := &s.assistantStream
	if st.rebaseline {
		return false
	}
	if st.published[ev.Seq] {
		return false
	}
	// Older pages do not run the live fold; still recognize their durable
	// identities when a replay overlaps the already-loaded history window.
	for _, msg := range s.messages {
		if msg.Seq == ev.Seq && len(msg.Raw) != 0 {
			return false
		}
	}
	s.bumpHistoryThroughSeq(ev.Seq)
	attempt := st.active
	appendSurface := string(ev.SurfaceOp) == `"append"`
	if attempt != nil && (ev.Type == "assistant/attempt" || (ev.Type == "assistant/message" && appendSurface)) &&
		ev.Seq > attempt.StartedAfterSeq && ev.Data.Turn == attempt.Turn && ev.Data.Step == attempt.Step {
		if st.pending == nil {
			st.pending = make(map[float64]json.RawMessage)
		}
		if _, exists := st.pending[ev.Seq]; exists {
			return s.rebaselineAssistantLocked()
		}
		st.pending[ev.Seq] = append(json.RawMessage(nil), raw...)
		return false
	}
	s.observeLiveQuotaLocked(raw)
	changed := s.publishAssistantDurableLocked(raw, ev.Seq)
	if retained := st.retained; retained != nil && ev.Type == "step/end" && ev.Data.Turn == retained.Turn && ev.Data.Step == retained.Step {
		s.retireAssistantAttemptLocked(retained)
		st.retained = nil
		changed = true
	}
	return changed
}

func (s *Store) publishAssistantDurableLocked(raw json.RawMessage, seq float64) bool {
	if s.assistantStream.published == nil {
		s.assistantStream.published = make(map[float64]bool)
	}
	s.assistantStream.published[seq] = true
	return s.applyPersistenceEventLocked(raw)
}

func (s *Store) applyAssistantStreamLocked(raw json.RawMessage) bool {
	s.invalidateStreamViewLocked()
	var wrap struct {
		Frame struct {
			Type            string          `json:"type"`
			AttemptID       string          `json:"attemptId"`
			Revision        *float64        `json:"revision"`
			StartedAfterSeq float64         `json:"startedAfterSeq"`
			Turn            float64         `json:"turn"`
			Step            float64         `json:"step"`
			Index           *int            `json:"index"`
			Time            float64         `json:"time"`
			Chunk           json.RawMessage `json:"chunk"`
			Outcome         struct {
				Kind      string  `json:"kind"`
				EventType string  `json:"eventType"`
				Seq       float64 `json:"seq"`
			} `json:"outcome"`
		} `json:"frame"`
	}
	if json.Unmarshal(raw, &wrap) != nil {
		return false
	}
	f, st := wrap.Frame, &s.assistantStream
	if st.rebaseline {
		return false
	}
	if f.Revision == nil || *f.Revision != st.revision+1 {
		return s.rebaselineAssistantLocked()
	}
	st.revision = *f.Revision
	if f.AttemptID == "" {
		return s.rebaselineAssistantLocked()
	}
	switch f.Type {
	case "start":
		if st.active != nil || st.retained != nil || len(st.pending) != 0 {
			return s.rebaselineAssistantLocked()
		}
		if s.inspector.client != s.client || s.inspector.session != s.selectedID || s.inspector.generation != s.followGeneration {
			s.resetInspectorLocked()
		}
		st.active = &assistantStreamAttempt{AttemptID: f.AttemptID, StartedAfterSeq: f.StartedAfterSeq, Turn: f.Turn, Step: f.Step}
		return false
	case "chunk":
		a := st.active
		// A follow mounted after start cannot reconstruct a suffix. Its settlement
		// publishes directly; unrelated/late attempts cannot corrupt the current one.
		if a == nil || a.AttemptID != f.AttemptID {
			return false
		}
		if f.Index == nil || *f.Index != a.NextIndex {
			return s.rebaselineAssistantLocked()
		}
		a.NextIndex++
		if a.inspectorAfterSeq == nil {
			a.inspectorAfterSeq = s.inspectorCursorLocked()
		}
		record, _ := json.Marshal(map[string]any{"type": "chunk", "time": f.Time, "chunk": f.Chunk})
		a.Stream = append(a.Stream, record)
		s.applyAssistantChunkLocked(a, f.Chunk, f.Time)
		// Raw-only chunks also change the inspector, even when Chat text is unchanged.
		return true
	case "end":
		a := st.active
		if a == nil || a.AttemptID != f.AttemptID {
			return false
		}
		if f.Index == nil || *f.Index != a.NextIndex {
			return s.rebaselineAssistantLocked()
		}
		if f.Outcome.Kind == "abandoned" {
			if len(st.pending) != 0 {
				return s.rebaselineAssistantLocked()
			}
			st.active = nil
			s.retireAssistantAttemptLocked(a)
			return true
		}
		if f.Outcome.Kind != "committed" {
			return s.rebaselineAssistantLocked()
		}
		raw, found := st.pending[f.Outcome.Seq]
		var ev assistantStreamEvent
		if !found || json.Unmarshal(raw, &ev) != nil || !validRawJSONArrayOrNull(ev.Data.Stream) || ev.Type != f.Outcome.EventType || len(st.pending) != 1 {
			return s.rebaselineAssistantLocked()
		}
		delete(st.pending, f.Outcome.Seq)
		st.active = nil
		if ev.Type == "assistant/message" && !ev.Data.Interrupted {
			// Keep timing/prefix rows until step/end like the original fold, but the
			// UI projects the canonical settlement immediately and exactly once.
			a.settlementSeq = ev.Seq
			st.retained = a
		} else {
			s.retireAssistantAttemptLocked(a)
		}
		s.publishAssistantDurableLocked(raw, ev.Seq)
		if ev.Type != "assistant/message" || ev.Data.Interrupted {
			s.settleInspectorAttemptLocked(a, ev.Seq)
		}
		return true
	default:
		return s.rebaselineAssistantLocked()
	}
}

func (s *Store) restoreAssistantStreamLocked(baseline *assistantStreamBaseline) bool {
	s.invalidateViewLocked()
	if baseline == nil {
		return true
	} // Older hosts did not include this optional field.
	st := &s.assistantStream
	if !assistantSafeInt(baseline.Revision) || baseline.Revision < 0 {
		return s.rebaselineAssistantLocked()
	}
	st.revision = baseline.Revision
	a := baseline.ActiveAttempt
	if a == nil {
		return true
	}
	if a.AttemptID == "" || a.NextIndex < 0 {
		return s.rebaselineAssistantLocked()
	}
	chunks, ok := expandAssistantStream(a.Stream)
	// A Host snapshot may retain a compacted prefix whose nextIndex is ahead
	// of the visible chunk slice. Reject only impossible gaps where the slice
	// already extends beyond the advertised dense index.
	if !ok || len(chunks) > a.NextIndex {
		return s.rebaselineAssistantLocked()
	}
	a.blocks, a.rows = nil, nil
	a.tools = toolPreparingState{}
	if len(chunks) > 0 {
		a.inspectorAfterSeq = s.inspectorCursorLocked()
	}
	st.active = a
	for _, chunk := range chunks {
		s.applyAssistantChunkLocked(a, chunk.chunk, chunk.time)
	}
	// Snapshot chunks are historical; preparing tool rows are a live-only
	// projection and must reopen only from subsequent live deltas.
	a.tools = toolPreparingState{}
	return true
}

func (s *Store) applyAssistantChunkLocked(a *assistantStreamAttempt, raw json.RawMessage, time float64) bool {
	s.invalidateStreamViewLocked()
	if a.trajectoryFirstToken == nil {
		a.trajectoryFirstToken = trajectoryTokenTime(raw, time)
	}
	toolChanged := foldToolPreparingChunk(&a.tools, raw, time)
	if a.blocks == nil {
		a.blocks = make(map[int]assistantStreamBlock)
	}
	index, changed := foldAssistantChunk(a.blocks, raw, time)
	if !changed {
		return toolChanged
	}
	block := a.blocks[index]
	if a.rows == nil {
		a.rows = make(map[int]string)
	}
	id := a.rows[index]
	if id == "" {
		id = newRequestID()
		a.rows[index] = id
	}
	msg := assistantBlockMessage(Message{ID: id, TurnSeq: s.editTurnSeq, Streaming: true, EventTime: block.time, ConversationTurn: &a.Turn, ConversationStep: &a.Step}, block)
	for i := range s.messages {
		if s.messages[i].ID == id {
			s.messages[i] = msg
			return true
		}
	}
	s.messages = append(s.messages, msg)
	return true
}

type assistantMsgCacheEntry struct {
	id          string
	turnSeq     float64
	turnOpening bool
	raw         string
	rows        []Message
}

func (s *Store) cachedProjectAssistantMessageLocked(msg Message) []Message {
	if msg.EventType != "assistant/message" || msg.Seq <= 0 || len(msg.Raw) == 0 {
		return projectAssistantMessage(msg)
	}
	if s.assistantMsgCache != nil {
		if entry, ok := s.assistantMsgCache[msg.Seq]; ok && entry.id == msg.ID && entry.turnSeq == msg.TurnSeq && entry.turnOpening == msg.TurnOpening && bytes.Equal([]byte(entry.raw), msg.Raw) {
			return cloneMessages(entry.rows)
		}
	} else {
		s.assistantMsgCache = make(map[float64]assistantMsgCacheEntry, 64)
	}
	rows := projectAssistantMessage(msg)
	s.assistantMsgCache[msg.Seq] = assistantMsgCacheEntry{
		id:          msg.ID,
		turnSeq:     msg.TurnSeq,
		turnOpening: msg.TurnOpening,
		raw:         string(msg.Raw),
		rows:        cloneMessages(rows),
	}
	return rows
}

// Projection is identity-based. Equal text from distinct turns remains distinct.
func (s *Store) projectAssistantMessagesLocked(messages []Message) []Message {
	hidden := map[string]bool{}
	var settlement *Message
	if a := s.assistantStream.retained; a != nil {
		for _, id := range a.rows {
			hidden[id] = true
		}
		for i := range messages {
			if messages[i].Seq == a.settlementSeq && messages[i].EventType == "assistant/message" {
				copy := messages[i]
				settlement = &copy
				break
			}
		}
	}
	out := make([]Message, 0, len(messages))
	anchored := false
	for _, msg := range messages {
		if hidden[msg.ID] {
			if !anchored && assistantMessageVisible(msg) && settlement != nil {
				out = append(out, s.cachedProjectAssistantMessageLocked(*settlement)...)
				anchored = true
			}
			continue
		}
		if anchored && settlement != nil && msg.Seq == settlement.Seq && msg.EventType == "assistant/message" {
			continue
		}
		out = append(out, s.cachedProjectAssistantMessageLocked(msg)...)
	}
	return out
}

func projectAssistantMessage(msg Message) []Message {
	// Failed attempts remain in the journal for inspection but are not answers.
	// The original assistant-step node only matches live-chunk and message.
	if msg.EventType == "assistant/attempt" {
		return nil
	}
	if msg.EventType != "assistant/message" {
		if !msg.Streaming || assistantMessageVisible(msg) {
			return []Message{msg}
		}
		return nil
	}
	var ev assistantStreamEvent
	if json.Unmarshal(msg.Raw, &ev) != nil || !validRawJSONArrayOrNull(ev.Data.Stream) {
		return []Message{msg}
	}
	blocks := assistantDurableBlocks(ev)
	var out []Message
	textIDUsed := false
	for i, block := range blocks {
		if block.text == "" && block.imageRef == "" && len(block.raw) == 0 {
			continue
		}
		row := assistantBlockMessage(msg, block)
		row.SourceMessageID = msg.ID
		if block.kind == "text" && !textIDUsed {
			textIDUsed = true
		} else {
			row.ID = fmt.Sprintf("%s/%s/%d", msg.ID, block.kind, i)
		}
		out = append(out, row)
	}
	return out
}

func assistantBlockMessage(msg Message, block assistantStreamBlock) Message {
	msg.Body, msg.Kind, msg.Role = block.text, MessageKindText, "assistant"
	msg.ImageRef = block.imageRef
	msg.ContentBlock = append(json.RawMessage(nil), block.raw...)
	if block.kind == "image" && block.imageRef != "" {
		msg.Kind = MessageKindImage
	} else if len(block.raw) != 0 {
		msg.Kind = MessageKindUnknown
	}
	if block.kind == "reasoning" {
		msg.Kind, msg.Role = MessageKindThinking, "thinking"
	}
	return msg
}

// This canonical row keeps raw content for Snapshot and history pagination;
// reasoning is not flattened into an assistant answer at classification time.
func classifyAssistantPersistence(raw json.RawMessage) (Message, bool) {
	var ev assistantStreamEvent
	if json.Unmarshal(raw, &ev) != nil || !validRawJSONArrayOrNull(ev.Data.Stream) || (ev.Type != "assistant/message" && ev.Type != "assistant/attempt") {
		return Message{}, false
	}
	id := ev.Data.Message.ID
	if id == "" {
		id = fmt.Sprintf("assistant-event/%g", ev.Seq)
	}
	msg := Message{ID: id, Kind: MessageKindText, Role: "assistant", Seq: ev.Seq, EventType: ev.Type, Raw: append(json.RawMessage(nil), raw...), Model: ev.Data.Message.Source.Model, EventTime: ev.Time}
	for _, block := range assistantDurableBlocks(ev) {
		if block.kind == "text" {
			msg.Body += block.text
		}
	}
	return msg, true
}

func assistantDurableBlocks(ev assistantStreamEvent) []assistantStreamBlock {
	// Legacy hosts may carry top-level content/text instead of message.content.
	if ev.Type == "assistant/message" && ev.Data.Message.Content == nil {
		if ev.Data.Content != nil {
			ev.Data.Message.Content = ev.Data.Content
		} else if ev.Data.Text != "" {
			return []assistantStreamBlock{{kind: "text", text: ev.Data.Text, time: ev.Time}}
		}
	}
	if ev.Type == "assistant/message" && ev.Data.Message.Content != nil {
		var out []assistantStreamBlock
		for _, raw := range ev.Data.Message.Content {
			if block, ok := readAssistantContentBlock(raw, ev.Time); ok {
				out = append(out, block)
			}
		}
		return out
	}
	var stream []json.RawMessage
	if len(ev.Data.Stream) > 0 && !bytes.Equal(bytes.TrimSpace(ev.Data.Stream), []byte("null")) {
		if json.Unmarshal(ev.Data.Stream, &stream) != nil {
			return nil
		}
	}
	chunks, ok := expandAssistantStream(stream)
	if !ok {
		return nil
	}
	blocks := make(map[int]assistantStreamBlock)
	for _, chunk := range chunks {
		foldAssistantChunk(blocks, chunk.chunk, chunk.time)
	}
	keys := make([]int, 0, len(blocks))
	for index := range blocks {
		keys = append(keys, index)
	}
	sort.Ints(keys)
	out := make([]assistantStreamBlock, 0, len(keys))
	for _, index := range keys {
		out = append(out, blocks[index])
	}
	return out
}

// FeedbackMessageID keeps render-row keys out of the Host feedback API.
func (msg Message) FeedbackMessageID() string {
	if msg.SourceMessageID != "" {
		return msg.SourceMessageID
	}
	return msg.ID
}

func assistantMessageVisible(msg Message) bool {
	return msg.Body != "" || msg.ImageRef != "" || len(msg.ContentBlock) != 0
}

func readAssistantContentBlock(raw json.RawMessage, time float64) (assistantStreamBlock, bool) {
	var b struct {
		Type       string `json:"type"`
		Text       string `json:"text"`
		Attachment struct {
			ID string `json:"attachmentId"`
		} `json:"attachment"`
	}
	if json.Unmarshal(raw, &b) != nil || string(raw) == "null" {
		return assistantStreamBlock{}, false
	}
	block := assistantStreamBlock{kind: b.Type, time: time}
	switch b.Type {
	case "text", "reasoning":
		block.text = b.Text
	case "tool-call":
		return assistantStreamBlock{}, false // Rendered by the tool lifecycle projection.
	default:
		block.raw = append(json.RawMessage(nil), raw...)
		if b.Type == "image" {
			if b.Attachment.ID != "" {
				block.imageRef = "attachment:" + b.Attachment.ID
			} else {
				block.imageRef = imagePathFromBlock(raw)
			}
		}
	}
	return block, true
}

func foldAssistantChunk(blocks map[int]assistantStreamBlock, raw json.RawMessage, time float64) (int, bool) {
	var c struct {
		Type      string          `json:"type"`
		Index     int             `json:"index"`
		Text      string          `json:"text"`
		BlockType string          `json:"blockType"`
		Block     json.RawMessage `json:"block"`
	}
	if json.Unmarshal(raw, &c) != nil || c.Index < 0 {
		return 0, false
	}
	previous := blocks[c.Index]
	switch c.Type {
	case "block-start":
		blocks[c.Index] = assistantStreamBlock{kind: c.BlockType, time: time}
	case "text-delta", "reasoning-delta", "text":
		kind := "text"
		if c.Type == "reasoning-delta" {
			kind = "reasoning"
		}
		if previous.kind != kind {
			previous = assistantStreamBlock{kind: kind, time: time}
		}
		previous.text += c.Text
		blocks[c.Index] = previous
	case "block-end":
		block, ok := readAssistantContentBlock(c.Block, time)
		if !ok {
			blocks[c.Index] = assistantStreamBlock{time: time}
		} else {
			blocks[c.Index] = block
		}
	default:
		return c.Index, false // Tool/usage/control frames count in nextIndex, not answer text.
	}
	return c.Index, true
}

type assistantTimedChunk struct {
	time  float64
	chunk json.RawMessage
}

func assistantSafeInt(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Trunc(v) == v && math.Abs(v) <= 9007199254740991
}

// Expand packed runs without losing chunk boundaries: nextIndex counts every
// member, including invisible tool/usage frames, rather than text characters.
func expandAssistantStream(records []json.RawMessage) ([]assistantTimedChunk, bool) {
	var out []assistantTimedChunk
	for _, raw := range records {
		var r struct {
			Type  string          `json:"type"`
			Time  float64         `json:"time"`
			Time0 float64         `json:"time0"`
			Index int             `json:"index"`
			DT    []float64       `json:"dt"`
			Texts []string        `json:"texts"`
			Args  []string        `json:"args"`
			ID    string          `json:"id"`
			Name  *string         `json:"name"`
			Chunk json.RawMessage `json:"chunk"`
		}
		if json.Unmarshal(raw, &r) != nil {
			return nil, false
		}
		if r.Type == "chunk" {
			var obj map[string]json.RawMessage
			if !assistantSafeInt(r.Time) || json.Unmarshal(r.Chunk, &obj) != nil || obj == nil {
				return nil, false
			}
			out = append(out, assistantTimedChunk{r.Time, append(json.RawMessage(nil), r.Chunk...)})
			continue
		}
		texts, typ := r.Texts, "text-delta"
		switch r.Type {
		case "text-chunks":
		case "reasoning-chunks":
			typ = "reasoning-delta"
		case "tool-call-chunks":
			texts, typ = r.Args, "tool-call-delta"
			if r.ID == "" {
				return nil, false
			}
		default:
			return nil, false
		}
		if r.Index < 0 || !assistantSafeInt(r.Time0) || len(texts) == 0 || len(r.DT) != len(texts)-1 {
			return nil, false
		}
		time := r.Time0
		for i, text := range texts {
			if i > 0 {
				if !assistantSafeInt(r.DT[i-1]) {
					return nil, false
				}
				time += r.DT[i-1]
			}
			if !assistantSafeInt(time) {
				return nil, false
			}
			chunk := map[string]any{"type": typ, "index": r.Index, "text": text}
			if typ == "tool-call-delta" {
				delete(chunk, "text")
				chunk["id"], chunk["argumentsDelta"] = r.ID, text
				if r.Name != nil {
					chunk["name"] = *r.Name
				}
			}
			b, _ := json.Marshal(chunk)
			out = append(out, assistantTimedChunk{time, b})
		}
	}
	return out, true
}
