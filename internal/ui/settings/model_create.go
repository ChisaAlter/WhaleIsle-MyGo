package settings

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

var configurationRoutePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// configurationProtocolLabels names the wire protocols a Host can declare, as the
// reference does; an unknown id is shown as it is.
func configurationProtocolLabel(id string) string {
	switch id {
	case "openai-completions":
		return "OpenAI Chat Completions"
	case "openai-responses":
		return "OpenAI Responses"
	case "anthropic-messages":
		return "Anthropic Messages"
	}
	return id
}

// configurationCreateHint is the first unmet requirement of the create card in
// the reference's words ("" when the card is ready or the blocking field already
// says what is wrong beside itself).
func configurationCreateHint(route, endpoint string, rows []any, taken bool, keyBad bool) string {
	switch {
	case route == "" || !configurationRoutePattern.MatchString(route) || taken:
		return ""
	case endpoint != "" && !configurationHTTPURL(endpoint):
		return ""
	case keyBad:
		return ""
	case endpoint == "":
		return providerCopy("自定义模型 API 需要填写 API 地址。", "A custom provider needs a base URL.")
	}
	if message := configurationModelError(rows); message != "" {
		return message
	}
	if len(rows) == 0 {
		return providerCopy("自定义模型 API 至少需要一个模型。", "A custom provider needs at least one model.")
	}
	return ""
}

func configurationHTTPURL(endpoint string) bool {
	u, err := url.Parse(endpoint)
	return err == nil && u.Host != "" && (u.Scheme == "http" || u.Scheme == "https")
}

