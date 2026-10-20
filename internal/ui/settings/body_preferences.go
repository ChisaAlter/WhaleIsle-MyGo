package settings

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

// BodySettingsAccess projects described, non-secret feature preferences into
// their original settings rows. The application retains the Host write owner.
type BodySettingsAccess struct {
	Owner    uint64
	Snapshot store.ViewState
	OnChange func(namespace, key, value string)
	OnPatch  func(namespace string, values map[string]string)
}

func bodyHeading(c *ui.Context, title, hint string, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg)
		if hint != "" {
			ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
	})
}

// A Setting-Cell is a flat, hairline-separated title/description and compact
// control. Measured content width controls stacking, including inside a modal.
func bodyPreferenceRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, controls func()) ui.Element {
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	width := row.Bounds().W
	row.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.W != width {
			c.Invalidate()
		}
	})
	row.Children(func() {
		content := func() {
			copy := ui.Column(c).MinWidth(0).Gap(4)
			if width >= 420 {
				copy.Grow(1)
			} else {
				copy.FillWidth()
			}
			copy.Children(func() {
				ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).TextColor(fg)
				if hint != "" {
					ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
			})
			if controls != nil {
				control := ui.Column(c).MinWidth(0).Gap(6)
				if width >= 420 {
					control.Width(180).Shrink(0)
				} else {
					control.FillWidth()
				}
				control.Children(controls)
			}
		}
		if width >= 420 {
			ui.Row(c).FillWidth().MinWidth(0).Gap(24).Padding(16, 0).AlignItems(ui.Center).Children(content)
		} else {
			ui.Column(c).FillWidth().MinWidth(0).Gap(10).Padding(14, 0).Children(content)
		}
		ui.Box(c).FillWidth().Height(.5).Background(ui.RGBA(muted.R, muted.G, muted.B, .25))
	})
	return row
}

// bodyDisclosure folds a secondary group behind one row. The chevron and the
// hover fill are what make the row read as openable, so a folded group stays
// discoverable next to the rows that are always visible.
func bodyDisclosure(c *ui.Context, key, title string, body func()) {
	anchor := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(8)
	open := ui.Local(anchor, "open", func() bool { return false })
	muted := c.Theme().TextMuted
	anchor.Children(func() {
		toggle := ui.Button(c, "").Label(title).FillWidth().Height(34).Padding(0, 8, 0, 4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Tooltip(title)
		toggle.Children(func() {
			if *open {
				ui.Icon(c, bodyIcon("down")).Size(12, 12).TextColor(muted).PassThrough()
			} else {
				ui.Icon(c, bodyIcon("right")).Size(12, 12).TextColor(muted).PassThrough()
			}
			ui.Text(c, " "+title).FontSize(theme.InterfaceSize(13)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
		})
		if toggle.Hovered() {
			toggle.Background(c.Theme().SurfaceHover)
		}
		if toggle.Clicked() {
			*open = !*open
		}
		if *open {
			body()
		}
	})
}

func bodyHostState(c *ui.Context, a *BodySettingsAccess, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	if s.Loading {
		ui.Text(c, providerCopy("正在读取设置…", "Loading settings…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if !s.ClientReady {
		ui.Text(c, providerCopy("Host 未连接；保留上次设置。", "Host disconnected; showing retained settings.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	} else if !s.Writable {
		ui.Text(c, providerCopy("此 Host 的设置为只读。", "Settings on this Host are read-only.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if s.Saving {
		ui.Text(c, providerCopy("正在保存…", "Saving…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
	if s.Err != "" {
		ui.Text(c, s.Err).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
}

func bodyHostValue(a *BodySettingsAccess, namespace, key string) (store.Field, string, bool) {
	if a == nil {
		return store.Field{}, "", false
	}
	for _, ns := range a.Snapshot.Namespaces {
		if ns.NS != namespace {
			continue
		}
		for _, field := range ns.Fields {
			if field.Key != key || SecretLikeKey(field.Key) {
				continue
			}
			var values map[string]json.RawMessage
			_ = json.Unmarshal(ns.Value, &values)
			value := strings.TrimSpace(string(values[key]))
			var text string
			if json.Unmarshal(values[key], &text) == nil {
				value = text
			}
			return field, value, true
		}
	}
	return store.Field{}, "", false
}

func bodyHostDisabled(a *BodySettingsAccess) bool {
	return a == nil || a.OnChange == nil || !a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading || a.Snapshot.Saving
}

func bodyHostPreference(c *ui.Context, a *BodySettingsAccess, namespace, key, title, hint string, fg, muted ui.Color) bool {
	field, value, found := bodyHostValue(a, namespace, key)
	if !found {
		return false
	}
	if hint == "" {
		hint = field.Description
	}
	bodyPreferenceRow(c, fmt.Sprintf("preference-%d-%s-%s", a.Owner, namespace, key), title, hint, fg, muted, func() {
		switch field.Kind {
		case store.FieldBool:
			checked := value == "true"
			if ui.Switch(c, &checked).Label(title).Disabled(bodyHostDisabled(a)).Changed() {
				a.OnChange(namespace, key, fmt.Sprint(checked))
			}
		case store.FieldEnum:
			options := field.EnumOptions
			if key == "transcriptView" {
				options = nil
				for _, option := range field.EnumOptions {
					if option != "normal" && option != "expanded" {
						options = append(options, option)
					}
				}
				if value == "normal" || value == "expanded" {
					for _, option := range options {
						if option == "detailed" {
							value = option
							break
						}
					}
				}
			}
			if ui.Select(c, &value, options).Label(title).FillWidth().Disabled(bodyHostDisabled(a) || len(options) == 0).Changed() {
				a.OnChange(namespace, key, value)
			}
		default:
			anchor := ui.Column(c.Key(fmt.Sprintf("draft-%d-%s-%s", a.Owner, namespace, key))).FillWidth().Gap(4)
			type draftState struct{ Value, Base string }
			draft := ui.Local(anchor, "value", func() draftState { return draftState{value, value} })
			if draft.Value == value || draft.Value == draft.Base && value != draft.Base {
				draft.Value, draft.Base = value, value
			}
			commit := func() {
				if !bodyHostDisabled(a) {
					a.OnChange(namespace, key, draft.Value)
				}
			}
			anchor.Children(func() {
				ui.TextInput(c, &draft.Value).Label(title).FillWidth().Disabled(!a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading).OnSubmit(commit)
				if ui.Button(c, providerCopy("保存", "Save")).Disabled(bodyHostDisabled(a) || draft.Value == value).Clicked() {
					commit()
				}
			})
		}
	})
	return true
}

func bodyLocalToggle(c *ui.Context, key, title, hint string, value *bool, disabled bool, fg, muted ui.Color, changed func(bool)) {
	if value == nil {
		return
	}
	bodyPreferenceRow(c, key, title, hint, fg, muted, func() {
		if ui.Switch(c, value).Label(title).Disabled(disabled).Changed() && changed != nil {
			changed(*value)
		}
	})
}
