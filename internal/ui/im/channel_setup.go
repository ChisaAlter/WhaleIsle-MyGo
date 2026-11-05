package im

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/theme"
)

type channelPageState struct {
	credentials, submitted, loaded bool
	copyNotice, copyError          string
}

func targetedProvision(p imonboard.Provision) bool {
	return p.AttemptID != "" && p.BotID != "" && (p.Operation == "callback_repair" || p.Operation == "group_message_permission")
}

func channelAction(s *imonboard.Store, v imonboard.View, method, bot, mode, list string) {
	s.Review(v.Scope, method, bot, mode, list)
	if method != "bot.delete" {
		s.Confirm()
	}
}

func channelSurface(c *ui.Context, body func()) {
	channelSurfaceBase(c).Children(body)
}

func channelSurfaceBase(c *ui.Context) ui.Element {
	return ui.Column(c).FillWidth().MinWidth(0).Padding(12).Gap(12).Radius(12).Border(1, c.Theme().Border)
}

func channelPanelWidth(c *ui.Context, owner ui.Element) float32 {
	width := owner.Bounds().W
	if width <= 0 {
		width, _ = c.Size()
	}
	owner.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.W != width {
			c.Invalidate()
		}
	})
	return width
}

func channelFeedback(c *ui.Context, text string, failed bool) {
	ink := c.Theme().TextMuted
	if failed {
		ink = c.Theme().Danger
	}
	ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(ink).Selectable()
}

func channelHeading(c *ui.Context, m Model, v imonboard.View, page *channelPageState, blocked bool, fg, muted ui.Color) {
	p := imonboard.PlatformByID(v.Channel)
	s := &m.Store.Onboarding
	connected := 0
	for _, bot := range v.Bots {
		if bot.Connected {
			connected++
		}
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		if p.QR {
			if ui.PrimaryButton(c, channelText("扫码接入机器人", "Connect with QR")).Disabled(blocked || v.Provision.AttemptID != "").Clicked() {
				page.credentials = false
				channelAction(s, v, "provision.begin", "", "", "")
			}
		}
		if p.Secret != "" {
			caption := channelText("手动接入", "Connect manually")
			if page.credentials {
				caption = channelText("收起凭据", "Hide credentials")
			}
			if ui.Button(c, caption).Disabled(blocked || v.Provision.AttemptID != "").Clicked() {
				page.credentials = !page.credentials
				if !page.credentials {
					s.ClearCredentialDraft(v.Scope)
				}
			}
		}
		ui.Box(c).Grow(1)
		if len(v.Bots) > 0 {
			ui.Text(c, fmt.Sprintf(channelText("%d / %d 在线", "%d / %d online"), connected, len(v.Bots))).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if ui.Button(c, channelText("刷新", "Refresh")).Disabled(blocked).Clicked() {
			s.Refresh()
		}
	})
}

func channelEmptyView(c *ui.Context, m Model, v imonboard.View, page *channelPageState, blocked bool, fg, muted ui.Color) {
	copy := copyForChannel(v.Channel)
	channelSurface(c, func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(18).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(8).Children(func() {
				ui.Text(c, channelText("尚未接入", "Not connected")).TextColor(muted).FontSize(theme.InterfaceSize(12))
				ui.Text(c, copy.title).FillWidth().FontSize(theme.InterfaceSize(17)).Bold().TextColor(fg)
				ui.Text(c, copy.description).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(muted)
				if imonboard.PlatformByID(v.Channel).QR {
					if ui.PrimaryButton(c, channelText("生成二维码", "Generate QR code")).Disabled(blocked).Clicked() {
						channelAction(&m.Store.Onboarding, v, "provision.begin", "", "", "")
					}
				} else if ui.Button(c, channelText("填写 Bot Token", "Enter Bot Token")).Disabled(blocked).Clicked() {
					page.credentials = true
				}
			})
		})
	})
}

