package main

import "github.com/egoist/mygo/ui"

func (s *mainState) renderPluginModelExtras(c *ui.Context, provider, namespace string) {
	if owner := s.pluginNavigationOwner(); owner != nil {
		owner.RenderModelExtras(c, provider, namespace)
	}
}
func (s *mainState) renderPluginModelFooter(c *ui.Context) { s.renderPluginModelExtras(c, "", "") }
