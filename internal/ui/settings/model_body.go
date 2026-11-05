package settings

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// modelNamespaceBody retains the rich model/provider editors and scopes every
// draft callback to the namespace and Host that supplied the displayed schema.
func modelNamespaceBody(c *ui.Context, m Model, namespace string) bool {
	return modelNamespaceEditor(c, m, namespace, nil)
}

func modelNamespaceEditor(c *ui.Context, m Model, namespace string, onCancel func()) bool {
	if m.Store == nil || m.Models == nil {
		return false
	}
	container := ui.Column(c.Key("model-namespace/" + m.Store.ProviderPageOwner() + "/" + namespace + "/" + m.Models.configurationProvider)).FillWidth().MinWidth(0).Gap(16)
	ownedBinding := ui.Local(container, "provider-draft", func() *store.NamespaceBinding {
		return m.Store.BindProviderEditor(namespace, m.Models.configurationProvider, m.Models.configurationPath)
	})
	binding := *ownedBinding
	if m.Models.configurationBinding != nil {
		binding = m.Models.configurationBinding
	}
	if binding == nil {
		return false
	}
	snap := binding.Snapshot()
	if len(snap.Namespaces) == 0 {
		return false
	}
	ns := snap.Namespaces[0]
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	inputs := ui.Local(container, "inputs", func() map[string]*string { return map[string]*string{} })
	probe := ui.Local(container, "model-probe", func() modelProbeState { return modelProbeState{} })
	if m.Models.configurationProbe != nil {
		probe = m.Models.configurationProbe
	}
	submitted := ui.Local(container, "submitted", func() bool { return false })
	createKey := ui.Local(container, "create-api-key", func() string { return "" })
	if m.Models.configurationCreateKey != nil {
		createKey = m.Models.configurationCreateKey
	}
	creating := ui.Local(container, "creating-provider", func() bool { return false })
	if m.Models.configurationCreating != nil {
		creating = m.Models.configurationCreating
	}
	createUsesKey := ui.Local(container, "create-uses-key", func() bool { return false })
	keyPending, keyBusy, keyError := false, false, ""
	if credentials := m.ReferencedCredentials; credentials != nil {
		if m.Models.configurationCreateOnly && *createUsesKey {
			keyError = credentials.Snapshot.Err
		}
		for _, row := range credentials.Snapshot.Rows {
			if row.ID == credentials.Snapshot.SelectedID && row.Namespace == namespace && row.Provider == m.Models.configurationProvider {
				keyPending = credentials.Snapshot.Draft != "" || credentials.Snapshot.ClearPending
				keyBusy, keyError = credentials.Snapshot.Busy, credentials.Snapshot.Err
				break
			}
		}
	}
	if *submitted && !snap.Saving && !snap.Dirty && snap.Err == "" && !keyPending && !keyBusy && keyError == "" {
		*submitted = false
		if m.Models.configurationCreateOnly {
			binding.SetForm("@provider-create", "{}")
			binding.SetForm("@provider-create-committed", "")
		}
		m.Store.RefreshProviderDirectory(context.Background())
		if onCancel != nil {
			onCancel()
		}
	}
	models := *m.Models
	models.configurationBinding = binding
	models.configurationCreateKey = createKey
	models.configurationProbe = probe
	discardKey := func() {
		credentials := m.ReferencedCredentials
		if credentials == nil || credentials.OnEdit == nil || credentials.Snapshot.Busy {
			return
		}
		for _, row := range credentials.Snapshot.Rows {
			if row.ID == credentials.Snapshot.SelectedID && row.Namespace == namespace && (row.Provider == m.Models.configurationProvider || m.Models.configurationCreateOnly && row.Provider == snap.Drafts["@provider-create-committed"]) {
				credentials.OnEdit(credentials.Snapshot.Ticket, "")
				return
			}
		}
	}
	models.configurationOnCancel = func() {
		*submitted = false
		*createKey = ""
		*createUsesKey = false
		binding.Discard()
		probe.cancel()
		discardKey()
		if onCancel != nil {
			onCancel()
		}
	}
	models.configurationResetCreate = func() {
		*createKey, *createUsesKey, *submitted = "", false, false
		discardKey()
		probe.cancel()
	}
	if credentials := m.ReferencedCredentials; credentials != nil && credentials.OnCreateProvider != nil {
		models.configurationOnCreate = func(provider, key string) {
			*creating = true
			*createUsesKey = true
			credentials.OnCreateProvider(binding, provider, key, func(accepted, saved bool) {
				*creating = false
				if accepted {
					*createKey = ""
				}
				*submitted = saved
			})
		}
	}
	models.configurationOnSubmit = func() {
		*submitted = true
		if credentials := m.ReferencedCredentials; keyPending && credentials != nil && credentials.OnSaveProvider != nil {
			credentials.OnSaveProvider(credentials.Snapshot.Ticket, binding)
		} else {
			binding.Save(context.Background())
		}
	}
	m.Models = &models
	snap.Saving = snap.Saving || *creating || keyBusy || probe.busy
	m.OnDraft = binding.SetDraft
	m.DraftPtr = func(key, seed string) *string {
		p := (*inputs)[key]
		if p == nil {
			p = new(string)
			(*inputs)[key] = p
		}
		*p = seed
		return p
	}
	// The card leads with the credential field; everything else lives behind
	// the reference's "自定义设置" disclosure.
	fields := []store.Field{}
	for _, field := range ns.Fields {
		if m.Models.configurationProvider != "" && field.Key != "models" && field.Key != "modelOverrides" && field.Key != "providers" && field.Key != "baseURL" {
			continue
		}
		if m.Models.configurationProvider != "" && (field.Role == "credential-ref" || field.Key == "apiKeyEnv") {
			continue
		}
		if m.Models.configurationCreateOnly && field.Key != "providers" {
			continue
		}
		fields = append(fields, field)
	}
	primary, advanced := splitModelFields(fields)
	if m.Models.configurationProvider != "" {
		primary, advanced = nil, fields
	}
	render := func(field store.Field) {
		if configurationField(c, m, snap, ns, field, fg, muted) {
			return
		}
		modelBoundScalar(c, binding, snap, field, fg, muted)
	}
	container.Children(func() {
		if m.Models.configurationCreateOnly && keyError != "" {
			ui.Text(c, keyError).FillWidth().TextColor(c.Theme().Danger)
			if *createKey == "" && m.ReferencedCredentials != nil && m.ReferencedCredentials.OnRefresh != nil && ui.Button(c, providerCopy("刷新密钥状态", "Refresh key status")).Disabled(snap.Saving).Clicked() {
				m.ReferencedCredentials.OnRefresh()
			}
		}
		fieldOwnsHint := false
		if namespace != "llm-deepseek-account" && !m.Models.configurationCreateOnly {
			credentials := m.ReferencedCredentials
			if credentials != nil && m.Models.configurationProvider != "" {
				filtered := *credentials
				filtered.formOwned, filtered.locked = credentials.OnSaveProvider != nil, snap.Saving
				filtered.envNames, filtered.envPresent = providerEnvNames(m.Models, namespace)
				filtered.envHint, fieldOwnsHint = providerEnvHint(m.Models, namespace, binding, primary), true
				filtered.Snapshot.Rows = nil
				for _, row := range credentials.Snapshot.Rows {
					if row.Namespace == namespace && row.Provider == m.Models.configurationProvider {
						filtered.Snapshot.Rows = append(filtered.Snapshot.Rows, row)
					}
				}
				credentials = &filtered
			}
			referencedCredentialsPane(c, credentials, namespace, fg, muted)
		}
		settingsNotice(c, "model_body:184", "error", snap.Err != "", func() string { return snap.Err })
		if !snap.Writable {
			modelHint(c, providerCopy("此供应商设置为只读。", "Provider settings are read-only."), "")
		}
		envNames, envPresent := providerEnvNames(m.Models, namespace)
		for _, field := range primary {
			if (field.Role != "credential-ref" && field.Key != "apiKeyEnv") && (SecretLikeKey(field.Key) || SecretLikeKey(field.Label)) {
				modelAPIKeyField(c, binding, snap, field, envNames, envPresent, fg, muted)
				continue
			}
			render(field)
		}
		if hint := providerEnvHint(m.Models, namespace, binding, primary); hint != "" && !keyPending && !fieldOwnsHint {
			modelHint(c, hint, "warn")
		}
		if len(advanced) > 0 {
			modelDisclosure(c, "model-advanced/"+binding.Owner(), i18n.T(i18n.KeySettingsModelsCustomize), func() {
				for _, field := range advanced {
					render(field)
				}
			})
		}
		if m.Models.configurationCreateOnly {
			return
		}
		ui.Row(c).FillWidth().Gap(8).Wrap().Justify(ui.End).Children(func() {
			if modelPill(c, i18n.T(i18n.KeySettingsWhaleCancel), false, false, snap.Saving || m.ReferencedCredentials != nil && m.ReferencedCredentials.Snapshot.Busy).Clicked() {
				models.configurationOnCancel()
			}
			// Like the reference, Save is only blocked while the card cannot
			// write (no Host, read-only, busy) or a row is invalid. With nothing
			// edited it simply closes the card.
			if modelPill(c, i18n.T(i18n.KeySettingsSave), true, false, !snap.ClientReady || !snap.Writable || snap.Saving || snap.Loading || keyBusy || configurationInvalid(snap, m.Models.configurationPath)).Clicked() {
				m.Models.configurationOnSubmit()
			}
		})
	})
	return true
}
func modelBoundScalar(c *ui.Context, binding *store.NamespaceBinding, snap store.ViewState, f store.Field, fg, muted ui.Color) {
	f.Label = configurationLabel(f.Label)
	disabled := !snap.ClientReady || !snap.Writable || snap.Saving || snap.Loading
	if f.Key == "baseURL" {
		modelBaseURLField(c, binding, snap, f, disabled)
		return
	}
	ui.Column(c.Key("model-scalar/" + f.Key)).FillWidth().MinWidth(0).Gap(4).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			ui.Text(c, f.Label).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).TextColor(theme.Color(modelColors(c).Secondary)).Grow(1).MinWidth(0)
			if ui.Button(c, providerCopy("继承", "Inherit")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Disabled(disabled).Clicked() {
				binding.Reset(f.Key)
			}
		})
		if f.Description != "" {
			ui.Text(c, f.Description).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		value := snap.Drafts[f.Key]
		placeholder := ""
		if f.Key == "baseURL" {
			display, inherited := binding.ProviderInput([]string{f.Key}, value)
			if inherited || snap.Resets[f.Key] {
				placeholder = value
				value = ""
			} else if display != nil {
				value = fmt.Sprint(display)
			}
		}
		switch f.Kind {
		case store.FieldBool:
			checked := value == "true"
			if ui.Switch(c, &checked).Label(f.Label).Disabled(disabled).Changed() {
				if checked {
					binding.SetDraft(f.Key, "true")
				} else {
					binding.SetDraft(f.Key, "false")
				}
			}
		case store.FieldEnum:
			if ui.Select(c, &value, f.EnumOptions).FillWidth().Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(theme.InterfaceSize(14)).Label(f.Label).Disabled(disabled).Changed() {
				binding.SetDraft(f.Key, value)
			}
		default:
			if f.Kind == store.FieldJSON || MultilineStringField(f) || strings.Contains(value, "\n") {
				if ui.TextArea(c, &value).Label(f.Label).FillWidth().Lines(3, 12).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(theme.InterfaceSize(14)).ReadOnly(disabled).Changed() {
					binding.SetDraft(f.Key, value)
				}
			} else {
				input := ui.TextInput(c, &value).Label(f.Label).Placeholder(placeholder).FillWidth().Height(36).Radius(8).Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(theme.InterfaceSize(14)).ReadOnly(disabled)
				if f.Role != "credential-ref" && f.Key != "apiKeyEnv" && (SecretLikeKey(f.Key) || SecretLikeKey(f.Label)) {
					input.Password()
				}
				if input.Changed() {
					if f.Key == "baseURL" && value == "" {
						binding.Reset(f.Key)
					} else {
						binding.SetDraft(f.Key, value)
					}
				}
			}
		}
	})
}

