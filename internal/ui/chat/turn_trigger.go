package chat

import (
	"encoding/json"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TurnTriggerView identifies the durable nonhuman input that opened a Turn.
// Its source changes attribution only; the body is retained, read-only text.
func TurnTriggerView(c *ui.Context, msg session.Message, scope string, palette theme.Colors) {
	source, _ := contextMessageParts(msg)
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
	t := design.For(c)
	muted, secondary := t.MutedForeground, t.Foreground
	root := chatCardSurface(c, ui.Column(c.Key("turn-trigger:"+scope+":"+msg.ID)).FillWidth().MinWidth(0), design.RadiusXL2).
		Transition(ui.ElementTransition{Colors: true, Duration: 100 * time.Millisecond, Ease: contextColorEase})
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		header := ui.ButtonBase(c).Label(title).Expanded(*open).Role(ui.RoleDisclosure).
			FillWidth().MinWidth(0).Padding(design.Sp(3), design.Sp(4)).Gap(design.Sp(2.5)).Radius(design.RadiusXL2).
			Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer).FocusRing(false)
		rootHovered, headerHovered := root.Hovered(), header.Hovered()
		if rootHovered {
			root.Background(t.Accent)
		}
		target := float32(0)
		if headerHovered {
			target = 1
		}
		ink := muted.Mix(secondary, header.AnimateWith("trigger-hover", target, 100*time.Millisecond, contextColorEase))
		if header.Clicked() {
			*open = !*open
			c.Invalidate()
		}
		header.Children(func() {
			ui.Icon(c, icon).Size(design.IconXS, design.IconXS).TextColor(ink).Shrink(0).Role(ui.RoleNone)
			ui.Text(c, title).Font(theme.RenderPreferences().Interface.Family).FontSize(theme.RenderPreferences().Chat.Size).LineHeight(0).TextColor(ink).Shrink(0).SingleLine()
			if stamp := formatMessageClock(msg.EventTime, c.Now()); stamp != "" {
				ui.Text(c, stamp).Font(theme.RenderPreferences().Interface.Family).FontSize(design.TextXS.Size).FixedLineHeight(18).TextColor(ink).SingleLine().Shrink(0).Margin(0, 0, 0, ui.Auto)
			}
			chevron := ui.Icon(c, contextChevronIcon).Size(12, 12).TextColor(ink).Shrink(0).Role(ui.RoleNone)
			if *open {
				chevron.Rotate(180)
			}
		})
		if !*open {
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(0, design.Sp(4), design.Sp(3), design.Sp(10)).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatTriggerExplanation)).Font(theme.RenderPreferences().Interface.Family).FontSize(design.TextXS.Size).FixedLineHeight(18).TextColor(t.MutedForeground).Margin(design.Sp(2), 0)
			style := contextBodyPaletteFor(c, palette)
			style.Font, style.Size, style.LineHeight = theme.RenderPreferences().Interface.Family, design.TextXS.Size, 18
			ui.Scroll(c).ScrollOverflowX().FillWidth().MinWidth(0).MaxHeight(240).Children(func() {
				paintNoticeContext(c, msg, style)
			})
		})
	})
}
