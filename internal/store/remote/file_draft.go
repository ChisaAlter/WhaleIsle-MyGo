package remote

// FileDraftBuffer belongs to one resource occurrence. It publishes committed
// native input independently of the next widget frame; marked IME text is not
// part of CommittedText. Implementations must be safe from any goroutine.
type FileDraftBuffer interface {
	CommittedText() string
	ReplaceText(string)
}

type fileDraftOwner struct{ buffer FileDraftBuffer }

// EditBuffer returns the buffer retained by the exact captured resource. A
// stale widget cannot attach to a replacement session, remote identity or tab.
// The factory runs only once for that resource and must not call back into Files.
func (s *Files) EditBuffer(captured FileSnapshot, create func(string) FileDraftBuffer) FileDraftBuffer {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.states[s.key]
	if v == nil || v.BindingID != captured.BindingID || v.SessionID != captured.SessionID || v.RemoteIdentity != captured.RemoteIdentity || v.FilePath != captured.FilePath || !v.FileLoaded || v.File.Binary || v.File.Truncated {
		return nil
	}
	i, j := resourceIndex(v, v.FilePath), resourceIndex(&captured, captured.FilePath)
	if i < 0 || j < 0 || v.Tabs[i].draftOwner == nil || v.Tabs[i].draftOwner != captured.Tabs[j].draftOwner {
		return nil
	}
	r := &v.Tabs[i]
	if r.draftOwner.buffer == nil && create != nil {
		r.draftOwner.buffer = create(r.Draft)
	}
	return r.draftOwner.buffer
}

// syncResourceDrafts is always called under Files.mu. Sources remain attached
// to their captured resource even while a different session is selected.
func syncResourceDrafts(v *FileSnapshot) {
	if v == nil {
		return
	}
	stashResource(v)
	for i := range v.Tabs {
		r := &v.Tabs[i]
		if r.draftOwner != nil && r.draftOwner.buffer != nil {
			r.Draft = r.draftOwner.buffer.CommittedText()
			if r.Path == v.FilePath {
				v.Draft = r.Draft
			}
		}
	}
}

func (s *Files) syncDraftsLocked() {
	for _, v := range s.states {
		syncResourceDrafts(v)
	}
}

// Explicit model replacement (read, discard, SetDraft) updates only this
// resource's native buffer. Ordinary capture never resets undo or composition.
func replaceResourceDraft(v *FileSnapshot, text string) {
	v.Draft = text
	if i := resourceIndex(v, v.FilePath); i >= 0 {
		if owner := v.Tabs[i].draftOwner; owner != nil && owner.buffer != nil {
			owner.buffer.ReplaceText(text)
		}
	}
}
