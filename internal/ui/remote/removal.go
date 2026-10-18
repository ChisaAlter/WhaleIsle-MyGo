package remote

import remotestore "whaleisle/internal/store/remote"

// Keep the review and unsaved editor intact on failure. Only the matching
// successful deletion may close them, never a different Host's late result.
func (e *MachineEditor) syncRemoval(s remotestore.Snapshot) {
	if e.PendingRemove == "" {
		return
	}
	if e.RemovalGeneration != s.Generation {
		e.PendingRemove = ""
		e.Removing = false
		return
	}
	if !e.Removing || s.Busy {
		return
	}
	e.Removing = false
	if s.RemovedID != e.PendingRemove {
		return
	}
	if e.Draft.ID == e.PendingRemove {
		e.Open = false
		e.Draft = remotestore.Draft{}
	}
	e.PendingRemove = ""
}