// providerEnvWarnInk is the reference's orange "no environment variable" hint.
// theme.Colors has no warning token yet, so the amber lives here.
var providerEnvWarnInk = ui.RGB(217, 119, 6)

// splitModelFields keeps the credential field on the card face and moves the
// rest behind the optional-settings disclosure.
func splitModelFields(fields []store.Field) (primary, advanced []store.Field) {
	for i, field := range fields {
		if len(primary) == 0 && ((field.Role != "credential-ref" && field.Key != "apiKeyEnv") && (SecretLikeKey(field.Key) || SecretLikeKey(field.Label))) {
			primary = append(primary, fields[i])
			continue
		}
		advanced = append(advanced, fields[i])
	}
	if len(primary) == 0 && len(advanced) > 0 {
		primary, advanced = advanced[:1], advanced[1:]
	}
	return primary, advanced
}

// providerEnvHint reports the orange line when this provider reads its key from
// environment variables that this process does not have.
func providerEnvHint(models *ModelsAccess, namespace string, binding *store.NamespaceBinding, primary []store.Field) string {
	if models == nil || models.ProviderEnv == nil {
		return ""
	}
	names, present := models.ProviderEnv(namespace)
	if present || len(names) == 0 {
		return ""
	}
	// A typed key would make the warning stale, so only show it while the
	// credential field is still empty.
	snap := binding.Snapshot()
	for _, field := range primary {
		if value := strings.TrimSpace(snap.Drafts[field.Key]); value != "" {
			return ""
		}
	}
	return fmt.Sprintf(i18n.T(i18n.KeySettingsModelsEnvHint), names[0])
}

