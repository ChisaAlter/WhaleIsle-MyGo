package settings

import (
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
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
	CanSave     func() bool
	Ready       func() bool
	OnReload    func()
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
	kHeading(c, i18n.T(i18n.KeySettingsCredTitle))
	kHint(c, i18n.T(i18n.KeySettingsCredHint))
	settingsAlert(c, "credentials:38", d.Err != "", providerCopy("没能读取凭据列表，请重试。", "The credential list could not be read. Try again."), d.Err)
	settingsNotice(c, "credentials:41", d.LastAction != "", func() string { return settingsActionText(d.LastAction) })
	if d.Ready != nil && !d.Ready() {
		if d.Err == "" {
			kHint(c, providerCopy("正在读取凭据…", "Loading credentials…"))
		}
		if d.OnReload != nil && kButton(c, providerCopy("重新读取文件", "Reload file")).AlignSelf(ui.Start).Clicked() {
			d.OnReload()
		}
		return
	}
	if d.HarnessNote != "" {
		kHint(c, d.HarnessNote)
	}
	if d.ShowSecrets != nil {
		kCheckbox(c, d.ShowSecrets, i18n.T(i18n.KeySettingsCredShow), false)
	}
	show := d.ShowSecrets != nil && *d.ShowSecrets
	secretRow(c, i18n.T(i18n.KeySettingsCredAPIKey), d.Fields.APIKey, show, fg)
	desktopRow(c, i18n.T(i18n.KeySettingsCredBaseURL), func() {
		if d.Fields.BaseURL == nil {
			return
		}
		kInput(c, d.Fields.BaseURL, coss.InputOptions{Label: i18n.T(i18n.KeySettingsCredBaseURL)})
	}, fg, muted)
	secretRow(c, i18n.T(i18n.KeySettingsCredGithub), d.Fields.GithubToken, show, fg)
	secretRow(c, i18n.T(i18n.KeySettingsCredRemote), d.Fields.RemoteToken, show, fg)
	secretRow(c, i18n.T(i18n.KeySettingsCredRelay), d.Fields.RemoteRelayToken, show, fg)
	ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
		saveDisabled := d.OnSave == nil || d.CanSave != nil && !d.CanSave()
		if kPrimary(c, i18n.T(i18n.KeySettingsSave), saveDisabled).Clicked() && d.OnSave != nil {
			d.OnSave()
		}
		if d.OnReload != nil && kButton(c, providerCopy("重新读取文件", "Reload file")).Clicked() {
			d.OnReload()
		}
		if d.OnDescribe != nil && kButton(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
			d.OnDescribe()
		}
		if kDanger(c, i18n.T(i18n.KeySettingsCredClearAPIKey), false).Clicked() {
			if d.Fields.APIKey != nil {
				*d.Fields.APIKey = ""
			}
		}
		if kDanger(c, i18n.T(i18n.KeySettingsCredClearSecrets), false).Clicked() {
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
	ui.Column(c).Gap(design.Sp(1)).Children(func() {
		kLabel(c, label, fg)
		if value == nil {
			return
		}
		if show {
			kInput(c, value, coss.InputOptions{Label: label})
			return
		}
		kText(c, maskSecret(*value), design.TextXS, design.WeightNormal, fg)
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
