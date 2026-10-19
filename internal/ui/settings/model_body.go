package settings

import (
	"context"
	"github.com/egoist/mygo/ui"
	"strings"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
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
	container.Children(func() {
		if snap.Err != "" {
			ui.Text(c, snap.Err).TextColor(fg)
		}
		if !snap.Writable {
			ui.Text(c, providerCopy("此供应商设置为只读。", "Provider settings are read-only.")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		for _, field := range ns.Fields {
			if configurationField(c, m, snap, ns, field, fg, muted) {
				continue
			}
			modelBoundScalar(c, binding, snap, field, fg, muted)
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.PrimaryButton(c, providerCopy("保存", "Save")).Disabled(!snap.ClientReady || !snap.Writable || snap.Saving || snap.Loading || !snap.Dirty || configurationInvalid(snap)).Clicked() {
				binding.Save(context.Background())
			}
			if ui.Button(c, providerCopy("放弃草稿", "Discard drafts")).Disabled(snap.Saving || !snap.Dirty).Clicked() {
				binding.Discard()
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
