package session

import (
	"encoding/json"
	"time"
	"unsafe"

	"whaleisle/internal/dsh"
)

const (
	defaultSessionCacheLimit    = 24
	defaultSessionCacheMaxBytes = 64 << 20 // 64 MiB estimated memory budget
)

type sessionCacheKey struct {
	client         *dsh.Client
	hostGeneration uint64
	sessionID      string
}

type cachedSessionState struct {
	key                     sessionCacheKey
	estBytes                int64
	messages                []Message
	trajectoryBoundaries    map[float64]json.RawMessage
	workflowBoundaries      map[float64]json.RawMessage
	systemPromptAnchors     map[float64]float64
	planBoard               PlanBoard
	planHistoryBinding      string
	turnFilesHistoryBinding string
	historyThroughSeq       float64
	historyHasMore          bool
	messageRatings          map[string]string
	durableHasEdits         bool
}

type sessionLRUCache struct {
	limit      int
	maxBytes   int64
	totalBytes int64
	entries    []*cachedSessionState // oldest at 0, most recently used at len-1
}

func (c *sessionLRUCache) maxEntries() int {
	if c.limit > 0 {
		return c.limit
	}
	return defaultSessionCacheLimit
}

func (c *sessionLRUCache) byteBudget() int64 {
	if c.maxBytes > 0 {
		return c.maxBytes
	}
	return defaultSessionCacheMaxBytes
}

func estimateCachedSessionBytes(entry *cachedSessionState) int64 {
	if entry == nil {
		return 0
	}
	const messageStructBytes = int64(unsafe.Sizeof(Message{}))
	total := int64(unsafe.Sizeof(*entry)) + int64(len(entry.messages))*messageStructBytes
	for i := range entry.messages {
		m := &entry.messages[i]
		total += int64(len(m.ID) + len(m.SourceMessageID) + len(m.ImageRef) + len(m.ContentBlock) +
			len(m.Role) + len(m.Body) + len(m.EventType) + len(m.Raw) + len(m.Model) + len(m.Args))
		for _, f := range m.Files {
			total += int64(len(f)) + 16
		}
	}
	for _, raw := range entry.trajectoryBoundaries {
		total += int64(len(raw)) + 48
	}
	for _, raw := range entry.workflowBoundaries {
		total += int64(len(raw)) + 48
	}
	total += int64(len(entry.systemPromptAnchors)) * 32
	for k, v := range entry.messageRatings {
		total += int64(len(k)+len(v)) + 48
	}
	total += int64(len(entry.planBoard.Todos)*64 + len(entry.planBoard.Queue)*96)
	return total
}

func (c *sessionLRUCache) get(key sessionCacheKey) (*cachedSessionState, bool) {
	for i, e := range c.entries {
		if e.key == key {
			if i != len(c.entries)-1 {
				copy(c.entries[i:], c.entries[i+1:])
				c.entries[len(c.entries)-1] = e
			}
			return e, true
		}
	}
	return nil, false
}

func (c *sessionLRUCache) put(entry *cachedSessionState) {
	if entry == nil || entry.key.sessionID == "" {
		return
	}
	entry.estBytes = estimateCachedSessionBytes(entry)
	for i, e := range c.entries {
		if e.key == entry.key {
			c.totalBytes -= e.estBytes
			copy(c.entries[i:], c.entries[i+1:])
			c.entries[len(c.entries)-1] = nil
			c.entries = c.entries[:len(c.entries)-1]
			break
		}
	}
	budget := c.byteBudget()
	if entry.estBytes > budget {
		// Single session exceeds the entire cache budget; evict older entries
		// rather than exceeding the budget with stale sessions.
		c.clear()
		return
	}
	c.entries = append(c.entries, entry)
	c.totalBytes += entry.estBytes
	limit := c.maxEntries()
	for len(c.entries) > 0 && (len(c.entries) > limit || c.totalBytes > budget) {
		evicted := c.entries[0]
		c.totalBytes -= evicted.estBytes
		copy(c.entries, c.entries[1:])
		c.entries[len(c.entries)-1] = nil
		c.entries = c.entries[:len(c.entries)-1]
	}
	if c.totalBytes < 0 {
		c.totalBytes = 0
	}
}

func (c *sessionLRUCache) removeSession(sessionID string) {
	if sessionID == "" || len(c.entries) == 0 {
		return
	}
	kept := c.entries[:0]
	for _, e := range c.entries {
		if e.key.sessionID != sessionID {
			kept = append(kept, e)
		} else {
			c.totalBytes -= e.estBytes
		}
	}
	for i := len(kept); i < len(c.entries); i++ {
		c.entries[i] = nil
	}
	c.entries = kept
	if c.totalBytes < 0 {
		c.totalBytes = 0
	}
}

func (c *sessionLRUCache) clear() {
	for i := range c.entries {
		c.entries[i] = nil
	}
	c.entries = c.entries[:0]
	c.totalBytes = 0
}

func cloneCachedTrajectoryBoundaries(in map[float64]json.RawMessage) map[float64]json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	var out map[float64]json.RawMessage
	for k, v := range in {
		// Drop bulky request/header tool-schema payloads on backgrounding; the
		// Host follow snapshot rehydrates full trajectory boundaries on return.
		if typ, ok := peekEventType(v); ok && typ == "request/header" {
			continue
		}
		if out == nil {
			out = make(map[float64]json.RawMessage, len(in))
		}
		out[k] = v
	}
	return out
}

func cloneRawBoundaryMap(in map[float64]json.RawMessage) map[float64]json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	out := make(map[float64]json.RawMessage, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneFloatMap(in map[float64]float64) map[float64]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[float64]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStepIndicesMap(in map[string][]int) map[string][]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]int, len(in))
	for k, v := range in {
		out[k] = append([]int(nil), v...)
	}
	return out
}