// providerEnvNames reports the environment variables this provider reads its
// key from and whether one is present (nil access answers nothing).
func providerEnvNames(models *ModelsAccess, namespace string) ([]string, bool) {
	if models == nil || models.ProviderEnv == nil {
		return nil, false
	}
	return models.ProviderEnv(namespace)
}

// modelAPIKeyField is the reference's credential row: a password field whose
// placeholder names the environment variable this process would otherwise use.
func modelAPIKeyField(c *ui.Context, binding *store.NamespaceBinding, snap store.ViewState, field store.Field, envNames []string, envPresent bool, fg, muted ui.Color) {
	disabled := !snap.ClientReady || !snap.Writable || snap.Saving || snap.Loading
	value := snap.Drafts[field.Key]
	ui.Column(c.Key("model-secret/" + field.Key)).FillWidth().MinWidth(0).Gap(4).Children(func() {
		modelFieldLabel(c, i18n.T(i18n.KeySettingsModelsAPIKey))
		placeholder := ""
		if len(envNames) > 0 && !envPresent {
			placeholder = fmt.Sprintf(i18n.T(i18n.KeySettingsModelsAPIKeyPlaceholder), envNames[0])
		}
		if apiKeyInput(c, field.Key, &value, placeholder, disabled) {
			binding.SetDraft(field.Key, value)
		}
	})
}
