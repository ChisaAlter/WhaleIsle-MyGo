package settings

import (
	"context"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/api"
	store "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
)

// ModelsAccess is the dedicated Models settings page (session/modelCatalog).
type ModelsAccess struct {
	configurationBinding     *store.NamespaceBinding
	configurationProvider    string
	configurationPath        []string
	configurationCreateOnly  bool
	configurationOnSubmit    func()
	configurationOnCancel    func()
	configurationCreateKey   *string
	configurationOnCreate    func(string, string)
	configurationResetCreate func()
	configurationCreating    *bool
	configurationProbe       *modelProbeState
	ProviderDiscovery        *ProviderDiscoveryAccess
	ProviderProfiles         *ProviderProfilesAccess
	Vision                   *VisionAccess
	// ProviderEnv reports the environment variables a provider namespace reads
	// its API key from, and whether this process has one set: the orange hint on
	// a card that still needs a key. Nil leaves the page saying nothing about the
	// environment, as before the Host is described.
	ProviderEnv func(namespace string) (names []string, present bool)
	Labels      []string
	Selected    *string
	SessionID   string
	Err         string
	LastAction  string
	// OnConfigure is a notification only. Provider editing stays in this page.
	OnConfigure    func(namespace string)
	OnRefresh      func()
	OnApply        func()
	ProviderExtras func(*ui.Context, string, string)
	FooterExtras   func(*ui.Context)
}

func modelsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Models == nil {
		return
	}
	a := m.Models
	owner := ""
	if m.Store != nil {
		owner = m.Store.ProviderPageOwner()
	}
	page := ui.Column(c.Key("models-page/" + owner)).FillWidth().MinWidth(0).Gap(12)
	editing := ui.Local(page, "editing-namespace", func() string { return "" })
	adding := ui.Local(page, "adding-provider", func() bool { return false })
	dismissedSetup := ui.Local(page, "dismissed-setup", func() map[string]bool { return map[string]bool{} })
	// setupOpen is the provider whose card was opened as the first-run setup card. It keeps that
	// place until it closes, even after the key it asked for is saved: an editor that moved to a
	// row card mid-save would lose its state (including the flag that closes it).
	setupOpen := ui.Local(page, "setup-open", func() string { return "" })
	page.Children(func() {
		ui.Text(c, providerCopy("模型", "Models")).FontSize(fonttheme.InterfaceSize(16)).FixedLineHeight(fonttheme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
		ui.Text(c, providerCopy("填入各提供商的 API 密钥即可使用其模型。", "Enter your API keys to use models from the following providers.")).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(muted)
		settingsNotice(c, "models:63", "error", a.Err != "", func() string { return a.Err })
		settingsNotice(c, "models:66", "", a.LastAction != "", func() string { return settingsActionText(a.LastAction) })
		providerProfilesPane(c, a.ProviderProfiles, fg, muted)
		if m.Store != nil {
			m.Store.EnsureProviderDirectory(context.Background())
			state := m.Store.Snapshot()
			_, directoryLoading, directoryError := m.Store.ConfigurableProviders()
			rows, anyUsable, readinessKnown := modelProviderDirectory(m)
			_, catalogLoading, catalogError := m.Store.ModelConfigurationCatalog()
			// A failed load is a toast and a 28 DIP retry button in the page's
			// corner, as in the reference, not a line that pushes the cards down.
			settingsNotice(c, "models-directory", "error", directoryError != "", func() string { return directoryError })
			settingsNotice(c, "models-catalog", "error", catalogError != "", func() string { return catalogError })
			if (directoryError != "" || catalogError != "") && modelsRetryCorner(c, directoryLoading || catalogLoading || state.Loading || state.Saving) {
				if directoryError != "" {
					m.Store.RefreshProviderDirectory(context.Background())
				}
				if catalogError != "" {
					m.Store.RefreshModelConfigurationCatalog(context.Background())
				}
			}
			needsSetup := func(row modelProviderRow) bool {
				return !anyUsable && readinessKnown && row.entry.Provider != "deepseek-account" && len(row.entry.SettingsPath) == 0 && !row.keyConfigured && !(*dismissedSetup)[row.entry.Provider]
			}
			if *editing == "" || *setupOpen != *editing {
				*setupOpen = ""
			}
			if *editing == "" && !*adding {
				for _, row := range rows {
					if needsSetup(row) {
						*editing, *setupOpen = row.entry.Provider, row.entry.Provider
						selectModelCredential(m.ReferencedCredentials, row.entry)
						break
					}
				}
			}
			// The reference lists the providers in one column: 12 DIP below the intro and 8 DIP between cards.
			providerList := func() {
				for _, row := range rows {
					entry, namespace := row.entry, row.namespace
					title := entry.DisplayName
					if entry.Provider == "deepseek-account" {
						title = providerCopy("DeepSeek 账号", "DeepSeek Account")
					}
					open := *editing == entry.Provider
					// The editor card: the title with its route id, then the body.
					editor := func() {
						modelSetupCard(c, "model-setup-"+entry.Provider, func() {
							ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
								ui.Text(c, title).FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(500).TextColor(fg)
								ui.Text(c, entry.Provider).MinWidth(0).SingleLine().Padding(3.4, 0, 0, 0).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
							})
							configured := *a
							configured.configurationProvider = entry.Provider
							configured.configurationPath = entry.SettingsPath
							body := m
							body.Models = &configured
							modelNamespaceEditor(c, body, namespace.NS, func() { (*dismissedSetup)[entry.Provider] = true; *editing = "" })
						})
					}
					providerCard := ui.Column(c.Key("model-provider-" + entry.Provider)).FillWidth().MinWidth(0).Gap(8)
					providerCard.Children(func() {
						if open && *setupOpen == entry.Provider {
							// First run: the setup card IS the provider's presence on the page.
							if entry.Error != nil && *entry.Error != "" {
								modelHint(c, *entry.Error, "error")
							}
							editor()
						} else {
							modelRowCard(c, "model-row-"+entry.Provider, func() {
								ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
									ui.Text(c, title).Grow(1).MinWidth(0).SingleLine().FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).FontWeight(500).TextColor(fg)
									if entry.Declared != nil && *entry.Declared {
										ui.Text(c, providerCopy("自定义", "Custom")).Shrink(0).Padding(0, 4).Radius(4).Border(.667, fonttheme.Color(modelColors(c).BorderL3)).
											FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(fonttheme.Color(modelColors(c).Secondary))
									}
									if row.accountAvailable {
										ui.Text(c, providerCopy("已登录", "Signed in")).FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().Success)
									}
									modelProviderCredentialStatus(c, m, entry, muted)
									if modelPill(c, providerCopy("编辑", "Edit"), false, true, !open && (state.Loading || state.Saving)).Label(providerCopy("编辑 ", "Edit ") + title).Clicked() {
										if open {
											*editing = ""
										} else {
											*adding = false
											*editing = entry.Provider
											selectModelCredential(m.ReferencedCredentials, entry)
											if a.OnConfigure != nil {
												a.OnConfigure(namespace.NS)
											}
										}
									}
									providerRemovalButton(c, a.ProviderProfiles, entry.SettingsNs, entry.Provider, state.Loading || state.Saving)
								})
								if entry.Error != nil && *entry.Error != "" {
									modelHint(c, *entry.Error, "error")
								}
								if open {
									// The reference spaces the card list 12 DIP; the card itself keeps 8.
									ui.Column(c).FillWidth().MinWidth(0).Padding(4, 0, 0, 0).Children(editor)
								}
							})
						}
						if a.ProviderExtras != nil {
							a.ProviderExtras(c, entry.Provider, entry.SettingsNs)
						}
					})
				}
			}
			if len(rows) > 0 {
				ui.Column(c.Key("model-provider-list")).FillWidth().MinWidth(0).Gap(8).Padding(12, 0, 0, 0).Children(providerList)
			}
			if len(rows) == 0 && !state.Loading && !directoryLoading && directoryError == "" {
				ui.Text(c, providerCopy("当前 Host 尚未提供模型供应商配置。", "This Host exposes no provider configuration.")).TextColor(muted)
			}
			modelAddProviderCard(c, m, a, editing, adding, fg, muted)
		} else {
			modelAddProviderCard(c, m, a, editing, adding, fg, muted)
		}
		visionPane(c, a.Vision, fg, muted)
		advanced := ui.Column(c.Key("model-provider-tools")).FillWidth().MinWidth(0).Gap(12)
		open := ui.Local(advanced, "open", func() bool { return false })
		advanced.Children(func() {
			label := providerCopy("供应商密钥与模型探测", "Provider keys and model discovery")
			if modelLinkPill(c, label, false).AlignSelf(ui.Start).Clicked() {
				*open = !*open
			}
			if *open && *editing == "" && !*adding {
				referencedCredentialsPane(c, m.ReferencedCredentials, "", fg, muted)
				providerDiscoveryPane(c, a.ProviderDiscovery, fg, muted)
			}
		})
		if a.FooterExtras != nil {
			a.FooterExtras(c)
		}
	})
}

