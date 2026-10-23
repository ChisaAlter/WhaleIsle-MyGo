package chat

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TurnTriggerView identifies the durable nonhuman input that opened a Turn.
// Its source changes attribution only; the body is retained, read-only text.
func TurnTriggerView(c *ui.Context, msg session.Message, scope string, palette theme.Colors) {
	source, content := contextMessageParts(msg)
	field := func(name string) string {
		var text string
		_ = json.Unmarshal(source[name], &text)
		return text
	}
	label := i18n.KeyChatTriggerRequest
	icon := contextInjectionIcon
	switch field("kind") {
	case "goal":
		label, icon = i18n.KeyChatTriggerGoal, triggerIcons["goal"]
	case "agent-message":
		label, icon = i18n.KeyChatTriggerAgent, triggerIcons["agent"]
	case "team-message":
		label, icon = i18n.KeyChatTriggerTeam, triggerIcons["team"]
	case "subagent-settled":
		label, icon = i18n.KeyChatTriggerSubagent, triggerIcons["team"]
	case "webhook":
		label, icon = i18n.KeyChatTriggerWebhook, triggerIcons["webhook"]
		if field("provider") == "github" {
			label, icon = i18n.KeyChatTriggerGithub, triggerIcons["github"]
		}
	case "schedule":
		label, icon = i18n.KeyChatTriggerSchedule, triggerIcons["schedule"]
	case "tool-jobs":
		label, icon = i18n.KeyChatTriggerJob, triggerIcons["job"]
	case "cordis-host-runner":
		label, icon = i18n.KeyChatTriggerPlugin, triggerIcons["plugin"]
	}
	title := i18n.T(label)
	muted, secondary := chrome.Color(palette.Muted), chrome.Color(palette.Secondary)
	root := ui.Column(c.Key("turn-trigger:"+scope+":"+msg.ID)).FillWidth().MinWidth(0).
		Border(.5, chrome.Color(palette.Border).Alpha(.5)).Radius(20).Background(chrome.Color(palette.Field))
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		header := ui.Button(c, "").Label(title).Expanded(*open).Role(ui.RoleDisclosure).
			FillWidth().MinWidth(0).Padding(12, 16).Gap(10).BorderWidth(0).Radius(20).
			Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Start)
		ink := muted
		if header.Hovered() {
			ink = secondary
			root.Background(chrome.Color(palette.Hover))
		}
		if header.Clicked() {
			*open = !*open
			c.Invalidate()
		}
		header.Children(func() {
			ui.Icon(c, icon).Size(14, 14).TextColor(ink).Shrink(0).Role(ui.RoleNone)
			ui.Text(c, title).FontSize(theme.CurrentPreferences().Chat.Size).TextColor(ink).Shrink(1).MinWidth(0).SingleLine()
			ui.Box(c).Grow(1).MinWidth(0)
			if stamp := formatEventTime(msg.EventTime); stamp != "" {
				ui.Text(c, stamp).FontSize(12).TextColor(ink).SingleLine().Shrink(0)
			}
			chevron := ui.Icon(c, contextChevronIcon).Size(12, 12).TextColor(ink).Shrink(0).Role(ui.RoleNone)
			if *open {
				chevron.Rotate(180)
			}
		})
		if !*open {
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(0, 16, 12, 40).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatTriggerExplanation)).FontSize(12).TextColor(secondary).Margin(8, 0)
			body := msg.Body
			if len(content) > 0 {
				var text strings.Builder
				allText := true
				for _, block := range content {
					allText = allText && block.Type == "text"
					text.WriteString(block.Text)
				}
				if allText {
					body = text.String()
				}
			}
			ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(240).Children(func() {
				ui.Text(c, clipReadOnlyText(body, 20000)).FontSize(12).TextColor(secondary).Selectable()
			})
		})
	})
}
