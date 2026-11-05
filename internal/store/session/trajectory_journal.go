package session

import "sort"

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
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Seq == 0 {
			return false
		}
		if rows[j].Seq == 0 {
			return true
		}
		return rows[i].Seq < rows[j].Seq
	})
	return rows
}
