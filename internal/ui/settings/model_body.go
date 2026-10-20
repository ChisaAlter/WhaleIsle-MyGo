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
	"whaleisle/internal/ui/widgets"
)

// modelNamespaceBody retains the rich model/provider editors and scopes every
// draft callback to the namespace and Host that supplied the displayed schema.
func modelNamespaceBody(c *ui.Context, m Model, namespace string) bool {
	if m.Store == nil || m.Models == nil {
		return false
	}
	binding := m.Store.BindNamespace(namespace)
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
	container := ui.Column(c.Key("model-namespace/" + binding.Owner())).FillWidth().MinWidth(0).Gap(12)
	inputs := ui.Local(container, "inputs", func() map[string]*string { return map[string]*string{} })
	models := *m.Models
	models.configurationBinding = binding
	m.Models = &models
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
	primary, advanced := splitModelFields(ns.Fields)
	render := func(field store.Field) {
		if configurationField(c, m, snap, ns, field, fg, muted) {
			return
		}
		modelBoundScalar(c, binding, snap, field, fg, muted)
	}
	container.Children(func() {
		if snap.Err != "" {
			ui.Text(c, snap.Err).TextColor(fg)
		}
		if !snap.Writable {
			ui.Text(c, providerCopy("此供应商设置为只读。", "Provider settings are read-only.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		envNames, envPresent := providerEnvNames(m.Models, namespace)
		for _, field := range primary {
			if SecretLikeKey(field.Key) || SecretLikeKey(field.Label) {
				modelAPIKeyField(c, binding, snap, field, envNames, envPresent, fg, muted)
				continue
			}
			render(field)
		}
		if hint := providerEnvHint(m.Models, namespace, binding, primary); hint != "" {
			ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(providerEnvWarnInk)
		}
		if len(advanced) > 0 {
			bodyDisclosure(c, "model-advanced/"+binding.Owner(), i18n.T(i18n.KeySettingsModelsCustomize), func() {
				for _, field := range advanced {
					render(field)
				}
			})
		}
		ui.Row(c).FillWidth().Gap(8).Wrap().Justify(ui.End).Children(func() {
			if widgets.Button(c, colors, i18n.T(i18n.KeySettingsWhaleCancel),
				widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall,
					Disabled: snap.Saving || !snap.Writable}).Clicked() {
				binding.Discard()
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeySettingsSave),
				widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall,
					Disabled: !snap.ClientReady || !snap.Writable || snap.Saving || snap.Loading || !snap.Dirty || configurationInvalid(snap)}).Clicked() {
				binding.Save(context.Background())
			}
		})
	})
	return true
}
func modelBoundScalar(c *ui.Context, binding *store.NamespaceBinding, snap store.ViewState, f store.Field, fg, muted ui.Color) {
	disabled := !snap.ClientReady || !snap.Writable || snap.Saving || snap.Loading
	ui.Column(c.Key("model-scalar/" + f.Key)).FillWidth().MinWidth(0).Gap(4).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Wrap().Children(func() {
			ui.Text(c, f.Label).FontSize(theme.InterfaceSize(13)).TextColor(fg).Grow(1).MinWidth(0)
			if ui.Button(c, providerCopy("继承", "Inherit")).Disabled(disabled).Clicked() {
				binding.Reset(f.Key)
			}
		})
		if f.Description != "" {
			ui.Text(c, f.Description).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		value := snap.Drafts[f.Key]
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
			if ui.Select(c, &value, f.EnumOptions).FillWidth().Label(f.Label).Disabled(disabled).Changed() {
				binding.SetDraft(f.Key, value)
			}
		default:
			if f.Kind == store.FieldJSON || MultilineStringField(f) || strings.Contains(value, "\n") {
				if ui.TextArea(c, &value).Label(f.Label).FillWidth().Lines(3, 12).ReadOnly(disabled).Changed() {
					binding.SetDraft(f.Key, value)
				}
			} else {
				input := ui.TextInput(c, &value).Label(f.Label).FillWidth().ReadOnly(disabled)
				if SecretLikeKey(f.Key) || SecretLikeKey(f.Label) {
					input.Password()
				}
				if input.Changed() {
					binding.SetDraft(f.Key, value)
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
		if len(primary) == 0 && (SecretLikeKey(field.Key) || SecretLikeKey(field.Label)) {
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
	return fmt.Sprintf(i18n.T(i18n.KeySettingsModelsEnvHint), strings.Join(names, " / "))
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
		ui.Text(c, i18n.T(i18n.KeySettingsModelsAPIKey)).FontSize(theme.InterfaceSize(13)).TextColor(fg)
		input := ui.TextInput(c, &value).Label(i18n.T(i18n.KeySettingsModelsAPIKey)).FillWidth().ReadOnly(disabled).Password()
		if len(envNames) > 0 && !envPresent {
			input.Placeholder(fmt.Sprintf(i18n.T(i18n.KeySettingsModelsAPIKeyPlaceholder), strings.Join(envNames, " / ")))
		}
		if input.Changed() {
			binding.SetDraft(field.Key, value)
		}
	})
}