// saveCurrentSessionToLRULocked snapshots the currently selected session's
// authoritative durable history into the bounded LRU cache before switching away.
func (s *Store) saveCurrentSessionToLRULocked() {
	if s.selectedID == "" || !s.historyReady || s.historyErr != nil {
		return
	}
	durable, _, splitOK := s.splitVolatileTailFastLocked()
	if !splitOK {
		durable = make([]Message, 0, len(s.messages))
		for _, m := range s.messages {
			if !volatileMessageRow(m) {
				durable = append(durable, m)
			}
		}
	} else {
		durable = append([]Message(nil), durable...)
	}
	var cachedBoundaries map[float64]json.RawMessage
	if !s.trajectoryDisabled {
		cachedBoundaries = cloneCachedTrajectoryBoundaries(s.trajectoryBoundaries)
	}
	entry := &cachedSessionState{
		key: sessionCacheKey{
			client:         s.client,
			hostGeneration: s.eventsGeneration,
			sessionID:      s.selectedID,
		},
		messages:                durable,
		trajectoryBoundaries:    cachedBoundaries,
		workflowBoundaries:      cloneRawBoundaryMap(s.workflowBoundaries),
		systemPromptAnchors:     cloneFloatMap(s.systemPromptAnchors),
		planBoard:               clonePlanBoard(s.planBoard),
		planHistoryBinding:      s.planHistoryBinding,
		turnFilesHistoryBinding: s.turnFilesHistoryBinding,
		historyThroughSeq:       s.historyThroughSeq,
		historyHasMore:          s.historyHasMore,
		messageRatings:          cloneStringStringMap(s.messageRatings),
		durableHasEdits:         s.durableHasEdits,
	}
	// Release rebuildable derived caches when backgrounding a session so
	// inactive entries only hold authoritative durable state.
	s.sessionLRU.put(entry)
}

// restoreSessionFromLRULocked populates the store immediately from the bounded
// LRU cache when switching back to a recently visited session, eliminating the
// blank-screen flash while background follow catches up with any new events.
func (s *Store) restoreSessionFromLRULocked(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	entry, ok := s.sessionLRU.get(sessionCacheKey{
		client:         s.client,
		hostGeneration: s.eventsGeneration,
		sessionID:      sessionID,
	})
	if !ok {
		return false
	}
	s.messages = append([]Message(nil), entry.messages...)
	s.trajectoryBoundaries = cloneRawBoundaryMap(entry.trajectoryBoundaries)
	s.workflowBoundaries = cloneRawBoundaryMap(entry.workflowBoundaries)
	s.systemPromptAnchors = cloneFloatMap(entry.systemPromptAnchors)
	s.planBoard = clonePlanBoard(entry.planBoard)
	s.planHistoryBinding = entry.planHistoryBinding
	s.turnFilesHistoryBinding = entry.turnFilesHistoryBinding
	s.historyThroughSeq = entry.historyThroughSeq
	s.historyHasMore = entry.historyHasMore
	s.messageRatings = cloneStringStringMap(entry.messageRatings)
	s.historyReady = true
	s.historyLoading = false
	s.durableHasEdits = entry.durableHasEdits
	s.durableTimedCache = nil
	s.durableUnmappedCompactions = nil
	s.durableStepIndices = nil
	s.durableCacheValid = false
	s.msgProjCache = nil
	s.workflowRunsCache = nil
	s.markHistoryDirtyLocked()
	s.markBoundaryDirtyLocked()
	s.markWorkflowDirtyLocked()
	return true
}

// sameSessionWorkspaceLocked reports whether prevID and nextID belong to the
// same known workspace (registered workspace ID or identical normalized non-empty
// session cwd), allowing workspace-scoped skills and slash commands to be reused
// across session switches without flashing empty.
func (s *Store) sameSessionWorkspaceLocked(prevID, nextID string) bool {
	if prevID == "" || nextID == "" {
		return false
	}
	if prevID == nextID {
		return true
	}
	if len(s.workspaceMemberships) > 0 {
		var prevWS, nextWS string
		for wsID, m := range s.workspaceMemberships {
			for _, sid := range m.SessionIDs {
				if sid == prevID {
					prevWS = wsID
				}
				if sid == nextID {
					nextWS = wsID
				}
			}
		}
		if prevWS != "" && prevWS == nextWS {
			return true
		}
		if prevWS != "" || nextWS != "" {
			return false
		}
	}
	var prevCwd, nextCwd string
	for _, it := range s.items {
		if it.SessionID == prevID {
			prevCwd = it.Cwd
		} else if it.SessionID == nextID {
			nextCwd = it.Cwd
		}
		if prevCwd != "" && nextCwd != "" {
			break
		}
	}
	return prevCwd != "" && nextCwd != "" && hostPathKey(prevCwd) == hostPathKey(nextCwd)
}

// streamNotifyInterval bounds high-frequency assistant-stream UI notifications
// to a 60fps (~16ms) coalescing window while keeping state updates synchronous
// and flushing terminal/error/interactive events immediately.
const streamNotifyInterval = 16 * time.Millisecond

func (s *Store) stopStreamNotifyTimerLocked() {
	if s.streamNotifyTimer != nil {
		s.streamNotifyTimer.Stop()
		s.streamNotifyTimer = nil
	}
	s.streamNotifyPending = false
}

func (s *Store) flushStreamNotifyLocked() {
	s.stopStreamNotifyTimerLocked()
	s.lastStreamNotify = time.Now()
}
