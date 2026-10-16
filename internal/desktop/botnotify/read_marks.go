package botnotify

// ReadCandidates mirrors the original UI: initialize old profiles at their
// current activity cursor, and acknowledge new activity in the visible session.
// Unavailable inspections never establish a baseline.
func ReadCandidates(contacts []ContactActivity, sessionID string) []ContactActivity {
	var out []ContactActivity
	for _, a := range contacts {
		if a.Error == "" && a.ActivitySeq >= 0 && (!a.ReadInitialized || (a.SessionID == sessionID && a.ActivitySeq > a.LastSeenSeq)) {
			out = append(out, a)
		}
	}
	return out
}
