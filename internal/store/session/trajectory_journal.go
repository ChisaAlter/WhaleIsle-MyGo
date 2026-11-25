package session

import "sort"

// The journal and the chat transcript have different replacement rules. Keep
// the journal's frozen durable rows separately, while sharing only the parsed
// timing inputs. Streaming changes then project the active suffix, not history.
type trajectoryMessageCache struct {
	rows        []Message
	stepIndices map[string][]int
}

func (s *Store) trajectoryMessagesViewLocked(historyChanged bool) []Message {
	durable, volatile, tail := s.splitVolatileTailFastLocked()
	if !tail || s.assistantStream.retained != nil || s.assistantStream.rebaseline {
		s.trajectoryMessageCache = nil
		return s.trajectoryJournalMessagesLocked()
	}
	cache := s.trajectoryMessageCache
	if historyChanged || cache == nil {
		rows := s.projectAssistantMessagesLocked(durable)
		rows, unmapped, stepIndices := s.projectDurableTimingLocked(rows)
		// Preserve the timing decoder's exact identity rules, including a Raw
		// record that differs from the retained boundary at the same sequence.
		liveIDs := make(map[string]string)
		for opID, indices := range stepIndices {
			for _, i := range indices {
				liveIDs[rows[i].ID] = opID
			}
		}
		rows = append(rows, unmapped...)
		rows = cloneMessages(s.projectSystemPromptsLocked(rows))
		sortTrajectoryMessages(rows)
		cache = &trajectoryMessageCache{rows: rows, stepIndices: make(map[string][]int)}
		for i, row := range rows {
			if opID := liveIDs[row.ID]; opID != "" {
				cache.stepIndices[opID] = append(cache.stepIndices[opID], i)
			}
		}
		s.trajectoryMessageCache = cache
	}
	rows := make([]Message, len(cache.rows), len(cache.rows)+len(volatile))
	copy(rows, cache.rows)
	_, ops, _ := s.ensureTrajectoryTimingCacheLocked()
	opID, timing, live := s.liveAttemptTimingLocked(ops)
	if timing != nil {
		for _, i := range cache.stepIndices[opID] {
			rows[i].TrajectoryTime = cloneTrajectoryTiming(timing)
		}
	}
	for _, message := range volatile {
		for _, row := range projectAssistantMessage(message) {
			row = cloneMessage(row)
			if timing := live[row.ID]; timing != nil {
				row.TrajectoryTime = cloneTrajectoryTiming(timing)
			}
			rows = append(rows, row)
		}
	}
	beforePreparing := len(rows)
	rows = s.projectPreparingToolsLocked(rows)
	for i := beforePreparing; i < len(rows); i++ {
		rows[i] = cloneMessage(rows[i])
	}
	return rows[:len(rows):len(rows)]
}

// Trajectory owns its event ledger, not Chat's replacement surface. Reuse the
// same assistant/timing decoders while deliberately retaining the whole journal.
func (s *Store) trajectoryJournalMessagesLocked() []Message {
	rows := s.projectAssistantMessagesLocked(s.messages)
	rows = s.projectPreparingToolsLocked(rows)
	rows = s.projectTrajectoryTimingLocked(rows)
	rows = s.projectSystemPromptsLocked(rows)
	rows = cloneMessages(rows)
	if rows == nil {
		rows = []Message{}
	}
	sortTrajectoryMessages(rows)
	return rows
}

func sortTrajectoryMessages(rows []Message) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Seq == 0 {
			return false
		}
		if rows[j].Seq == 0 {
			return true
		}
		return rows[i].Seq < rows[j].Seq
	})
}
