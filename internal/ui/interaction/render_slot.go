package interaction

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
)

// A slot keeps the same pending owner as the visible native takeover. The
// source request identity is intentionally distinct from waterfall EventID.
type SlotRenderer func(c *ui.Context, slot, agentID, sourceRequestID, callID string, fallback func())

func renderInteractionSlot(c *ui.Context, pending interaction.Pending, slot string, fallback func(), renderers ...SlotRenderer) {
	if len(renderers) > 0 && renderers[0] != nil {
		renderers[0](c, slot, pending.AgentID, pending.SourceRequestID, pending.CallID, fallback)
		return
	}
	fallback()
}
