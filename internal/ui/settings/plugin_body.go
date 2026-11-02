package settings

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// NamespaceBody renders only a served namespace's native fields and Save
// controls. It never selects the settings rail or borrows its selected draft.
// A false result means that namespace is not served by this Host.
func NamespaceBody(c *ui.Context, m Model, namespace string) bool {
	if m.Store == nil {
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
	disabled := !snap.ClientReady || !snap.Writable || snap.Loading || snap.Saving
	ui.Column(c.Key("plugin-namespace-" + namespace)).FillWidth().MinWidth(0).Gap(14).Children(func() {
		if snap.Err != "" {
			ui.Text(c, snap.Err).FillWidth().TextColor(c.Theme().Danger)
		}
		if snap.LastAction != "" {
			ui.Text(c, settingsActionText(snap.LastAction)).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		for _, field := range ns.Fields {
			f := field
			ui.Column(c.Key(f.Key)).FillWidth().MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, f.Label).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(fg)
					if ui.Button(c, providerCopy("重置为继承值", "Reset to inherited")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(12.5)).Disabled(disabled).Clicked() {
						binding.Reset(f.Key)
					}
				})
				if f.Description != "" {
					ui.Text(c, f.Description).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
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
					if ui.Select(c, &value, f.EnumOptions).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(disabled).Changed() {
						binding.SetDraft(f.Key, value)
					}
				case store.FieldJSON:
					if ui.TextArea(c, &value).FillWidth().MinHeight(100).Lines(4, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Label(f.Label).ReadOnly(disabled).Changed() {
						binding.SetDraft(f.Key, value)
					}
				default:
					if f.Kind == store.FieldString && (MultilineStringField(f) || strings.Contains(value, "\n")) {
						if ui.TextArea(c, &value).FillWidth().MinHeight(80).Lines(3, 10).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Label(f.Label).ReadOnly(disabled).Changed() {
							binding.SetDraft(f.Key, value)
						}
					} else {
						input := ui.TextInput(c, &value).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Label(f.Label).ReadOnly(disabled)
						if SecretLikeKey(f.Key) || SecretLikeKey(f.Label) {
							input.Password()
						}
						if input.Changed() {
							binding.SetDraft(f.Key, value)
						}
					}
				}
				if snap.Resets[f.Key] {
					ui.Text(c, providerCopy("保存后将使用继承值。", "Will inherit after Save.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
			})
		}
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			if ui.PrimaryButton(c, providerCopy("保存", "Save")).Height(32).Padding(0, 14).Radius(8).FontSize(theme.InterfaceSize(13)).Disabled(disabled || !snap.Dirty).Clicked() {
				binding.Save(context.Background())
			}
			if ui.Button(c, providerCopy("放弃草稿", "Discard drafts")).Height(32).Padding(0, 14).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(theme.InterfaceSize(13)).Disabled(snap.Saving || !snap.Dirty).Clicked() {
				binding.Discard()
			}
		})
		if !snap.Writable {
			ui.Text(c, providerCopy("此配置为只读。", "This configuration is read-only.")).FillWidth().TextColor(muted)
		}
	})
	return true
}
