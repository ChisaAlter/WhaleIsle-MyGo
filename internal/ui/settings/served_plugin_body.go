package settings

import (
	"context"
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// ServedPluginBody is one original plugins.item form, including its coupled
// namespaces. It never changes the settings rail's selected namespace.
func ServedPluginBody(c *ui.Context, m Model, key string) bool {
	if m.Store == nil {
		return false
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	names := []string{}
	switch key {
	case "subagent":
		names = []string{"subagent-model-selection-settings", "subagent"}
	case "web-search":
		if m.Store.BindNamespace("web-search-deepseek") == nil {
			return false
		}
		names = []string{"web-search-deepseek", "web-deepseek-api", "web"}
	case "agent-loop":
		names = []string{"agent-loop"}
	case "shell":
		if m.Store.BindNamespace("pwsh-sandbox") != nil {
			names = []string{"pwsh-sandbox"}
		} else {
			names = []string{"bash-sandbox"}
		}
	default:
		return false
	}
	bindings := []*store.NamespaceBinding{}
	states := map[string]store.ViewState{}
	for _, ns := range names {
		if b := m.Store.BindNamespace(ns); b != nil {
			bindings = append(bindings, b)
			states[ns] = b.Snapshot()
		}
	}
	if len(bindings) == 0 {
		return false
	}
	dirty, disabled, invalid, saving := false, false, false, false
	for _, state := range states {
		dirty = dirty || state.Dirty
		disabled = disabled || !state.ClientReady || !state.Writable || state.Loading || state.Saving
		invalid = invalid || store.ServedDraftError(state) != ""
		saving = saving || state.Saving
	}
	var credential *store.WebSearchCredential
	var keyState store.WebSearchCredentialView
	if key == "web-search" {
		credential = m.Store.BindWebSearchCredential()
		if credential != nil {
			keyState = credential.Snapshot()
			if !keyState.Known && !keyState.Loading && keyState.Error == "" {
				credential.Refresh(context.Background())
			}
			dirty = dirty || keyState.Draft != ""
			invalid = invalid || keyState.Draft != "" && (!keyState.Known || !keyState.Writable || keyState.Loading)
		}
	}
	ui.Column(c.Key("served-card/" + bindings[0].Owner())).FillWidth().MinWidth(0).Gap(12).Children(func() {
		for _, state := range states {
			if state.Err != "" {
				ui.Text(c, state.Err).FillWidth().TextColor(c.Theme().Danger)
				break
			}
		}
		field := func(namespace, key, zh, en, hint string) {
			state, ok := states[namespace]
			if !ok {
				return
			}
			b := m.Store.BindNamespace(namespace)
			for _, f := range state.Namespaces[0].Fields {
				if f.Key == key {
					f.Label = providerCopy(zh, en)
					f.Description = hint
					servedScalar(c, b, state, f, fg, muted)
					return
				}
			}
		}
		switch key {
		case "subagent":
			field("subagent", "maxDepth", "最大嵌套深度", "Maximum nesting depth", providerCopy("0 禁止委派；留空继承默认值。", "0 disables delegation; blank inherits the default."))
			field("subagent", "maxActiveSubagents", "最多活动子代理", "Maximum active subagents", providerCopy("至少为 1；留空继承默认值。", "At least 1; blank inherits the default."))
			if state, ok := states["subagent-model-selection-settings"]; ok {
				b := m.Store.BindNamespace(state.SelectedNS)
				field(state.SelectedNS, "enabled", "允许代理选择模型", "Let agents choose models", providerCopy("关闭时子代理继承父代理模型。", "When disabled, subagents inherit the parent's model."))
				if state.Drafts["enabled"] == "true" {
					m.Store.EnsureModelConfigurationCatalog(context.Background())
					servedSubagentModels(c, m.Store, b, state, fg, muted)
				}
			}
		case "web-search":
			web, switchAvailable := states["web"]
			_, apiAvailable := states["web-deepseek-api"]
			useAPI := switchAvailable && apiAvailable && web.Drafts["searchProvider"] == "deepseek-search-api"
			if switchAvailable && apiAvailable {
				bodyPreferenceRow(c, "web-api-switch", providerCopy("使用 Search / Fetch API", "Use Search / Fetch API"), providerCopy("同时切换搜索和抓取服务。", "Switches search and fetch providers together."), fg, muted, func() {
					if ui.Switch(c, &useAPI).Label("Search / Fetch API").Disabled(disabled || invalid || web.Err != "").Changed() {
						b := m.Store.BindNamespace("web")
						targets := map[string]string{"searchProvider": "deepseek-official", "fetchProvider": "http"}
						if useAPI {
							targets = map[string]string{"searchProvider": "deepseek-search-api", "fetchProvider": "deepseek-fetch-api"}
						}
						var base map[string]string
						_ = json.Unmarshal(web.Namespaces[0].Base, &base)
						for k, v := range targets {
							if base[k] == v {
								b.Reset(k)
							} else {
								b.SetDraft(k, v)
							}
						}
					}
				})
			}
			ns := "web-search-deepseek"
			if useAPI {
				ns = "web-deepseek-api"
			}
			if credential != nil {
				bodyPreferenceRow(c, "web-search-key", "API Key", providerCopy("密钥不会写入设置文件；留空保留已有密钥。", "Keys are not stored in settings; blank keeps the existing key."), fg, muted, func() {
					text := keyState.Draft
					if ui.TextInput(c, &text).FillWidth().Password().Label("Web Search API Key").Disabled(disabled || !keyState.Known || !keyState.Writable || keyState.Loading).Changed() {
						credential.Edit(text)
					}
					status := providerCopy("尚未配置密钥；账户模型可使用默认接口搜索。", "No key configured; account models can use the default search endpoint.")
					if keyState.Configured {
						status = providerCopy("已配置密钥。", "Key configured.")
					}
					if !keyState.Known {
						status = providerCopy("正在读取密钥状态…", "Reading key status…")
					}
					ui.Text(c, status).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
					if keyState.Error != "" {
						ui.Text(c, keyState.Error).FillWidth().TextColor(c.Theme().Danger)
					}
					if ui.Button(c, providerCopy("刷新密钥状态", "Refresh key status")).Disabled(keyState.Loading || disabled).Clicked() {
						credential.Refresh(context.Background())
					}
				})
			}
			field(ns, "baseURL", "API 基础地址", "API base URL", providerCopy("包含 API 前缀；留空恢复默认地址。", "Include the API prefix; blank restores the default endpoint."))
			if useAPI {
				field(ns, "count", "每次搜索结果数", "Results per search", providerCopy("1–50 的整数，默认 10。", "Whole number from 1–50; default 10."))
			} else {
				field(ns, "maxUses", "单次请求最多搜索次数", "Searches per request", providerCopy("仅适用于 Messages API；留空继承。", "Messages API only; blank inherits the default."))
			}
		case "agent-loop":
			field("agent-loop", "maxParallelToolCalls", "并行工具调用上限", "Parallel tool-call limit", providerCopy("限制每步同时执行的并行安全工具调用；留空继承。", "Limits parallel-safe tool calls per step; blank inherits the default."))
		case "shell":
			field(names[0], "timeoutMs", "前台命令超时（毫秒）", "Foreground timeout (ms)", "")
			field(names[0], "maxOutputBytes", "每个输出流的内存上限（字节）", "Per-stream output cap (bytes)", "")
		}
		for _, state := range states {
			if message := store.ServedDraftError(state); message != "" {
				ui.Text(c, message).FillWidth().TextColor(c.Theme().Danger)
			}
		}
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.PrimaryButton(c, providerCopy("保存", "Save")).Disabled(disabled || invalid || !dirty).Clicked() {
				if credential != nil {
					credential.Save(context.Background(), bindings[0], bindings[1:]...)
				} else {
					bindings[0].SaveSequence(context.Background(), bindings[1:]...)
				}
			}
			if ui.Button(c, providerCopy("放弃草稿", "Discard drafts")).Disabled(saving || !dirty).Clicked() {
				for _, b := range bindings {
					b.Discard()
				}
				if credential != nil {
					credential.Discard()
				}
			}
		})
		if disabled {
			for _, state := range states {
				if !state.Writable {
					ui.Text(c, providerCopy("此配置为只读。", "This configuration is read-only.")).TextColor(muted)
					break
				}
			}
		}
	})
	return true
}

