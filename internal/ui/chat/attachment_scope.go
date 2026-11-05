package chat

import "whaleisle/internal/store/session"

var attachmentResolver func(binding, sessionID string) session.AttachmentRead

// SetAttachmentResolver supplies immutable render-time canonical owners. The
// application installs it once; no asynchronous load consults selectedID.
func SetAttachmentResolver(resolve func(string, string) session.AttachmentRead) {
	attachmentResolver = resolve
}

// Rendering is synchronous on the UI thread. Nested child occurrences replace
// and restore this lexical seat; workers retain only the immutable copied read.
var renderedAttachmentRead session.AttachmentRead

func WithAttachmentRead(read session.AttachmentRead, render func()) {
	previous := renderedAttachmentRead
	renderedAttachmentRead = read
	defer func() { renderedAttachmentRead = previous }()
	render()
}
func renderAttachmentSnapshot(snapshot session.ViewState, render func()) {
	if attachmentResolver == nil {
		render()
		return
	}
	WithAttachmentRead(attachmentResolver(snapshot.PlanHistoryBinding, snapshot.SelectedID), render)
}
func attachmentImageRead() session.AttachmentRead {
	if renderedAttachmentRead.Key != "" {
		return renderedAttachmentRead
	}
	// Existing standalone callers retain their explicitly installed loader. The
	// production application uses SetAttachmentResolver instead of this legacy seat.
	if attachmentResolver != nil {
		return session.AttachmentRead{}
	}
	attachmentFetcher.mu.Lock()
	defer attachmentFetcher.mu.Unlock()
	return session.AttachmentRead{Fetch: attachmentFetcher.fn}
}
