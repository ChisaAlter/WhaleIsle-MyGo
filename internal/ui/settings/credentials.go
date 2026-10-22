package settings

import (
	"github.com/egoist/mygo/ui"
	fonttheme "whaleisle/internal/ui/theme"

	"whaleisle/internal/ui/i18n"
)

// CredentialsFields are editable secrets from credentials.json.
type CredentialsFields struct {
	APIKey           *string
	BaseURL          *string
	GithubToken      *string
	RemoteToken      *string
	RemoteRelayToken *string
}

// CredentialsAccess reads/writes credentials.json via main callbacks.
type CredentialsAccess struct {
	Fields      CredentialsFields
	ShowSecrets *bool
	OnSave      func()
	OnDescribe  func()
	HarnessNote string
	LastAction  string
	Err         string
}

func credentialsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Credentials == nil {
		return
	}
	d := m.Credentials
	ui.Text(c, i18n.T(i18n.KeySettingsCredTitle)).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsCredHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if d.Err != "" {
		ui.Text(c, d.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if d.LastAction != "" {
		ui.Text(c, settingsActionText(d.LastAction)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if d.HarnessNote != "" {
		ui.Text(c, d.HarnessNote).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if d.ShowSecrets != nil {
		ui.Checkbox(c, d.ShowSecrets, i18n.T(i18n.KeySettingsCredShow))
	}
	show := d.ShowSecrets != nil && *d.ShowSecrets
	secretRow(c, i18n.T(i18n.KeySettingsCredAPIKey), d.Fields.APIKey, show, fg)
	desktopRow(c, i18n.T(i18n.KeySettingsCredBaseURL), func() {
		if d.Fields.BaseURL == nil {
			return
		}
		ui.TextInput(c, d.Fields.BaseURL).FillWidth()
	}, fg, muted)
	secretRow(c, i18n.T(i18n.KeySettingsCredGithub), d.Fields.GithubToken, show, fg)
	secretRow(c, i18n.T(i18n.KeySettingsCredRemote), d.Fields.RemoteToken, show, fg)
	secretRow(c, i18n.T(i18n.KeySettingsCredRelay), d.Fields.RemoteRelayToken, show, fg)
	ui.Row(c).Gap(8).Children(func() {
		save := ui.Button(c, i18n.T(i18n.KeySettingsSave))
		if d.OnSave == nil {
			save = save.Disabled(true)
		}
		if save.Clicked() && d.OnSave != nil {
			d.OnSave()
		}
		if d.OnDescribe != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			d.OnDescribe()
		}
		if ui.Button(c, i18n.T(i18n.KeySettingsCredClearAPIKey)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			if d.Fields.APIKey != nil {
				*d.Fields.APIKey = ""
			}
		}
		if ui.Button(c, i18n.T(i18n.KeySettingsCredClearSecrets)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			clearSecret := func(p *string) {
				if p != nil {
					*p = ""
				}
			}
			clearSecret(d.Fields.APIKey)
			clearSecret(d.Fields.GithubToken)
			clearSecret(d.Fields.RemoteToken)
			clearSecret(d.Fields.RemoteRelayToken)
		}
	})
}

func secretRow(c *ui.Context, label string, value *string, show bool, fg ui.Color) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		if value == nil {
			return
		}
		if show {
			ui.TextInput(c, value).FillWidth()
			return
		}
		ui.Text(c, maskSecret(*value)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	})
}

func maskSecret(s string) string {
	if s == "" {
		return "(empty)"
	}
	if len(s) <= 4 {
		return "••••"
	}
	return s[:2] + "••••" + s[len(s)-2:]
}
