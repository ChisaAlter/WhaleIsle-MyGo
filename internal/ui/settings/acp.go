package settings

import (
	"encoding/json"
	"sort"
	"strings"

	"whaleisle/internal/desktop/acp"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

type ACPAccess struct {
	Catalog                 acp.Catalog
	Registry                []acp.RegistryAgent
	Selected, Error, Status string
	Busy, Available, Loaded bool
	OnSelect                func(string)
	OnAction                func(string, map[string]any)
}
type acpDraft struct {
	ID                                               string
	Name, Command, Args, Profile, RegistryID, Models string
	Policy, Source                                   string
	Env                                              map[string]string
	EnvName, EnvValue                                string
	Error                                            string
	Baseline                                         string
}

func newACPDraft(agent acp.Agent) acpDraft {
	env := map[string]string{}
	for k, v := range agent.Env {
		env[k] = v
	}
	models := "[]"
	if len(agent.ConfiguredModels) > 0 && string(agent.ConfiguredModels) != "null" {
		models = string(agent.ConfiguredModels)
	}
	d := acpDraft{Name: agent.Name, Command: agent.Command, Args: strings.Join(agent.Args, "\n"), Profile: agent.ProfileDir, RegistryID: agent.RegistryAgentID, Source: agent.RegistrySource, Policy: agent.PermissionPolicy, Env: env, Models: models}
	d.Baseline = d.configuration()
	return d
}
func (d acpDraft) configuration() string {
	var models any = d.Models
	_ = json.Unmarshal([]byte(d.Models), &models)
	raw, _ := json.Marshal([]any{d.ID, d.Name, d.Command, d.Args, d.Profile, d.RegistryID, d.Source, d.Policy, d.Env, models})
	return string(raw)
}

func acpSettings(c *ui.Context, a *ACPAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	ui.Column(c.Key("acp-settings")).FillWidth().MinWidth(0).Gap(design.Sp(2.5)).Children(func() {
		kPageTitle(c, providerCopy("外部 Agent", "External agents"))
		kBody(c, providerCopy("连接 ACP 编程 Agent，在对话中选择 Agent、模型与推理强度。", "Connect ACP coding agents and choose the agent, model and reasoning effort in a conversation.")).TextColor(muted)
		if !a.Available {
			kHint(c, providerCopy("连接 Host 后管理外部 Agent。", "Connect the Host to manage external agents."))
			return
		}
		if a.Error != "" {
			settingsAuthoredFailure(c, a.Error)
		}
		if a.Status != "" {
			kHint(c, a.Status)
		}
		call := func(method string, payload map[string]any) {
			if a.OnAction != nil {
				a.OnAction(method, payload)
			}
		}
		ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
			if kButtonDisabled(c, providerCopy("刷新 Agent", "Refresh agents"), a.Busy).Clicked() {
				call("catalog", map[string]any{})
			}
			if kPrimary(c, providerCopy("添加自定义 Agent", "Add custom agent"), a.Busy).Clicked() {
				a.OnSelect("new")
			}
		})
		for _, agent := range a.Catalog.Agents {
			kSurface(c, ui.Column(c.Key("acp-"+agent.ID)).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Padding(design.Sp(2.5))).Children(func() {
				ui.Row(c).Gap(design.Sp(2)).Wrap().AlignItems(ui.Center).Children(func() {
					if kChoiceButton(c, agent.Name, a.Selected == agent.ID, a.Busy).Clicked() {
						if a.Selected == agent.ID {
							a.OnSelect("")
						} else {
							a.OnSelect(agent.ID)
						}
					}
					enabled := agent.Enabled
					if kCheckbox(c, &enabled, providerCopy("启用", "Enabled"), a.Busy).Changed() {
						call("agent/toggle", map[string]any{"id": agent.ID, "enabled": enabled})
					}
					state := providerCopy("未登录", "Signed out")
					if agent.Account != nil && agent.Account.Authenticated {
						state = providerCopy("已登录", "Signed in")
					}
					if !agent.BinaryFound {
						state = providerCopy("未找到命令", "Command unavailable")
					}
					coss.Badge(c, state, coss.BadgeOptions{Variant: coss.BadgeOutline, Size: coss.BadgeMD})
				})
				if a.Selected == agent.ID {
					acpAgentEditor(c, a, agent, call, fg, muted)
					acpLogin(c, a, agent, call, fg, muted)
					for _, session := range a.Catalog.ActiveSessions {
						if session.AgentID == agent.ID {
							ui.Text(c, session.Cwd+" · "+session.CurrentModel).FontSize(theme.InterfaceSize(12)).TextColor(muted).Selectable()
							if kDanger(c, providerCopy("断开会话", "Disconnect session")+" "+session.SessionID, a.Busy).AlignSelf(ui.Start).Clicked() {
								call("sessions/stop", map[string]any{"agentId": agent.ID, "sessionId": session.SessionID})
							}
						}
					}
				}
			})
		}
		if a.Selected == "new" {
			acpAgentEditor(c, a, acp.Agent{RegistrySource: "local", PermissionPolicy: "ask", Env: map[string]string{}}, call, fg, muted)
		}
		group := ui.Column(c.Key("acp-registry")).FillWidth().MinWidth(0).Gap(design.Sp(1.5))
		query := ui.Local(group, "query", func() string { return "" })
		group.Children(func() {
			regIn := kInput(c, query, coss.InputOptions{
				Label:       "ACP registry search",
				Placeholder: providerCopy("搜索 ACP Agent", "Search ACP agents"),
			})
			if regIn.Submitted() && !regIn.Composing() && !a.Busy {
				call("agent/registry-search", map[string]any{"query": *query})
			}
			if kButtonDisabled(c, providerCopy("搜索 Agent 目录", "Search agent registry"), a.Busy).AlignSelf(ui.Start).Clicked() {
				call("agent/registry-search", map[string]any{"query": *query})
			}
			for _, item := range a.Registry {
				ui.Row(c.Key(item.ID)).Gap(design.Sp(2)).Wrap().AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.Name).TextColor(fg)
					if kButtonDisabled(c, providerCopy("添加", "Add")+" "+item.Name, a.Busy).Clicked() {
						call("agent/save", map[string]any{"agent": map[string]any{"id": item.ID, "name": item.Name, "registrySource": "registry", "registryAgentId": item.ID, "enabled": false, "permissionPolicy": "ask"}})
					}
				})
			}
		})
	})
}
func acpAgentEditor(c *ui.Context, a *ACPAccess, agent acp.Agent, call func(string, map[string]any), fg, muted ui.Color) {
	box := ui.Column(c.Key("acp-editor-" + agent.ID)).FillWidth().MinWidth(0).Gap(design.Sp(1.5))
	d := ui.Local(box, "draft", func() acpDraft { return newACPDraft(agent) })
	incoming := newACPDraft(agent)
	if incoming.Baseline != d.Baseline && d.configuration() == d.Baseline {
		*d = incoming
	}
	if incoming.Baseline == d.configuration() {
		d.Baseline = incoming.Baseline
	}
	box.Children(func() {
		input := func(label string, value *string) {
			kHint(c, label)
			kInput(c, value, coss.InputOptions{
				Label:    "ACP " + label,
				Disabled: a.Busy,
			})
		}
		if agent.ID == "" {
			input(providerCopy("Agent ID（小写字母、数字、- 或 _）", "Agent ID (lowercase letters, numbers, - or _)"), &d.ID)
		}
		input(providerCopy("显示名称", "Display name"), &d.Name)
		ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
			for _, source := range []string{"registry", "local"} {
				label := providerCopy("目录管理", "Registry managed")
				if source == "local" {
					label = providerCopy("本地命令", "Local command")
				}
				if kChoiceButton(c, label, d.Source == source, a.Busy || d.Source == source).Clicked() {
					d.Source = source
				}
			}
		})
		if d.Source == "registry" {
			input(providerCopy("目录 Agent ID", "Registry agent ID"), &d.RegistryID)
		}
		input(providerCopy("命令或可执行文件", "Command or executable"), &d.Command)
		kHint(c, providerCopy("参数（每行一个）", "Arguments (one per line)"))
		kTextarea(c, &d.Args, coss.TextareaOptions{
			Label:    "ACP arguments",
			MaxLines: 4,
			Disabled: a.Busy,
		}).Lines(1, 4)
		input(providerCopy("独立配置目录", "Profile directory"), &d.Profile)
		ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
			for _, policy := range []string{"ask", "auto-allow-read", "auto-allow"} {
				label := map[string]string{"ask": providerCopy("逐次询问", "Ask each time"), "auto-allow-read": providerCopy("自动允许读取", "Allow reads"), "auto-allow": providerCopy("自动允许", "Allow automatically")}[policy]
				if kChoiceButton(c, label, d.Policy == policy, a.Busy || d.Policy == policy).Clicked() {
					d.Policy = policy
				}
			}
		})
		keys := make([]string, 0, len(d.Env))
		for key := range d.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := d.Env[key]
			ui.Row(c.Key("env-" + key)).FillWidth().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, key).TextColor(muted)
				v := value
				if kInputRow(c, &v, coss.InputOptions{
					Label:    "ACP env " + key,
					Password: true,
					Disabled: a.Busy,
				}).Changed() {
					d.Env[key] = v
				}
				if coss.Button(c, "×", coss.ButtonOptions{
					Variant:  coss.ButtonGhost,
					Size:     coss.ButtonSM,
					Label:    "Remove env " + key,
					Disabled: a.Busy,
				}).Clicked() {
					delete(d.Env, key)
				}
			})
		}
		ui.Row(c).FillWidth().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
			kInputRow(c, &d.EnvName, coss.InputOptions{
				Label:       "Environment variable name",
				Placeholder: "NAME",
			})
			kInputRow(c, &d.EnvValue, coss.InputOptions{
				Label:    "Environment variable value",
				Password: true,
			})
			if kButtonDisabled(c, "+", a.Busy || strings.TrimSpace(d.EnvName) == "").Label("Add environment variable").Clicked() {
				d.Env[strings.TrimSpace(d.EnvName)] = d.EnvValue
				d.EnvName = ""
				d.EnvValue = ""
			}
		})
		advanced := ui.Local(box, "models-open", func() bool { return false })
		if kButton(c, providerCopy("模型配置", "Model configuration")).AlignSelf(ui.Start).Clicked() {
			*advanced = !*advanced
		}
		if *advanced {
			kHint(c, providerCopy("自定义模型列表（JSON，保留 Host 的推理配置）", "Custom model list (JSON, preserving Host reasoning options)"))
			kTextarea(c, &d.Models, coss.TextareaOptions{
				Label:    "ACP models JSON",
				MaxLines: 8,
				Disabled: a.Busy,
			}).Lines(3, 8)
			for _, model := range agent.AllModels {
				ui.Row(c.Key("model-" + model.ID)).Gap(design.Sp(2)).Wrap().AlignItems(ui.Center).Children(func() {
					ui.Text(c, model.Name).TextColor(fg)
					for _, setting := range []struct {
						key, label string
						values     []string
					}{
						{"favoriteModels", providerCopy("收藏", "Favorite"), agent.FavoriteModels},
						{"hiddenModels", providerCopy("隐藏", "Hidden"), agent.HiddenModels},
					} {
						checked := false
						for _, id := range setting.values {
							if id == model.ID {
								checked = true
							}
						}
						if kCheckbox(c, &checked, setting.label+" "+model.ID, a.Busy).Changed() {
							next := []string{}
							for _, id := range setting.values {
								if id != model.ID {
									next = append(next, id)
								}
							}
							if checked {
								next = append(next, model.ID)
							}
							call("agent/save", map[string]any{"agent": agent.Patch(map[string]any{setting.key: next})})
						}
					}
				})
			}
		}
		if d.Error != "" {
			settingsAuthoredFailure(c, d.Error)
		}
		payload := func() map[string]any {
			if agent.ID == "" {
				if !validACPAgentID(d.ID) {
					d.Error = providerCopy("Agent ID 需为 1–48 个小写字母、数字、- 或 _，且不能以 - 开头或结尾。", "Use 1–48 lowercase letters, numbers, - or _ for the agent ID, with no leading or trailing -.")
					return nil
				}
				for _, existing := range a.Catalog.Agents {
					if existing.ID == d.ID {
						d.Error = providerCopy("该 Agent ID 已存在，请换一个。", "This agent ID already exists. Choose another.")
						return nil
					}
				}
			}
			var models []json.RawMessage
			if err := json.Unmarshal([]byte(d.Models), &models); err != nil {
				d.Error = errcopy.Sentence(err)
				return nil
			}
			d.Error = ""
			args := []string{}
			for _, line := range strings.Split(d.Args, "\n") {
				if line != "" {
					args = append(args, strings.TrimSuffix(line, "\r"))
				}
			}
			fields := map[string]any{"name": d.Name, "command": d.Command, "args": args, "env": d.Env, "profileDir": d.Profile, "registrySource": d.Source, "registryAgentId": d.RegistryID, "permissionPolicy": d.Policy, "models": models}
			if agent.ID == "" {
				fields["enabled"] = false
				fields["id"] = d.ID
			}
			return map[string]any{"agent": agent.Patch(fields)}
		}
		ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
			if d.configuration() != d.Baseline && kGhost(c, providerCopy("放弃草稿", "Discard draft"), a.Busy).Clicked() {
				*d = newACPDraft(agent)
			}
			if kPrimary(c, providerCopy("保存 Agent", "Save agent"), a.Busy || strings.TrimSpace(d.Name) == "").Clicked() {
				if p := payload(); p != nil {
					call("agent/save", p)
				}
			}
			if agent.ID != "" {
				if kButtonDisabled(c, providerCopy("测试 ACP", "Test ACP"), a.Busy).Clicked() {
					if p := payload(); p != nil {
						p["id"] = agent.ID
						call("agent/test", p)
					}
				}
				if agent.Builtin && kButtonDisabled(c, providerCopy("添加独立账号", "Add separate account"), a.Busy).Clicked() {
					call("agent/create-account", map[string]any{"baseAgentId": agent.ID})
				}
				remove := ui.Local(box, "remove", func() bool { return false })
				if kDanger(c, providerCopy("移除或重置", "Remove or reset"), a.Busy).Clicked() {
					*remove = !*remove
				}
				if *remove {
					kHint(c, providerCopy("会断开该 Agent 的运行会话；内置项恢复默认，自定义项移除。", "Disconnects this agent's sessions; built-ins reset and custom entries are removed."))
					if kDanger(c, providerCopy("确认移除或重置", "Confirm removal or reset"), a.Busy).Clicked() {
						call("agent/delete", map[string]any{"id": agent.ID})
						*remove = false
					}
				}
			}
		})
	})
}

func validACPAgentID(id string) bool {
	if len(id) < 1 || len(id) > 48 || id[0] == '-' || id[len(id)-1] == '-' {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
