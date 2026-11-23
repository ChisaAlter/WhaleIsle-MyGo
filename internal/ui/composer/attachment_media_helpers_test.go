package composer

import (
	"time"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/store/session"
)

// settleAttachment waits for an attachment's media request without starting
// its animation clock, then copies the result as a frame would.
func settleAttachment(v *attachmentDisplay) *attachmentDisplay {
	if v == nil || !v.image {
		return v
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && composerMedia().Request(nil, v.key, v.src).State() == media.Pending {
		if composerMedia().RunPending() == 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	v.refresh(nil, time.Time{})
	return v
}

// readSettled is readAttachmentDisplay followed by its decode.
func readSettled(a session.ComposerAttachment, full ...bool) *attachmentDisplay {
	return settleAttachment(readAttachmentDisplay(a, full...))
}

// advance shows the frame due at now and returns the wait to the next.
func (v *attachmentDisplay) advance(now time.Time) time.Duration { return v.refresh(nil, now) }
