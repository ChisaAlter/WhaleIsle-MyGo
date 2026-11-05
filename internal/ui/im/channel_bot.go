package im

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func channelBotCard(c *ui.Context, m Model, v imonboard.View, bot imonboard.Bot, blocked bool, fg, muted ui.Color) {
	channelSurface(c, func() {
		title := bot.Bot.Name
		if title == "" {
			title = bot.BotID
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			ui.Icon(c, channelLogos[v.Channel]).Size(24, 24).Shrink(0)
			ui.Text(c, title).Grow(1).MinWidth(0).Bold().TextColor(fg)
			state := channelText("未连接", "Disconnected")
			dot := muted
			if bot.Connected {
				state, dot = channelText("已连接", "Connected"), ui.RGB(34, 160, 107)
			}
			ui.Box(c).Size(6, 6).Radius(3).Background(dot)
			ui.Text(c, state).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		})
		ui.Text(c, bot.BotID).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
		if bot.Health.Summary != "" {
			ui.Text(c, bot.Health.Summary).FillWidth().TextColor(muted)
		}
		channelHealthDetails(c, bot, fg, muted, c.Theme().Danger)
		channelBindingControls(c, m, v, bot, blocked, fg, muted)
		if v.Channel == "feishu" {
			channelFeishuPermissions(c, m, v, bot, blocked, muted)
		}
		if v.Channel == "telegram" || v.Channel == "whatsapp" {
			channelAccessPolicy(c, m, v, bot, blocked, muted)
		}
		if targetedProvision(v.Provision) && v.Provision.BotID == bot.BotID {
			channelProvisionView(c, m, v, blocked, fg, muted)
		}
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.Button(c, channelText("重新连接", "Reconnect")).Disabled(blocked || v.Provision.AttemptID != "").Clicked() {
				channelAction(&m.Store.Onboarding, v, "bot.reconnect", bot.BotID, "", "")
			}
			if ui.Button(c, channelText("发送测试消息", "Send test message")).Disabled(blocked || v.Provision.AttemptID != "").Clicked() {
				channelAction(&m.Store.Onboarding, v, "bot.reconnect-test", bot.BotID, "", "")
			}
			if ui.Button(c, channelText("断开", "Disconnect")).Disabled(blocked || !bot.Connected).Clicked() {
				channelAction(&m.Store.Onboarding, v, "bot.disconnect", bot.BotID, "", "")
			}
			if ui.Button(c, channelText("删除机器人…", "Delete bot…")).TextColor(c.Theme().Danger).Disabled(blocked || v.Provision.AttemptID != "").Clicked() {
				channelAction(&m.Store.Onboarding, v, "bot.delete", bot.BotID, "", "")
			}
		})
	})
}