func channelCredentials(c *ui.Context, m Model, v imonboard.View, page *channelPageState, blocked bool, fg, muted ui.Color) {
	p, copy := imonboard.PlatformByID(v.Channel), copyForChannel(v.Channel)
	s := &m.Store.Onboarding
	root := channelSurfaceBase(c)
	width := channelPanelWidth(c, root)
	root.Children(func() {
		ui.Text(c, channelText("手动接入", "Connect manually")+" "+copy.name).Bold().TextColor(fg)
		ui.Text(c, copy.description).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if v.Channel == "slack" {
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				if ui.Button(c, channelText("复制 Manifest", "Copy Manifest")).Disabled(blocked).Clicked() {
					if err := mygo.Clipboard.Write(transfer.TextData(slackAppManifest), mygo.ClipboardOptions{}); err != nil {
						page.copyError = err.Error()
						page.copyNotice = ""
					} else {
						page.copyError = ""
						page.copyNotice = channelText("已复制 Manifest", "Manifest copied")
					}
				}
				ui.Link(c, channelText("打开 Slack 创建页", "Create a Slack app"), "https://api.slack.com/apps?new_app=1")
			})
			if page.copyError != "" {
				channelFeedback(c, page.copyError, true)
			} else if page.copyNotice != "" {
				channelFeedback(c, page.copyNotice, false)
			}
		}
		draft := v.Draft
		field := func(label string, value *string, secret bool) {
			column := ui.Column(c.Key(label)).MinWidth(0).Gap(6)
			if width <= 680 {
				column.FillWidth()
			} else {
				column.Grow(1).MinWidth(200)
			}
			column.Children(func() {
				ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				input := ui.TextInput(c, value).Label(label).FillWidth().Height(36).Disabled(blocked)
				if secret {
					input.Password()
				}
			})
		}
		fields := func() {
			if p.Identity != "" {
				field(p.Identity, &draft.Identity, false)
			}
			field(p.Secret, &draft.Secret, true)
			if p.Extra != "" {
				field(p.Extra, &draft.Extra, true)
			}
		}
		if width <= 680 {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(fields)
		} else {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(12).Children(fields)
		}
		if p.Extra != "" {
			ui.Text(c, channelText("Bot Token 来自 OAuth & Permissions；App Token 来自 Basic Information，必须包含 connections:write。", "Find the Bot Token in OAuth & Permissions and the App Token in Basic Information. The App Token needs connections:write.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if draft != v.Draft {
			s.Edit(v.Scope, draft)
		}
		missing := strings.TrimSpace(draft.Secret) == "" || p.Identity != "" && strings.TrimSpace(draft.Identity) == "" || p.Extra != "" && strings.TrimSpace(draft.Extra) == ""
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.PrimaryButton(c, channelText("绑定并连接", "Bind and connect")).Disabled(blocked || missing).Clicked() {
				s.Edit(v.Scope, draft)
				page.submitted = true
				channelAction(s, v, "bot.bind-credentials", "", "", "")
			}
			if ui.Button(c, channelText("取消", "Cancel")).Disabled(v.Busy).Clicked() {
				page.credentials, page.submitted = false, false
				s.ClearCredentialDraft(v.Scope)
			}
		})
	})
}

