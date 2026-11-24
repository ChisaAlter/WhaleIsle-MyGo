package session

import (
	"encoding/json"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/workflowrun"
)

const defaultSessionCacheLimit = 24

type sessionCacheKey struct {
	client         *dsh.Client
	hostGeneration uint64
	sessionID      string
}

type cachedSessionState struct {
	key                        sessionCacheKey
	messages                   []Message
	trajectoryBoundaries       map[float64]json.RawMessage
	workflowBoundaries         map[float64]json.RawMessage
	systemPromptAnchors        map[float64]float64
	planBoard                  PlanBoard
	planHistoryBinding         string
	turnFilesHistoryBinding    string
	historyThroughSeq          float64
	historyHasMore             bool
	messageRatings             map[string]string
	durableTimedCache          []Message
	durableUnmappedCompactions []Message
	durableStepIndices         map[string][]int
	durableCacheValid          bool
	durableHasEdits            bool
	msgProjCache               []Message
	workflowRunsCache          []workflowrun.Run
}

type sessionLRUCache struct {
	limit   int
	entries []*cachedSessionState // oldest at 0, most recently used at len-1
}

func (c *sessionLRUCache) maxEntries() int {
	if c.limit > 0 {
		return c.limit
	}
	return defaultSessionCacheLimit
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
	for i, e := range c.entries {
		if e.key == entry.key {
			copy(c.entries[i:], c.entries[i+1:])
			c.entries[len(c.entries)-1] = entry
			return
		}
	}
	limit := c.maxEntries()
	if len(c.entries) >= limit {
		copy(c.entries, c.entries[1:])
		c.entries[len(c.entries)-1] = entry
		return
	}
	c.entries = append(c.entries, entry)
}

func (c *sessionLRUCache) removeSession(sessionID string) {
	if sessionID == "" || len(c.entries) == 0 {
		return
	}
	kept := c.entries[:0]
	for _, e := range c.entries {
		if e.key.sessionID != sessionID {
			kept = append(kept, e)
		}
	}
	for i := len(kept); i < len(c.entries); i++ {
		c.entries[i] = nil
	}
	c.entries = kept
}

func (c *sessionLRUCache) clear() {
	for i := range c.entries {
		c.entries[i] = nil
	}
	c.entries = c.entries[:0]
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
	durable, volatile, splitOK := s.splitVolatileTailFastLocked()
	if !splitOK {
		durable = make([]Message, 0, len(s.messages))
		for _, m := range s.messages {
			if !volatileMessageRow(m) {
				durable = append(durable, m)
			}
		}
		volatile = nil
	} else {
		durable = append([]Message(nil), durable...)
	}
	entry := &cachedSessionState{
		key: sessionCacheKey{
			client:         s.client,
			hostGeneration: s.eventsGeneration,
			sessionID:      s.selectedID,
		},
		messages:                durable,
		trajectoryBoundaries:    cloneRawBoundaryMap(s.trajectoryBoundaries),
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
	if s.durableCacheValid && !s.durableHasEdits {
		entry.durableTimedCache = s.durableTimedCache
		entry.durableUnmappedCompactions = s.durableUnmappedCompactions
		entry.durableStepIndices = cloneStepIndicesMap(s.durableStepIndices)
		entry.durableCacheValid = true
	}
	if len(volatile) == 0 && s.historyValid && s.historySelectedID == s.selectedID {
		entry.msgProjCache = s.msgProjCache
		entry.workflowRunsCache = s.workflowRunsCache
	}
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
	if entry.durableCacheValid {
		s.durableTimedCache = entry.durableTimedCache
		s.durableUnmappedCompactions = entry.durableUnmappedCompactions
		s.durableStepIndices = cloneStepIndicesMap(entry.durableStepIndices)
		s.durableCacheValid = true
	}
	if entry.msgProjCache != nil {
		s.msgProjCache = entry.msgProjCache
		s.workflowRunsCache = entry.workflowRunsCache
	}
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
