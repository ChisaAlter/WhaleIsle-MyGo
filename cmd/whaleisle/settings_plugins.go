package main

import (
	"strings"

	"whaleisle/internal/dsh/api"
)

// Inspection shares the generation-bound operation gate with lifecycle changes,
// so a stale Host answer cannot replace the current plugin page's feedback.
func (s *mainState) inspectInstalledPlugin(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	op, _ := s.beginMarketAttempt("", nil, false)
	if op == nil {
		return
	}
	op.method = "pluginManager/inspect"
	go func() {
		defer s.releaseMarket(op)
		var out api.PluginSpecInspection
		err := op.client.Call(op.ctx, op.method, map[string]any{"spec": name}, &out)
		message := name
		if err != nil {
			message = err.Error()
		} else if len(out) > 0 {
			message = string(out)
		}
		s.finishMarket(op, marketResult{message: message}, false)
	}()
}
