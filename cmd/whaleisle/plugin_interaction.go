package main

import "github.com/egoist/mygo/ui"

func (s *mainState) renderPluginInteractionSlot(c *ui.Context, slot, agentID, sourceRequestID, callID string, fallback func()) {
	if owner := s.currentNativePluginSessionOwner(); owner != nil && owner.RenderNativeInteraction(c, slot, agentID, sourceRequestID, callID) {
		return
	}
	if fallback != nil {
		fallback()
	}
}
