package settings

import (
	"strings"

	"whaleisle/internal/desktop/acp"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

func acpLogin(c *ui.Context, a *ACPAccess, agent acp.Agent, call func(string, map[string]any), fg, muted ui.Color) {
	box := ui.Column(c.Key("acp-auth-" + agent.ID)).FillWidth().MinWidth(0).Gap(design.Sp(2))
	box.Children(func() {
		kHeading(c, providerCopy("账号登录", "Account sign-in"))
		methods := []acp.Method{}
		if agent.Account != nil {
			methods = agent.Account.AvailableMethods
		}
		if len(methods) == 0 {
			methods = []acp.Method{{ID: agent.AuthMethod, Label: providerCopy("登录", "Sign in")}}
		}
		for _, method := range methods {
			row := ui.Column(c.Key("auth-method-" + method.ID)).FillWidth().MinWidth(0).Gap(design.Sp(1.5))
			values := ui.Local(row, "values", func() map[string]string { return map[string]string{} })
			row.Children(func() {
				for _, field := range method.Fields {
					key := field.Key
					if key == "" {
						key = field.Name
					}
					v := (*values)[key]
					kHint(c, field.Label)
					if kInput(c, &v, coss.InputOptions{
						Label:    field.Label,
						Password: true,
						Disabled: a.Busy,
					}).Changed() {
						(*values)[key] = v
					}
				}
				if kPrimary(c, method.Label, a.Busy || !agent.Enabled).AlignSelf(ui.Start).Clicked() {
					p := map[string]any{"id": agent.ID, "methodId": method.ID}
					if len(*values) > 0 {
						p["credentials"] = *values
					}
					call("agent/login", p)
				}
				if len(method.Fields) > 0 && kButtonDisabled(c, providerCopy("仅保存凭据", "Save credentials only")+" · "+method.Label, a.Busy).AlignSelf(ui.Start).Clicked() {
					call("agent/login", map[string]any{"id": agent.ID, "methodId": method.ID, "credentials": *values, "onlySaveCredentials": true})
				}
			})
		}
		flow := agent.AuthFlow
		if flow != nil {
			ui.Text(c, flow.Message).TextColor(muted).Selectable()
			if flow.Active() && kButtonDisabled(c, providerCopy("取消登录", "Cancel sign-in"), a.Busy).AlignSelf(ui.Start).Clicked() {
				call("agent/login/cancel", map[string]any{"id": agent.ID})
			}
			url := flow.AuthorizationURL
			interaction := flow.Interaction
			if interaction != nil {
				if interaction.URL != "" {
					url = interaction.URL
				}
				if interaction.UserCode != "" {
					ui.Text(c, interaction.UserCode).FontSize(theme.InterfaceSize(18)).TextColor(fg).Selectable()
				}
				if interaction.Output != "" {
					ui.Text(c, interaction.Output).FontSize(theme.InterfaceSize(12)).MaxLines(20).TextColor(muted).Selectable()
				}
				if interaction.RequiresConsent && kPrimary(c, providerCopy("允许浏览器登录", "Allow browser sign-in"), a.Busy).AlignSelf(ui.Start).Clicked() {
					call("agent/login/respond", map[string]any{"id": agent.ID, "response": map[string]any{"type": "browser", "action": "accept"}})
				}
				if interaction.Type == "credentials" {
					fields := ui.Local(box, "flow-fields-"+interaction.ID, func() map[string]string { return map[string]string{} })
					for _, field := range interaction.Fields {
						v := (*fields)[field.Name]
						kHint(c, field.Label)
						if kInput(c, &v, coss.InputOptions{
							Label:    "Login " + field.Name,
							Password: true,
							Disabled: a.Busy,
						}).Changed() {
							(*fields)[field.Name] = v
						}
					}
					if kPrimary(c, providerCopy("提交登录信息", "Continue sign-in"), a.Busy).AlignSelf(ui.Start).Clicked() {
						call("agent/login/respond", map[string]any{"id": agent.ID, "response": map[string]any{"type": "credentials", "values": *fields}})
					}
				}
				if interaction.Type == "browser" && interaction.AcceptsCallback {
					callback := ui.Local(box, "callback-"+interaction.ID, func() string { return "" })
					kInput(c, callback, coss.InputOptions{
						Label:       "OAuth callback URL",
						Placeholder: providerCopy("粘贴登录回调地址", "Paste callback URL"),
					})
					if kPrimary(c, providerCopy("完成回调", "Complete sign-in"), a.Busy || strings.TrimSpace(*callback) == "").AlignSelf(ui.Start).Clicked() {
						call("agent/login/complete", map[string]any{"id": agent.ID, "callbackUrl": *callback})
					}
				}
			}
			if url != "" {
				ui.Text(c, url).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
				if kButtonDisabled(c, providerCopy("打开登录页面", "Open sign-in page"), a.Busy || interaction != nil && interaction.RequiresConsent).AlignSelf(ui.Start).Clicked() {
					call("agent/open-url", map[string]any{"url": url})
				}
			}
		}
		if agent.Enabled {
			reauth := ui.Local(box, "reauth", func() bool { return false })
			if kButtonDisabled(c, providerCopy("重新授权", "Reauthorize"), a.Busy).AlignSelf(ui.Start).Clicked() {
				*reauth = !*reauth
			}
			if *reauth {
				kHint(c, providerCopy("将断开会话并清除 CLI 登录状态，保留 API 密钥，然后重新登录。", "Disconnect sessions and clear CLI sign-in, keep API keys, then sign in again."))
				if kPrimary(c, providerCopy("确认重新授权", "Confirm reauthorization"), a.Busy).AlignSelf(ui.Start).Clicked() {
					call("agent/login/reauth", map[string]any{"id": agent.ID})
					*reauth = false
				}
			}
		}
		if agent.Account != nil && agent.Account.Authenticated {
			confirm := ui.Local(box, "logout", func() bool { return false })
			if kDanger(c, providerCopy("退出登录", "Sign out"), a.Busy).AlignSelf(ui.Start).Clicked() {
				*confirm = !*confirm
			}
			if *confirm {
				clearKeys := ui.Local(box, "clear-api-keys", func() bool { return false })
				kCheckbox(c, clearKeys, providerCopy("同时移除已保存的 API 密钥", "Also remove saved API keys"), a.Busy)
				kHint(c, providerCopy("该账号的 ACP 会话将断开。", "This account's ACP sessions will disconnect."))
				if kDanger(c, providerCopy("确认退出登录", "Confirm sign-out"), a.Busy).AlignSelf(ui.Start).Clicked() {
					call("agent/logout", map[string]any{"id": agent.ID, "clearApiKeys": *clearKeys})
					*confirm = false
				}
			}
		}
	})
}