// modelsRetryCorner is the reference's 28 DIP refresh button in the page's top
// right corner, offered while the provider directory or the model catalog failed
// to load. It is out of the flow, so it moves nothing.
func modelsRetryCorner(c *ui.Context, disabled bool) bool {
	label := providerCopy("重试", "Retry")
	button := ui.Button(c.Key("models-retry"), "").Label(label).Tooltip(label).Absolute().Top(0).Right(0).Size(28, 28).Shrink(0).Padding(0).Radius(14).
		Border(.667, fonttheme.Color(modelColors(c).BorderL3)).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover)
	}
	button.Children(func() {
		ui.Icon(c, bodyIcon("refresh")).Size(16, 16).TextColor(c.Theme().Text).PassThrough()
	})
	return button.Clicked()
}

func modelProviderTitle(namespace string) string {
	switch namespace {
	case "llm-deepseek", "llm-deepseek-api-key":
		return "DeepSeek API"
	case "llm-deepseek-account":
		return providerCopy("DeepSeek 账号", "DeepSeek Account")
	case "llm-pi-ai":
		return providerCopy("自定义供应商", "Custom providers")
	case "subagent-model-selection-settings":
		return providerCopy("子代理模型", "Subagent models")
	default:
		return namespace
	}
}

func selectModelCredential(a *ReferencedCredentialsAccess, entry api.ListConfigurableProvidersResultItem) {
	if a == nil || a.OnSelect == nil || !a.Snapshot.Ready || a.Snapshot.Busy {
		return
	}
	for _, row := range a.Snapshot.Rows {
		if row.Namespace == entry.SettingsNs && row.Provider == entry.Provider {
			a.OnSelect(row.ID)
			return
		}
	}
}

// modelProviderCredentialStatus is the 8 DIP dot after a provider's name: green
// when its key is stored, red when it still needs one.
func modelProviderCredentialStatus(c *ui.Context, m Model, entry api.ListConfigurableProvidersResultItem, muted ui.Color) {
	if m.ReferencedCredentials == nil {
		return
	}
	for _, row := range m.ReferencedCredentials.Snapshot.Rows {
		if row.Namespace != entry.SettingsNs || row.Provider != entry.Provider || !row.Known || row.KeylessNativeAuth() || !row.NamedReference && row.NativeAuth != nil {
			continue
		}
		label, ink := providerCopy("API 密钥缺失", "API key missing"), c.Theme().Danger
		if row.Configured {
			label, ink = providerCopy("API 密钥已配置", "API key configured"), c.Theme().Success
		}
		ui.Box(c).Size(8, 8).Shrink(0).Radius(4).Background(ink).Label(label).Tooltip(label)
		return
	}
}

// modelAddProviderCard is the reference's full-width add row. Providers
// still come from the Host, so without a selection target the card explains
// itself instead of implying a local form.
func modelAddProviderCard(c *ui.Context, m Model, a *ModelsAccess, editing *string, open *bool, fg, muted ui.Color) {
	modelAddProviderEditor(c, m, a, editing, open, fg, muted)
}

func providerPathValue(value any, path []string) (any, bool) {
	for _, key := range path {
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = obj[key]
		if !ok {
			return nil, false
		}
	}
	return value, true
}
func setProviderPath(value map[string]any, path []string) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		value[path[0]] = map[string]any{}
		return
	}
	child, ok := value[path[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		value[path[0]] = child
	}
	setProviderPath(child, path[1:])
}
