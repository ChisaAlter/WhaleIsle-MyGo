package settings

import (
	"context"
	"strings"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
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
		settingsAlert(c, "plugin_body:34", snap.Err != "", providerCopy("没能读取插件详情，请重试。", "Plugin details could not be read. Try again."), snap.Err)
		settingsNotice(c, "plugin_body:37", snap.LastAction != "", func() string { return settingsActionText(snap.LastAction) })
		for _, field := range ns.Fields {
			f := field
			ui.Column(c.Key(f.Key)).FillWidth().MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, f.Label).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(fg)
					if kButtonDisabled(c, providerCopy("重置为继承值", "Reset to inherited"), disabled).Clicked() {
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
					if kSelect(c, &value, f.EnumOptions, f.Label, disabled).FillWidth().Changed() {
						binding.SetDraft(f.Key, value)
					}
				case store.FieldJSON:
					if kTextarea(c, &value, coss.TextareaOptions{
						Label:    f.Label,
						ReadOnly: disabled,
						MaxLines: 12,
					}).MinHeight(100).Lines(4, 12).Changed() {
						binding.SetDraft(f.Key, value)
					}
				default:
					if f.Kind == store.FieldString && (MultilineStringField(f) || strings.Contains(value, "\n")) {
						if kTextarea(c, &value, coss.TextareaOptions{
							Label:    f.Label,
							ReadOnly: disabled,
							MaxLines: 10,
						}).MinHeight(80).Lines(3, 10).Changed() {
							binding.SetDraft(f.Key, value)
						}
					} else {
						if kInput(c, &value, coss.InputOptions{
							Label:    f.Label,
							ReadOnly: disabled,
							Password: SecretLikeKey(f.Key) || SecretLikeKey(f.Label),
						}).Changed() {
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
			if kPrimary(c, providerCopy("保存", "Save"), disabled || !snap.Dirty).Clicked() {
				binding.Save(context.Background())
			}
			if kButtonDisabled(c, providerCopy("放弃草稿", "Discard drafts"), snap.Saving || !snap.Dirty).Clicked() {
				binding.Discard()
			}
		})
		if !snap.Writable {
			ui.Text(c, providerCopy("此配置为只读。", "This configuration is read-only.")).FillWidth().TextColor(muted)
		}
	})
	return true
}
