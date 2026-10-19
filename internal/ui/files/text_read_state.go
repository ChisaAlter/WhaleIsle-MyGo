package files

// Explicit activation/Refresh adopts a complete local disk baseline while
// retaining the dirty draft, as FilePreview does. Failed/incomplete reads and
// Host-owned resources do not replace a dirty local editor's baseline.
type textReadState struct {
	preview, draft             string
	truncated, remote, blocked bool
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
		if result.err == nil && !result.truncated && !result.remote && !before.remote {
			out.preview = result.text
		} else if result.err == nil && result.text != before.preview {
			out.blocked = true
		}
		return out
	}
	if result.err == nil {
		out.preview, out.draft = result.text, result.text
		out.remote = result.remote
	}
	return out
}
