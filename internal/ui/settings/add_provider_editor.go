package settings

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// modelAddProviderEditor is the reference's add row: the full-width outlined pill
// while closed, replaced by the add card (mode switch, hint, panel) while open.
// Each visited mode holds its own card-local binding until close/save.
func modelAddProviderEditor(c *ui.Context, m Model, a *ModelsAccess, editing *string, open *bool, fg, muted ui.Color) {
	if m.Store == nil {
		return
	}
	anchor := ui.Column(c.Key("add-provider/" + m.Store.ProviderPageOwner())).FillWidth().MinWidth(0).Gap(12)
	mode := ui.Local(anchor, "mode", func() string { return "catalog" })
	selected := ui.Local(anchor, "catalog-selection", func() string { return "" })
	bindings := ui.Local(anchor, "bindings", func() map[string]*store.NamespaceBinding { return map[string]*store.NamespaceBinding{} })
	key := ui.Local(anchor, "custom-key", func() string { return "" })
	creating := ui.Local(anchor, "creating", func() bool { return false })
	probe := ui.Local(anchor, "probe", func() modelProbeState { return modelProbeState{} })
	closeAll := func() {
		*open = false
		*key = ""
		probe.cancel()
		for _, b := range *bindings {
			b.Discard()
		}
		*bindings = map[string]*store.NamespaceBinding{}
	}
	anchor.Children(func() {
		busy := *creating || probe.busy || a.ProviderProfiles != nil && a.ProviderProfiles.Snapshot.Busy || m.ReferencedCredentials != nil && m.ReferencedCredentials.Snapshot.Busy
		for _, b := range *bindings {
			busy = busy || b.Snapshot().Saving
		}
		if !*open {
			// Another card took over (an Edit opened): the add draft goes with it.
			if len(*bindings) > 0 || *key != "" {
				closeAll()
			}
			if modelAddPill(c, strings.TrimPrefix(i18n.T(i18n.KeySettingsModelsAddProvider), "+ "), busy).Clicked() {
				*open = true
				*editing = ""
				*mode = "catalog"
			}
			return
		}
		modelSetupCard(c, "add-provider-editor", func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				options := []string{providerCopy("第三方模型提供商", "Third-party model provider"), providerCopy("自定义模型 API", "Custom model API")}
				active := 0
				if *mode == "custom" {
					active = 1
				}
				if picked := modelSegmented(c, "add-provider-mode", options, active, busy); picked >= 0 {
					*mode = "catalog"
					if picked == 1 {
						*mode = "custom"
					}
				}
				if *mode == "custom" {
					modelHint(c, providerCopy("连接中转站、自部署服务或其他兼容 OpenAI / Anthropic 协议的接口，需填写 API 地址、协议和模型。", "Connect a relay, a self-hosted server, or any other OpenAI- or Anthropic-compatible endpoint by its base URL, protocol, and models."), "")
				} else {
					modelHint(c, providerCopy("从内置目录中选择 OpenAI、Anthropic、Kimi 等提供商，填入其 API 密钥即可使用。", "Pick OpenAI, Anthropic, Kimi, or another provider from the built-in catalog and enter its API key."), "")
				}
			})
			configured := *a
			configured.configurationCreating = creating
			configured.configurationCreateKey = key
			configured.configurationProbe = probe
			if *mode == "custom" {
				b := (*bindings)["custom"]
				if b == nil {
					b = m.Store.BindProviderEditor("llm-pi-ai", "@custom", nil)
					(*bindings)["custom"] = b
				}
				if b == nil {
					ui.Text(c, providerCopy("Host 未提供自定义供应商配置。", "The Host exposes no custom provider settings.")).TextColor(muted)
					return
				}
				configured.configurationCreateOnly = true
				configured.configurationBinding = b
				body := m
				body.Models = &configured
				modelNamespaceEditor(c, body, "llm-pi-ai", closeAll)
				return
			}
			m.Store.EnsureProviderDirectory(context.Background())
			directory, loading, _ := m.Store.ConfigurableProviders()
			snap := m.Store.Snapshot()
			labels := []string{}
			targets := map[string]int{}
			names := map[string]int{}
			for _, entry := range directory {
				names[entry.DisplayName]++
			}
			for i, entry := range directory {
				for _, ns := range snap.Namespaces {
					if ns.NS != entry.SettingsNs {
						continue
					}
					var value any
					_ = json.Unmarshal(ns.Value, &value)
					label := entry.DisplayName
					if names[label] > 1 {
						label += " (" + entry.Provider + ")"
					}
					_, exists := providerPathValue(value, entry.SettingsPath)
					// Keep the submitting card mounted through its accepted receipt.
					// Its child consumes success and closes both add-mode drafts.
					if !exists || label == *selected && (*bindings)[entry.Provider] != nil {
						labels = append(labels, label)
						targets[label] = i
					}
					break
				}
			}
			if _, ok := targets[*selected]; !ok && len(labels) > 0 {
				*selected = labels[0]
			}
			if len(labels) > 0 {
				ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
					modelFieldLabel(c, providerCopy("提供商", "Provider"))
					ui.Select(c, selected, labels).Label(providerCopy("选择目录供应商", "Choose catalog provider")).FillWidth().Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).Disabled(busy || loading)
				})
			}
			if i, ok := targets[*selected]; ok {
				entry := directory[i]
				b := (*bindings)[entry.Provider]
				if b == nil {
					b = m.Store.BindProviderEditor(entry.SettingsNs, entry.Provider, entry.SettingsPath)
					(*bindings)[entry.Provider] = b
				}
				if b != nil {
					configured.configurationBinding = b
					configured.configurationProvider = entry.Provider
					configured.configurationPath = entry.SettingsPath
					// Materialize only inside this uncommitted card. Host and the main list
					// remain untouched until its own Apply completes.
					bs := b.Snapshot()
					if len(entry.SettingsPath) > 0 {
						top := entry.SettingsPath[0]
						obj := configurationObject(bs.Drafts[top])
						if _, exists := providerPathValue(obj, entry.SettingsPath[1:]); !exists {
							setProviderPath(obj, entry.SettingsPath[1:])
							raw, _ := json.Marshal(obj)
							b.SetDraft(top, string(raw))
						}
					}
					if credentials := m.ReferencedCredentials; credentials != nil {
						for _, row := range credentials.Snapshot.Rows {
							if row.Namespace == entry.SettingsNs && row.Provider == entry.Provider && row.ID != credentials.Snapshot.SelectedID {
								selectModelCredential(credentials, entry)
								break
							}
						}
					}
					body := m
					body.Models = &configured
					modelNamespaceEditor(c, body, entry.SettingsNs, closeAll)
				}
			} else {
				ui.Text(c, providerCopy("没有尚未配置的目录供应商。", "No unconfigured catalog providers.")).TextColor(muted)
				ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
					if modelPill(c, providerCopy("取消", "Cancel"), false, false, busy).Clicked() {
						closeAll()
					}
				})
			}
		})
	})
}