func servedScalar(c *ui.Context, b *store.NamespaceBinding, s store.ViewState, f store.Field, fg, muted ui.Color) {
	disabled := !s.ClientReady || !s.Writable || s.Loading || s.Saving
	bodyPreferenceRow(c, "served-"+b.Owner()+f.Key, f.Label, f.Description, fg, muted, func() {
		value := s.Drafts[f.Key]
		if f.Kind == store.FieldBool {
			checked := value == "true"
			if ui.Switch(c, &checked).Label(f.Label).Disabled(disabled).Changed() {
				if checked {
					b.SetDraft(f.Key, "true")
				} else {
					b.SetDraft(f.Key, "false")
				}
			}
		} else if ui.TextInput(c, &value).FillWidth().Label(f.Label).Disabled(disabled).Changed() {
			if strings.TrimSpace(value) == "" {
				b.Reset(f.Key)
			} else {
				b.SetDraft(f.Key, strings.TrimSpace(value))
			}
		}
		var user map[string]json.RawMessage
		_ = json.Unmarshal(s.Namespaces[0].User, &user)
		if _, ok := user[f.Key]; ok && !s.Resets[f.Key] {
			ui.Text(c, providerCopy("已覆盖", "Overridden")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		if ui.Button(c, providerCopy("恢复默认", "Reset")).Disabled(disabled).Clicked() {
			b.Reset(f.Key)
		}
	})
}