// configurationCreateProvider is the reference's "自定义模型 API" card: Provider
// ID, display name, API address, protocol, API key and the model catalog, then
// 取消 / 创建提供商. The first unmet requirement is the disabled button's
// tooltip; fields that are wrong say so beside themselves.
func configurationCreateProvider(c *ui.Context, m Model, s store.ViewState, providers map[string]any, fields, models []store.Field, fg, muted ui.Color) {
	const key = "@provider-create"
	draft := configurationObject(s.Drafts[key])
	save := func() { raw, _ := json.Marshal(draft); configurationSetForm(m, key, string(raw)) }
	createOnly := m.Models != nil && m.Models.configurationCreateOnly
	committed := s.Drafts["@provider-create-committed"]
	disabled := !s.Writable || s.Saving
	var protocols []string
	for _, field := range fields {
		if field.Key == "api" {
			protocols = field.EnumOptions
		}
	}
	protocol := configurationText(draft["api"])
	if protocol == "" && len(protocols) > 0 {
		protocol = protocols[0]
	}
	route := strings.TrimSpace(configurationText(draft["route"]))
	endpoint := strings.TrimSpace(configurationText(draft["baseURL"]))
	rows := configurationRows(draft["models"])
	existing := providers
	if createOnly && len(s.Namespaces) > 0 {
		existing = configurationObjectValue(configurationObject(string(s.Namespaces[0].Value))["providers"])
		if committed == route {
			copy := map[string]any{}
			for id, value := range existing {
				if id != route {
					copy[id] = value
				}
			}
			existing = copy
		}
	}
	_, taken := existing[route]
	keyValue := ""
	if m.Models != nil && m.Models.configurationCreateKey != nil {
		keyValue = *m.Models.configurationCreateKey
	}
	keyBad := keyValue != "" && strings.TrimSpace(keyValue) == ""
	routeInvalid := route != "" && !configurationRoutePattern.MatchString(route)
	baseInvalid := endpoint != "" && !configurationHTTPURL(endpoint)
	ready := configurationCreateError(route, endpoint, protocol, rows, existing) == "" && !keyBad
	ui.Column(c.Key("provider-create")).FillWidth().MinWidth(0).Gap(16).Children(func() {
		if committed != "" {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				modelFieldLabel(c, configurationLabel("route"))
				ui.Text(c, route).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
			})
		} else {
			configurationInput(c, m, s, "create/route", "route", draft["route"], false, func(v any) { configurationSet(draft, "route", v); save() }, fg, muted, "acme-gateway")
			modelHint(c, providerCopy("以小写字母开头的标识，在请求中唯一标识该提供商，并用于派生凭据名。", "A lowercase identifier that names the provider in requests and derives its credential name."), "")
			if routeInvalid {
				modelHint(c, providerCopy("需以小写字母开头，之后可用小写字母、数字和短横线。", "Start with a lowercase letter, then lowercase letters, digits and hyphens."), "error")
			} else if taken {
				modelHint(c, providerCopy("已有提供商使用了这个 ID。", "Another provider already uses this ID."), "error")
			}
		}
		namePlaceholder := route
		if namePlaceholder == "" {
			namePlaceholder = configurationLabel("displayName")
		}
		configurationInput(c, m, s, "create/displayName", "displayName", draft["displayName"], false, func(v any) { configurationSet(draft, "displayName", v); save() }, fg, muted, namePlaceholder)
		basePlaceholder := "https://gateway.example/v1"
		if protocol == "anthropic-messages" {
			basePlaceholder = "https://gateway.example"
		}
		configurationInput(c, m, s, "create/baseURL", "baseURL", draft["baseURL"], false, func(v any) { configurationSet(draft, "baseURL", v); save() }, fg, muted, basePlaceholder)
		if baseInvalid {
			modelHint(c, providerCopy("请输入有效的 HTTP 或 HTTPS 地址。", "Enter a valid HTTP or HTTPS address."), "error")
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			modelFieldLabel(c, providerCopy("API 协议", "API protocol"))
			labels := make([]string, len(protocols))
			selected := ""
			for i, id := range protocols {
				labels[i] = configurationProtocolLabel(id)
				if id == protocol {
					selected = labels[i]
				}
			}
			if ui.Select(c, &selected, labels).Label(providerCopy("API 协议", "API protocol")).FillWidth().Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(theme.InterfaceSize(14)).Disabled(disabled).Changed() {
				for i, label := range labels {
					if label == selected {
						draft["api"] = protocols[i]
						save()
					}
				}
			}
		})
		if m.Models != nil && createOnly && m.Models.configurationOnCreate != nil && m.Models.configurationCreateKey != nil {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				modelFieldLabel(c, providerCopy("API 密钥", "API key"))
				apiKeyInput(c, "create", m.Models.configurationCreateKey, providerCopy("输入 API 密钥", "Enter API key"), disabled)
			})
			if keyBad {
				modelHint(c, providerCopy("请输入 API 密钥；若该提供商以其他方式鉴权，可以留空。", "Enter the API key, or leave the field empty if this provider authenticates another way."), "error")
			}
		}
		configurationModels(c, m, s, "create", rows, models, func(v []any) { draft["models"] = v; save() }, nil, fg, muted)
		ui.Row(c).FillWidth().Gap(8).Wrap().Justify(ui.End).Children(func() {
			if modelPill(c, providerCopy("取消", "Cancel"), false, false, s.Saving).Clicked() {
				if createOnly && m.Models != nil && m.Models.configurationOnCancel != nil {
					m.Models.configurationOnCancel()
				} else {
					configurationSetForm(m, key, "{}")
				}
			}
			label := providerCopy("创建提供商", "Create provider")
			if !createOnly {
				label = providerCopy("添加到草稿", "Add to draft")
			}
			create := modelPill(c, label, true, false, !s.ClientReady || s.Loading || disabled || !ready)
			if hint := configurationCreateHint(route, endpoint, rows, taken, keyBad); hint != "" && !ready {
				create.Tooltip(hint)
			}
			if create.Clicked() {
				profile := map[string]any{"api": protocol, "baseURL": endpoint, "models": rows}
				if committed == route {
					profile = configurationObjectValue(providers[route])
					profile["api"], profile["baseURL"], profile["models"] = protocol, endpoint, rows
				}
				if name := configurationText(draft["displayName"]); name != "" {
					profile["displayName"] = name
				}
				providers[route] = profile
				configurationWrite(m, "providers", providers)
				if !createOnly {
					configurationSetForm(m, key, "{}")
				}
				if createOnly && m.Models != nil {
					if m.Models.configurationOnCreate != nil && m.Models.configurationCreateKey != nil && *m.Models.configurationCreateKey != "" {
						m.Models.configurationOnCreate(route, *m.Models.configurationCreateKey)
					} else if m.Models.configurationOnSubmit != nil {
						m.Models.configurationOnSubmit()
					}
				}
			}
		})
	})
}
