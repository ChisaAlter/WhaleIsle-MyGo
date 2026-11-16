package files

// Explicit activation/Refresh adopts a complete local disk baseline while
// retaining the dirty draft, as FilePreview does. Failed/incomplete reads and
// Host-owned resources do not replace a dirty local editor's baseline.
type textReadState struct {
	preview, draft             string
	conflictContent            string
	truncated, remote, blocked bool
	hasConflict, conflictTrunc bool
}

func reconcileTextRead(before textReadState, result textReadResult) textReadState {
	out := before
	out.blocked = result.err != nil
	if result.err == nil {
		out.truncated = result.truncated
		if result.remote {
			out.remote = true
		}
	}
	if before.preview != before.draft {
		if result.err != nil {
			out.blocked = true
			return out
		}
		if result.truncated {
			out.blocked = true
			out.hasConflict = false
			out.conflictContent = ""
			out.conflictTrunc = false
			return out
		}
		if result.text != before.preview {
			// Retain the previous baseline (before.preview) and unsaved draft
			// separately from disk V2 so autosave can never advance Preview to V2
			// and overwrite external changes without explicit user resolution.
			out.blocked = true
			out.hasConflict = true
			out.conflictContent = result.text
			out.conflictTrunc = false
			return out
		}
		out.hasConflict = false
		out.conflictContent = ""
		out.conflictTrunc = false
		return out
	}
	if result.err == nil {
		out.preview, out.draft = result.text, result.text
		out.remote = result.remote
		out.hasConflict = false
		out.conflictContent = ""
		out.conflictTrunc = false
	}
	return out
}
