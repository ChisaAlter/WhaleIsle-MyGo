package im

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/png"
	"net/url"
	"strings"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func onboardingView(c *ui.Context, m Model) {
	s := &m.Store.Onboarding
	v := s.Snapshot()
	p := imonboard.PlatformByID(v.Channel)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	blocked := v.Busy || !v.Available || m.PluginDisabled || !m.HarnessReady
	ui.Column(c.Key(fmt.Sprintf("im-setup-%d", v.Scope))).Grow(1).MinHeight(0).Background(chrome.Color(colors.BG)).Children(func() {
		ui.Row(c).Padding(10).Gap(8).Children(func() {
			ui.Text(c, p.Label+" 原生接入").Bold().Grow(1).TextColor(fg)
			if ui.Button(c, "刷新连接状态").Disabled(blocked).Clicked() {
				s.Refresh()
			}
			if ui.Button(c, "返回 IM").Clicked() {
				s.Hide()
			}
			if m.OnClose != nil && ui.Button(c, "关闭").Clicked() {
				s.Hide()
				m.OnClose()
			}
		})
		ui.Scroll(c).Grow(1).MinHeight(0).Padding(16).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				for start := 0; start < len(imonboard.Platforms); start += 3 {
					end := start + 3
					if end > len(imonboard.Platforms) {
						end = len(imonboard.Platforms)
					}
					ui.Row(c).Gap(6).Children(func() {
						for _, platform := range imonboard.Platforms[start:end] {
							if ui.Button(c, platform.Label).Disabled(v.Busy || platform.ID == v.Channel).Clicked() {
								s.Show(platform.ID)
							}
						}
					})
				}
				ui.Text(c, p.Guide).FontSize(theme.InterfaceSize(12)).TextColor(fg)
				ui.Text(c, "当前 Host："+v.Host).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if v.Channel == "slack" {
					ui.Link(c, "打开 Slack 创建应用页面", "https://api.slack.com/apps?new_app=1")
					if ui.Button(c, "复制原版 Slack Manifest").Clicked() {
						mygo.Clipboard.WriteText(slackAppManifest)
					}
				}
				ui.Text(c, "这里直接调用已有 Host 平台接口。平台应用创建、管理员授权和风控验证仍需你在对应平台完成。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if m.OnOpenPluginWeb != nil && ui.Button(c, "打开原版平台向导 / 外部注册说明").Clicked() {
					s.Hide()
					m.OnOpenPluginWeb()
				}
				if v.Notice != "" {
					ui.Text(c, v.Notice).TextColor(fg)
				}
				if v.TestNotice != "" {
					ui.Text(c, v.TestNotice).TextColor(fg)
				}
				if v.Error != "" {
					ui.Text(c, v.Error).TextColor(chrome.Color(colors.Accent))
				}
				if !v.Available {
					ui.Text(c, "请先启动并连接 Host。").TextColor(muted)
				}
				locked := blocked || v.Review != ""
				if v.Review != "" {
					ui.Text(c, "确认操作："+v.Review+" · "+p.Label+" "+v.Target).Bold().TextColor(fg)
					if v.Review == "bot.bind-credentials" {
						ui.Text(c, "凭据将交给当前 Host 验证并存入 Host 凭据存储；本地面板不会写入磁盘。").TextColor(muted)
					}
					if v.Review == "bot.reconnect-test" {
						ui.Text(c, "将重连此飞书机器人，并由 Host 向该机器人已有的私聊测试目标发送一条测试消息；请在飞书会话中核对。若没有可用私聊目标，不会改为群发。").TextColor(muted)
					}
					if v.Review == "bot.delete" {
						ui.Text(c, "删除该平台机器人连接及 Host 保存的绑定；需要重新接入才能恢复。").TextColor(muted)
					}
					if v.Review == "bot.access-policy.set" {
						ui.Text(c, "新的访问模式："+v.Mode+"；允许名单：\n"+v.Allowlist).TextColor(fg)
					}
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, "确认执行").Disabled(blocked).Clicked() {
							s.Confirm()
						}
						if ui.Button(c, "取消操作").Disabled(v.Busy).Clicked() {
							s.Dismiss()
						}
					})
				}
				if p.QR {
					if ui.Button(c, "生成 / 更换接入二维码").Disabled(locked).Clicked() {
						s.Review(v.Scope, "provision.begin", "", "", "")
					}
					if v.Provision.AttemptID != "" {
						provision := v.Provision
						ui.Text(c, "接入状态："+provision.Status).Bold().TextColor(fg)
						if !provision.Deadline.IsZero() {
							ui.Text(c, "二维码有效期至："+provision.Deadline.Local().Format("15:04:05")).TextColor(muted)
						}
						if provision.QR != "" {
							renderPairingQR(c, provision.QR)
						}
						if parsed, err := url.Parse(provision.URL); err == nil && parsed.Scheme == "https" && parsed.Host != "" {
							ui.Link(c, "打开平台授权 / 验证页面", provision.URL)
							if ui.Button(c, "复制平台授权链接").Clicked() {
								mygo.Clipboard.WriteText(provision.URL)
							}
						}
						if provision.QR == "" && provision.URL != "" {
							ui.Text(c, "Host 当前提供授权链接，可点击上方链接完成平台授权。").TextColor(muted)
						}
						if ui.Button(c, "取消当前接入").Disabled(locked).Clicked() {
							s.Review(v.Scope, "provision.cancel", "", "", "")
						}
					}
				}
				draft := v.Draft
				input := func(label string, value *string, password bool) {
					ui.Text(c, label).TextColor(fg)
					field := ui.TextInput(c.Key(label), value).FillWidth().Disabled(locked)
					if password {
						field.Password()
					}
				}
				if p.Secret != "" {
					ui.Text(c, "绑定已有机器人凭据").Bold().TextColor(fg)
					if p.Identity != "" {
						input(p.Identity, &draft.Identity, false)
					}
					input(p.Secret, &draft.Secret, true)
					if p.Extra != "" {
						input(p.Extra, &draft.Extra, true)
					}
					missing := strings.TrimSpace(draft.Secret) == "" || (p.Identity != "" && strings.TrimSpace(draft.Identity) == "") || (p.Extra != "" && strings.TrimSpace(draft.Extra) == "")
					if ui.Button(c, "验证并绑定…").Disabled(locked || missing).Clicked() {
						s.Edit(v.Scope, draft)
						s.Review(v.Scope, "bot.bind-credentials", "", "", "")
					}
					ui.Text(c, "仅当前页面临时保留输入；提交、返回或切换平台后清空。").FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
				if v.Channel == "weixin" && (v.Provision.VerificationRequired || v.Provision.Status == "needs_verification") {
					input("微信配对验证码", &draft.VerifyCode, true)
					if ui.Button(c, "提交微信验证码…").Disabled(locked || strings.TrimSpace(draft.VerifyCode) == "").Clicked() {
						s.Edit(v.Scope, draft)
						s.Review(v.Scope, "provision.verify", "", "", "")
					}
				}
				if draft != v.Draft {
					s.Edit(v.Scope, draft)
				}
				ui.Text(c, "Host 平台连接").Bold().TextColor(fg)
				if len(v.Bots) == 0 {
					ui.Text(c, "尚未读取到已绑定机器人。").TextColor(muted)
				}
				for _, bot := range v.Bots {
					b := bot
					ui.Column(c.Key(b.BotID)).Border(1, chrome.Color(colors.Border)).Padding(10).Gap(8).Children(func() {
						state := b.State
						if b.Connected {
							state = "connected"
						}
						ui.Text(c, b.Bot.Name+" · "+b.BotID+" · "+state).Bold().TextColor(fg)
						ui.Text(c, b.Health.Summary).TextColor(muted)
						action := func(label, method string) {
							if ui.Button(c, label).Disabled(locked).Clicked() {
								s.Review(v.Scope, method, b.BotID, "", "")
							}
						}
						ui.Row(c).Gap(8).Children(func() {
							action("重新连接（不发消息）…", "bot.reconnect")
							if v.Channel == "feishu" {
								action("重连并发送测试消息…", "bot.reconnect-test")
								action("断开连接…", "bot.disconnect")
							}
							action("删除绑定…", "bot.delete")
						})
						if v.Channel == "feishu" {
							ui.Text(c, "群响应模式："+b.GroupResponseMode).TextColor(muted)
							ui.Row(c).Gap(8).Children(func() {
								action("修复权限与回调…", "bot.callback-repair.begin")
								action("开通群消息权限…", "bot.group-message-permission.begin")
							})
							ui.Row(c).Gap(8).Children(func() {
								for _, mode := range []string{"mention", "all"} {
									if ui.Button(c, "群响应："+mode).Disabled(locked || b.GroupResponseMode == mode).Clicked() {
										if mode == "all" && !b.GroupMessagePermissionGranted {
											s.Review(v.Scope, "bot.group-message-permission.begin", b.BotID, "", "")
										} else {
											s.Review(v.Scope, "bot.group-response-mode.set", b.BotID, mode, "")
										}
									}
								}
							})
						}
						if v.Channel == "whatsapp" || v.Channel == "telegram" {
							entries := b.AccessPolicy.AllowedNumbers
							if v.Channel == "telegram" {
								entries = b.AccessPolicy.AllowedUsers
							}
							ui.Text(c, "当前访问模式："+b.AccessPolicy.AccessMode+"；名单："+strings.Join(entries, ", ")).TextColor(muted)
							text := strings.Join(entries, "\n")
							if value, ok := v.PolicyDrafts[b.BotID]; ok {
								text = value
							}
							ui.Text(c, "允许名单（每行一个；Telegram User ID / WhatsApp 国际号码，不含 +）").TextColor(fg)
							ui.TextArea(c.Key("allowlist"), &text).FillWidth().Lines(2, 5).Disabled(locked)
							s.EditPolicy(v.Scope, b.BotID, text)
							modes := []string{"self-only", "private-allowlist", "open"}
							if v.Channel == "telegram" {
								modes = []string{"compatible", "private-allowlist"}
							}
							ui.Row(c).Gap(8).Children(func() {
								for _, mode := range modes {
									if ui.Button(c, "保存模式："+mode+"…").Disabled(locked).Clicked() {
										s.Review(v.Scope, "bot.access-policy.set", b.BotID, mode, text)
									}
								}
							})
						}
					})
				}
			})
		})
	})
}
func renderPairingQR(c *ui.Context, source string) {
	header, body, ok := strings.Cut(source, ",")
	if !ok || !strings.HasPrefix(header, "data:image/") || !strings.Contains(header, ";base64") {
		return
	}
	data, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return
	}
	if strings.Contains(header, "svg+xml") {
		if svg, err := ui.ParseSVG(data); err == nil {
			ui.Image(c, svg).Width(260).Height(260).Fit(ui.Contain)
		}
		return
	}
	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		ui.Image(c, ui.NewBitmap(img)).Width(260).Height(260).Fit(ui.Contain)
	}
}
