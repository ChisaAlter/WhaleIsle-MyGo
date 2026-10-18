package agents

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

func paintAgentAncestors(c *ui.Context, ancestors []session.AgentNavigationAncestor, jump func(int)) {
	if len(ancestors) == 0 {
		return
	}
	ui.Row(c).FillWidth().Wrap().Padding(4, 12).Gap(4).Children(func() {
		for _, ancestor := range ancestors {
			label := ancestor.Title
			if label == "" {
				label = ancestor.SessionID
			}
			if ui.Button(c, label).Label("Agent ancestor "+ancestor.SessionID).Tooltip(ancestor.SessionID).FontSize(theme.InterfaceSize(11)).Clicked() && jump != nil {
				jump(ancestor.Depth)
			}
			ui.Text(c, "›")
		}
	})
}
