package pluginui

import "time"

// A screenshot's focus sample is older than later typed acknowledgements. It
// must never roll the stable native text client back when its node is rebuilt.
func (c *MirrorTextClient) applyTextSampleLocked(focus *mirrorTextFocus, at time.Time) {
	if at.IsZero() || at.Before(c.updated) {
		return
	}
	c.applyFocusLocked(focus)
	c.updated = at
}
