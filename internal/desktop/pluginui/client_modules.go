package pluginui

import (
	"errors"
	"whaleisle/internal/desktop/pluginslots"
)

// ClientModules describes only this retained source page. A package being
// enabled is not evidence that its browser/client implementation loaded.
func (o *Owner) ClientModules() pluginslots.ClientModulesSnapshot {
	if o == nil {
		return pluginslots.ClientModulesSnapshot{}
	}
	o.mu.Lock()
	g, closed := o.gateway, o.closed
	o.mu.Unlock()
	if closed || g == nil {
		return pluginslots.ClientModulesSnapshot{}
	}
	return g.ClientModules()
}
func (o *Owner) RetryClientModules(page uint64) error {
	if o == nil {
		return errors.New("plugin source page unavailable")
	}
	o.mu.Lock()
	g, closed := o.gateway, o.closed
	o.mu.Unlock()
	if closed || g == nil {
		return errors.New("plugin source page unavailable")
	}
	return g.RetryClientModules(page)
}
