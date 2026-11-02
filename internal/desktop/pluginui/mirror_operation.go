package pluginui

import "context"

// awaitMirrorCapture waits only on a worker. A pending input marks the source
// reserved before reaching here, so no new capture can overtake it. The channel
// closes at the actual CapturePreview completion, never at context expiry.
func (o *Owner) awaitMirrorCapture(ctx context.Context, state *mirrorState) error {
	o.mu.Lock()
	if state == nil || o.closed || o.mirror != state {
		o.mu.Unlock()
		return errSessionUnavailable
	}
	done := state.captureDone
	o.mu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
