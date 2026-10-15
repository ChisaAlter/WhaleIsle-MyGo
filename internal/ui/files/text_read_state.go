package files

// Reconcile only complete owned read results. An editor's dirty baseline is
// never replaced by a reread, even when the disk changed or became truncated.
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
		if result.err == nil && result.text != before.preview {
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
