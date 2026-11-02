package session

// BelongsTo reports the immutable Store owner of this render-pass handle. It
// does not read a view, extend the pass lifetime, or permit cross-pass reuse.
func (f *FrameSnapshot) BelongsTo(store *Store) bool {
	return f != nil && store != nil && f.store == store
}