func channelProvisionView(c *ui.Context, m Model, v imonboard.View, blocked bool, fg, muted ui.Color) {
	p, copy, s := v.Provision, copyForChannel(v.Channel), &m.Store.Onboarding
	root := channelSurfaceBase(c)
	width := channelPanelWidth(c, root)
	root.Children(func() {
		if p.Applying() || p.Status == "connecting" || p.Status == "creating" {
			title := channelText("已确认，正在连接机器人", "Confirmed; connecting the bot")
			detail := channelText("正在安全保存凭据并检查消息通道，其他机器人不会中断。", "Saving credentials and checking the message connection. Other bots are unaffected.")
			if p.Operation == "callback_repair" || p.Operation == "group_message_permission" {
				title = channelText("配置已提交，正在完成授权", "Submitted; completing authorization")
				detail = channelText("正在保存权限、验证回调并重连此机器人。此阶段无法取消，其他机器人不会中断。", "Saving permissions, verifying callbacks, and reconnecting this bot. This stage cannot be cancelled; other bots are unaffected.")
			}
			ui.Text(c, title).FillWidth().Bold().TextColor(fg)
			ui.Text(c, detail).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(muted)
			if p.Cancelable() && ui.Button(c, channelText("取消接入", "Cancel setup")).Disabled(blocked).Clicked() {
				channelAction(s, v, "provision.cancel", "", "", "")
			}
			return
		}
		if p.Status == "failed" || p.Status == "expired" || p.Status == "cancelled" {
			ui.Text(c, channelText("接入尚未完成", "Setup did not finish")).Bold().TextColor(fg)
			message := p.Error.Message
			if message == "" {
				message = channelText("平台尚未完成授权，请重试。", "Platform authorization did not finish. Try again.")
			}
			if p.Status == "expired" {
				message = channelText("二维码已过期，请重新生成。", "The QR code expired. Generate a new one.")
			}
			if p.Status == "cancelled" {
				message = channelText("已取消本次接入。", "This setup was cancelled.")
			}
			channelFeedback(c, message, p.Status != "cancelled")
			if p.Error.Code != "" {
				ui.Text(c, p.Error.Code).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
			}
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				if ui.PrimaryButton(c, channelText("重新生成二维码", "Generate a new QR code")).Disabled(blocked).Clicked() {
					channelAction(s, v, "provision.begin", "", "", "")
				}
				if ui.Button(c, channelText("关闭", "Close")).Disabled(blocked).Clicked() {
					s.CloseProvision(v.Scope)
				}
			})
			return
		}
		if v.Channel == "weixin" && (p.VerificationRequired || p.Status == "needs_verification") {
			channelVerification(c, m, v, blocked, fg, muted)
			return
		}
		if p.Operation == "callback_repair" {
			copy.title = channelText("使用飞书扫码补全权限与回调", "Repair permissions with Feishu")
			copy.description = channelText("更新现有应用，增量补全卡片按钮回调、im:message:readonly 和 im:resource；不创建新应用。完成后仅此机器人短暂重连。", "Updates this existing app with missing card callbacks, im:message:readonly and im:resource. It does not create another app. Only this bot briefly reconnects.")
		} else if p.Operation == "group_message_permission" {
			copy.title = channelText("使用飞书确认群消息权限", "Authorize Feishu group messages")
			copy.description = channelText("更新现有应用并增量开通“获取群组中所有消息”权限；确认后自动启用“响应所有群消息”，其他机器人不受影响。", "Adds permission to read all group messages to the existing app. Confirmation enables all-message responses for this bot; other bots are unaffected.")
		}
		var layout ui.Element
		if width <= 680 {
			layout = ui.Column(c).FillWidth().MinWidth(0).Gap(24)
		} else {
			layout = ui.Row(c).FillWidth().MinWidth(0).Gap(24)
		}
		layout.Children(func() {
			qrColumn := ui.Column(c).Width(min(float32(300), max(float32(0), width-24))).Shrink(0).Gap(10).AlignItems(ui.Center)
			if width <= 680 {
				qrColumn.FillWidth()
			}
			qrColumn.Children(func() {
				if p.QR != "" {
					renderPairingQRSize(c, p.QR, min(float32(260), max(float32(0), width-24)))
				} else {
					ui.Text(c, channelText("二维码未就绪，请打开授权链接。", "QR code unavailable. Open the authorization link.")).FillWidth().TextColor(muted)
				}
				remaining := max(0, int(p.Deadline.Sub(c.Now()).Seconds()))
				ui.Text(c, fmt.Sprintf(channelText("二维码有效时间  %02d:%02d", "QR code valid for  %02d:%02d"), remaining/60, remaining%60)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				duration := p.Deadline.Sub(p.StartedAt).Seconds()
				progress := float32(0)
				if duration > 0 {
					progress = min(float32(1), max(float32(0), float32(p.Deadline.Sub(c.Now()).Seconds()/duration)))
				}
				ui.Box(c).FillWidth().Height(4).Radius(2).Background(c.Theme().Border).Children(func() {
					ui.Box(c).WidthPercent(progress * 100).Height(4).Radius(2).Background(c.Theme().Accent)
				})
				c.After(time.Second)
			})
			copyColumn := ui.Column(c).MinWidth(0).Gap(10)
			if width <= 680 {
				copyColumn.FillWidth()
			} else {
				copyColumn.Grow(1)
			}
			copyColumn.Children(func() {
				ui.Text(c, copy.title).FillWidth().FontSize(theme.InterfaceSize(17)).Bold().TextColor(fg)
				ui.Text(c, copy.description).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(muted)
				for index, step := range copy.steps {
					ui.Text(c, fmt.Sprintf("%d. %s", index+1, step)).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(muted)
				}
				ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
					if parsed, err := url.Parse(p.URL); err == nil && parsed.Scheme == "https" && parsed.Host != "" {
						ui.Link(c, channelText("打开授权链接", "Open authorization link"), p.URL)
					}
					if ui.Button(c, channelText("换一个二维码", "Replace QR code")).Disabled(blocked).Clicked() {
						channelAction(s, v, "provision.begin", "", "", "")
					}
					if ui.Button(c, channelText("取消", "Cancel")).Disabled(blocked).Clicked() {
						channelAction(s, v, "provision.cancel", "", "", "")
					}
				})
			})
		})
	})
}

func channelVerification(c *ui.Context, m Model, v imonboard.View, blocked bool, fg, muted ui.Color) {
	ui.Text(c, channelText("输入手机微信显示的数字", "Enter the digits shown in WeChat")).Bold().TextColor(fg)
	ui.Text(c, channelText("配对码只用于本次扫码轮询，不写入配置或日志。", "The pairing code is only used for this scan and is not saved to configuration or logs.")).FillWidth().TextColor(muted)
	draft := v.Draft
	if ui.TextInput(c.Key("weixin-verification"), &draft.VerifyCode).Label(channelText("微信配对码", "WeChat pairing code")).FillWidth().Disabled(blocked).AutoFocus().Changed() {
		filtered := ""
		for _, digit := range draft.VerifyCode {
			if digit >= '0' && digit <= '9' && len(filtered) < 8 {
				filtered += string(digit)
			}
		}
		draft.VerifyCode = filtered
		m.Store.Onboarding.Edit(v.Scope, draft)
	}
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		if ui.PrimaryButton(c, channelText("继续连接", "Continue")).Disabled(blocked || !imonboard.ValidVerificationCode(draft.VerifyCode)).Clicked() {
			m.Store.Onboarding.Edit(v.Scope, draft)
			channelAction(&m.Store.Onboarding, v, "provision.verify", "", "", "")
		}
		if ui.Button(c, channelText("取消绑定", "Cancel setup")).Disabled(blocked).Clicked() {
			channelAction(&m.Store.Onboarding, v, "provision.cancel", "", "", "")
		}
	})
}
