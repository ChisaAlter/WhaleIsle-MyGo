package interaction

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func approvalFilePath(args string) string {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return ""
	}
	var payload struct {
		FilePath  string `json:"file_path"`
		FilePath2 string `json:"filePath"`
		Path      string `json:"path"`
	}
	if json.Unmarshal([]byte(trimmed), &payload) == nil {
		for _, candidate := range []string{payload.FilePath, payload.FilePath2, payload.Path} {
			if text := strings.TrimSpace(candidate); text != "" {
				return text
			}
		}
	}
	return ""
}

func approvalArguments(c *ui.Context, p interaction.Pending, store CardStore, keys *approvalKeys, fg, muted, accent ui.Color) {
	if p.Args == "" {
		return
	}
	ui.Column(c).FillWidth().Padding(6, 8, 6, 8).Background(accent.Alpha(0.06)).Gap(4).Children(func() {
		if path := approvalFilePath(p.Args); path != "" {
			ui.Text(c, path).Font(theme.RenderPreferences().Code.Family).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
		}
		ui.Column(c.Key("approval-arguments:" + p.AgentID + ":" + p.EventID)).FillWidth().MinWidth(0).Children(func() {
			ui.Text(c, p.Args).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).Selectable().TextColor(fg)
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
