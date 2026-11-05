package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

// AccountMenu exposes only metadata until a selected action mounts its original
// component. The shipped remote action is the single native phone popup route.
func (s *mainState) pluginAccountLauncherActions(menu *ui.Menu) {
	contributions := s.pluginPublicContributions()
	invoke := contributions.Invoke
	contributions.Invoke = func(action pluginui.PublicMenuAction) {
		if action.ID == "remote" && (action.Registrant == "@deepseek-ai/dsh-client-ui-settings-remote" || action.Registrant == pluginui.BundledRegistrant) {
			if owner := s.pluginNavigationOwner(); owner != nil && owner.PublicMenuActionCurrent(action) {
				s.openPhonePopup()
			}
			return
		}
		if invoke != nil {
			invoke(action)
		}
	}
	contributions.RenderMenu(menu, "settings.launcher.action", "")
}

func (s *mainState) renderPluginAccountLauncherPopup(c *ui.Context, anchor ui.Element) {
	s.renderPluginPublicSlot(c, "settings.launcher.action", "", nil)
	s.renderPhonePopup(c, anchor)
}
