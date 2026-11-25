package composer

import (
	"strconv"

	"whaleisle/internal/store/session"
)

// Reference discovery never runs on the renderer. Each draft delegates its
// background query lifecycle to session.ReferenceQueryController.
type referenceQueryState struct {
	ctrl session.ReferenceQueryController
}

func (s *referenceQueryState) cancelQuery() {
	s.ctrl.Cancel()
}

func (s *referenceQueryState) query(m Model, q string, quoted bool) []session.ReferenceCandidate {
	scope := strconv.FormatUint(m.Snapshot.ComposerHostGeneration, 10) + ":" + strconv.FormatUint(m.Snapshot.ComposerSessionGeneration, 10) + "\x00" + m.Snapshot.SelectedID
	return s.ctrl.Query(scope, q, quoted, m.OnQueryReferences, m.referenceInvalidate)
}

func referenceQueryActive(m Model) bool { return m.OnQueryReferences != nil && m.State != nil }

func referenceQueryStatus(m Model) (bool, string) {
	if !referenceQueryActive(m) {
		return false, ""
	}
	return m.State.references.ctrl.Status()
}
