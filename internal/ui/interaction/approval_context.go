package interaction

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func approvalArguments(c *ui.Context, p interaction.Pending, store *interaction.Store, keys *approvalKeys, fg, muted, accent ui.Color) {
	if p.ApprovalContext != "" {
		ui.Text(c, p.ApprovalContext).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if p.Args == "" {
		return
	}
	ui.Column(c).FillWidth().Padding(6, 8, 6, 8).Background(accent.Alpha(0.06)).Gap(4).Children(func() {
		ui.Scroll(c.Key("approval-arguments:" + p.AgentID + ":" + p.EventID)).FillWidth().MaxHeight(220).Children(func() {
			ui.Text(c, p.Args).FontSize(theme.InterfaceSize(11)).TextColor(fg)
		})
		copyButton := ui.Button(c, i18n.T(i18n.KeyInteractionCopyArgs)).FontSize(theme.InterfaceSize(11))
		if keys != nil {
			copyButton.HandleInput(approvalHandler(keys, p, store, true))
		}
		if copyButton.Clicked() && strings.TrimSpace(p.Args) != "" {
			mygo.Clipboard.WriteText(p.Args)
		}
	})
}