func channelFeishuPermissions(c *ui.Context, m Model, v imonboard.View, bot imonboard.Bot, blocked bool, muted ui.Color) {
	s := &m.Store.Onboarding
	disabled := blocked || v.Provision.AttemptID != ""
	mention := channelText("仅在 @机器人时响应（推荐）", "Respond only when mentioned (recommended)")
	all := channelText("响应所有群消息", "Respond to all group messages")
	selected := mention
	if bot.GroupResponseMode == "all" {
		selected = all
	}
	ui.Text(c, channelText("群消息响应", "Group responses")).Bold()
	if ui.Select(c, &selected, []string{mention, all}).FillWidth().Disabled(disabled).Changed() {
		mode := "mention"
		if selected == all {
			mode = "all"
		}
		if mode == "all" && !bot.GroupMessagePermissionGranted {
			channelAction(s, v, "bot.group-message-permission.begin", bot.BotID, "", "")
		} else {
			channelAction(s, v, "bot.group-response-mode.set", bot.BotID, mode, "")
		}
	}
	help := channelText("私聊始终响应；群聊仅处理明确 @当前机器人的消息。选择全部消息后会打开飞书官方授权流程。", "Private chats always respond; groups only respond to explicit mentions of this bot. Selecting all messages opens Feishu authorization.")
	if bot.GroupMessagePermissionGranted {
		help = channelText("私聊始终响应；群消息权限已开通，切换模式无需再次授权。", "Private chats always respond. Group-message permission is granted; changing modes needs no new authorization.")
	}
	if bot.GroupResponseMode == "all" {
		help = channelText("尚未确认“获取群组中所有消息”权限（im:message.group_msg），请完成飞书授权。", "Permission to read all group messages (im:message.group_msg) is not confirmed. Complete Feishu authorization.")
		caption := channelText("授权读取全部群消息…", "Authorize all group messages…")
		if bot.GroupMessagePermissionGranted {
			help = channelText("已开通“获取群组中所有消息”权限（im:message.group_msg）；机器人会处理群聊中的所有可见消息。", "Permission to read all group messages (im:message.group_msg) is granted. This bot handles all visible group messages.")
			caption = channelText("重新授权群消息权限…", "Reauthorize group messages…")
		}
		if ui.Button(c, caption).Disabled(disabled).Clicked() {
			channelAction(s, v, "bot.group-message-permission.begin", bot.BotID, "", "")
		}
	}
	ui.Text(c, help).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	if ui.Button(c, channelText("补全权限与卡片回调…", "Repair permissions and card callbacks…")).Disabled(disabled).Clicked() {
		channelAction(s, v, "bot.callback-repair.begin", bot.BotID, "", "")
	}
	ui.Text(c, channelText("最多增量补全 card.action.trigger、读取图片/文件所需 im:message:readonly，以及上传图片/文件所需 im:resource。确认页只显示缺失项，不创建新应用；完成后仅此机器人短暂重连。", "Incrementally adds missing card.action.trigger, im:message:readonly for reading media, and im:resource for uploads. The confirmation shows only missing items and creates no new app. Only this bot briefly reconnects.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
}

type accessPolicyDraft struct{ mode, original string }

func channelAccessPolicy(c *ui.Context, m Model, v imonboard.View, bot imonboard.Bot, blocked bool, muted ui.Color) {
	owner := ui.Column(c.Key("access-policy")).FillWidth().MinWidth(0).Gap(8)
	state := ui.Local(owner, "policy", func() accessPolicyDraft {
		return accessPolicyDraft{mode: bot.AccessPolicy.AccessMode, original: bot.AccessPolicy.AccessMode}
	})
	if state.original != bot.AccessPolicy.AccessMode {
		state.mode, state.original = bot.AccessPolicy.AccessMode, bot.AccessPolicy.AccessMode
	}
	modes := []string{"compatible", "private-allowlist"}
	captions := []string{channelText("兼容模式", "Compatible"), channelText("仅白名单私聊", "Allowlisted private chats")}
	if v.Channel == "whatsapp" {
		modes, captions = []string{"self-only", "private-allowlist", "open"}, []string{channelText("仅自己", "Self only"), channelText("私聊白名单", "Private allowlist"), channelText("开放模式", "Open")}
	}
	if state.mode == "" {
		state.mode = modes[0]
	}
	selected := captions[0]
	for index, mode := range modes {
		if state.mode == mode {
			selected = captions[index]
		}
	}
	text := strings.Join(bot.AccessPolicy.AllowedUsers, "\n")
	if v.Channel == "whatsapp" {
		text = strings.Join(bot.AccessPolicy.AllowedNumbers, "\n")
	}
	if draft, ok := v.PolicyDrafts[bot.BotID]; ok {
		text = draft
	}
	disabled := blocked || v.Provision.AttemptID != ""
	owner.Children(func() {
		ui.Text(c, channelText("消息访问策略", "Message access policy")).Bold()
		if ui.Select(c, &selected, captions).FillWidth().Disabled(disabled).Changed() {
			for index, caption := range captions {
				if selected == caption {
					state.mode = modes[index]
				}
			}
		}
		explanation := channelText("兼容模式：私聊直接响应；群聊仅响应 @机器人或回复机器人。白名单模式：忽略所有群消息，仅允许所列用户私聊。", "Compatible mode responds to private messages and group mentions/replies. Allowlist mode ignores all groups and accepts private messages only from listed users.")
		if v.Channel == "whatsapp" {
			explanation = channelText("仅自己：只响应自己的自聊。私聊白名单：保留自聊并允许所列号码私聊，忽略群消息。开放模式：响应所有私聊、自己的群消息，以及群内其他人的 @ 或回复。", "Self only responds to your self-chat. Private allowlist keeps self-chat and listed private contacts, ignoring groups. Open mode accepts all private messages, your own group messages, and others’ group mentions/replies.")
		}
		ui.Text(c, explanation).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if state.mode == "private-allowlist" {
			caption := channelText("允许的 Telegram 用户 ID（每行一个，或逗号分隔）", "Allowed Telegram user IDs (one per line or comma-separated)")
			warning := channelText("白名单为空时，此机器人不会响应任何消息。", "An empty allowlist rejects every incoming message for this bot.")
			if v.Channel == "whatsapp" {
				caption = channelText("允许的国际电话号码（可带 +，每行一个或逗号分隔）", "Allowed international phone numbers (optional +, one per line or comma-separated)")
				warning = channelText("白名单为空时，仅保留自己的自聊。号码会按国际号码规范化，策略仅影响此机器人。", "An empty allowlist keeps self-chat only. Numbers are normalized internationally; this policy applies only to this bot.")
			}
			ui.Text(c, caption).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if ui.TextArea(c, &text).FillWidth().Height(84).Disabled(disabled).Changed() {
				m.Store.Onboarding.EditPolicy(v.Scope, bot.BotID, text)
			}
			if strings.TrimSpace(text) == "" {
				ui.Text(c, warning).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Danger)
			}
		}
		if ui.Button(c, channelText("保存访问策略", "Save access policy")).Disabled(disabled).Clicked() {
			channelAction(&m.Store.Onboarding, v, "bot.access-policy.set", bot.BotID, state.mode, text)
		}
	})
}

func channelReviewDialog(c *ui.Context, m Model, v imonboard.View, fg, muted ui.Color) {
	if v.Review != "bot.delete" {
		return
	}
	title := v.Target
	for _, bot := range v.Bots {
		if bot.BotID == v.Target && bot.Bot.Name != "" {
			title = bot.Bot.Name
		}
	}
	open := true
	ui.DialogBaseWithOptions(c.Key("im-bot-delete"), &open, ui.DialogOptions{DismissDisabled: v.Busy, OnDismiss: func() { m.Store.Onboarding.Dismiss() }}, func(backdrop, panel ui.Element) {
		_, height := c.Size()
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 440, 20).MaxHeight(height - 24).Gap(14).Children(func() {
			ui.Text(c, channelText("删除机器人？", "Delete bot?")).Bold().TextColor(fg)
			ui.Text(c, title+" · "+v.Target).FillWidth().TextColor(fg).Selectable()
			ui.Text(c, channelText("删除此机器人的凭据、连接与聊天绑定。工作目录和原会话内容保留。", "Removes this bot’s credentials, connection and chat bindings. Workspace files and existing sessions are retained.")).FillWidth().TextColor(muted)
			if v.Error != "" {
				channelFeedback(c, v.Error, true)
			}
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, channelText("取消", "Cancel")).Disabled(v.Busy).Clicked() {
					m.Store.Onboarding.Dismiss()
				}
				if ui.Button(c, channelText("确认删除", "Delete")).TextColor(c.Theme().Danger).Disabled(v.Busy).Clicked() {
					m.Store.Onboarding.Confirm()
				}
			})
		})
	})
}
