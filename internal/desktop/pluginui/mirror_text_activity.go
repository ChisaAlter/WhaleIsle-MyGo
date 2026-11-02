package pluginui

import "time"

// Native IME may query/update composition between asynchronous acknowledgements.
// Capturing during that interval can wait for an occluded WebView surface and
// starve its next text operation; keep text ahead until composition ends.
func (s *mirrorTextState) captureDeferred() bool {
	if s == nil {
		return false
	}
	if s.busy || len(s.queue) > 0 || time.Since(s.lastInput) < 1200*time.Millisecond {
		return true
	}
	for _, client := range s.clients {
		if client.pending > 0 || client.marked != nil {
			return true
		}
	}
	return false
}
